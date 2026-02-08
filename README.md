# Array Banking API

> **Banking REST API for Array Candidate Assessment Project**
> Version 1.0 | Last Updated: October 2025

A comprehensive, banking REST API built with Go, designed specifically for technical interviews and developer assessments at Array. This project demonstrates real-world banking domain complexity with production-ready architecture, security practices, and comprehensive API coverage.

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18%2B-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=flat)](LICENSE)

---

## 📋 Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [API Documentation](#api-documentation)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
- [Project Structure](#project-structure)
- [NorthWind Bank Integration](#northwind-bank-integration)
- [License](#license)

---

## 🎯 Overview

The Array Banking API is a production-quality foundation designed to enable meaningful technical assessments while demonstrating enterprise-grade development practices. Unlike generic toy APIs, this system provides authentic banking domain complexity with clear extension points for candidates to showcase their skills.

---

## ✨ Key Features

### Development Experience

- **🐳 Docker Support**
  - Complete Docker Compose setup for development and production
  - Automated database migrations with golang-migrate
  - Hot reload with Air for rapid development
  - PostgreSQL 18 with persistent volumes
  - Comprehensive Docker documentation in [README.docker.md](README.docker.md)

- **🧪 Testing**
  - Comprehensive test coverage
  - Mock repositories with gomock
  - Postman collection for API testing

- **🔧 Development Tools**
  - Makefile with common tasks
  - Automatic API documentation generation
  - Code linting with golangci-lint
  - Environment-based configuration

---

## 🛠️ Tech Stack

### Core Technologies

| Category | Technology | Version | Purpose |
|----------|-----------|---------|---------|
| **Language** | Go | 1.24+ | Primary application language |
| **Framework** | Echo | v4 | High-performance HTTP framework |
| **Database** | PostgreSQL | 16+ | Primary data store |
| **ORM** | GORM | Latest | Database abstraction layer |
| **Authentication** | JWT | golang-jwt/v5 | Stateless authentication |
| **Validation** | go-playground/validator | v10 | Request validation |

### Infrastructure & DevOps

- **Containerization**: Docker, Docker Compose
- **Migrations**: golang-migrate
- **Logging**: Go standard library `log/slog`
- **Metrics**: Prometheus with Go client
- **Hot Reload**: Air (development)
- **API Docs**: swaggo/swag (OpenAPI 3.1)

### Testing & Quality

- **Testing Framework**: Go standard library `testing`
- **Assertions**: testify
- **Mocking**: gomock, sqlmock
- **Fake Data**: gofakeit
- **API Testing**: Postman collection

### Key Libraries

```go
github.com/labstack/echo/v4              // HTTP framework
github.com/golang-jwt/jwt/v5             // JWT authentication
github.com/go-playground/validator/v10   // Request validation
github.com/shopspring/decimal            // Precise decimal arithmetic
github.com/google/uuid                   // UUID generation
gorm.io/gorm                             // ORM
gorm.io/driver/postgres                  // PostgreSQL driver
github.com/prometheus/client_golang      // Prometheus metrics
github.com/swaggo/swag                   // OpenAPI documentation
```

---

## 🚀 Quick Start

### Prerequisites

- **Go**: 1.24 or higher
- **PostgreSQL**: 16 or higher (or use Docker)
- **Docker** (optional but recommended): 20.10+
- **Make** (optional): For using Makefile commands
- **Git**: For version control

### Installation

#### Option 1: Docker (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone <repository-url>
cd array_interview_day_2

# Create environment file
cp .env.example .env

# Start the application with Docker Compose
docker compose up -d

# API is now running at http://localhost:8080
```

The Docker setup includes:
- PostgreSQL database with automatic migrations
- API server with hot reload (Air)
- Seed data with sample users and accounts
- Persistent volumes for data

See [README.docker.md](README.docker.md) for detailed Docker documentation.

#### Option 2: Local Development

If you prefer to run without Docker:

```bash
# Clone the repository
git clone <repository-url>
cd array_interview_day_2

# Install dependencies
go mod download

# Set up environment variables
cp .env.example .env
# Edit .env with your PostgreSQL credentials

# Run database migrations
# (Ensure PostgreSQL is running)

# Generate API documentation
make docs

# Build and run
make build
./api
```

### Verify Installation

```bash
# Check health endpoint
curl http://localhost:8080/api/v1/health

# Expected response:
# {"status": "healthy", "database": "connected"}

# View API documentation
open http://localhost:8080/docs
```

### Default Test Users

When running with Docker and `SEED_DATABASE=true`, the following test users are available:

| Email | Password | Role |
|-------|----------|------|
| john.doe@example.com | Password123! | admin |
| jane.smith@example.com | Password123! | customer |
| bob.johnson@example.com | Password123! | customer |
| alice.williams@example.com | Password123! | customer |
| charlie.brown@example.com | Password123! | customer |

### First API Request

```bash
# Login to get JWT token
curl.exe -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "password": "Password123!"
  }'

# Response includes accessToken - use it for authenticated requests:
# {
#   "accessToken": "eyJhbGc...",
#   "refreshToken": "...",
#   "tokenType": "Bearer",
#   "expiresAt": "2025-10-24T13:00:00Z"
# }

# Get current user profile (use accessToken from login response)
curl.exe -s http://localhost:8080/api/v1/customers/me -H "Authorization: Bearer <your-token-here>"
```

---

## 🏗️ Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  (Web/Mobile Apps, Postman, cURL)                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        │ HTTPS/TLS
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway/Load Balancer                 │
│  (Rate Limiting, CORS, Security Headers)                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                     Echo HTTP Server                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Middleware Chain                        │   │
│  │  • Request Logging                                   │   │
│  │  • CORS                                              │   │
│  │  • Rate Limiting                                     │   │
│  │  • JWT Authentication                                │   │
│  │  • Request Validation                                │   │
│  │  • Error Handling                                    │   │
│  │  • Prometheus Metrics                                │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  Handlers Layer                      │   │
│  │  • Auth Handler    • Account Handler                │   │
│  │  • Customer Handler • Transaction Handler           │   │
│  │  • Admin Handler    • Docs Handler                  │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                     Services Layer                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  • AuthService      • AccountService                │   │
│  │  • CustomerService  • TransactionService            │   │
│  │  • PasswordService  • TokenService                  │   │
│  │  • AuditService     • ProcessingQueue               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Business Logic, Transaction Management, Domain Rules        │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                  Repositories Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  • UserRepository       • AccountRepository         │   │
│  │  • TransactionRepository • TransferRepository       │   │
│  │  • AuditLogRepository   • RefreshTokenRepository    │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  Data Access Layer, Query Building                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                      GORM ORM                                │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   PostgreSQL Database                        │
│  • Users & Authentication  • Accounts & Balances            │
│  • Transactions & Transfers • Audit Logs                    │
└─────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### 1. Handlers Layer (`internal/handlers/`)
- HTTP request/response handling
- Request binding and validation
- Calling appropriate services
- Response formatting
- Error handling (with middleware)

#### 2. Services Layer (`internal/services/`)
- Business logic implementation
- Transaction orchestration
- Cross-repository operations
- Domain rules enforcement
- Audit logging

#### 3. Repositories Layer (`internal/repositories/`)
- Data access abstraction
- Database query construction
- CRUD operations
- Data mapping (models ↔ database)

#### 4. Models Layer (`internal/models/`)
- Domain entities
- Business objects
- Data structures
- Validation rules

#### 5. Middleware Layer (`internal/middleware/`)
- Request preprocessing
- Authentication/authorization
- Rate limiting
- Logging and metrics
- Error handling

### Design Patterns Used

- **Repository Pattern**: Data access abstraction
- **Service Layer Pattern**: Business logic encapsulation
- **Middleware Pattern**: Cross-cutting concerns
- **Dependency Injection**: Loose coupling between layers
- **Circuit Breaker**: Resilience for external operations
- **Retry Pattern**: Exponential backoff for transient failures
- **Optimistic Locking**: Concurrent transaction safety

---

## 📖 API Documentation

### Interactive Documentation

The API includes comprehensive interactive documentation powered by OpenAPI 3.1 and Scalar UI:

**Access Documentation**: http://localhost:8080/docs

Features:
- Complete endpoint reference with request/response examples
- Try-it-out functionality for testing endpoints
- Authentication flow documentation
- Error code reference
- Response schema visualization

### API Endpoints Overview

#### Authentication & Authorization

```
POST   /api/v1/auth/register         Register new user
POST   /api/v1/auth/login            Login and get JWT token
POST   /api/v1/auth/refresh          Refresh access token
POST   /api/v1/auth/logout           Logout (invalidate token) [Auth Required]
```

#### Account Management

```
POST   /api/v1/accounts                          Create new account [Auth Required]
GET    /api/v1/accounts                          List user's accounts [Auth Required]
GET    /api/v1/accounts/:accountId               Get account details [Auth Required]
PATCH  /api/v1/accounts/:accountId/status        Update account status [Auth Required]
DELETE /api/v1/accounts/:accountId               Close account [Auth Required]
POST   /api/v1/accounts/:accountId/transactions  Create transaction [Auth Required]
GET    /api/v1/accounts/:accountId/transactions  List transactions [Auth Required]
GET    /api/v1/accounts/:accountId/transactions/:id  Get transaction details [Auth Required]
POST   /api/v1/accounts/:accountId/transfer      Initiate transfer [Auth Required]
```

#### Account Summary & Statements

```
GET    /api/v1/accounts/summary                  Get account summary [Auth Required]
GET    /api/v1/accounts/metrics                  Get account metrics [Auth Required]
GET    /api/v1/accounts/:accountId/statements    Get account statement [Auth Required]
```

#### Customer Management (Admin Only)

```
GET    /api/v1/customers/search                  Search customers [Admin]
POST   /api/v1/customers                         Create customer [Admin]
GET    /api/v1/customers/:id                     Get customer profile [Admin]
PUT    /api/v1/customers/:id                     Update customer profile [Admin]
DELETE /api/v1/customers/:id                     Delete customer [Admin]
GET    /api/v1/customers/:id/accounts            Get customer accounts [Admin]
POST   /api/v1/customers/:id/accounts            Create account for customer [Admin]
GET    /api/v1/customers/:id/activity            Get customer activity [Admin]
PUT    /api/v1/customers/:id/password/reset      Reset customer password [Admin]
```

#### Self-Service Customer Endpoints

```
GET    /api/v1/customers/me                      Get my profile [Auth Required]
PUT    /api/v1/customers/me/email                Update my email [Auth Required]
GET    /api/v1/customers/me/accounts             Get my accounts [Auth Required]
GET    /api/v1/customers/me/transfers            Get my transfer history [Auth Required]
GET    /api/v1/customers/me/activity             Get my activity [Auth Required]
PUT    /api/v1/customers/me/password             Update my password [Auth Required]
```

#### Admin Operations

```
GET    /api/v1/admin/users                       List all users [Admin]
GET    /api/v1/admin/users/:userId               Get user details [Admin]
POST   /api/v1/admin/users/:userId/unlock        Unlock user account [Admin]
DELETE /api/v1/admin/users/:userId               Delete user [Admin]
GET    /api/v1/admin/accounts                    List all accounts [Admin]
GET    /api/v1/admin/accounts/:accountId         Get account details [Admin]
GET    /api/v1/admin/users/:userId/accounts      Get user's accounts [Admin]
POST   /api/v1/accounts/:accountId/transfer-ownership  Transfer account ownership [Admin]
```

#### Development Endpoints (Non-Production Only)

```
POST   /api/v1/dev/accounts/:accountId/generate-test-data  Generate test transactions [Auth Required]
DELETE /api/v1/dev/accounts/:accountId/test-data           Clear test data [Auth Required]
```

#### System

```
GET    /api/v1/health                Health check endpoint
GET    /docs                         Interactive API documentation (Scalar UI)
GET    /docs/swagger.json            OpenAPI 3.1 specification
```

### Error Codes

All API errors follow a standardized format with specific error codes. See [docs/error-codes.md](docs/error-codes.md) for the complete error code reference.

Example error response:
```json
{
  "error": {
    "code": "ACCOUNT_003",
    "message": "Insufficient account balance",
    "details": ["Required: $500.00, Available: $250.00"],
    "trace_id": "550e8400-e29b-41d4-a716-446655440002"
  }
}
```

### Postman Collection

A comprehensive Postman collection is available in the `postman/` directory:

```bash
# Import collection into Postman
postman/Array-Banking-API.postman_collection.json

# Import environment variables
postman/Array-Banking-API-Local.postman_environment.json
```

The collection includes:
- Pre-configured authentication flows
- All API endpoints with examples
- Automated tests for happy paths
- Error case scenarios
- Performance assertions

---

## 💻 Development

### Project Structure

```
array_interview_day_2/
├── cmd/
│   └── api/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/                     # Configuration management
│   ├── database/                   # Database initialization
│   ├── dto/                        # Data Transfer Objects
│   ├── errors/                     # Error handling utilities
│   ├── handlers/                   # HTTP handlers
│   ├── middleware/                 # HTTP middleware
│   ├── models/                     # Domain models
│   ├── repositories/               # Data access layer
│   ├── services/                   # Business logic layer
│   └── validation/                 # Request validation
├── db/
│   ├── migrations/                 # Database migration files
│   └── seeds/                      # Seed data
├── docs/                           # Generated API documentation
├── postman/                        # Postman collections
├── scripts/                        # Utility scripts
├── .env.example                    # Environment variables template
├── .air.toml                       # Hot reload configuration
├── docker-compose.yml              # Docker Compose configuration
├── Dockerfile                      # Docker image definition
├── Makefile                        # Build automation
├── go.mod                          # Go module definition
└── README.md                       # This file
```

### Building from Source

```bash
# Install dependencies
go mod download

# Generate API documentation
make docs

# Build binary
make build

# Run binary
./api
```

### Development Workflow

#### With Docker (Hot Reload)

```bash
# Start development environment
docker compose up -d

# Make code changes - Air automatically rebuilds
# View logs
docker compose logs -f api

# Stop environment
docker compose down
```

#### Without Docker

```bash
# Install Air for hot reload
go install github.com/air-verse/air@latest

# Run with hot reload
air

# Or run directly
go run cmd/api/main.go
```

### Code Generation

```bash
# Generate API documentation
make docs

# Generate mocks (if using gomock)
go generate ./...

# Generate Postman collection
make postman
```

### Database Migrations

#### Create New Migration

```bash
# Create migration files
migrate create -ext sql -dir db/migrations -seq add_new_feature
```

This creates two files:
- `XXXXXX_add_new_feature.up.sql` (forward migration)
- `XXXXXX_add_new_feature.down.sql` (rollback migration)

#### Run Migrations

With Docker:
```bash
# Automatic on startup when AUTO_MIGRATE=true
docker compose up -d
```

Manual:
```bash
# Using golang-migrate CLI
migrate -path db/migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" up

# Or via application startup (AUTO_MIGRATE=true)
```

### Environment Variables

Key environment variables (see `.env.example` for complete list):

```bash
# Application
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=arraybank
DB_PASSWORD=arraybank_dev_password
DB_NAME=arraybank_dev
DB_SSL_MODE=disable

# JWT (RSA keypair; base64-encoded PEM. See .env.example for generation.)
JWT_PRIVATE_KEY=<base64-encoded-private-pem>
JWT_PUBLIC_KEY=<base64-encoded-public-pem>
JWT_ISSUER=banking-api
JWT_ACCESS_TOKEN_DURATION=24h
JWT_REFRESH_TOKEN_DURATION=168h

# Features
AUTO_MIGRATE=true
SEED_DATABASE=true

# Server
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# CORS (app reads CORS_ALLOW_ORIGINS)
CORS_ALLOW_ORIGINS=http://localhost:3000,http://localhost:8080

# Rate limiting
RATE_LIMIT_PER_SECOND=10
RATE_LIMIT_BURST=20

# NorthWind Bank Integration
NORTHWIND_BASE_URL=https://northwind.dev.array.io
NORTHWIND_API_KEY=<your-northwind-api-key>
NORTHWIND_POLL_INTERVAL_SECONDS=10

# Regulator Webhook
REGULATOR_WEBHOOK_URL=http://regulator:9000/webhook
REGULATOR_RETRY_INITIAL_SECONDS=2
REGULATOR_RETRY_MAX_SECONDS=60
```

### Code Quality

```bash
# Run linter
make lint

# Or directly with golangci-lint
golangci-lint run ./...

# Format code
go fmt ./...

# Vet code
go vet ./...
```

---

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/handlers/...

# Run with race detection
go test -race ./...

# Run with verbose output
go test -v ./...
```

### Test Coverage

The project aims for >80% test coverage. View coverage report:

```bash
make test-coverage
open coverage.html
```

### Test Structure

```
internal/
├── handlers/
│   ├── account_handler.go
│   └── account_handler_test.go      # Handler unit tests
├── services/
│   ├── auth_service.go
│   └── auth_service_test.go         # Service unit tests
└── repositories/
    ├── user_repository.go
    └── user_repository_test.go      # Repository tests with mocks
```

### Writing Tests

Handler tests should use a **table-driven** style: one test function per handler method, a slice of test cases, and mocks created in the test. New tests should follow this pattern.

Example test structure (uses real `AccountHandler` constructor and mocks):

```go
func TestAccountHandler_CreateAccount(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockAccountService := service_mocks.NewMockAccountServiceInterface(ctrl)
    mockAuditLogger := service_mocks.NewMockAuditLoggerInterface(ctrl)
    mockMetrics := service_mocks.NewMockMetricsRecorderInterface(ctrl)
    handler := NewAccountHandler(mockAccountService, mockAuditLogger, mockMetrics)

    e := echo.New()
    e.Validator = validation.EchoValidator()

    tests := []struct {
        name           string
        requestBody    string
        expectedStatus int
        expectedCode   string
    }{
        {
            name:           "successful account creation",
            requestBody:    `{"accountType":"checking"}`,
            expectedStatus: 201,
        },
        {
            name:           "invalid account type",
            requestBody:    `{"accountType":"invalid"}`,
            expectedStatus: 400,
            expectedCode:   "VALIDATION_001",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(tt.requestBody))
            req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
            rec := httptest.NewRecorder()
            c := e.NewContext(req, rec)
            c.Set("user_id", uuid.New())
            c.Set("user_role", "user")

            if tt.expectedStatus == 201 {
                mockAccountService.EXPECT().CreateAccount(gomock.Any(), "checking", gomock.Any()).Return(&models.Account{ID: uuid.New(), AccountNumber: "1012345678", AccountType: "checking", Status: "active"}, nil)
            }

            err := handler.CreateAccount(c)
            require.NoError(t, err)
            require.Equal(t, tt.expectedStatus, rec.Code)

            if tt.expectedCode != "" {
                var errResp ErrorResponse
                require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
                require.Equal(t, tt.expectedCode, errResp.Error.Code)
            }
        })
    }
}
```

### API Testing with Postman

```bash
# Import collection and environment
# Run collection with Newman (CLI)
newman run postman/Array-Banking-API.postman_collection.json \
  -e postman/Array-Banking-API-Local.postman_environment.json
```

---

## 🚢 Deployment

### Docker Production Build

```bash
# Build production image
docker build -t array-banking-api:1.0 .

# Tag for registry
docker tag array-banking-api:1.0 registry.example.com/array-banking-api:1.0

# Push to registry
docker push registry.example.com/array-banking-api:1.0
```

### Production Deployment with Docker Compose

```bash
# Create production environment file
cp .env.production.example .env.production

# Edit with production values
nano .env.production

# Deploy
docker compose -f docker-compose.prod.yml --env-file .env.production up -d

# Check health
curl https://api.example.com/health
```

### Production Checklist

- [ ] Update `JWT_SECRET` with strong random value (32+ characters)
- [ ] Update `DB_PASSWORD` with strong database password
- [ ] Set `APP_ENV=production`
- [ ] Set `LOG_LEVEL=info` or `warn`
- [ ] Configure `CORS_ALLOWED_ORIGINS` with actual domains
- [ ] Disable seed data: `SEED_DATABASE=false`
- [ ] Enable HTTPS/TLS with reverse proxy (Nginx, Traefik, Caddy)
- [ ] Configure rate limiting appropriately
- [ ] Set up database backups
- [ ] Configure monitoring and alerting
- [ ] Review and update security headers
- [ ] Enable audit logging
- [ ] Configure secret management (HashiCorp Vault, AWS Secrets Manager)

### Kubernetes Deployment

Example Kubernetes manifests:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: banking-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: banking-api
  template:
    metadata:
      labels:
        app: banking-api
    spec:
      containers:
      - name: api
        image: array-banking-api:1.0
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres-service
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: banking-api-secrets
              key: jwt-secret
```

### Monitoring

#### Prometheus Metrics

Metrics available at `/metrics`:

```
# Request metrics
http_requests_total
http_request_duration_seconds

# Business metrics
accounts_created_total
transactions_processed_total
transfers_completed_total

# System metrics
go_goroutines
go_memstats_alloc_bytes
```

#### Health Checks

```bash
# Kubernetes liveness probe
GET /api/v1/health

# Expected response (healthy):
{
  "status": "healthy",
  "database": "connected",
  "timestamp": "2025-10-24T12:00:00Z"
}
```

---

## 📁 Project Structure

### Key Directories

#### `/cmd/api/`
Application entry point. Contains `main.go` which initializes the server, dependencies, and routes.

#### `/internal/`
Private application code not intended for external import.

- **`config/`**: Configuration loading and management
- **`database/`**: Database initialization, connection, migrations
- **`dto/`**: Data Transfer Objects for API requests/responses
- **`errors/`**: Error handling utilities and error codes
- **`handlers/`**: HTTP request handlers (controllers)
- **`middleware/`**: HTTP middleware (auth, logging, rate limiting)
- **`models/`**: Domain models and entities
- **`repositories/`**: Data access layer (database operations)
- **`services/`**: Business logic layer
- **`validation/`**: Request validation logic

#### `/db/`
Database-related files.

- **`migrations/`**: SQL migration files (up/down)
- **`seeds/`**: Seed data for development/testing

#### `/docs/`
Generated API documentation files (OpenAPI/Swagger).

#### `/postman/`
Postman collections and environment files.

#### `/scripts/`
Utility scripts for development and deployment.

### Key Files

- **`go.mod`**: Go module definition with dependencies
- **`Makefile`**: Build automation and common tasks
- **`Dockerfile`**: Multi-stage Docker build
- **`docker-compose.yml`**: Development environment
- **`docker-compose.prod.yml`**: Production environment
- **`.env.example`**: Environment variable template
- **`.air.toml`**: Hot reload configuration (Air)
- **`.gitignore`**: Git ignore patterns

---

## 📄 License

Copyright © 2025 Array. All rights reserved.

This is proprietary software developed for Array's internal use. Unauthorized copying, distribution, or use is strictly prohibited.

---

## 📞 Support

### Documentation Resources

- **API Documentation**: http://localhost:8080/docs
- **Docker Setup**: [README.docker.md](README.docker.md)
- **Error Codes**: [docs/error-codes.md](docs/error-codes.md)
- **DTO Reference**: [internal/dto/README.md](internal/dto/README.md)

### Troubleshooting

#### API Won't Start

```bash
# Check logs
docker compose logs api

# Verify database is running
docker compose ps postgres

# Check environment variables
cat .env

# Reset everything
docker compose down -v
docker compose up -d --build
```

#### Database Connection Issues

```bash
# Test database connection
docker compose exec postgres psql -U arraybank -d arraybank_dev

# Check database health
curl http://localhost:8080/health

# View migration status
docker compose logs api | grep -i migration
```

#### Port Already in Use

```bash
# Find process using port 8080
lsof -ti:8080

# Kill the process
lsof -ti:8080 | xargs kill -9

# Or change port in .env
echo "APP_PORT=8081" >> .env
```

For additional support, refer to the troubleshooting section in [README.docker.md](README.docker.md#troubleshooting).

---

## 🎓 Learning Resources

### Understanding the Codebase

Start with these files to understand the architecture:

1. **`cmd/api/main.go`** - Application initialization and routing
2. **`internal/handlers/auth_handler.go`** - Example handler implementation
3. **`internal/services/auth_service.go`** - Example service with business logic
4. **`internal/repositories/user_repository.go`** - Example data access pattern
5. **`internal/middleware/auth.go`** - JWT authentication implementation

### Key Concepts Demonstrated

- **Clean Architecture**: Separation of concerns with handlers, services, repositories
- **Dependency Injection**: Constructor-based DI for testability
- **Error Handling**: Centralized error handling with trace IDs
- **Security**: JWT auth, password hashing, rate limiting, input validation
- **Testing**: Unit tests, integration tests, mocking patterns
- **API Design**: RESTful principles, versioning, pagination
- **Database**: Migrations, transactions, optimistic locking
- **Observability**: Structured logging, metrics, audit trails
- **DevOps**: Dockerization, CI/CD ready, health checks

---

## NorthWind Bank Integration

This project includes a NorthWind Bank integration module for external account management, fund transfers, and regulator compliance notifications.

For full documentation, architecture details, API endpoints, compliance notes, and testing instructions, see **[NORTHWIND_README.md](NORTHWIND_README.md)**.

### Key capabilities:
- External account validation and registration via NorthWind API
- Fund transfer initiation, monitoring, cancellation, and reversal
- Regulator webhook notifications within 60 seconds of terminal transfer state
- Exponential backoff retries with full audit trail in PostgreSQL
- Background polling and retry workers with graceful shutdown
- Postman collection for end-to-end testing (`postman/NorthWind-Integration.postman_collection.json`)

### How to verify NorthWind features

You can confirm that Authentication, Transfer Funds, and Notify Regulators are working as follows. Prerequisites: API running (e.g. `http://localhost:8080`), `NORTHWIND_API_KEY` set in `.env`, and seed data loaded (e.g. **john.doe@example.com** / **Password123!**).

**1. Authentication**

- Login to get a JWT: `curl.exe -s -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"john.doe@example.com\",\"password\":\"Password123!\"}"` (or use a `login.json` file with `-d "@login.json"` on Windows).
- With the returned `accessToken`, call a protected NorthWind endpoint: `curl.exe -s http://localhost:8080/api/v1/northwind/health -H "Authorization: Bearer <accessToken>"`. Expect 200 and NorthWind health data (or an error if the API key is missing/invalid).
- Without a token: `curl.exe -s -w "\n%{http_code}" http://localhost:8080/api/v1/northwind/health` should return 401.

**2. Transfer funds**

- With a valid JWT: call `GET /api/v1/northwind/health` or `GET /api/v1/northwind/bank` to confirm NorthWind connectivity.
- Register an external account: `POST /api/v1/northwind/external-accounts/validate-and-register` (body must match NorthWind sandbox; see [NORTHWIND_README.md](NORTHWIND_README.md)).
- Create a transfer: `POST /api/v1/northwind/transfers` with required fields (amount, currency, direction, transfer_type, reference_number, source_account, destination_account). Expect 201 and a transfer in `pending` (or similar).
- List and get transfer: `GET /api/v1/northwind/transfers`, `GET /api/v1/northwind/transfers/:id`. After the background poller runs (default every 10s), status should update to a terminal state (e.g. `completed` or `failed`).

**3. Notify regulators**

- After a transfer reaches a terminal state, check the database for proof of notification and delivery attempts:
  - `SELECT id, transfer_id, terminal_status, delivered, attempt_count, created_at FROM regulator_notifications ORDER BY created_at DESC LIMIT 10;`
  - `SELECT notification_id, http_status, error_message, attempted_at FROM regulator_notification_attempts ORDER BY attempted_at DESC LIMIT 20;`
- Optional: run a local webhook receiver (e.g. on port 19999), set `REGULATOR_WEBHOOK_URL=http://localhost:19999/webhook`, then trigger a terminal transfer to see the POST and `delivered=true`. To test retries, use an unreachable URL and confirm rows in `regulator_notification_attempts` and backoff behavior. See [NORTHWIND_README.md](NORTHWIND_README.md) (e.g. "Simulating Regulator Downtime") for payload format and receiver examples.
