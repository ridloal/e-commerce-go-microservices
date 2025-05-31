package service

import (
	"context"
	"testing"

	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	whRepo "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWarehouseService_CreateWarehouse(t *testing.T) {
	mockRepo := new(mocks.MockWarehouseRepository)
	service := NewWarehouseService(mockRepo)
	ctx := context.TODO()
	req := domain.CreateWarehouseRequest{Name: "Main WH"}

	t.Run("Successful creation", func(t *testing.T) {
		mockRepo.On("CreateWarehouse", ctx, mock.MatchedBy(func(wh *domain.Warehouse) bool {
			return wh.Name == req.Name
		})).Return(nil).Once()

		wh, err := service.CreateWarehouse(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, wh)
		assert.Equal(t, "Main WH", wh.Name)
		assert.Equal(t, "mock-wh-id", wh.ID) // ID diset oleh mock
		mockRepo.AssertExpectations(t)
	})
}

func TestWarehouseService_ReserveStock(t *testing.T) {
	mockRepo := new(mocks.MockWarehouseRepository)
	service := NewWarehouseService(mockRepo)
	ctx := context.TODO()
	productID := "prod-reserve"
	quantityToReserve := 5

	mockTx := new(mocks.MockDBTX) // Mock untuk transaksi DB

	activeWarehouses := []domain.Warehouse{
		{ID: "wh1", Name: "WH1", IsActive: true},
		{ID: "wh2", Name: "WH2", IsActive: true},
	}
	stockInWh1 := &domain.ProductStock{ID: "stock1", WarehouseID: "wh1", ProductID: productID, Quantity: 10, ReservedQuantity: 2} // Available: 8
	stockInWh2 := &domain.ProductStock{ID: "stock2", WarehouseID: "wh2", ProductID: productID, Quantity: 3, ReservedQuantity: 0}  // Available: 3

	t.Run("Successful reservation from one warehouse", func(t *testing.T) {
		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockRepo.On("ListWarehouses", ctx).Return(activeWarehouses[:1], nil).Once() // Hanya WH1
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh1", productID).Return(stockInWh1, nil).Once()
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh1", productID, quantityToReserve).Return(nil).Once()
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Maybe() // Mungkin tidak dipanggil jika commit berhasil

		err := service.ReserveStock(ctx, productID, quantityToReserve)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("Successful reservation from multiple warehouses", func(t *testing.T) {
		qtyToReserveMore := 7 // WH1 bisa 8, tapi kita hanya butuh 7. WH1 available: 8, WH2 available: 3
		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockRepo.On("ListWarehouses", ctx).Return(activeWarehouses, nil).Once()
		// WH1
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh1", productID).Return(stockInWh1, nil).Once()
		// Misal, logic akan mencoba mengambil semua dari WH1 jika cukup.
		// Jika quantityToReserve = 7, dan WH1 punya 8 available (10-2). Maka 7 akan diambil dari WH1.
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh1", productID, 7).Return(nil).Once()
		// WH2 tidak akan dipanggil karena sudah cukup dari WH1
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Maybe()

		err := service.ReserveStock(ctx, productID, qtyToReserveMore)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)

		// Reset mock calls untuk skenario berikutnya
		mockRepo.ExpectedCalls = nil
		mockRepo.Calls = nil
		mockTx.ExpectedCalls = nil
		mockTx.Calls = nil

		// Skenario: butuh 10, WH1 punya 8, WH2 punya 3.
		// Ambil 8 dari WH1, lalu 2 dari WH2.
		qtyToReserveAcross := 10
		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockRepo.On("ListWarehouses", ctx).Return(activeWarehouses, nil).Once()
		// WH1
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh1", productID).Return(stockInWh1, nil).Once()
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh1", productID, 8).Return(nil).Once() // Ambil semua yang available (8)
		// WH2
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh2", productID).Return(stockInWh2, nil).Once()
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh2", productID, 2).Return(nil).Once() // Ambil sisa (2)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Maybe()

		err = service.ReserveStock(ctx, productID, qtyToReserveAcross)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("Insufficient stock", func(t *testing.T) {
		qtyToReserveTooMuch := 20
		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockRepo.On("ListWarehouses", ctx).Return(activeWarehouses, nil).Once()
		// WH1
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh1", productID).Return(stockInWh1, nil).Once()
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh1", productID, 8).Return(nil).Once() // Ambil 8
		// WH2
		mockRepo.On("GetProductStockForUpdate", ctx, mockTx, "wh2", productID).Return(stockInWh2, nil).Once()
		mockRepo.On("IncreaseReservedStock", ctx, mockTx, "wh2", productID, 3).Return(nil).Once() // Ambil 3
		// Total direservasi 11, tapi butuh 20. remainingToReserve akan > 0.
		mockTx.On("Rollback").Return(nil).Once() // Commit tidak akan dipanggil
		mockTx.On("Commit").Return(nil).Maybe()

		err := service.ReserveStock(ctx, productID, qtyToReserveTooMuch)
		assert.Error(t, err)
		assert.EqualError(t, err, whRepo.ErrInsufficientStock.Error())
		mockRepo.AssertExpectations(t)
	})
}
