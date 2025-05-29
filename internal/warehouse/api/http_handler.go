package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/service"
)

type WarehouseHandler struct {
	warehouseService service.WarehouseService
}

func NewWarehouseHandler(ws service.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{warehouseService: ws}
}

func (h *WarehouseHandler) RegisterRoutes(router *gin.RouterGroup) {
	whRoutes := router.Group("/warehouses")
	{
		whRoutes.POST("", h.CreateWarehouse)
		whRoutes.GET("", h.ListWarehouses)
		whRoutes.GET("/:id", h.GetWarehouse)
		whRoutes.PUT("/:id/activate", h.ActivateWarehouse)
		whRoutes.PUT("/:id/deactivate", h.DeactivateWarehouse)

		whRoutes.POST("/:id/stocks", h.AddStock)                       // Add stock to a specific warehouse
		whRoutes.GET("/:id/stocks/:product_id", h.GetStockInWarehouse) // Get stock for a product in a specific warehouse
	}

	stockOpsRoutes := router.Group("/stocks") // Grup baru untuk operasi stok umum
	{
		stockOpsRoutes.POST("/reserve", h.ReserveStock)
		stockOpsRoutes.POST("/release", h.ReleaseStock)
	}

	stockInfoRoutes := router.Group("/stock-info")
	{
		stockInfoRoutes.GET("/products/:product_id", h.GetAggregatedProductStock)
	}

}

func (h *WarehouseHandler) CreateWarehouse(c *gin.Context) {
	var req domain.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	wh, err := h.warehouseService.CreateWarehouse(c.Request.Context(), req)
	if err != nil {
		logger.Error("Hdl.CreateWarehouse: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create warehouse"})
		return
	}
	c.JSON(http.StatusCreated, wh)
}

func (h *WarehouseHandler) ListWarehouses(c *gin.Context) {
	warehouses, err := h.warehouseService.ListWarehouses(c.Request.Context())
	if err != nil {
		logger.Error("Hdl.ListWarehouses: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list warehouses"})
		return
	}
	c.JSON(http.StatusOK, warehouses)
}

func (h *WarehouseHandler) GetWarehouse(c *gin.Context) {
	id := c.Param("id")
	wh, err := h.warehouseService.GetWarehouse(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrWarehouseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Error("Hdl.GetWarehouse: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get warehouse"})
		return
	}
	c.JSON(http.StatusOK, wh)
}

func (h *WarehouseHandler) ActivateWarehouse(c *gin.Context) {
	id := c.Param("id")
	err := h.warehouseService.ActivateWarehouse(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrWarehouseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Error("Hdl.ActivateWarehouse: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate warehouse"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Warehouse activated"})
}

func (h *WarehouseHandler) DeactivateWarehouse(c *gin.Context) {
	id := c.Param("id")
	err := h.warehouseService.DeactivateWarehouse(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrWarehouseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Error("Hdl.DeactivateWarehouse: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate warehouse"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Warehouse deactivated"})
}

func (h *WarehouseHandler) AddStock(c *gin.Context) {
	warehouseID := c.Param("id")
	var req domain.AddStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	stock, err := h.warehouseService.AddProductStock(c.Request.Context(), warehouseID, req)
	if err != nil {
		// Handle specific errors like warehouse not found, product ID format invalid (if adding validation)
		logger.Error("Hdl.AddStock: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add stock: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stock)
}

func (h *WarehouseHandler) GetStockInWarehouse(c *gin.Context) {
	warehouseID := c.Param("id")
	productID := c.Param("product_id")

	stock, err := h.warehouseService.GetProductStockByWarehouse(c.Request.Context(), warehouseID, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductStockNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Error("Hdl.GetStockInWarehouse: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stock"})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *WarehouseHandler) GetAggregatedProductStock(c *gin.Context) {
	productID := c.Param("product_id")
	stockInfo, err := h.warehouseService.GetAggregatedProductStock(c.Request.Context(), productID)
	if err != nil {
		logger.Error("Hdl.GetAggregatedProductStock: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get aggregated stock"})
		return
	}
	c.JSON(http.StatusOK, stockInfo)
}

func (h *WarehouseHandler) ReserveStock(c *gin.Context) {
	var req domain.StockOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err := h.warehouseService.ReserveStock(c.Request.Context(), req.ProductID, req.Quantity)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStock) || errors.Is(err, repository.ErrProductStockNotFound) {
			c.JSON(http.StatusConflict, gin.H{"error": "Failed to reserve stock: " + err.Error()})
			return
		}
		logger.Error("Hdl.ReserveStock: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during stock reservation"})
		return
	}

	c.JSON(http.StatusOK, domain.StockOperationResponse{
		Message:   "Stock reserved successfully",
		ProductID: req.ProductID,
	})
}

func (h *WarehouseHandler) ReleaseStock(c *gin.Context) {
	var req domain.StockOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	err := h.warehouseService.ReleaseStock(c.Request.Context(), req.ProductID, req.Quantity)
	if err != nil {
		// Untuk release, error karena tidak ada yang di-release mungkin bukan critical server error
		if errors.Is(err, repository.ErrProductStockNotFound) || strings.Contains(err.Error(), "could not release full quantity") {
			c.JSON(http.StatusConflict, gin.H{"error": "Failed to release stock: " + err.Error()})
			return
		}
		logger.Error("Hdl.ReleaseStock: service error", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during stock release"})
		return
	}

	c.JSON(http.StatusOK, domain.StockOperationResponse{
		Message:   "Stock released successfully",
		ProductID: req.ProductID,
	})
}
