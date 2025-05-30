package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/config"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/database"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	productAPI "github.com/ridloal/e-commerce-go-microservices/internal/product/api"
	productRepo "github.com/ridloal/e-commerce-go-microservices/internal/product/repository"
	productService "github.com/ridloal/e-commerce-go-microservices/internal/product/service"
)

func main() {
	// Load Config
	dbCfg := config.LoadProductDBConfig()
	serverCfg := config.LoadServerConfig("8082")

	warehouseServiceURL := config.GetEnv("WAREHOUSE_SERVICE_URL", "http://localhost:8083")

	// Setup Logger
	logger.Info("Starting Product Service...")

	// Setup Database
	db, err := database.Connect(dbCfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database for Product Service", err, nil)
		return
	}
	defer db.Close()

	// Setup Dependencies
	whClient := productService.NewWarehouseServiceClient(warehouseServiceURL) // Buat client warehouse
	prodRepository := productRepo.NewPostgresProductRepository(db)
	prodService := productService.NewProductService(prodRepository, whClient) // Inject client
	productHandler := productAPI.NewProductHandler(prodService)

	// Setup Gin Router
	router := gin.Default()
	router.RedirectTrailingSlash = false

	apiV1 := router.Group("/api/v1")
	productHandler.RegisterRoutes(apiV1)

	logger.Info("Product Service running on port " + serverCfg.Port)
	logger.Info("Product Service connecting to Warehouse Service at " + warehouseServiceURL)
	if err := router.Run(serverCfg.Port); err != nil {
		logger.Error("Failed to run Product Service server", err, nil)
	}
}
