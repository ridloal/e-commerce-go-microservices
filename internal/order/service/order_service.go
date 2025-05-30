package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	// Ganti dengan path yang benar
	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	warehouseDomain "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/robfig/cron/v3"
)

var (
	ErrOrderCreationFailed    = errors.New("order creation failed")
	ErrStockReservationFailed = errors.New("stock reservation failed for one or more items")
	ErrOrderCannotBeConfirmed = errors.New("order cannot be confirmed, invalid current status or order not found")
	ErrStockDeductionFailed   = errors.New("stock deduction failed for one or more items")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error)
	ProcessPaymentTimeouts(ctx context.Context) // Fungsi untuk scheduler
	ConfirmPayment(ctx context.Context, orderID string) (*domain.Order, error)
}

type orderServiceImpl struct {
	orderRepo              repository.OrderRepository
	warehouseClient        WarehouseClient
	scheduler              *cron.Cron
	paymentTimeoutDuration time.Duration
}

func NewOrderService(or repository.OrderRepository, wc WarehouseClient, paymentTimeout time.Duration) OrderService {
	s := &orderServiceImpl{
		orderRepo:              or,
		warehouseClient:        wc,
		scheduler:              cron.New(cron.WithSeconds()), // Menggunakan opsi WithSeconds() jika perlu granularitas detik
		paymentTimeoutDuration: paymentTimeout,
	}
	s.initScheduler()
	return s
}

func (s *orderServiceImpl) initScheduler() {
	spec := "*/30 * * * * *" // Setiap 30 detik untuk testing
	s.scheduler.AddFunc(spec, func() {
		logger.Info("Scheduler: Running ProcessPaymentTimeouts job...")
		// Gunakan context.Background() karena ini adalah background job
		s.ProcessPaymentTimeouts(context.Background())
	})
	s.scheduler.Start()
	logger.Info(fmt.Sprintf("Payment timeout scheduler initialized with spec '%s' and timeout duration %v", spec, s.paymentTimeoutDuration))
}

func (s *orderServiceImpl) ProcessPaymentTimeouts(ctx context.Context) {
	logger.Info("Processing payment timeouts...")
	orders, err := s.orderRepo.GetPendingOrdersOlderThan(ctx, s.paymentTimeoutDuration)
	if err != nil {
		logger.Error("ProcessPaymentTimeouts: failed to get pending orders", err, nil)
		return
	}

	if len(orders) == 0 {
		logger.Info("ProcessPaymentTimeouts: No orders found past payment timeout.")
		return
	}

	logger.Info(fmt.Sprintf("ProcessPaymentTimeouts: Found %d orders to process for timeout.", len(orders)))

	for _, order := range orders {
		logger.Info(fmt.Sprintf("Processing timeout for order ID: %s", order.ID))

		// 1. Dapatkan item-item order
		items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("ProcessPaymentTimeouts: Failed to get items for order %s", order.ID), err, nil)
			// Pertimbangkan untuk menandai order ini sebagai FAILED atau perlu investigasi manual
			// s.orderRepo.UpdateOrderStatus(ctx, order.ID, domain.StatusFailed) // contoh
			continue // Lanjut ke order berikutnya
		}

		// 2. Lepaskan stok untuk setiap item
		allReleased := true
		for _, item := range items {
			logger.Info(fmt.Sprintf("Releasing stock for ProductID: %s, Quantity: %d (Order: %s)", item.ProductID, item.Quantity, order.ID))
			err := s.warehouseClient.ReleaseStock(ctx, item.ProductID, item.Quantity)
			if err != nil {
				// Ini masalah jika pelepasan gagal, bisa menyebabkan inkonsistensi
				// Mungkin stok sudah dilepas, atau warehouse service error. Perlu logging detail.
				logger.Error(fmt.Sprintf("CRITICAL: Failed to release stock for ProductID: %s, OrderID: %s during timeout processing", item.ProductID, order.ID), err, nil)
				allReleased = false
				// Jangan hentikan proses update status order, tapi catat ini
			}
		}

		// 3. Update status order menjadi PAYMENT_TIMEOUT
		// Lakukan ini bahkan jika beberapa pelepasan stok gagal, agar tidak diproses lagi.
		// Kegagalan pelepasan stok harus dimonitor dan ditangani secara terpisah jika terjadi.
		err = s.orderRepo.UpdateOrderStatus(ctx, order.ID, domain.StatusPaymentTimeout)
		if err != nil {
			logger.Error(fmt.Sprintf("ProcessPaymentTimeouts: Failed to update order status for %s", order.ID), err, nil)
		} else {
			if allReleased {
				logger.Info(fmt.Sprintf("Order %s marked as PAYMENT_TIMEOUT and stock released.", order.ID))
			} else {
				logger.Warn(fmt.Sprintf("Order %s marked as PAYMENT_TIMEOUT, but some stock items may not have been released successfully. Needs review.", order.ID))
			}
		}
	}
}

func (s *orderServiceImpl) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}

	// 1. (TODO Nantinya): Validasi produk dan dapatkan harga terbaru dari ProductService
	// Untuk sekarang, kita asumsikan harga di request valid.

	// 2. Reservasi stok untuk setiap item via WarehouseService
	// Jika salah satu gagal, seluruh order gagal.
	// Tidak ada rollback otomatis untuk reservasi yang sudah berhasil di item lain dalam tahap ini.
	// Ini akan ditangani oleh mekanisme timeout pembayaran nanti (Tahap 4).
	successfullyReservedItems := []domain.CreateOrderItemRequest{}

	for _, itemReq := range req.Items {
		logger.Info(fmt.Sprintf("Attempting to reserve stock for ProductID: %s, Quantity: %d", itemReq.ProductID, itemReq.Quantity))
		err := s.warehouseClient.ReserveStock(ctx, itemReq.ProductID, itemReq.Quantity)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to reserve stock for ProductID: %s", itemReq.ProductID), err, nil)

			// PENTING: Melepaskan stok yang sudah berhasil direservasi untuk item sebelumnya dalam order ini
			// Ini membuat proses lebih atomik dari perspektif Order Service
			if len(successfullyReservedItems) > 0 {
				logger.Info(fmt.Sprintf("Rolling back reservations for %d successfully reserved items due to failure on ProductID: %s", len(successfullyReservedItems), itemReq.ProductID))
				for _, reservedItem := range successfullyReservedItems {
					// Gunakan context baru atau background context jika ctx asli sudah selesai
					// Ini adalah "best-effort" rollback. Kegagalan di sini kompleks untuk ditangani.
					releaseErr := s.warehouseClient.ReleaseStock(context.Background(), reservedItem.ProductID, reservedItem.Quantity)
					if releaseErr != nil {
						// Log error parah ini, karena bisa menyebabkan inkonsistensi stok
						logger.Error(fmt.Sprintf("CRITICAL: Failed to release previously reserved stock for ProductID: %s after order failure.", reservedItem.ProductID), releaseErr, nil)
					}
				}
			}
			return nil, fmt.Errorf("%w: product_id %s, quantity %d. %v", ErrStockReservationFailed, itemReq.ProductID, itemReq.Quantity, err)
		}
		successfullyReservedItems = append(successfullyReservedItems, itemReq) // Tambahkan ke daftar yang berhasil
		logger.Info(fmt.Sprintf("Successfully reserved stock for ProductID: %s, Quantity: %d", itemReq.ProductID, itemReq.Quantity))
	}

	// 3. Hitung total amount
	var totalAmount float64
	orderItems := make([]domain.OrderItem, len(req.Items))
	for i, itemReq := range req.Items {
		totalAmount += itemReq.Price * float64(itemReq.Quantity)
		orderItems[i] = domain.OrderItem{
			ProductID:       itemReq.ProductID,
			Quantity:        itemReq.Quantity,
			PriceAtPurchase: itemReq.Price,
		}
	}

	// 4. Buat Order di database
	newOrder := &domain.Order{
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Status:      domain.StatusPendingPayment, // Status awal
	}

	err := s.orderRepo.CreateOrderWithItems(ctx, newOrder, orderItems)
	if err != nil {
		logger.Error("CreateOrder: failed to save order to repository", err, nil)
		// Jika penyimpanan order gagal SETELAH stok direservasi, ini adalah masalah.
		// Stok yang sudah direservasi akan "tergantung". Mekanisme timeout harus menangani ini.
		// Untuk sekarang, log error ini dengan serius.
		// Idealnya, kita juga coba release stok di sini, tapi itu menambah kompleksitas.
		return nil, fmt.Errorf("%w: %v", ErrOrderCreationFailed, err)
	}

	return &domain.CreateOrderResponse{Order: *newOrder}, nil
}

func (s *orderServiceImpl) ConfirmPayment(ctx context.Context, orderID string) (*domain.Order, error) {
	// 1. Dapatkan order
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderCannotBeConfirmed
		}
		logger.Error(fmt.Sprintf("ConfirmPayment: failed to get order %s", orderID), err, nil)
		return nil, err
	}

	// 2. Validasi status order (hanya PENDING_PAYMENT yang bisa dikonfirmasi)
	if order.Status != domain.StatusPendingPayment {
		logger.Warn(fmt.Sprintf("ConfirmPayment: attempt to confirm order %s with invalid status %s", orderID, order.Status), nil)
		return nil, ErrOrderCannotBeConfirmed
	}

	// 3. Dapatkan item-item order
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil || len(items) == 0 {
		logger.Error(fmt.Sprintf("ConfirmPayment: failed to get items for order %s or order has no items", orderID), err, nil)
		// Mungkin update status order ke FAILED di sini jika item tidak ada
		return nil, fmt.Errorf("failed to retrieve items for order %s: %w", orderID, err)
	}

	productIDsInOrder := make([]string, 0, len(items))
	itemMapByProductID := make(map[string]domain.OrderItem) // Untuk akses mudah ke quantity per item
	for _, item := range items {
		productIDsInOrder = append(productIDsInOrder, item.ProductID)
		itemMapByProductID[item.ProductID] = item
	}

	// 4. Cari gudang mana saja yang memiliki reservasi untuk produk-produk dalam order ini
	warehouseReservations, err := s.warehouseClient.FindWarehousesWithReservations(ctx, productIDsInOrder)
	if err != nil {
		logger.Error(fmt.Sprintf("ConfirmPayment: Failed to find warehouses with reservations for order %s", orderID), err, nil)
		// Ini bukan error fatal untuk status order, tapi stock deduction akan gagal.
		// Kita bisa lanjutkan update status order dan catat error ini untuk investigasi.
		// Alternatif: GAGALKAN konfirmasi jika ini tidak bisa didapatkan? Tergantung kebutuhan bisnis.
		// Untuk simulasi, kita log dan lanjutkan, tapi tandai bahwa deduction mungkin tidak terjadi.
	}

	logger.Info(fmt.Sprintf("ConfirmPayment: Found %d potential warehouse locations with reservations for products in order %s", len(warehouseReservations), orderID))

	// 5. Iterasi per item order, lalu iterasi per gudang yang punya reservasi untuk item tsb,
	//    dan coba kurangi stok.
	allDeductionsSuccessful := true
	for _, item := range items {
		quantityToDeductForItem := item.Quantity // Total yang perlu dikurangi untuk item ini
		if quantityToDeductForItem <= 0 {
			continue
		} // Lewati jika sudah 0 (seharusnya tidak terjadi)

		successfullyDeductedForThisItem := 0

		// Cari semua gudang yang punya reservasi untuk item.ProductID ini
		for _, whReservation := range warehouseReservations {
			if whReservation.ProductID == item.ProductID && whReservation.Reserved > 0 {
				if quantityToDeductForItem <= 0 {
					break
				} // Sudah cukup dikurangi untuk item ini

				// Jumlah yang bisa dikurangi dari gudang ini adalah min(yang perlu dikurangi, yang direservasi di gudang ini)
				var amountToDeductFromThisWarehouse int
				if quantityToDeductForItem <= whReservation.Reserved {
					amountToDeductFromThisWarehouse = quantityToDeductForItem
				} else {
					amountToDeductFromThisWarehouse = whReservation.Reserved
				}

				if amountToDeductFromThisWarehouse > 0 {
					logger.Info(fmt.Sprintf("ConfirmPayment: Attempting to deduct %d of ProductID %s from WarehouseID %s for OrderID %s",
						amountToDeductFromThisWarehouse, item.ProductID, whReservation.WarehouseID, orderID))

					deductReq := warehouseDomain.DeductStockRequest{
						ProductID:   item.ProductID,
						Quantity:    amountToDeductFromThisWarehouse,
						WarehouseID: whReservation.WarehouseID,
					}
					err := s.warehouseClient.DeductStock(ctx, deductReq) // Memanggil client
					if err != nil {
						logger.Error(fmt.Sprintf("ConfirmPayment: Failed to deduct stock for ProductID %s from WarehouseID %s (Order %s)",
							item.ProductID, whReservation.WarehouseID, orderID), err, nil)
						allDeductionsSuccessful = false
						// PENTING: Lanjutkan ke gudang lain atau item lain. JANGAN return error di sini.
						// Kita ingin mencoba mengurangi sebanyak mungkin.
					} else {
						logger.Info(fmt.Sprintf("ConfirmPayment: Successfully deducted %d of ProductID %s from WarehouseID %s",
							amountToDeductFromThisWarehouse, item.ProductID, whReservation.WarehouseID))
						quantityToDeductForItem -= amountToDeductFromThisWarehouse
						successfullyDeductedForThisItem += amountToDeductFromThisWarehouse
					}
				}
			}
		}
		// Setelah mencoba semua warehouse untuk item ini:
		if quantityToDeductForItem > 0 {
			logger.Error(fmt.Sprintf("ConfirmPayment: Could not fully deduct stock for ProductID %s (Order %s). Needed %d, remaining to deduct %d.",
				item.ProductID, orderID, item.Quantity, quantityToDeductForItem), nil, nil)
			allDeductionsSuccessful = false
		}
	}

	if !allDeductionsSuccessful {
		// Log error/warning tingkat tinggi bahwa tidak semua stok berhasil dikurangi
		logger.Error(fmt.Sprintf("ConfirmPayment: One or more stock deductions failed for OrderID %s. Manual review may be needed.", orderID), nil, nil)
		// Bergantung pada kebijakan bisnis, ini bisa jadi alasan untuk TIDAK mengupdate status ke PAYMENT_CONFIRMED
		// atau memindahkannya ke status khusus "PARTIALLY_ALLOCATED" atau "AWAITING_STOCK_VERIFICATION".
		// UNTUK SIMULASI INI, kita tetap lanjutkan update status, tapi dengan catatan.
	}

	// 6. Update status order menjadi PAYMENT_CONFIRMED
	newStatus := domain.StatusPaymentConfirmed
	err = s.orderRepo.UpdateOrderStatus(ctx, order.ID, newStatus)
	if err != nil {
		logger.Error(fmt.Sprintf("ConfirmPayment: failed to update order status for %s after payment", orderID), err, nil)
		// Pembayaran berhasil tapi status gagal update. Perlu mekanisme retry atau alert.
		return nil, fmt.Errorf("failed to update order status for %s: %w", orderID, err)
	}

	order.Status = newStatus
	order.UpdatedAt = time.Now() // Seharusnya dihandle oleh UpdateOrderStatus di repo
	logger.Info(fmt.Sprintf("Order %s payment confirmed. Status updated to %s.", orderID, newStatus))
	return order, nil
}
