# ---- Builder Stage ----
    FROM golang:1.23.4-alpine AS builder

    # Set environment variables
    ENV CGO_ENABLED=0
    ENV GOOS=linux
    ENV GOARCH=amd64
    
    WORKDIR /build
    
    # Copy go.mod and go.sum first to leverage Docker cache
    COPY ../../go.mod ../../go.sum ./
    RUN go mod download
    
    # Copy the entire project source code
    # (Kita copy semua karena internal dependencies mungkin ada di luar cmd/api_gateway)
    COPY ../../ ./
    
    # Build the application
    # Sesuaikan path ke main.go untuk setiap service
    RUN go build -ldflags="-w -s" -o /app/api_gateway ./cmd/api_gateway/main.go
    
    # ---- Final Stage ----
    FROM alpine:latest
    
    # Install ca-certificates untuk HTTPS dan koneksi aman lainnya
    RUN apk --no-cache add ca-certificates
    
    WORKDIR /app
    
    # Copy binary dari builder stage
    COPY --from=builder /app/api_gateway /app/api_gateway
    
    # Expose port yang digunakan oleh API Gateway
    # Ini akan di-override oleh variabel lingkungan jika ada, tapi baik untuk dokumentasi
    EXPOSE 8080
    
    # Command untuk menjalankan aplikasi
    ENTRYPOINT ["/app/api_gateway"]