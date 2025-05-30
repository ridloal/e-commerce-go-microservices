package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq" // Untuk pq.Error
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/warehouse/domain"
)

var (
	ErrWarehouseNotFound      = errors.New("warehouse not found")
	ErrProductStockNotFound   = errors.New("product stock entry not found for this warehouse and product")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrStockConflict          = errors.New("stock entry conflict, possibly unique constraint violation")
	ErrUpdateStockOutOfBounds = errors.New("update results in negative quantity or reserved quantity")
)

type WarehouseRepository interface {
	CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error
	GetWarehouseByID(ctx context.Context, id string) (*domain.Warehouse, error)
	ListWarehouses(ctx context.Context) ([]domain.Warehouse, error)
	UpdateWarehouseStatus(ctx context.Context, id string, isActive bool) error

	// Stock Management
	CreateOrUpdateProductStock(ctx context.Context, stock *domain.ProductStock) error
	GetProductStock(ctx context.Context, warehouseID, productID string) (*domain.ProductStock, error)
	GetTotalAvailableStockByProductID(ctx context.Context, productID string) (int, error)
	TransferStock(ctx context.Context, productID, sourceWarehouseID, targetWarehouseID string, quantity int) error

	// Internal methods for more complex stock operations (typically within a transaction)
	// These may need to be called by the service layer with db tx object
	IncreaseProductStockQuantity(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error
	DecreaseProductStockQuantity(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error // For actual sale deduction
	IncreaseReservedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error
	DecreaseReservedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error // For releasing reservation
	GetProductStockForUpdate(ctx context.Context, dbops DBTX, warehouseID, productID string) (*domain.ProductStock, error)
	DeductCommittedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, quantityToDeduct int) error

	BeginTx(ctx context.Context) (DBTX, error)
}

// DBTX adalah interface yang bisa berupa *sql.DB atau *sql.Tx
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

type postgresWarehouseRepository struct {
	db *sql.DB
}

func NewPostgresWarehouseRepository(db *sql.DB) WarehouseRepository {
	return &postgresWarehouseRepository{db: db}
}

// --- Warehouse Methods ---
func (r *postgresWarehouseRepository) CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	query := `INSERT INTO warehouses (name, location, is_active, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`
	warehouse.IsActive = true // Default
	warehouse.CreatedAt = time.Now()
	warehouse.UpdatedAt = time.Now()

	var location sql.NullString
	if warehouse.Location != nil {
		location = sql.NullString{String: *warehouse.Location, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query, warehouse.Name, location, warehouse.IsActive, warehouse.CreatedAt, warehouse.UpdatedAt).
		Scan(&warehouse.ID, &warehouse.CreatedAt, &warehouse.UpdatedAt)
	if err != nil {
		logger.Error("CreateWarehouse: failed to insert warehouse", err, nil)
		return err
	}
	return nil
}

func (r *postgresWarehouseRepository) GetWarehouseByID(ctx context.Context, id string) (*domain.Warehouse, error) {
	query := `SELECT id, name, location, is_active, created_at, updated_at FROM warehouses WHERE id = $1`
	var w domain.Warehouse
	var location sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&w.ID, &w.Name, &location, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		logger.Error("GetWarehouseByID: query failed", err, nil)
		return nil, err
	}
	if location.Valid {
		w.Location = &location.String
	}
	return &w, nil
}

func (r *postgresWarehouseRepository) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	query := `SELECT id, name, location, is_active, created_at, updated_at FROM warehouses ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		logger.Error("ListWarehouses: query failed", err, nil)
		return nil, err
	}
	defer rows.Close()

	warehouses := []domain.Warehouse{}
	for rows.Next() {
		var w domain.Warehouse
		var location sql.NullString
		if err := rows.Scan(&w.ID, &w.Name, &location, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			logger.Error("ListWarehouses: scan failed", err, nil)
			return nil, err
		}
		if location.Valid {
			w.Location = &location.String
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, rows.Err()
}

func (r *postgresWarehouseRepository) UpdateWarehouseStatus(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE warehouses SET is_active = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, isActive, time.Now(), id)
	if err != nil {
		logger.Error("UpdateWarehouseStatus: exec failed", err, nil)
		return err
	}
	return nil
}

// --- Stock Methods ---

// CreateOrUpdateProductStock is used for initially adding stock or adjusting it.
// For transactional reservations/deductions, use specific methods with DBTX.
func (r *postgresWarehouseRepository) CreateOrUpdateProductStock(ctx context.Context, stock *domain.ProductStock) error {
	query := `
        INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
        quantity = product_stocks.quantity + EXCLUDED.quantity, --  Adding to existing quantity
        updated_at = EXCLUDED.updated_at
        RETURNING id, quantity, reserved_quantity, created_at, updated_at`

	stock.CreatedAt = time.Now()
	stock.UpdatedAt = time.Now()
	// initial reserved_quantity is 0 when adding stock this way
	stock.ReservedQuantity = 0

	err := r.db.QueryRowContext(ctx, query,
		stock.WarehouseID, stock.ProductID, stock.Quantity, stock.ReservedQuantity,
		stock.CreatedAt, stock.UpdatedAt,
	).Scan(&stock.ID, &stock.Quantity, &stock.ReservedQuantity, &stock.CreatedAt, &stock.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" { // foreign_key_violation
			logger.Error("CreateOrUpdateProductStock: foreign key violation", err, nil)
			return fmt.Errorf("warehouse or product does not exist: %w", err)
		}
		logger.Error("CreateOrUpdateProductStock: failed to upsert stock", err, nil)
		return err
	}
	return nil
}

func (r *postgresWarehouseRepository) GetProductStock(ctx context.Context, warehouseID, productID string) (*domain.ProductStock, error) {
	query := `SELECT id, warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at
              FROM product_stocks WHERE warehouse_id = $1 AND product_id = $2`
	var ps domain.ProductStock
	err := r.db.QueryRowContext(ctx, query, warehouseID, productID).Scan(
		&ps.ID, &ps.WarehouseID, &ps.ProductID, &ps.Quantity, &ps.ReservedQuantity, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductStockNotFound
		}
		logger.Error("GetProductStock: query failed", err, nil)
		return nil, err
	}
	return &ps, nil
}

func (r *postgresWarehouseRepository) GetTotalAvailableStockByProductID(ctx context.Context, productID string) (int, error) {
	query := `
        SELECT COALESCE(SUM(ps.quantity - ps.reserved_quantity), 0)
        FROM product_stocks ps
        JOIN warehouses w ON ps.warehouse_id = w.id
        WHERE ps.product_id = $1 AND w.is_active = TRUE`
	var totalAvailable int
	err := r.db.QueryRowContext(ctx, query, productID).Scan(&totalAvailable)
	if err != nil {
		logger.Error("GetTotalAvailableStockByProductID: query failed for product_id "+productID, err, nil)
		return 0, err
	}
	return totalAvailable, nil
}

func (r *postgresWarehouseRepository) TransferStock(ctx context.Context, productID, sourceWarehouseID, targetWarehouseID string, quantity int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("TransferStock: failed to begin transaction", err, nil)
		return fmt.Errorf("failed to begin transaction for stock transfer: %w", err)
	}
	defer tx.Rollback() // Rollback jika tidak di-commit

	// 1. Kunci baris stok di gudang sumber
	sourceStock, err := r.GetProductStockForUpdate(ctx, tx, sourceWarehouseID, productID)
	if err != nil {
		if errors.Is(err, ErrProductStockNotFound) {
			return fmt.Errorf("product %s not found in source warehouse %s: %w", productID, sourceWarehouseID, err)
		}
		return fmt.Errorf("failed to get stock from source warehouse %s for product %s: %w", sourceWarehouseID, productID, err)
	}

	// 2. Cek apakah stok cukup di gudang sumber (total quantity, bukan available)
	if sourceStock.Quantity < quantity {
		return fmt.Errorf("insufficient total stock for product %s in source warehouse %s. Available: %d, Requested: %d: %w",
			productID, sourceWarehouseID, sourceStock.Quantity, quantity, ErrInsufficientStock)
	}
	// Pastikan Reserved Quantity tidak melebihi quantity baru setelah dikurangi
	if (sourceStock.Quantity - quantity) < sourceStock.ReservedQuantity {
		return fmt.Errorf("transfer failed: reducing quantity for product %s in source warehouse %s below reserved quantity (%d < %d): %w",
			productID, sourceWarehouseID, (sourceStock.Quantity - quantity), sourceStock.ReservedQuantity, ErrInsufficientStock)
	}

	// 3. Kurangi stok dari gudang sumber
	// Langsung mengurangi 'quantity', bukan 'reserved_quantity'
	querySource := `UPDATE product_stocks SET quantity = quantity - $1, updated_at = NOW()
                    WHERE warehouse_id = $2 AND product_id = $3`
	_, err = tx.ExecContext(ctx, querySource, quantity, sourceWarehouseID, productID)
	if err != nil {
		logger.Error("TransferStock: failed to decrease stock from source", err, nil)
		return fmt.Errorf("failed to decrease stock from source warehouse %s: %w", sourceWarehouseID, err)
	}

	// 4. Tambah atau update stok di gudang tujuan (buat entri jika belum ada)
	// Kunci baris entri stok di gudang tujuan jika sudah ada, atau siapkan untuk insert
	// Ini bisa menggunakan ON CONFLICT DO UPDATE
	queryTarget := `
        INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at)
        VALUES ($1, $2, $3, 0, NOW(), NOW())
        ON CONFLICT (warehouse_id, product_id) DO UPDATE SET
        quantity = product_stocks.quantity + EXCLUDED.quantity,
        updated_at = NOW()
        WHERE product_stocks.warehouse_id = $1 AND product_stocks.product_id = $2`

	_, err = tx.ExecContext(ctx, queryTarget, targetWarehouseID, productID, quantity)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" { // foreign_key_violation for targetWarehouseID
			logger.Error("TransferStock: target warehouse does not exist", err, map[string]interface{}{"target_warehouse_id": targetWarehouseID})
			return fmt.Errorf("target warehouse %s does not exist: %w", targetWarehouseID, err)
		}
		logger.Error("TransferStock: failed to increase/update stock in target", err, nil)
		return fmt.Errorf("failed to update stock in target warehouse %s: %w", targetWarehouseID, err)
	}

	return tx.Commit()
}

// --- Transactional Stock Methods ---
func (r *postgresWarehouseRepository) BeginTx(ctx context.Context) (DBTX, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *postgresWarehouseRepository) GetProductStockForUpdate(ctx context.Context, dbops DBTX, warehouseID, productID string) (*domain.ProductStock, error) {
	query := `SELECT id, warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at
              FROM product_stocks WHERE warehouse_id = $1 AND product_id = $2 FOR UPDATE`
	var ps domain.ProductStock
	err := dbops.QueryRowContext(ctx, query, warehouseID, productID).Scan(
		&ps.ID, &ps.WarehouseID, &ps.ProductID, &ps.Quantity, &ps.ReservedQuantity, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Attempt to create a zero-stock entry if it doesn't exist, so it can be locked and updated
			// This might be needed if a product can exist in a warehouse with 0 stock initially

			// zeroStockQuery := `
			//     INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at)
			//     VALUES ($1, $2, 0, 0, NOW(), NOW())
			//     ON CONFLICT (warehouse_id, product_id) DO NOTHING --  if it was created by another tx concurrently
			//     RETURNING id, warehouse_id, product_id, quantity, reserved_quantity, created_at, updated_at`

			// Re-query after insert attempt to lock the row
			// This logic might be complex. Simpler for now: if it doesn't exist, it can't be locked.
			// Or, the service layer should ensure it exists OR CreateOrUpdateProductStock is called before such ops.
			// For now, let's return ErrProductStockNotFound if it truly doesn't exist during a FOR UPDATE.
			logger.Error("GetProductStockForUpdate: product stock not found for lock", sql.ErrNoRows, fmt.Sprintf(" WID: %s, PID: %s", warehouseID, productID))
			return nil, ErrProductStockNotFound
		}
		logger.Error("GetProductStockForUpdate: query failed", err, nil)
		return nil, err
	}
	return &ps, nil
}

func (r *postgresWarehouseRepository) IncreaseProductStockQuantity(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error {
	// Ensure stock item exists, if not, create with 0, then update
	// This is a simplified version, assumes stock item exists or GetProductStockForUpdate handled it
	query := `UPDATE product_stocks SET quantity = quantity + $1, updated_at = NOW()
              WHERE warehouse_id = $2 AND product_id = $3`
	res, err := dbops.ExecContext(ctx, query, amount, warehouseID, productID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation (e.g. quantity < 0)
			logger.Error("IncreaseProductStockQuantity: check violation", err, nil)
			return ErrUpdateStockOutOfBounds
		}
		logger.Error("IncreaseProductStockQuantity: exec failed", err, nil)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrProductStockNotFound // Or handle more gracefully, e.g. try to insert it
	}
	return nil
}

// DecreaseProductStockQuantity (for actual sale)
func (r *postgresWarehouseRepository) DecreaseProductStockQuantity(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error {
	query := `UPDATE product_stocks SET quantity = quantity - $1, updated_at = NOW()
              WHERE warehouse_id = $2 AND product_id = $3 AND (quantity - $1) >= reserved_quantity AND (quantity - $1) >= 0`
	res, err := dbops.ExecContext(ctx, query, amount, warehouseID, productID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation
			logger.Error("DecreaseProductStockQuantity: check violation", err, nil)
			// This can also mean quantity became less than reserved_quantity, which is bad if not handled
			return ErrInsufficientStock
		}
		logger.Error("DecreaseProductStockQuantity: exec failed", err, nil)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrInsufficientStock // or product not found, or (quantity - amount) < 0 condition failed
	}
	return nil
}

// IncreaseReservedStock (for order reservation)
func (r *postgresWarehouseRepository) IncreaseReservedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error {
	query := `UPDATE product_stocks SET reserved_quantity = reserved_quantity + $1, updated_at = NOW()
              WHERE warehouse_id = $2 AND product_id = $3 AND (quantity - (reserved_quantity + $1)) >= 0`
	res, err := dbops.ExecContext(ctx, query, amount, warehouseID, productID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation
			logger.Error("IncreaseReservedStock: check violation", err, nil)
			return ErrInsufficientStock
		}
		logger.Error("IncreaseReservedStock: exec failed", err, nil)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrInsufficientStock // or product not found, or no available stock to reserve
	}
	return nil
}

// DecreaseReservedStock (for releasing reservation or completing sale)
func (r *postgresWarehouseRepository) DecreaseReservedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, amount int) error {
	query := `UPDATE product_stocks SET reserved_quantity = reserved_quantity - $1, updated_at = NOW()
              WHERE warehouse_id = $2 AND product_id = $3 AND (reserved_quantity - $1) >= 0`
	res, err := dbops.ExecContext(ctx, query, amount, warehouseID, productID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation
			logger.Error("DecreaseReservedStock: check violation", err, nil)
			return ErrUpdateStockOutOfBounds // e.g. reserved_quantity went negative
		}
		logger.Error("DecreaseReservedStock: exec failed", err, nil)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrStockConflict // or product not found, or reserved_quantity couldn't be decreased that much
	}
	return nil
}

func (r *postgresWarehouseRepository) DeductCommittedStock(ctx context.Context, dbops DBTX, warehouseID, productID string, quantityToDeduct int) error {
	// Metode ini mengurangi reserved_quantity DAN quantity aktual.
	query := `UPDATE product_stocks
              SET quantity = quantity - $1,
                  reserved_quantity = reserved_quantity - $1,
                  updated_at = NOW()
              WHERE warehouse_id = $2 AND product_id = $3
                AND (reserved_quantity - $1) >= 0
                AND (quantity - $1) >= 0`
	// Mungkin perlu juga AND quantity >= reserved_quantity

	res, err := dbops.ExecContext(ctx, query, quantityToDeduct, warehouseID, productID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation
			logger.Error("DeductCommittedStock: check violation (negative stock/reserved)", err, nil)
			return ErrInsufficientStock // Atau error yang lebih spesifik
		}
		logger.Error("DeductCommittedStock: exec failed", err, nil)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		// Ini bisa berarti stok tidak cukup lagi, atau reservasi kurang, atau produk tidak ada
		logger.Error("DeductCommittedStock: no rows affected, potential inconsistency or insufficient reserved stock", nil, map[string]interface{}{
			"warehouse_id": warehouseID, "product_id": productID, "quantity": quantityToDeduct,
		})
		return ErrInsufficientStock // Atau error yang lebih spesifik
	}
	return nil
}
