# ---- Builder Stage ----
    FROM golang:1.23.4-alpine AS builder
    ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
    WORKDIR /build
    COPY ../../go.mod ../../go.sum ./
    RUN go mod download
    COPY ../../ ./
    RUN go build -ldflags="-w -s" -o /app/warehouse_service ./cmd/warehouse_service/main.go
    
    # ---- Final Stage ----
    FROM alpine:latest
    RUN apk --no-cache add ca-certificates
    WORKDIR /app
    COPY --from=builder /app/warehouse_service /app/warehouse_service
    EXPOSE 8083
    ENTRYPOINT ["/app/warehouse_service"]