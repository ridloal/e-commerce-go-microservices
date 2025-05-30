package config

import (
	"os"
	"strconv"
)

type ServerConfig struct {
	Port string
}

type DBConfig struct {
	DSN string // Data Source Name
}

// Untuk User Service
func LoadUserDBConfig() DBConfig {
	// DSN: "postgres://username:password@host:port/dbname?sslmode=disable"
	// Database: user_db
	dsn := "postgres://postgres:postgres@127.0.0.1:5432/user_db?sslmode=disable"
	if envDSN := os.Getenv("USER_DB_DSN"); envDSN != "" {
		dsn = envDSN
	}
	return DBConfig{DSN: dsn}
}

// Untuk Product Service
func LoadProductDBConfig() DBConfig {
	// Database: product_db
	dsn := "postgres://postgres:postgres@127.0.0.1:5432/product_db?sslmode=disable"
	if envDSN := os.Getenv("PRODUCT_DB_DSN"); envDSN != "" {
		dsn = envDSN
	}
	return DBConfig{DSN: dsn}
}

// Untuk Warehouse Service
func LoadWarehouseDBConfig() DBConfig {
	// Database: warehouse_db
	dsn := "postgres://postgres:postgres@127.0.0.1:5432/warehouse_db?sslmode=disable"
	if envDSN := os.Getenv("WAREHOUSE_DB_DSN"); envDSN != "" {
		dsn = envDSN
	}
	return DBConfig{DSN: dsn}
}

func LoadServerConfig(defaultPort string) ServerConfig {
	port := defaultPort
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		port = envPort
	}
	return ServerConfig{Port: ":" + port}
}

func LoadOrderDBConfig() DBConfig {
	dsn := "postgres://postgres:postgres@127.0.0.1:5432/order_db?sslmode=disable"
	if envDSN := os.Getenv("ORDER_DB_DSN"); envDSN != "" {
		dsn = envDSN
	}
	return DBConfig{DSN: dsn}
}

// Helper untuk mendapatkan port Environment Variable jika ada, atau default
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvAsInt(key string, fallback int) int {
	strValue := GetEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}

type ServiceEndpoint struct {
	Name string
	URL  string
	// Bisa ditambahkan prefix path jika layanan di-host dengan prefix
}

type GatewayConfig struct {
	ListenPort          string
	UserServiceURL      string
	ProductServiceURL   string
	WarehouseServiceURL string
	OrderServiceURL     string
}

func LoadGatewayConfig() GatewayConfig {
	return GatewayConfig{
		ListenPort:          GetEnv("API_GATEWAY_PORT", "8080"), // API Gateway akan berjalan di port 8080
		UserServiceURL:      GetEnv("USER_SERVICE_URL", "http://localhost:8081"),
		ProductServiceURL:   GetEnv("PRODUCT_SERVICE_URL", "http://localhost:8082"),
		WarehouseServiceURL: GetEnv("WAREHOUSE_SERVICE_URL", "http://localhost:8083"),
		OrderServiceURL:     GetEnv("ORDER_SERVICE_URL", "http://localhost:8084"),
	}
}
