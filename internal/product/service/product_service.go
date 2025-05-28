package service

import (
	"context"

	"github.com/ridloal/e-commerce-go-microservices/internal/product/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/repository"
)

type ProductService interface {
	ListProducts(ctx context.Context) ([]domain.Product, error)
	GetProductDetails(ctx context.Context, productID string) (*domain.Product, error)
}

type productServiceImpl struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productServiceImpl{repo: repo}
}

func (s *productServiceImpl) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.repo.ListProducts(ctx)
}

func (s *productServiceImpl) GetProductDetails(ctx context.Context, productID string) (*domain.Product, error) {
	return s.repo.GetProductByID(ctx, productID)
}
