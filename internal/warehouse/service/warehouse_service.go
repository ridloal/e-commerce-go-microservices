package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/repository"
)

var (
	ErrStockOperationFailed   = errors.New("stock operation failed")
	ErrNoActiveWarehouseFound = errors.New("no active warehouse found to fulfill stock operation")
)

type WarehouseService interface {
	CreateWarehouse(ctx context.Context, req domain.CreateWarehouseRequest) (*domain.Warehouse, error)
	GetWarehouse(ctx context.Context, id string) (*domain.Warehouse, error)
	ListWarehouses(ctx context.Context) ([]domain.Warehouse, error)
	ActivateWarehouse(ctx context.Context, id string) error
	DeactivateWarehouse(ctx context.Context, id string) error

	AddProductStock(ctx context.Context, warehouseID string, req domain.AddStockRequest) (*domain.ProductStock, error)
	GetProductStockByWarehouse(ctx context.Context, warehouseID, productID string) (*domain.ProductStock, error)
	GetAggregatedProductStock(ctx context.Context, productID string) (*domain.ProductStockInfo, error)
	TransferProductStock(ctx context.Context, req domain.TransferStockRequest) error

	// Internal methods for Order Service (will require transactions)
	ReserveStock(ctx context.Context, productID string, quantityToReserve int) error
	ReleaseStock(ctx context.Context, productID string, quantityToRelease int) error
	DeductStockAfterSale(ctx context.Context, req domain.DeductStockRequest) error
}

type warehouseServiceImpl struct {
	repo repository.WarehouseRepository
}

func NewWarehouseService(repo repository.WarehouseRepository) WarehouseService {
	return &warehouseServiceImpl{repo: repo}
}

// --- Warehouse Management ---
func (s *warehouseServiceImpl) CreateWarehouse(ctx context.Context, req domain.CreateWarehouseRequest) (*domain.Warehouse, error) {
	w := &domain.Warehouse{
		Name:     req.Name,
		Location: req.Location,
	}
	err := s.repo.CreateWarehouse(ctx, w)
	if err != nil {
		logger.Error("Svc.CreateWarehouse: repo error", err, nil)
		return nil, err
	}
	return w, nil
}

func (s *warehouseServiceImpl) GetWarehouse(ctx context.Context, id string) (*domain.Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *warehouseServiceImpl) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	return s.repo.ListWarehouses(ctx)
}

func (s *warehouseServiceImpl) ActivateWarehouse(ctx context.Context, id string) error {
	_, err := s.repo.GetWarehouseByID(ctx, id) // Check if exists
	if err != nil {
		return err
	}
	return s.repo.UpdateWarehouseStatus(ctx, id, true)
}

func (s *warehouseServiceImpl) DeactivateWarehouse(ctx context.Context, id string) error {
	_, err := s.repo.GetWarehouseByID(ctx, id) // Check if exists
	if err != nil {
		return err
	}
	return s.repo.UpdateWarehouseStatus(ctx, id, false)
}

// --- Stock Management ---
func (s *warehouseServiceImpl) AddProductStock(ctx context.Context, warehouseID string, req domain.AddStockRequest) (*domain.ProductStock, error) {
	// Check if warehouse exists and is active (optional check here, or rely on FK constraint)
	// _, err := s.repo.GetWarehouseByID(ctx, warehouseID)
	// if err != nil {
	// 	return nil, err
	// }

	stock := &domain.ProductStock{
		WarehouseID: warehouseID,
		ProductID:   req.ProductID,
		Quantity:    req.Quantity, // This is the amount to ADD
	}
	err := s.repo.CreateOrUpdateProductStock(ctx, stock)
	if err != nil {
		logger.Error("Svc.AddProductStock: repo error", err, nil)
		return nil, err
	}
	return stock, nil
}

func (s *warehouseServiceImpl) GetProductStockByWarehouse(ctx context.Context, warehouseID, productID string) (*domain.ProductStock, error) {
	return s.repo.GetProductStock(ctx, warehouseID, productID)
}

func (s *warehouseServiceImpl) GetAggregatedProductStock(ctx context.Context, productID string) (*domain.ProductStockInfo, error) {
	totalAvailable, err := s.repo.GetTotalAvailableStockByProductID(ctx, productID)
	if err != nil {
		logger.Error("Svc.GetAggregatedProductStock: repo error", err, nil)
		return nil, err
	}
	return &domain.ProductStockInfo{
		ProductID:      productID,
		TotalAvailable: totalAvailable,
	}, nil
}

func (s *warehouseServiceImpl) TransferProductStock(ctx context.Context, req domain.TransferStockRequest) error {
	if req.SourceWarehouseID == req.TargetWarehouseID {
		return errors.New("source and target warehouse IDs cannot be the same for a transfer")
	}
	// Validasi tambahan: cek apakah warehouse sumber dan tujuan ada dan aktif (opsional, repo bisa handle FK)
	// _, err := s.repo.GetWarehouseByID(ctx, req.SourceWarehouseID)
	// if err != nil { return fmt.Errorf("source warehouse not found: %w", err) }
	// _, err = s.repo.GetWarehouseByID(ctx, req.TargetWarehouseID)
	// if err != nil { return fmt.Errorf("target warehouse not found: %w", err) }

	err := s.repo.TransferStock(ctx, req.ProductID, req.SourceWarehouseID, req.TargetWarehouseID, req.Quantity)
	if err != nil {
		logger.Error("Svc.TransferProductStock: repo error", err, map[string]interface{}{
			"product_id": req.ProductID,
			"source_wh":  req.SourceWarehouseID,
			"target_wh":  req.TargetWarehouseID,
			"quantity":   req.Quantity,
		})
		return err
	}
	return nil
}

// ReserveStock attempts to reserve stock for a product across active warehouses.
// This is a simplified version. A more robust one would pick specific warehouses.
// For Tahap 2, we'll assume reservation happens from an aggregated pool,
// and the actual deduction will need to pinpoint warehouses.
// This method should be transactional.
func (s *warehouseServiceImpl) ReserveStock(ctx context.Context, productID string, quantityToReserve int) error {
	if quantityToReserve <= 0 {
		return errors.New("quantity to reserve must be positive")
	}

	// 1. Find active warehouses that MIGHT have the product (or just try them all for simplicity now)
	// This is complex for choosing WHICH warehouse. For now, let's assume a strategy:
	// Try to reserve from the first warehouse that has enough stock.
	// A better approach for later: distribute reservation or use a "virtual" aggregated stock for reservation,
	// then a an allocation job. For now, simpler: pick one.

	// This simplified ReserveStock will try to reserve from any available stock in any active warehouse
	// It's not ideal as it doesn't specify which warehouse.
	// The repository methods for increase/decrease reserved stock need a warehouse_id.

	// Let's make a conceptual simplification for Tahap 2:
	// The `ReserveStock` in the service must iterate through warehouses,
	// lock rows, and update. THIS IS A COMPLEX TRANSACTION.

	// To simplify for now, let's simulate reserving from the "first suitable warehouse"
	// This will be refactored for Order Service later with proper Saga/transaction management if across warehouses,
	// or within a single transaction if only one warehouse is involved for the whole reservation.

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		logger.Error("Svc.ReserveStock: begin tx failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Find warehouses that have the product. For simplicity, let's get all active warehouses.
	// A more optimized query would be to find warehouses that stock this product and are active.
	activeWarehouses, err := s.repo.ListWarehouses(ctx) // Ideally filter for active here
	if err != nil {
		logger.Error("Svc.ReserveStock: list warehouses failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}

	remainingToReserve := quantityToReserve
	reservedInThisTx := false

	for _, wh := range activeWarehouses {
		if !wh.IsActive {
			continue
		}
		if remainingToReserve <= 0 {
			break
		}

		// Lock the product stock row for this warehouse
		stockItem, err := s.repo.GetProductStockForUpdate(ctx, tx, wh.ID, productID)
		if err != nil {
			if errors.Is(err, repository.ErrProductStockNotFound) {
				continue // Product not in this warehouse
			}
			logger.Error("Svc.ReserveStock: GetProductStockForUpdate failed", err, fmt.Sprintf("WID: %s, PID: %s", wh.ID, productID))
			return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
		}

		availableInWarehouse := stockItem.Quantity - stockItem.ReservedQuantity
		if availableInWarehouse > 0 {
			canReserveFromThisWH := 0
			if availableInWarehouse >= remainingToReserve {
				canReserveFromThisWH = remainingToReserve
			} else {
				canReserveFromThisWH = availableInWarehouse
			}

			if canReserveFromThisWH > 0 {
				err = s.repo.IncreaseReservedStock(ctx, tx, wh.ID, productID, canReserveFromThisWH)
				if err != nil {
					logger.Error("Svc.ReserveStock: IncreaseReservedStock failed", err, fmt.Sprintf("WID: %s, PID: %s", wh.ID, productID))
					// if one part fails, the whole transaction should roll back
					return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
				}
				remainingToReserve -= canReserveFromThisWH
				reservedInThisTx = true
			}
		}
	}

	if remainingToReserve > 0 {
		// Not enough stock could be reserved across all warehouses
		// Rollback has already been deferred, so it will happen.
		return repository.ErrInsufficientStock
	}

	if !reservedInThisTx && quantityToReserve > 0 { // Edge case: 0 to reserve, or product genuinely not found in any active WH
		return repository.ErrProductStockNotFound // Or a more specific "Product has no stock in any active warehouse"
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Svc.ReserveStock: commit tx failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}

	return nil
}

// ReleaseStock - similar logic to ReserveStock but for decreasing reserved_quantity
func (s *warehouseServiceImpl) ReleaseStock(ctx context.Context, productID string, quantityToRelease int) error {
	if quantityToRelease <= 0 {
		return errors.New("quantity to release must be positive")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		logger.Error("Svc.ReleaseStock: begin tx failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}
	defer tx.Rollback()

	activeWarehouses, err := s.repo.ListWarehouses(ctx)
	if err != nil {
		logger.Error("Svc.ReleaseStock: list warehouses failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}

	remainingToRelease := quantityToRelease
	releasedInThisTx := false

	for _, wh := range activeWarehouses {
		if !wh.IsActive {
			continue
		}
		if remainingToRelease <= 0 {
			break
		}

		stockItem, err := s.repo.GetProductStockForUpdate(ctx, tx, wh.ID, productID)
		if err != nil {
			if errors.Is(err, repository.ErrProductStockNotFound) {
				continue
			}
			logger.Error("Svc.ReleaseStock: GetProductStockForUpdate failed", err, fmt.Sprintf("WID: %s, PID: %s", wh.ID, productID))
			return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
		}

		if stockItem.ReservedQuantity > 0 {
			canReleaseFromThisWH := 0
			if stockItem.ReservedQuantity >= remainingToRelease {
				canReleaseFromThisWH = remainingToRelease
			} else {
				canReleaseFromThisWH = stockItem.ReservedQuantity
			}

			if canReleaseFromThisWH > 0 {
				err = s.repo.DecreaseReservedStock(ctx, tx, wh.ID, productID, canReleaseFromThisWH)
				if err != nil {
					logger.Error("Svc.ReleaseStock: DecreaseReservedStock failed", err, fmt.Sprintf("WID: %s, PID: %s", wh.ID, productID))
					return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
				}
				remainingToRelease -= canReleaseFromThisWH
				releasedInThisTx = true
			}
		}
	}

	if remainingToRelease > 0 {
		// This means we tried to release more than was actually reserved, or product not found
		// This could be an error state or just a notification for idempotency
		logger.Info(fmt.Sprintf("Svc.ReleaseStock: Could not release full quantity %d for product %s, %d still remaining. Might be normal if less was reserved.", quantityToRelease, productID, remainingToRelease))
		// Depending on strictness, this might not be an error if we successfully released what was available.
		// For now, let's consider it an issue if we needed to release X but couldn't.
		// return repository.ErrStockOperationFailed // Too generic
		return fmt.Errorf("could not release full quantity, %d remaining", remainingToRelease)
	}

	if !releasedInThisTx && quantityToRelease > 0 {
		return repository.ErrProductStockNotFound // Or "No reserved stock found for this product in any active warehouse"
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Svc.ReleaseStock: commit tx failed", err, nil)
		return fmt.Errorf("%w: %v", ErrStockOperationFailed, err)
	}

	return nil
}

func (s *warehouseServiceImpl) DeductStockAfterSale(ctx context.Context, req domain.DeductStockRequest) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for stock deduction: %w", err)
	}
	defer tx.Rollback()

	// Kunci baris untuk update
	_, err = s.repo.GetProductStockForUpdate(ctx, tx, req.WarehouseID, req.ProductID)
	if err != nil {
		return fmt.Errorf("failed to lock stock for deduction (WH: %s, Prod: %s): %w", req.WarehouseID, req.ProductID, err)
	}

	err = s.repo.DeductCommittedStock(ctx, tx, req.WarehouseID, req.ProductID, req.Quantity)
	if err != nil {
		return fmt.Errorf("failed to deduct committed stock (WH: %s, Prod: %s, Qty: %d): %w", req.WarehouseID, req.ProductID, req.Quantity, err)
	}

	return tx.Commit()
}
