package service

import (
	"context"
	"sync" // Untuk Fan-out Fan-in pattern

	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/domain"
	"github.com/ridloal/e-commerce-go-microservices/internal/product/repository"
)

type ProductService interface {
	ListProducts(ctx context.Context) ([]domain.Product, error)
	GetProductDetails(ctx context.Context, productID string) (*domain.Product, error)
	// CreateProduct, UpdateProduct (tanpa stock), DeleteProduct - sementara manual dari db
}

type productServiceImpl struct {
	repo                   repository.ProductRepository
	warehouseServiceClient *WarehouseServiceClient
}

func NewProductService(repo repository.ProductRepository, whClient *WarehouseServiceClient) ProductService {
	return &productServiceImpl{
		repo:                   repo,
		warehouseServiceClient: whClient,
	}
}

func (s *productServiceImpl) ListProducts(ctx context.Context) ([]domain.Product, error) {
	products, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	// Fan-out: Panggil Warehouse Service untuk setiap produk secara concurrent
	var wg sync.WaitGroup
	type result struct {
		index int
		stock int
		err   error
	}
	resultsChan := make(chan result, len(products))

	for i, p := range products {
		wg.Add(1)
		go func(idx int, prodID string) {
			defer wg.Done()
			stockInfo, err := s.warehouseServiceClient.GetProductStockInfo(ctx, prodID)
			if err != nil {
				// Log error tapi jangan gagalkan seluruh list, mungkin produk belum ada stoknya
				logger.Error("ListProducts: failed to get stock for product "+prodID, err, nil)
				resultsChan <- result{index: idx, stock: 0, err: err} // Kirim 0 atau nilai default
				return
			}
			resultsChan <- result{index: idx, stock: stockInfo.TotalAvailable, err: nil}
		}(i, p.ID)
	}

	wg.Wait()
	close(resultsChan)

	// Fan-in: Kumpulkan hasil
	for res := range resultsChan {
		if res.err == nil { // Hanya update jika tidak ada error signifikan dari warehouse service
			products[res.index].StockQuantity = res.stock
		} else {
			// Jika ada error saat fetch stock, biarkan StockQuantity di product apa adanya (dari DB product),
			// atau set ke nilai default seperti 0 atau -1 untuk menandakan data tidak tersedia.
			// Untuk sekarang, kita biarkan 0 jika error saat fetch.
			products[res.index].StockQuantity = 0
		}
	}

	return products, nil
}

func (s *productServiceImpl) GetProductDetails(ctx context.Context, productID string) (*domain.Product, error) {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Dapatkan info stok dari Warehouse Service
	stockInfo, err := s.warehouseServiceClient.GetProductStockInfo(ctx, productID)
	if err != nil {
		// Jika gagal mendapatkan stok, kita bisa memilih untuk mengembalikan produk tanpa info stok
		// atau mengembalikan error. Untuk sekarang, log error dan set stok ke 0.
		logger.Error("GetProductDetails: failed to get stock for product "+productID, err, nil)
		product.StockQuantity = 0 // Atau tandai sebagai tidak diketahui
	} else {
		product.StockQuantity = stockInfo.TotalAvailable
	}

	return product, nil
}
