package domain

import (
	"time"
)

type OrderStatus string

const (
	StatusPendingPayment   OrderStatus = "PENDING_PAYMENT"
	StatusPaymentTimeout   OrderStatus = "PAYMENT_TIMEOUT"
	StatusAwaitingShipment OrderStatus = "AWAITING_SHIPMENT"
	StatusShipped          OrderStatus = "SHIPPED"
	StatusDelivered        OrderStatus = "DELIVERED"
	StatusCancelled        OrderStatus = "CANCELLED"
	StatusFailed           OrderStatus = "FAILED"
)

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"` // UUID
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items,omitempty"` // Di-populate saat get order details
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"-"`          // Biasanya tidak perlu di JSON item, sudah ada di Order
	ProductID       string    `json:"product_id"` // UUID
	Quantity        int       `json:"quantity"`
	PriceAtPurchase float64   `json:"price_at_purchase"`
	CreatedAt       time.Time `json:"created_at"`
}

// Untuk request pembuatan order
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
	// Harga bisa diambil dari Product Service saat checkout nanti, atau dikirim client
	// Untuk sekarang, asumsikan client mengirimkan harga, atau kita butuh call ke Product Service
	Price float64 `json:"price" binding:"required,gt=0"` // Harga satuan produk
}

type CreateOrderRequest struct {
	UserID string                   `json:"user_id" binding:"required"` // Didapat dari auth token idealnya
	Items  []CreateOrderItemRequest `json:"items" binding:"required,dive"`
}

// Response setelah order dibuat
type CreateOrderResponse struct {
	Order
}
