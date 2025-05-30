package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/service"
)

type ProductHandler struct {
	productService service.ProductService
}

func NewProductHandler(ps service.ProductService) *ProductHandler {
	return &ProductHandler{productService: ps}
}

func (h *ProductHandler) RegisterRoutes(router *gin.RouterGroup) {
	productRoutes := router.Group("/products")
	{
		productRoutes.GET("", h.ListProducts)
		productRoutes.GET("/", h.ListProducts)
		productRoutes.GET("/:id", h.GetProduct)
	}
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	products, err := h.productService.ListProducts(c.Request.Context())
	if err != nil {
		logger.Error("ListProducts: service error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID := c.Param("id")
	product, err := h.productService.GetProductDetails(c.Request.Context(), productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Error("GetProduct: service error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		return
	}
	c.JSON(http.StatusOK, product)
}
