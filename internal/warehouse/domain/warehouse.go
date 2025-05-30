package domain

import (
	"time"
)

type Warehouse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name" binding:"required"`
	Location  *string   `json:"location,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateWarehouseRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location,omitempty"`
}

type ProductStock struct {
	ID               string    `json:"id"`
	WarehouseID      string    `json:"warehouse_id"`
	ProductID        string    `json:"product_id"` // UUID from Product Service
	Quantity         int       `json:"quantity"`
	ReservedQuantity int       `json:"reserved_quantity"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AddStockRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"` // Must be greater than 0
}

// Digunakan untuk Product Service mengambil info stok
type ProductStockInfo struct {
	ProductID      string `json:"product_id"`
	TotalAvailable int    `json:"total_available"`
}

// Untuk update stok internal (reservasi, dll.)
type UpdateStockInternalRequest struct {
	ProductID        string
	WarehouseID      string // Spesifik warehouse jika diperlukan, atau bisa diagregat
	ChangeInQuantity int    // Bisa positif (menambah stok) atau negatif (mengurangi/mereservasi)
	ChangeInReserved int    // Bisa positif (mereservasi) atau negatif (melepas reservasi)
}

type StockOperationRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

// Response bisa sederhana atau mengembalikan status stok terbaru
type StockOperationResponse struct {
	Message   string `json:"message"`
	ProductID string `json:"product_id"`
	// Potentially new available stock, etc. (optional for now)
}

type TransferStockRequest struct {
	ProductID         string `json:"product_id" binding:"required"`
	SourceWarehouseID string `json:"source_warehouse_id" binding:"required"`
	TargetWarehouseID string `json:"target_warehouse_id" binding:"required"`
	Quantity          int    `json:"quantity" binding:"required,gt=0"`
}

type DeductStockRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,gt=0"`
	WarehouseID string `json:"warehouse_id" binding:"required"`
}

type ProductWarehouseReservationInfo struct {
	ProductID   string `json:"product_id"`
	WarehouseID string `json:"warehouse_id"`
	Reserved    int    `json:"reserved"` // Jumlah yang direservasi di gudang ini untuk produk ini
}

// Request untuk mencari gudang dengan reservasi
type FindWarehousesWithReservationsRequest struct {
	ProductIDs []string `json:"product_ids" binding:"required,dive,uuid"`
}
