package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/domain"
)

var ErrProductNotFound = errors.New("product not found")

type ProductRepository interface {
	ListProducts(ctx context.Context) ([]domain.Product, error)
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	// Add CreateProduct, UpdateProduct, DeleteProduct later if needed
}

type postgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) ProductRepository {
	return &postgresProductRepository{db: db}
}

func (r *postgresProductRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	query := `SELECT id, name, description, price, stock_quantity, created_at, updated_at FROM products ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		logger.Error("ListProducts: query failed", err)
		return nil, err
	}
	defer rows.Close()

	products := []domain.Product{}
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt); err != nil {
			logger.Error("ListProducts: scan failed", err)
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		logger.Error("ListProducts: rows iteration error", err)
		return nil, err
	}
	return products, nil
}

func (r *postgresProductRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `SELECT id, name, description, price, stock_quantity, created_at, updated_at FROM products WHERE id = $1`
	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		logger.Error("GetProductByID: query failed", err)
		return nil, err
	}
	return &p, nil
}
