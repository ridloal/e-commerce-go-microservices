package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	// Ganti dengan path yang benar
	"github.com/ridloal/e-commerce-go-microservices/internal/order/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
)

var ErrOrderNotFound = errors.New("order not found")

// DBTX interface untuk transaksi (bisa sama dengan yg di warehouse repo)
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

type OrderRepository interface {
	CreateOrderWithItems(ctx context.Context, order *domain.Order, items []domain.OrderItem) error
	// GetOrderByID, ListOrdersByUserID akan ditambahkan nanti
	BeginTx(ctx context.Context) (DBTX, error)

	GetPendingOrdersOlderThan(ctx context.Context, duration time.Duration) ([]domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, newStatus domain.OrderStatus) error
	GetOrderItemsByOrderID(ctx context.Context, orderID string) ([]domain.OrderItem, error)
}

type postgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) OrderRepository {
	return &postgresOrderRepository{db: db}
}

func (r *postgresOrderRepository) BeginTx(ctx context.Context) (DBTX, error) {
	return r.db.BeginTx(ctx, nil)
}

// CreateOrderWithItems menyimpan order dan item-itemnya dalam satu transaksi.
func (r *postgresOrderRepository) CreateOrderWithItems(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	tx, err := r.db.Begin() // Tidak menggunakan DBTX di sini karena manage transaksi internal
	if err != nil {
		logger.Error("CreateOrderWithItems: failed to begin tx", err, nil)
		return err
	}
	defer tx.Rollback() // Rollback jika tidak di-commit

	// 1. Simpan Order
	orderQuery := `INSERT INTO orders (user_id, total_amount, status, created_at, updated_at)
                   VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at, status`

	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	if order.Status == "" {
		order.Status = domain.StatusPendingPayment // Default status
	}

	err = tx.QueryRowContext(ctx, orderQuery, order.UserID, order.TotalAmount, order.Status, order.CreatedAt, order.UpdatedAt).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt, &order.Status)
	if err != nil {
		logger.Error("CreateOrderWithItems: failed to insert order", err, nil)
		return err
	}

	// 2. Simpan Order Items
	itemStmt, err := tx.PrepareContext(ctx, `INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase, created_at)
                                            VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`)
	if err != nil {
		logger.Error("CreateOrderWithItems: failed to prepare item statement", err, nil)
		return err
	}
	defer itemStmt.Close()

	for i := range items {
		items[i].OrderID = order.ID
		items[i].CreatedAt = time.Now() // Atau gunakan waktu order jika sama
		err = itemStmt.QueryRowContext(ctx, items[i].OrderID, items[i].ProductID, items[i].Quantity, items[i].PriceAtPurchase, items[i].CreatedAt).
			Scan(&items[i].ID, &items[i].CreatedAt)
		if err != nil {
			logger.Error("CreateOrderWithItems: failed to insert order item", err, map[string]interface{}{"item_product_id": items[i].ProductID})
			return err // Rollback akan terjadi
		}
	}
	order.Items = items // Assign items to order struct

	return tx.Commit()
}

func (r *postgresOrderRepository) GetPendingOrdersOlderThan(ctx context.Context, duration time.Duration) ([]domain.Order, error) {
	query := `SELECT id, user_id, total_amount, status, created_at, updated_at
              FROM orders
              WHERE status = $1 AND created_at < $2
              ORDER BY created_at ASC`

	thresholdTime := time.Now().Add(-duration)
	rows, err := r.db.QueryContext(ctx, query, domain.StatusPendingPayment, thresholdTime)
	if err != nil {
		logger.Error("GetPendingOrdersOlderThan: query failed", err, nil)
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			logger.Error("GetPendingOrdersOlderThan: scan failed", err, nil)
			// Lanjutkan proses order lain jika satu gagal di-scan
			continue
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *postgresOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, newStatus domain.OrderStatus) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, newStatus, orderID)
	if err != nil {
		logger.Error("UpdateOrderStatus: exec failed", err, map[string]interface{}{"order_id": orderID, "new_status": newStatus})
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *postgresOrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	query := `SELECT id, order_id, product_id, quantity, price_at_purchase, created_at
              FROM order_items WHERE order_id = $1`
	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		logger.Error("GetOrderItemsByOrderID: query failed", err, nil)
		return nil, err
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var i domain.OrderItem
		if err := rows.Scan(&i.ID, &i.OrderID, &i.ProductID, &i.Quantity, &i.PriceAtPurchase, &i.CreatedAt); err != nil {
			logger.Error("GetOrderItemsByOrderID: scan failed", err, nil)
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
