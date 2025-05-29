package service

import (
	"context"
	"errors"
	"fmt"

	// Ganti dengan path yang benar
	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
)

var (
	ErrOrderCreationFailed    = errors.New("order creation failed")
	ErrStockReservationFailed = errors.New("stock reservation failed for one or more items")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error)
}

type orderServiceImpl struct {
	orderRepo       repository.OrderRepository
	warehouseClient WarehouseClient // Interface dari client warehouse kita
	// productClient ProductClient // Nanti, untuk mengambil harga produk terbaru
}

func NewOrderService(or repository.OrderRepository, wc WarehouseClient) OrderService {
	return &orderServiceImpl{
		orderRepo:       or,
		warehouseClient: wc,
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
