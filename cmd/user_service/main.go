package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/config"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/database"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	userAPI "github.com/ridloal/e-commerce-go-microservices/internal/user/api"
	userRepo "github.com/ridloal/e-commerce-go-microservices/internal/user/repository"
	userService "github.com/ridloal/e-commerce-go-microservices/internal/user/service"
)

func main() {
	// Load Config
	dbCfg := config.LoadUserDBConfig()
	serverCfg := config.LoadServerConfig("8081") // User service default port 8081

	// Setup Logger
	logger.Info("Starting User Service...")

	// Setup Database
	db, err := database.Connect(dbCfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database", err)
		return // or panic
	}
	defer db.Close()

	// Setup Dependencies
	userRepository := userRepo.NewPostgresUserRepository(db)
	usrService := userService.NewUserService(userRepository)
	userHandler := userAPI.NewUserHandler(usrService)

	// Setup Gin Router
	router := gin.Default() // Default with Logger and Recovery middleware

	// Group routes under /api/v1
	apiV1 := router.Group("/api/v1")
	userHandler.RegisterRoutes(apiV1)

	logger.Info("User Service running on port " + serverCfg.Port)
	if err := router.Run(serverCfg.Port); err != nil {
		logger.Error("Failed to run server", err)
	}
}
