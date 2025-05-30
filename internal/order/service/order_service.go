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
	"github.com/robfig/cron/v3"
)

var (
	ErrOrderCreationFailed    = errors.New("order creation failed")
	ErrStockReservationFailed = errors.New("stock reservation failed for one or more items")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error)
	ProcessPaymentTimeouts(ctx context.Context) // Fungsi untuk scheduler
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
