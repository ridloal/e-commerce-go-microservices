package mocks

import (
	"context"
	"time"

	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/repository"
	"github.com/stretchr/testify/mock"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrderWithItems(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	args := m.Called(ctx, order, items)
	if order != nil && args.Error(0) == nil {
		order.ID = "mock-order-id"
		order.Status = domain.StatusPendingPayment
		// ...
	}
	return args.Error(0)
}
func (m *MockOrderRepository) GetPendingOrdersOlderThan(ctx context.Context, duration time.Duration) ([]domain.Order, error) {
	args := m.Called(ctx, duration)
	if o := args.Get(0); o != nil {
		return o.([]domain.Order), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, newStatus domain.OrderStatus) error {
	args := m.Called(ctx, orderID, newStatus)
	return args.Error(0)
}
func (m *MockOrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	args := m.Called(ctx, orderID)
	if oi := args.Get(0); oi != nil {
		return oi.([]domain.OrderItem), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockOrderRepository) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	args := m.Called(ctx, orderID)
	if o := args.Get(0); o != nil {
		return o.(*domain.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) BeginTx(ctx context.Context) (repository.DBTX, error) {
	args := m.Called(ctx)
	if tx := args.Get(0); tx != nil {
		return tx.(repository.DBTX), args.Error(1)
	}
	return nil, args.Error(1)
}
