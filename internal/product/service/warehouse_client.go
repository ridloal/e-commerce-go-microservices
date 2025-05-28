package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain" // Menggunakan domain dari Warehouse
)

type WarehouseServiceClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewWarehouseServiceClient(baseURL string) *WarehouseServiceClient {
	return &WarehouseServiceClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *WarehouseServiceClient) GetProductStockInfo(ctx context.Context, productID string) (*domain.ProductStockInfo, error) {
	reqURL := fmt.Sprintf("%s/api/v1/stock-info/products/%s", c.BaseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		logger.Error("WarehouseClient.GetProductStockInfo: NewRequest failed", err, nil)
		return nil, fmt.Errorf("failed to create request to warehouse service: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		logger.Error("WarehouseClient.GetProductStockInfo: HTTPClient.Do failed", err, nil)
		return nil, fmt.Errorf("failed to call warehouse service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log body for debugging if needed, but don't return it in the error directly
		logger.Error(fmt.Sprintf("WarehouseClient.GetProductStockInfo: warehouse service returned status %d for product %s", resp.StatusCode, productID), nil, nil)
		return nil, fmt.Errorf("warehouse service returned status: %d", resp.StatusCode)
	}

	var stockInfo domain.ProductStockInfo
	if err := json.NewDecoder(resp.Body).Decode(&stockInfo); err != nil {
		logger.Error("WarehouseClient.GetProductStockInfo: JSON decode failed", err, nil)
		return nil, fmt.Errorf("failed to decode response from warehouse service: %w", err)
	}
	return &stockInfo, nil
}
