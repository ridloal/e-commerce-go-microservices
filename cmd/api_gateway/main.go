package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ridloal/e-commerce-go-microservices/internal/platform/config"
	"github.com/ridloal/e-commerce-go-microservices/internal/platform/logger"
)

func newSingleHostReverseProxy(targetHost string) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL '%s': %w", targetHost, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Original director sudah cukup untuk set Scheme dan Host
	// proxy.Director = func(req *http.Request) {
	// 	req.URL.Scheme = targetURL.Scheme
	// 	req.URL.Host = targetURL.Host
	// 	req.Host = targetURL.Host // Penting untuk beberapa server target
	//  // Path akan diteruskan apa adanya
	// }

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		logger.Error(fmt.Sprintf("Gateway: proxy error for %s %s to %s", req.Method, req.URL.Path, targetURL), err, nil)
		http.Error(rw, "Service unavailable or proxy error", http.StatusBadGateway)
	}
	return proxy, nil
}

func main() {
	cfg := config.LoadGatewayConfig()
	logger.Info("Starting API Gateway on port " + cfg.ListenPort)

	mux := http.NewServeMux()

	// Rute dan target service
	serviceMappings := map[string]string{
		"/api/v1/users/":      cfg.UserServiceURL, // Trailing slash penting untuk ServeMux matching
		"/api/v1/products/":   cfg.ProductServiceURL,
		"/api/v1/stock-info/": cfg.WarehouseServiceURL,
		"/api/v1/warehouses/": cfg.WarehouseServiceURL,
		"/api/v1/stocks/":     cfg.WarehouseServiceURL,
		"/api/v1/orders/":     cfg.OrderServiceURL,
	}

	for pathPrefix, targetHost := range serviceMappings {
		proxy, err := newSingleHostReverseProxy(targetHost)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create reverse proxy for target %s (prefix %s)", targetHost, pathPrefix), err, nil)
			continue
		}

		// Karena layanan kita mengharapkan path lengkap (misal /api/v1/users/login),
		// dan ServeMux dengan trailing slash akan cocok dengan semua subpath,
		// kita tidak perlu strip prefix di sini.
		mux.Handle(pathPrefix, http.HandlerFunc(func(p *httputil.ReverseProxy, prefix string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Log request yang masuk ke gateway jika diperlukan
				// logger.Info(fmt.Sprintf("Gateway: a %s %s request.", r.Method, r.URL.Path))
				p.ServeHTTP(w, r)
			}
		}(proxy, pathPrefix))) // Capture proxy & prefix dalam closure

		logger.Info(fmt.Sprintf("Routing %s to %s", pathPrefix, targetHost))
	}

	server := &http.Server{
		Addr:    ":" + cfg.ListenPort,
		Handler: mux,
	}

	logger.Info(fmt.Sprintf("API Gateway successfully configured and listening on :%s", cfg.ListenPort))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("API Gateway failed to start or crashed", err, nil)
	}
}
