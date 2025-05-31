package mocks

import (
	"context"
	// Diperlukan jika DBTX adalah sql.DB atau sql.Tx
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain" // Untuk DBTX
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
	whRepo "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
	"github.com/stretchr/testify/mock"
)

type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	args := m.Called(ctx, warehouse)
	if warehouse != nil && args.Error(0) == nil {
		warehouse.ID = "mock-wh-id"
	}
	return args.Error(0)
}
func (m *MockWarehouseRepository) GetWarehouseByID(ctx context.Context, id string) (*domain.Warehouse, error) {
	args := m.Called(ctx, id)
	if wh := args.Get(0); wh != nil {
		return wh.(*domain.Warehouse), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWarehouseRepository) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	args := m.Called(ctx)
	if whList := args.Get(0); whList != nil {
		return whList.([]domain.Warehouse), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWarehouseRepository) UpdateWarehouseStatus(ctx context.Context, id string, isActive bool) error {
	args := m.Called(ctx, id, isActive)
	return args.Error(0)
}
func (m *MockWarehouseRepository) CreateOrUpdateProductStock(ctx context.Context, stock *domain.ProductStock) error {
	args := m.Called(ctx, stock)
	if stock != nil && args.Error(0) == nil {
		stock.ID = "mock-stock-id"
	}
	return args.Error(0)
}
func (m *MockWarehouseRepository) GetProductStock(ctx context.Context, warehouseID, productID string) (*domain.ProductStock, error) {
	args := m.Called(ctx, warehouseID, productID)
	if ps := args.Get(0); ps != nil {
		return ps.(*domain.ProductStock), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWarehouseRepository) GetTotalAvailableStockByProductID(ctx context.Context, productID string) (int, error) {
	args := m.Called(ctx, productID)
	return args.Int(0), args.Error(1)
}
func (m *MockWarehouseRepository) TransferStock(ctx context.Context, productID, sourceWarehouseID, targetWarehouseID string, quantity int) error {
	args := m.Called(ctx, productID, sourceWarehouseID, targetWarehouseID, quantity)
	return args.Error(0)
}
func (m *MockWarehouseRepository) BeginTx(ctx context.Context) (whRepo.DBTX, error) {
	args := m.Called(ctx)
	if tx := args.Get(0); tx != nil {
		return tx.(whRepo.DBTX), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWarehouseRepository) GetProductStockForUpdate(ctx context.Context, dbops whRepo.DBTX, warehouseID, productID string) (*domain.ProductStock, error) {
	args := m.Called(ctx, dbops, warehouseID, productID)
	if ps := args.Get(0); ps != nil {
		return ps.(*domain.ProductStock), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWarehouseRepository) IncreaseReservedStock(ctx context.Context, dbops whRepo.DBTX, warehouseID, productID string, amount int) error {
	args := m.Called(ctx, dbops, warehouseID, productID, amount)
	return args.Error(0)
}
func (m *MockWarehouseRepository) DecreaseReservedStock(ctx context.Context, dbops whRepo.DBTX, warehouseID, productID string, amount int) error {
	args := m.Called(ctx, dbops, warehouseID, productID, amount)
	return args.Error(0)
}
func (m *MockWarehouseRepository) DeductCommittedStock(ctx context.Context, dbops whRepo.DBTX, warehouseID, productID string, quantityToDeduct int) error {
	args := m.Called(ctx, dbops, warehouseID, productID, quantityToDeduct)
	return args.Error(0)
}
func (m *MockWarehouseRepository) FindWarehousesWithActiveReservations(ctx context.Context, productIDs []string) ([]domain.ProductWarehouseReservationInfo, error) {
	args := m.Called(ctx, productIDs)
	if infos := args.Get(0); infos != nil {
		return infos.([]domain.ProductWarehouseReservationInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

// Method yang hilang
func (m *MockWarehouseRepository) DecreaseProductStockQuantity(ctx context.Context, dbops repository.DBTX, warehouseID, productID string, amount int) error {
	args := m.Called(ctx, dbops, warehouseID, productID, amount)
	return args.Error(0)
}

func (m *MockWarehouseRepository) IncreaseProductStockQuantity(ctx context.Context, dbops repository.DBTX, warehouseID, productID string, amount int) error {
	args := m.Called(ctx, dbops, warehouseID, productID, amount)
	return args.Error(0)
}
