# NorthWind Bank Integration Module

> Extension to the Array Banking API for external bank transfers via the NorthWind Bank API.

---

## Overview

This module adds three major capabilities on top of the existing Array Banking API:

1. **External Account Registry** - Register and validate external bank accounts via NorthWind before transferring funds.
2. **External Transfers** - Initiate, monitor, cancel, and reverse ACH/wire transfers through NorthWind Bank.
3. **Regulator Webhook Notifications** - Automatically notify a regulator endpoint whenever a transfer reaches a terminal state (COMPLETED or FAILED) within 60 seconds, with retry logic and full audit trail.

---

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.24+ (for local development)
- A NorthWind API key

### Running with Docker

```bash
# Set your NorthWind API key
export NORTHWIND_API_KEY=your-api-key-here

# Start everything
docker-compose up --build
```

The API will be available at `http://localhost:8080`.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NORTHWIND_BASE_URL` | `https://northwind.dev.array.io` | NorthWind Bank API base URL |
| `NORTHWIND_API_KEY` | (required) | API key for NorthWind authentication |
| `NORTHWIND_POLL_INTERVAL_SECONDS` | `10` | How often to poll NorthWind for transfer status updates |
| `REGULATOR_WEBHOOK_URL` | `http://regulator:9000/webhook` | URL to POST regulator notifications |
| `REGULATOR_RETRY_INITIAL_SECONDS` | `2` | Initial backoff for failed regulator delivery |
| `REGULATOR_RETRY_MAX_SECONDS` | `60` | Maximum backoff cap for retries |

All existing environment variables (DB, JWT, etc.) remain unchanged. See `.env.example`.

### Running Tests

```bash
go test ./internal/integrations/northwind/... -v
go test ./internal/services/... -run TestRegulator -v
```

### Generating Swagger Docs

```bash
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal -ot yaml,json --v3.1
```

---

## Architecture

### Package Structure

```
internal/
├── config/config.go                    # Extended with NorthWind + Regulator config
├── integrations/northwind/
│   ├── client.go                       # HTTP client for NorthWind API
│   ├── client_test.go                  # Client unit tests with httptest
│   └── models.go                       # Request/response models matching NorthWind Swagger
├── models/
│   ├── northwind_external_account.go   # GORM model for registered external accounts
│   ├── northwind_transfer.go           # GORM model for tracked external transfers
│   └── regulator_notification.go       # GORM models for notifications + attempts
├── repositories/
│   ├── interfaces.go                   # Extended with 4 new repository interfaces
│   ├── northwind_external_account_repository.go
│   ├── northwind_transfer_repository.go
│   └── regulator_notification_repository.go
├── services/
│   ├── northwind_account_service.go    # Validate + register external accounts
│   ├── northwind_transfer_service.go   # Create + manage external transfers
│   ├── northwind_polling_service.go    # Background poller for transfer status
│   ├── regulator_service.go            # Webhook delivery with retry + audit
│   └── regulator_service_test.go       # Backoff/retry unit tests
├── handlers/
│   └── northwind_handler.go            # HTTP handlers for all NorthWind endpoints
└── errors/codes.go                     # Extended with NORTHWIND_* error codes
```

### Database Schema (4 new tables)

Migration files in `db/migrations/`:

| Table | Description |
|---|---|
| `northwind_external_accounts` | Registered external bank accounts, validated via NorthWind |
| `northwind_transfers` | External transfers with full lifecycle tracking |
| `regulator_notifications` | Webhook notification records with retry scheduling |
| `regulator_notification_attempts` | Individual delivery attempt audit records |

### Background Workers

Two new goroutines are started alongside the existing `TransactionProcessingService`:

1. **NorthWind Polling Service** (`northwind_polling_service.go`)
   - Runs every `NORTHWIND_POLL_INTERVAL_SECONDS` (default 10s)
   - Fetches PENDING/PROCESSING transfers from local DB
   - Calls NorthWind `GET /external/transfers/{id}` for each
   - Updates local status on change
   - Triggers regulator notification on terminal states

2. **Regulator Retry Service** (`regulator_service.go`)
   - Runs every 5 seconds
   - Picks up undelivered notifications where `next_attempt_at <= now()`
   - Attempts HTTP POST to regulator webhook
   - Records every attempt in `regulator_notification_attempts` (audit proof)
   - Uses exponential backoff with jitter (2s initial, 60s cap)

### Data Flow

```
User -> POST /northwind/transfers
          |
          v
   NorthwindTransferService
      1. ValidateTransfer (NorthWind API)
      2. GetAccountBalance (NorthWind API)
      3. InitiateTransfer (NorthWind API)
      4. Store in northwind_transfers (PENDING)
          |
          v  (background, every 10s)
   NorthwindPollingService
      1. Fetch PENDING transfers from DB
      2. GetTransferStatus (NorthWind API)
      3. Update DB if status changed
      4. If COMPLETED/FAILED -> RegulatorService
          |
          v
   RegulatorService
      1. Create regulator_notifications row
      2. Immediately attempt POST to webhook (first attempt within seconds)
      3. If fails, schedule retry with exponential backoff
      4. Every attempt logged in regulator_notification_attempts
```

---

## API Endpoints

All endpoints are under `/api/v1/northwind` and require JWT authentication (Bearer token).

### Bank Info & Health
| Method | Endpoint | Description |
|---|---|---|
| GET | `/northwind/bank` | Get NorthWind bank information |
| GET | `/northwind/domains` | Get NorthWind domains |
| GET | `/northwind/health` | Check NorthWind API health |

### External Accounts
| Method | Endpoint | Description |
|---|---|---|
| POST | `/northwind/external-accounts/validate-and-register` | Validate and register an external account |
| GET | `/northwind/external-accounts` | List user's registered external accounts |
| GET | `/northwind/external-accounts/accessible` | List accessible accounts from NorthWind (passthrough) |

### Transfers
| Method | Endpoint | Description |
|---|---|---|
| POST | `/northwind/transfers` | Initiate a new external transfer |
| GET | `/northwind/transfers` | List user's transfers (with filters) |
| GET | `/northwind/transfers/:id` | Get specific transfer details |
| POST | `/northwind/transfers/:id/cancel` | Cancel a pending transfer |
| POST | `/northwind/transfers/:id/reverse` | Reverse a completed transfer |

### Dev Only
| Method | Endpoint | Description |
|---|---|---|
| POST | `/northwind/reset` | Reset NorthWind sandbox state (non-production) |

---

## Compliance: 60-Second Notification Requirement

The system is designed to meet the requirement of notifying the regulator within 60 seconds of a transfer reaching a terminal state:

1. **Immediate first attempt**: When the polling service detects a terminal status, `CreateAndSendNotification()` creates the DB record AND immediately attempts HTTP delivery in the same call.

2. **Idempotency**: A unique constraint on `(transfer_id, terminal_status)` prevents duplicate notifications. The `event_id` in the payload allows the regulator to deduplicate.

3. **Retry with exponential backoff**: If the regulator is down, retries are scheduled with exponential backoff (2s, 4s, 8s, 16s, 32s, capped at 60s) plus jitter to avoid thundering herd.

4. **Audit proof**: Every single delivery attempt is recorded in `regulator_notification_attempts` with timestamp, HTTP status, error message, and response body (truncated to 1KB).

5. **At-least-once delivery**: The system guarantees at-least-once delivery. The regulator should handle duplicates using the `event_id`.

### Notification Payload Format

```json
{
  "event_id": "uuid",
  "transfer_id": "uuid (local)",
  "northwind_transfer_id": "uuid (NorthWind)",
  "status": "COMPLETED|FAILED",
  "amount": 1000.00,
  "currency": "USD",
  "direction": "INBOUND|OUTBOUND",
  "transfer_type": "ACH",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Tradeoffs & Design Decisions

1. **Polling vs Webhooks from NorthWind**: We use polling because the NorthWind API doesn't offer webhooks. The poll interval is configurable (default 10s). This introduces a small delay but is reliable and simple.

2. **Immediate + retry for regulator**: The first notification attempt is synchronous within the polling cycle. This minimizes latency while the background retry loop handles failures. The 5-second retry check interval plus immediate first attempt means typical notification latency is under 15 seconds.

3. **GORM for repositories**: Consistent with existing codebase. We reuse the same DB connection and patterns.

4. **No external message queue**: We use Postgres-backed polling instead of Kafka/RabbitMQ. This keeps the deployment simple (single DB) while providing durability. For higher scale, a message queue could replace the polling tables.

5. **Balance check is best-effort**: If the NorthWind balance endpoint fails, we proceed with the transfer and let NorthWind reject it. This avoids blocking transfers on transient API errors.

6. **Separate notification + attempt tables**: The `regulator_notifications` table tracks scheduling state while `regulator_notification_attempts` provides immutable audit records. This separation makes audit queries simple and prevents update conflicts.

---

## Postman Collection

Import `postman/NorthWind-Integration.postman_collection.json` for a ready-to-use collection demonstrating:

1. User registration and login (JWT)
2. External account validation and registration
3. Transfer creation and status polling
4. Transfer cancellation and reversal
5. NorthWind health and reset endpoints

The collection automatically saves the JWT token and transfer IDs between requests.

---

## Simulating Regulator Downtime

To test the retry behavior:

1. Set `REGULATOR_WEBHOOK_URL=http://localhost:19999/webhook` (unreachable)
2. Create a transfer and wait for it to complete
3. Observe retry attempts in logs and in the `regulator_notification_attempts` table:

```sql
SELECT * FROM regulator_notification_attempts ORDER BY attempted_at DESC;
SELECT * FROM regulator_notifications WHERE delivered = false;
```

4. Then start a listener on port 19999 and watch the next retry succeed:

```bash
# Simple webhook receiver for testing
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers['Content-Length'])
        print(self.rfile.read(length).decode())
        self.send_response(200)
        self.end_headers()
HTTPServer(('', 19999), H).serve_forever()
"
```
