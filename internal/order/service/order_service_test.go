package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	oRepo "github.com/ridloal/e-commerce-go-microservices/internal/order/repository"

	// mocks for order repo
	"github.com/ridloal/e-commerce-go-microservices/internal/order/repository/mocks"
	// mocks for warehouse client used by order service
	whClientOrderMocks "github.com/ridloal/e-commerce-go-microservices/internal/order/service/mocks"
	whDomain "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOrderService_CreateOrder(t *testing.T) {
	mockOrderRepo := new(mocks.MockOrderRepository)
	mockWhClient := new(whClientOrderMocks.MockWarehouseClientForOrder)
	paymentTimeout := 1 * time.Minute
	// NewOrderService tidak menginisialisasi scheduler secara langsung yang mudah di-mock
	// tapi ia memanggil s.initScheduler() yang menggunakan cron.New().
	// Untuk unit test CreateOrder, scheduler tidak terlalu relevan.
	orderServiceInstance := NewOrderService(mockOrderRepo, mockWhClient, paymentTimeout)
	// Hentikan scheduler yang mungkin dimulai oleh NewOrderService agar tidak mengganggu tes lain
	// Anda bisa membuat `orderServiceImpl` memiliki metode `StopScheduler()` atau mengembalikan `*cron.Cron` dari `NewOrderService`
	// Untuk contoh ini, kita asumsikan bisa mengabaikannya jika tidak ada interaksi langsung.
	// Jika `orderServiceImpl` memiliki field scheduler:
	// if osImpl, ok := orderServiceInstance.(*orderServiceImpl); ok {
	//     if osImpl.scheduler != nil {
	//         osImpl.scheduler.Stop()
	//     }
	// }

	ctx := context.TODO()
	createOrderReq := domain.CreateOrderRequest{
		UserID: "user123",
		Items: []domain.CreateOrderItemRequest{
			{ProductID: "prod1", Quantity: 2, Price: 10.0},
			{ProductID: "prod2", Quantity: 1, Price: 25.0},
		},
	}

	t.Run("Successful order creation", func(t *testing.T) {
		mockWhClient.On("ReserveStock", ctx, "prod1", 2).Return(nil).Once()
		mockWhClient.On("ReserveStock", ctx, "prod2", 1).Return(nil).Once()
		mockOrderRepo.On("CreateOrderWithItems", ctx, mock.AnythingOfType("*domain.Order"), mock.AnythingOfType("[]domain.OrderItem")).Return(nil).Once()

		resp, err := orderServiceInstance.CreateOrder(ctx, createOrderReq)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "mock-order-id", resp.ID) // ID dari mock
		assert.Equal(t, domain.StatusPendingPayment, resp.Status)
		assert.Equal(t, (2*10.0)+(1*25.0), resp.TotalAmount)
		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertExpectations(t)
	})

	t.Run("Stock reservation failed for one item, ensure rollback", func(t *testing.T) {
		mockWhClient.On("ReserveStock", ctx, "prod1", 2).Return(nil).Once()                             // Sukses item pertama
		mockWhClient.On("ReserveStock", ctx, "prod2", 1).Return(errors.New("stock unavailable")).Once() // Gagal item kedua

		// Expect ReleaseStock to be called for the successfully reserved item ("prod1")
		// context.Background() digunakan di service untuk rollback
		mockWhClient.On("ReleaseStock", context.Background(), "prod1", 2).Return(nil).Once()

		resp, err := orderServiceInstance.CreateOrder(ctx, createOrderReq)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrStockReservationFailed)
		assert.Contains(t, err.Error(), "prod2") // Error message should mention the failing product
		mockWhClient.AssertExpectations(t)
	})

	t.Run("CreateOrderWithItems fails after stock reservation", func(t *testing.T) {
		mockWhClient.On("ReserveStock", ctx, "prod1", 2).Return(nil).Once()
		mockWhClient.On("ReserveStock", ctx, "prod2", 1).Return(nil).Once()
		repoErr := errors.New("db transaction error")
		mockOrderRepo.On("CreateOrderWithItems", ctx, mock.AnythingOfType("*domain.Order"), mock.AnythingOfType("[]domain.OrderItem")).Return(repoErr).Once()

		// Service tidak melakukan rollback eksplisit di sini, mengandalkan payment timeout
		// "Stok yang sudah direservasi akan "tergantung". Mekanisme timeout harus menangani ini."

		resp, err := orderServiceInstance.CreateOrder(ctx, createOrderReq)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrOrderCreationFailed)
		assert.Contains(t, err.Error(), repoErr.Error())
		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertExpectations(t)
	})
}

func TestOrderService_ConfirmPayment(t *testing.T) {
	mockOrderRepo := new(mocks.MockOrderRepository)
	mockWhClient := new(whClientOrderMocks.MockWarehouseClientForOrder)
	orderServiceInstance := NewOrderService(mockOrderRepo, mockWhClient, 1*time.Minute)
	// if osImpl, ok := orderServiceInstance.(*orderServiceImpl); ok && osImpl.scheduler != nil { osImpl.scheduler.Stop() }

	ctx := context.TODO()
	orderID := "order-confirm-123"
	mockPendingOrder := &domain.Order{
		ID:          orderID,
		UserID:      "user1",
		Status:      domain.StatusPendingPayment,
		TotalAmount: 50.0,
	}
	mockOrderItems := []domain.OrderItem{
		{ID: "item1", ProductID: "prodA", Quantity: 1, PriceAtPurchase: 20.0},
		{ID: "item2", ProductID: "prodB", Quantity: 2, PriceAtPurchase: 15.0},
	}
	mockReservations := []whDomain.ProductWarehouseReservationInfo{
		{ProductID: "prodA", WarehouseID: "wh1", Reserved: 1},
		{ProductID: "prodB", WarehouseID: "wh1", Reserved: 2},
	}

	t.Run("Successful payment confirmation", func(t *testing.T) {
		mockOrderRepo.On("GetOrderByID", ctx, orderID).Return(mockPendingOrder, nil).Once()
		mockOrderRepo.On("GetOrderItemsByOrderID", ctx, orderID).Return(mockOrderItems, nil).Once()
		mockWhClient.On("FindWarehousesWithReservations", ctx, []string{"prodA", "prodB"}).Return(mockReservations, nil).Once()
		// Deduct stock for each item from its reserved warehouse
		deductReqA := whDomain.DeductStockRequest{ProductID: "prodA", Quantity: 1, WarehouseID: "wh1"}
		deductReqB := whDomain.DeductStockRequest{ProductID: "prodB", Quantity: 2, WarehouseID: "wh1"}
		mockWhClient.On("DeductStock", ctx, deductReqA).Return(nil).Once()
		mockWhClient.On("DeductStock", ctx, deductReqB).Return(nil).Once()
		mockOrderRepo.On("UpdateOrderStatus", ctx, orderID, domain.StatusPaymentConfirmed).Return(nil).Once()

		order, err := orderServiceInstance.ConfirmPayment(ctx, orderID)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, domain.StatusPaymentConfirmed, order.Status)
		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertExpectations(t)
	})

	t.Run("Order not found", func(t *testing.T) {
		mockOrderRepo.On("GetOrderByID", ctx, orderID).Return(nil, oRepo.ErrOrderNotFound).Once()

		order, err := orderServiceInstance.ConfirmPayment(ctx, orderID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, ErrOrderCannotBeConfirmed)
		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertNotCalled(t, "FindWarehousesWithReservations")
		mockWhClient.AssertNotCalled(t, "DeductStock")
		mockOrderRepo.AssertNotCalled(t, "UpdateOrderStatus")
	})

	t.Run("Order not in PENDING_PAYMENT status", func(t *testing.T) {
		shippedOrder := &domain.Order{ID: orderID, Status: domain.StatusShipped}
		mockOrderRepo.On("GetOrderByID", ctx, orderID).Return(shippedOrder, nil).Once()

		order, err := orderServiceInstance.ConfirmPayment(ctx, orderID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, ErrOrderCannotBeConfirmed)
		mockOrderRepo.AssertExpectations(t)
	})
}

func TestOrderService_ProcessPaymentTimeouts(t *testing.T) {
	mockOrderRepo := new(mocks.MockOrderRepository)
	mockWhClient := new(whClientOrderMocks.MockWarehouseClientForOrder)
	timeoutDuration := 30 * time.Minute
	orderServiceInstance := NewOrderService(mockOrderRepo, mockWhClient, timeoutDuration)
	// if osImpl, ok := orderServiceInstance.(*orderServiceImpl); ok && osImpl.scheduler != nil { osImpl.scheduler.Stop() }

	ctx := context.Background() // Sesuai penggunaan di service
	pendingOrder1 := domain.Order{ID: "timeout1", UserID: "userA", Status: domain.StatusPendingPayment, CreatedAt: time.Now().Add(-timeoutDuration * 2)}
	itemsForOrder1 := []domain.OrderItem{
		{ProductID: "prodX", Quantity: 1},
		{ProductID: "prodY", Quantity: 2},
	}

	t.Run("Successfully process one timed-out order", func(t *testing.T) {
		mockOrderRepo.On("GetPendingOrdersOlderThan", ctx, timeoutDuration).Return([]domain.Order{pendingOrder1}, nil).Once()
		mockOrderRepo.On("GetOrderItemsByOrderID", ctx, pendingOrder1.ID).Return(itemsForOrder1, nil).Once()
		mockWhClient.On("ReleaseStock", ctx, "prodX", 1).Return(nil).Once()
		mockWhClient.On("ReleaseStock", ctx, "prodY", 2).Return(nil).Once()
		mockOrderRepo.On("UpdateOrderStatus", ctx, pendingOrder1.ID, domain.StatusPaymentTimeout).Return(nil).Once()

		orderServiceInstance.ProcessPaymentTimeouts(ctx) // Ini void method

		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertExpectations(t)
	})

	t.Run("No orders past payment timeout", func(t *testing.T) {
		mockOrderRepo.On("GetPendingOrdersOlderThan", ctx, timeoutDuration).Return([]domain.Order{}, nil).Once()

		orderServiceInstance.ProcessPaymentTimeouts(ctx)

		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertNotCalled(t, "ReleaseStock")
		mockOrderRepo.AssertNotCalled(t, "GetOrderItemsByOrderID")
		mockOrderRepo.AssertNotCalled(t, "UpdateOrderStatus")
	})

	t.Run("Failed to release stock for an item", func(t *testing.T) {
		mockOrderRepo.On("GetPendingOrdersOlderThan", ctx, timeoutDuration).Return([]domain.Order{pendingOrder1}, nil).Once()
		mockOrderRepo.On("GetOrderItemsByOrderID", ctx, pendingOrder1.ID).Return(itemsForOrder1, nil).Once()
		mockWhClient.On("ReleaseStock", ctx, "prodX", 1).Return(errors.New("warehouse client error")).Once() // Gagal rilis prodX
		mockWhClient.On("ReleaseStock", ctx, "prodY", 2).Return(nil).Once()                                  // prodY tetap dirilis

		// Order status tetap diupdate menjadi PAYMENT_TIMEOUT
		mockOrderRepo.On("UpdateOrderStatus", ctx, pendingOrder1.ID, domain.StatusPaymentTimeout).Return(nil).Once()

		orderServiceInstance.ProcessPaymentTimeouts(ctx)

		mockOrderRepo.AssertExpectations(t)
		mockWhClient.AssertExpectations(t)
	})
}
