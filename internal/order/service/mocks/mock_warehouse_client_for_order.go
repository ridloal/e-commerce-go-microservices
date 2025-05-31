package mocks

import (
	"context"

	whDomain "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/stretchr/testify/mock"
)

type MockWarehouseClientForOrder struct {
	mock.Mock
}

func (m *MockWarehouseClientForOrder) ReserveStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}
func (m *MockWarehouseClientForOrder) ReleaseStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}
func (m *MockWarehouseClientForOrder) DeductStock(ctx context.Context, req whDomain.DeductStockRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
func (m *MockWarehouseClientForOrder) FindWarehousesWithReservations(ctx context.Context, productIDs []string) ([]whDomain.ProductWarehouseReservationInfo, error) {
	args := m.Called(ctx, productIDs)
	if res := args.Get(0); res != nil {
		return res.([]whDomain.ProductWarehouseReservationInfo), args.Error(1)
	}
	return nil, args.Error(1)
}
