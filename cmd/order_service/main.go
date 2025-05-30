package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/api"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/repository"
	"github.com/ridloal/e-commerce-go-microservices/internal/order/service"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/config"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/database"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
)

func main() {
	// Load Config
	dbCfg := config.LoadOrderDBConfig()
	serverCfg := config.LoadServerConfig("8084") // Order service default port 8084
	warehouseServiceURL := config.GetEnv("WAREHOUSE_SERVICE_URL", "http://localhost:8083")

	logger.Info("Starting Order Service...")

	// Setup Database
	db, err := database.Connect(dbCfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database for Order Service", err, nil)
		return
	}
	defer db.Close()

	paymentTimeoutStr := config.GetEnv("PAYMENT_TIMEOUT_MINUTES", "2") // Default 1 menit untuk testing
	paymentTimeoutMinutes, err := time.ParseDuration(paymentTimeoutStr + "m")
	if err != nil {
		logger.Error("Invalid PAYMENT_TIMEOUT_MINUTES. Defaulting to 1 minute.", err, nil)
		paymentTimeoutMinutes = 1 * time.Minute
	}

	// Setup Dependencies
	orderRepository := repository.NewPostgresOrderRepository(db)
	warehouseClient := service.NewHTTPWarehouseClient(warehouseServiceURL) // Client ke Warehouse Service
	ordService := service.NewOrderService(orderRepository, warehouseClient, paymentTimeoutMinutes)
	orderHandler := api.NewOrderHandler(ordService)

	// Setup Gin Router
	router := gin.Default()
	apiV1 := router.Group("/api/v1")
	orderHandler.RegisterRoutes(apiV1)

	logger.Info("Order Service running on port " + serverCfg.Port)
	logger.Info("Order Service connecting to Warehouse Service at " + warehouseServiceURL)
	if errSrv := router.Run(serverCfg.Port); errSrv != nil {
		logger.Error("Failed to run Order Service server", errSrv, nil)
	}
}
