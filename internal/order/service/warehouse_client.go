package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	// Ganti dengan path yang benar
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	warehouseDomain "github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
)

type WarehouseClient interface {
	ReserveStock(ctx context.Context, productID string, quantity int) error
	ReleaseStock(ctx context.Context, productID string, quantity int) error
}

type httpWarehouseClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHTTPWarehouseClient(baseURL string) WarehouseClient {
	return &httpWarehouseClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second, // Timeout mungkin perlu lebih lama untuk operasi stok
		},
	}
}

func (c *httpWarehouseClient) doStockOperation(ctx context.Context, operation string, productID string, quantity int) error {
	reqURL := fmt.Sprintf("%s/api/v1/stocks/%s", c.BaseURL, operation)

	payload := warehouseDomain.StockOperationRequest{
		ProductID: productID,
		Quantity:  quantity,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("WarehouseClient.%sStock: Marshal failed", operation), err, nil)
		return fmt.Errorf("failed to marshal %s stock request: %w", operation, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Error(fmt.Sprintf("WarehouseClient.%sStock: NewRequest failed", operation), err, nil)
		return fmt.Errorf("failed to create %s stock request: %w", operation, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("WarehouseClient.%sStock: HTTPClient.Do failed", operation), err, nil)
		return fmt.Errorf("failed to call warehouse service for %s stock: %w", operation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp warehouseDomain.StockOperationResponse // Atau map[string]string generik
		// Mencoba decode error response, tapi jangan sampai error decode menghalangi error utama
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		errMsg := fmt.Sprintf("warehouse service %s stock returned status %d", operation, resp.StatusCode)
		if errResp.Message != "" {
			errMsg = fmt.Sprintf("%s - %s", errMsg, errResp.Message)
		}
		logger.Error(errMsg, nil, fmt.Sprintf("ProductID: %s, Qty: %d", productID, quantity))
		return fmt.Errorf(errMsg) // Mengembalikan error yang lebih deskriptif
	}

	// Sukses jika status OK
	return nil
}

func (c *httpWarehouseClient) ReserveStock(ctx context.Context, productID string, quantity int) error {
	return c.doStockOperation(ctx, "reserve", productID, quantity)
}

func (c *httpWarehouseClient) ReleaseStock(ctx context.Context, productID string, quantity int) error {
	return c.doStockOperation(ctx, "release", productID, quantity)
}
