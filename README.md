# E-commerce & Stock Management System (Go Microservices)

Welcome to the E-commerce & Stock Management System project! This project is an implementation of an e-commerce case study built using a microservices architecture with Go (Golang) and PostgreSQL as the database. The main goal of this project is to create a modular, scalable, and maintainable platform.

This document will guide you through an overview of the project, prerequisites, how to run the services locally, and other relevant information.

## Table of Contents

- [Key Features](#key-features)
- [System Architecture](#system-architecture)
- [Technologies Used](#technologies-used)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Local Setup & Configuration](#local-setup--configuration)
  - [Environment Variables (.env)](#environment-variables-env)
  - [Database Setup](#database-setup)
  - [Running Database Migrations](#running-database-migrations)
- [How to Run Services](#how-to-run-services)
  - [Running Each Service Individually (Locally)](#running-each-service-individually-locally)
  - [Using Docker (For Deployment)](#using-docker-for-deployment)
- [Sample API Endpoints](#sample-api-endpoints)
- [Development Strategy](#development-strategy)
- [Testing Strategy](#testing-strategy)
- [Contributing](#contributing)
- [License](#license)

## Key Features

The system is designed to cover e-commerce and stock management functionalities through several separate services:

1.  **User Service**:
    * Handles simple user authentication (login using phone number or email).
    * Manages user data.
2.  **Product Service**:
    * Provides an API to display a list of products.
    * Displays product stock availability integrated with the Warehouse Service.
3.  **Order Service**:
    * Manages the customer checkout process.
    * Handles reservation (locking) of stock for ordered products.
    * Deducts stock after successful payment.
    * Includes a mechanism to release reserved stock if payment is not made within a specified time frame (N minutes).
4.  **Warehouse Service**:
    * Manages detailed product stock across multiple warehouses.
    * Allows product transfers between warehouses.
    * Manages warehouse status (active/inactive) and ensures stock from inactive warehouses is not counted.
5.  **API Gateway**:
    * Acts as a single entry point for all client requests.
    * Routes requests to the appropriate services.

## System Architecture

This project adopts a **Microservices Architecture** to ensure each component can be developed, deployed, and scaled independently.

### Key Components:
1.  **API Gateway**: Single entry point for all requests.
2.  **User Service**
3.  **Product Service**
4.  **Order Service**
5.  **Warehouse Service**
6.  **Database (PostgreSQL)**: Each service will have its own database schema (or separate database instances in future implementations) for data isolation.

## Technologies Used

* **Programming Language**: Golang (Go)
* **HTTP Framework**: Gin-Gonic (`gin`)
* **Database**: PostgreSQL
* **Database Driver (Go)**: `jackc/pgx/v5/stdlib`
* **Database Migration Tool**: `golang-migrate/migrate`
* **Password Hashing**: `golang.org/x/crypto/bcrypt`
* **JWT (JSON Web Tokens)**: `github.com/golang-jwt/jwt/v5`
* **Containerization (for Deployment)**: Docker

## Project Structure

The project uses a monorepo structure for ease of management:

```
/e-commerce-go-microservices/
├── cmd/                    # Entry points (main.go) for each service and API Gateway
│   ├── api_gateway/
│   ├── order_service/
│   ├── product_service/
│   ├── user_service/
│   └── warehouse_service/
├── internal/               # Internal code specific to each service
│   ├── <service_name>/     # e.g., user/, product/
│   │   ├── api/            # HTTP handlers (controllers)
│   │   ├── service/        # Core business logic (use cases)
│   │   ├── repository/     # Database access (data access layer)
│   │   └── domain/         # Core domain models and entities
│   └── platform/           # Shared platform code (DB connection, logger, config)
├── migrations/             # SQL database migration scripts per service
│   ├── order_service/
│   ├── product_service/
│   ├── user_service/
│   └── warehouse_service/
├── .env                    # Environment variable configuration file (do not commit if it contains secrets)
├── .env.example            # Example environment variable file
├── go.mod                  # Go module definition
├── go.sum                  # Go dependency checksums
├── Makefile                # (Optional) Helper commands for build, run, etc.
└── README.md               # This documentation
```

## Prerequisites

Before you begin, ensure your system has the following software installed:

* **Go**: Version 1.22 or higher (align with the version in your `go.mod`).
* **PostgreSQL**: An active and running PostgreSQL database server.
* **`golang-migrate/migrate` CLI**: Tool for running database migrations.
    * Installation instructions: [`golang-migrate` Documentation](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
* **Git**: Version control system.
* **(Optional) `make`**: If you wish to use the provided `Makefile`.
* **(Optional for Local Development, Required for Deployment) Docker & Docker Compose**: For running the application in containers.

## Local Setup & Configuration

Follow these steps to set up and run the project in your local environment.

1.  **Clone the Repository**:
    ```bash
    git clone [https://github.com/ridloal/e-commerce-go-microservices.git](https://github.com/ridloal/e-commerce-go-microservices.git)
    cd e-commerce-go-microservices
    ```

2.  **Environment Variables (`.env`)**:
    Copy the `.env.example` file (if it exists) to `.env` in the project's root directory. If not, create the `.env` file manually.
    Adjust the variables within to match your local configuration, especially for database connections and server ports.

    Example content for `.env` (adjust `your_...` values):
    ```env
    # ==== Global Settings ====
    # None for now

    # ==== API Gateway ====
    API_GATEWAY_PORT=8080
    USER_SERVICE_URL=http://localhost:8081
    PRODUCT_SERVICE_URL=http://localhost:8082
    WAREHOUSE_SERVICE_URL=http://localhost:8083
    ORDER_SERVICE_URL=http://localhost:8084

    # ==== User Service ====
    USER_SERVER_PORT=8081
    USER_DB_HOST=localhost
    USER_DB_PORT=5432
    USER_DB_USER=your_user_db_user
    USER_DB_PASSWORD=your_user_db_password
    USER_DB_NAME=user_db
    USER_DB_DSN=postgres://${USER_DB_USER}:${USER_DB_PASSWORD}@${USER_DB_HOST}:${USER_DB_PORT}/${USER_DB_NAME}?sslmode=disable
    JWT_SECRET_KEY=yourSuperSecretKeyForJWT123!

    # ==== Product Service ====
    PRODUCT_SERVER_PORT=8082
    PRODUCT_DB_HOST=localhost
    PRODUCT_DB_PORT=5432
    PRODUCT_DB_USER=your_product_db_user
    PRODUCT_DB_PASSWORD=your_product_db_password
    PRODUCT_DB_NAME=product_db
    PRODUCT_DB_DSN=postgres://${PRODUCT_DB_USER}:${PRODUCT_DB_PASSWORD}@${PRODUCT_DB_HOST}:${PRODUCT_DB_PORT}/${PRODUCT_DB_NAME}?sslmode=disable
    # WAREHOUSE_SERVICE_URL is already defined above

    # ==== Warehouse Service ====
    WAREHOUSE_SERVER_PORT=8083
    WAREHOUSE_DB_HOST=localhost
    WAREHOUSE_DB_PORT=5432
    WAREHOUSE_DB_USER=your_warehouse_db_user
    WAREHOUSE_DB_PASSWORD=your_warehouse_db_password
    WAREHOUSE_DB_NAME=warehouse_db
    WAREHOUSE_DB_DSN=postgres://${WAREHOUSE_DB_USER}:${WAREHOUSE_DB_PASSWORD}@${WAREHOUSE_DB_HOST}:${WAREHOUSE_DB_PORT}/${WAREHOUSE_DB_NAME}?sslmode=disable

    # ==== Order Service ====
    ORDER_SERVER_PORT=8084
    ORDER_DB_HOST=localhost
    ORDER_DB_PORT=5432
    ORDER_DB_USER=your_order_db_user
    ORDER_DB_PASSWORD=your_order_db_password
    ORDER_DB_NAME=order_db
    ORDER_DB_DSN=postgres://${ORDER_DB_USER}:${ORDER_DB_PASSWORD}@${ORDER_DB_HOST}:${ORDER_DB_PORT}/${ORDER_DB_NAME}?sslmode=disable
    PAYMENT_TIMEOUT_MINUTES=2
    # WAREHOUSE_SERVICE_URL is already defined above
    ```
    **Important**: Ensure the code in `internal/platform/config/config.go` reads these variables from the environment.

3.  **Database Setup**:
    Ensure your PostgreSQL server is running. Create the necessary databases for each service if they don't already exist:
    * `user_db`
    * `product_db`
    * `warehouse_db`
    * `order_db`

    You can use `psql` or your preferred database administration tool. Example:
    ```sql
    CREATE DATABASE user_db;
    CREATE DATABASE product_db;
    CREATE DATABASE warehouse_db;
    CREATE DATABASE order_db;
    ```

4.  **Running Database Migrations**:
    Use the `golang-migrate/migrate CLI` to apply the database schema and initial seed data (if any). Run these commands from the project's root directory.

    * **User Service Migrations**:
        ```bash
        migrate -database "${USER_DB_DSN}" -path ./migrations/user_service up
        ```
    * **Product Service Migrations**:
        ```bash
        migrate -database "${PRODUCT_DB_DSN}" -path ./migrations/product_service up
        ```
    * **Warehouse Service Migrations**:
        ```bash
        migrate -database "${WAREHOUSE_DB_DSN}" -path ./migrations/warehouse_service up
        ```
    * **Order Service Migrations**:
        ```bash
        migrate -database "${ORDER_DB_DSN}" -path ./migrations/order_service up
        ```
    *Note: Replace `${USER_DB_DSN}` and other DSN variables with the actual DSN values if you are not exporting these variables in your shell, or ensure your Go application can read from the `.env` file if `migrate-cli` doesn't directly (typically, `migrate-cli` takes the DSN directly from the command-line argument).*

## How to Run Services

### Running Each Service Individually (Locally)

After completing the setup and database migrations, you can run each Go service individually. Open a separate terminal for each service.

1.  **User Service**:
    ```bash
    go run ./cmd/user_service/main.go
    ```
    By default, it will run on the port defined by `USER_SERVER_PORT` (e.g., 8081).

2.  **Product Service**:
    ```bash
    go run ./cmd/product_service/main.go
    ```
    By default, it will run on the port defined by `PRODUCT_SERVER_PORT` (e.g., 8082).

3.  **Warehouse Service**:
    ```bash
    go run ./cmd/warehouse_service/main.go
    ```
    By default, it will run on the port defined by `WAREHOUSE_SERVER_PORT` (e.g., 8083).

4.  **Order Service**:
    ```bash
    go run ./cmd/order_service/main.go
    ```
    By default, it will run on the port defined by `ORDER_SERVER_PORT` (e.g., 8084).

5.  **API Gateway**:
    ```bash
    go run ./cmd/api_gateway/main.go
    ```
    By default, it will run on the port defined by `API_GATEWAY_PORT` (e.g., 8080). The API Gateway will route requests to the above services based on the configured URLs.

Ensure that all dependent services are running before starting a service that relies on them (e.g., the Product Service might need the Warehouse Service for stock information).

### Using Docker (For Deployment)

For deployment, Docker and Docker Compose are used to containerize and orchestrate the services. The setup includes `Dockerfile` for each service and a `docker-compose.yml` file to manage the multi-container application.

The `docker-compose.yml` is configured to:
* Build an image for each service.
* Set up a PostgreSQL database container for each service.
* Automatically run database migrations (including seed data) for each service upon startup using the `migrate/migrate` image.
* Manage environment variables using the central `.env` file.
* Establish a network for inter-service communication.

**Deployment Steps**:

1.  **Ensure Docker and Docker Compose are installed.**
2.  **Navigate to the project root directory.**
3.  **Build the Docker images**:
    ```bash
    docker-compose build
    ```
    (Or `make build` if using the Makefile).
4.  **Start all services in detached mode**:
    ```bash
    docker-compose up -d
    ```
    (Or `make up` if using the Makefile).
    This command will start all services defined in `docker-compose.yml`, including databases. The migration services will run, apply schema changes and seed data, and then exit. The application services will start once their dependent databases are healthy.

5.  **Check service status**:
    ```bash
    docker-compose ps
    ```
    (Or `make ps`).
6.  **View logs**:
    ```bash
    docker-compose logs -f
    # For a specific service:
    docker-compose logs -f user_service
    ```
    (Or `make logs` / `make logs user_service`).
7.  **Stop and remove containers**:
    ```bash
    docker-compose down
    ```
    (Or `make down`). To also remove volumes (database data), use `docker-compose down -v`.

## Sample API Endpoints

The following are some primary API endpoints. For a complete list, please refer to the handler code in each service or the API documentation (e.g., Swagger/OpenAPI, if available).

**The API Gateway runs at `http://localhost:8080` (by default when deployed via Docker, or as configured)**

* **User Service** (prefixed with `/api/v1/users`)
    * `POST /api/v1/users/register`: Register a new user.
    * `POST /api/v1/users/login`: Log in a user.
* **Product Service** (prefixed with `/api/v1/products`)
    * `GET /api/v1/products`: Display a list of all products.
    * `GET /api/v1/products/{product_id}`: Display details of a specific product.
* **Warehouse Service** (prefixed with `/api/v1/warehouses` or `/api/v1/stocks`)
    * `POST /api/v1/warehouses`: Create a new warehouse.
    * `GET /api/v1/warehouses`: Display a list of warehouses.
    * `POST /api/v1/warehouses/{warehouse_id}/stocks`: Add product stock to a warehouse.
    * `GET /api/v1/stock-info/products/{product_id}`: Get aggregated stock for a product.
    * `POST /api/v1/stocks/reserve`: Reserve stock.
    * `POST /api/v1/stocks/release`: Release stock reservation.
* **Order Service** (prefixed with `/api/v1/orders`)
    * `POST /api/v1/orders`: Create a new order.
    * `POST /api/v1/orders/{order_id}/confirm-payment`: Confirm payment for an order.

## Development Strategy

The project is developed using a phased approach as outlined in the `initial_overview.md` document, starting from foundational setup, core service implementation, to advanced functionalities and deployment preparation.

## Testing Strategy

* **Unit Tests**: Testing individual functions and methods within each service using Go's testing library and mocks for external dependencies.
* **Integration Tests**: Testing interactions between components within a single service (e.g., API handler -> service -> repository -> database) and interactions between services.
* **End-to-End (E2E) Tests**: Testing complete user flows through the API Gateway.

## Contributing

Currently, this project is individually managed. If you are interested in contributing in the future:
1.  Please create an *Issue* first to discuss the changes or features you wish to add.
2.  After discussion, you can *Fork* this repository.
3.  Create a new *branch* for your work.
4.  Once completed, submit a *Pull Request* to the `main` branch (or the designated development branch).

Ensure your code adheres to existing coding standards and includes relevant tests.

## License

This project is currently not under a specific license. (You can add a license like MIT, Apache 2.0, etc., here if desired).
