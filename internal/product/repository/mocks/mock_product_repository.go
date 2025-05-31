package mocks

import (
	"context"

	pDomain "github.com/ridloal/e-commerce-go-microservices/internal/product/domain"

	"github.com/stretchr/testify/mock"
)

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) ListProducts(ctx context.Context) ([]pDomain.Product, error) {
	args := m.Called(ctx)
	if res := args.Get(0); res != nil {
		return res.([]pDomain.Product), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, id string) (*pDomain.Product, error) {
	args := m.Called(ctx, id)
	if res := args.Get(0); res != nil {
		return res.(*pDomain.Product), args.Error(1)
	}
	return nil, args.Error(1)
}
