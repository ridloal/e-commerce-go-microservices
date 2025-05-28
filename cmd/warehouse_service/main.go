package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/config"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/database"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	warehouseAPI "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/api"
	warehouseRepo "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
	warehouseService "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/service"
)

func main() {
	// Load Config
	dbCfg := config.LoadWarehouseDBConfig()
	serverCfg := config.LoadServerConfig("8083") // Warehouse service default port 8083

	// Setup Logger
	logger.Info("Starting Warehouse Service...")

	// Setup Database
	db, err := database.Connect(dbCfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database for Warehouse Service", err, nil)
		return
	}
	defer db.Close()

	// Setup Dependencies
	whRepository := warehouseRepo.NewPostgresWarehouseRepository(db)
	whService := warehouseService.NewWarehouseService(whRepository)
	whHandler := warehouseAPI.NewWarehouseHandler(whService)

	// Setup Gin Router
	router := gin.Default()

	apiV1 := router.Group("/api/v1")
	whHandler.RegisterRoutes(apiV1) // Pass router, not the group directly to RegisterRoutes

	logger.Info("Warehouse Service running on port " + serverCfg.Port)
	if err := router.Run(serverCfg.Port); err != nil {
		logger.Error("Failed to run Warehouse Service server", err, nil)
	}
}
