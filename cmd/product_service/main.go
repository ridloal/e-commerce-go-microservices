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
	serverCfg := config.LoadServerConfig("8082") // Product service default port 8082

	// Setup Logger
	logger.Info("Starting Product Service...")

	// Setup Database
	db, err := database.Connect(dbCfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database", err)
		return
	}
	defer db.Close()

	// Setup Dependencies
	prodRepository := productRepo.NewPostgresProductRepository(db)
	prodService := productService.NewProductService(prodRepository)
	productHandler := productAPI.NewProductHandler(prodService)

	// Setup Gin Router
	router := gin.Default()

	apiV1 := router.Group("/api/v1")
	productHandler.RegisterRoutes(apiV1)

	logger.Info("Product Service running on port " + serverCfg.Port)
	if err := router.Run(serverCfg.Port); err != nil {
		logger.Error("Failed to run server", err)
	}
}
