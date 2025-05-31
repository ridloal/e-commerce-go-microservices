package mocks

import (
	"context"

	whDomain "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/stretchr/testify/mock"
)

type MockWarehouseServiceClientForProduct struct {
	mock.Mock
}

func (m *MockWarehouseServiceClientForProduct) GetProductStockInfo(ctx context.Context, productID string) (*whDomain.ProductStockInfo, error) {
	args := m.Called(ctx, productID)
	if res := args.Get(0); res != nil {
		return res.(*whDomain.ProductStockInfo), args.Error(1)
	}
	return nil, args.Error(1)
}
