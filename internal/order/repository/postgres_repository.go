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
