package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	// Ganti dengan path yang benar
	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/service"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(os service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: os}
}

func (h *OrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	orderRoutes := router.Group("/orders")
	{
		orderRoutes.POST("", h.CreateOrder)
		orderRoutes.POST("/:order_id/confirm-payment", h.ConfirmPayment)
		// Tambahkan GET /:id, GET /user/:user_id nanti
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req domain.CreateOrderRequest
	// Asumsikan UserID diambil dari JWT token di middleware pada production
	// Untuk sekarang, bisa dikirim di body atau hardcode untuk tes
	// if userID, exists := c.Get("userID"); exists {
	//     req.UserID = userID.(string)
	// } else {
	//     c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
	//     return
	// }

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("CreateOrder Hdl: bad request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Jika UserID tidak dari token, dan harus dari body, pastikan validasi `binding:"required"` ada di domain.CreateOrderRequest.UserID

	resp, err := h.orderService.CreateOrder(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrStockReservationFailed) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()}) // 409 Conflict
			return
		}
		if errors.Is(err, service.ErrOrderCreationFailed) {
			// Ini error internal yang lebih serius jika stok sudah direservasi tapi order gagal disimpan
			logger.Error("CreateOrder Hdl: order creation failed after potential reservation", err, nil)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order creation failed after stock operations"})
			return
		}
		logger.Error("CreateOrder Hdl: unhandled service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *OrderHandler) ConfirmPayment(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Di dunia nyata, di sini ada validasi apakah request ini sah (misal, dari payment gateway callback)

	order, err := h.orderService.ConfirmPayment(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, service.ErrOrderCannotBeConfirmed) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()}) // 409
			return
		}
		// ErrStockDeductionFailed juga bisa di-handle khusus jika perlu respons berbeda
		logger.Error(fmt.Sprintf("Hdl.ConfirmPayment: service error for order %s", orderID), err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm payment for order"})
		return
	}
	c.JSON(http.StatusOK, order)
}
