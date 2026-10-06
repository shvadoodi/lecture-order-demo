# Lecture Order Demo

A small Go REST API for the lecture **From Classroom Code to Production Software**.

The project intentionally uses an **in-memory repository** so students can see the architecture without needing Docker, PostgreSQL, Kafka, or cloud credentials.

## Architecture

```text
HTTP request
    ↓
Handler
    ↓
Service
    ↓
Repository → in-memory map (RAM)
    ↓
Order stored / updated / deleted

Service
    ↓
EventPublisher → log output (Kafka stand-in)
```

The interfaces make it possible to later replace the in-memory repository with PostgreSQL and the log publisher with Kafka, SNS, SQS, or EventBridge.

## Code layout

```text
cmd/api/                    executable entry point and OS signal handling
internal/
  app/                      dependency wiring and graceful server lifecycle
  config/                   environment configuration and validation
  order/                    models, validation, use cases, and dependency interfaces
  adapter/
    httpapi/                HTTP handlers, routes, JSON helpers, and Swagger UI
    memory/                 concurrency-safe in-memory repository
    eventlog/               log-based event publisher
docs/                       embedded Swagger specification
tools/generate.go           pinned mock and Swagger generator runner
.github/workflows/ci.yml     build, vet, unit tests, race detection, generation checks
```

The `order` package has no HTTP or storage implementation dependencies. Adapters implement its repository and publisher interfaces. HTTP handlers consume an `OrderService` interface. `internal/app` assembles the concrete implementations, while `cmd/api` loads configuration and handles shutdown signals.

Tests live beside the code they exercise. Files ending in `_moq_test.go` are generated mocks, used only by tests.

## Requirements

- Go 1.22+ to build, run, and test the app.
- Generation downloads a pinned Go 1.23.12 toolchain and pinned generators on first use; network access is required for that first download. No separate `moq` or `swag` installation is needed.
- Race detection requires cgo and a C compiler.

## Run

```bash
go run ./cmd/api
```

The API starts at:

```text
http://localhost:8080
```

Run commands from the repository root. To build a binary:

```bash
go build -o .tmp/order-api ./cmd/api
```

### Configuration

| Environment variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Listen address |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Maximum time to read request headers |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Grace period for active requests during shutdown |

Timeouts must be positive Go durations such as `500ms`, `5s`, or `1m`. An explicitly empty listen address is rejected.

For example, in PowerShell:

```powershell
$env:HTTP_ADDR = ":9090"
$env:HTTP_SHUTDOWN_TIMEOUT = "15s"
go run ./cmd/api
```

Ctrl+C or SIGTERM initiates graceful shutdown. If the grace period expires, the server closes active connections.

## Swagger UI

Open http://localhost:8080/swagger/index.html to browse and try the API.
The Swagger JSON is served at `/swagger/doc.json`. Generated docs are included in `docs/`, so `go run ./cmd/api` works without installing the generator.

After changing API annotations in `cmd/api/main.go` or `internal/adapter/httpapi/`, or request/response models in `internal/order/`, regenerate the specification:

```bash
go run tools/generate.go swag init -g main.go -d cmd/api,internal/adapter/httpapi,internal/order -o docs --parseInternal --outputTypes json
```

The generator writes only `docs/swagger.json`; `docs/docs.go` embeds that file and registers it with Swagger at startup.

## Test

```bash
go test ./...
go vet ./...
```

### Unit tests and mocks

Service tests use [moq](https://github.com/matryer/moq) mocks for the repository and publisher. HTTP tests mock the service, and lifecycle tests mock the server. Unexpected unconfigured mock calls panic, and tests assert payloads, call counts, context forwarding, and operation order.

Coverage includes CRUD success and dependency failures, all input validation rules, JSON and body-size boundaries, status codes, route methods, missing IDs, context cancellation, mutable-data isolation, concurrent repository access and ID generation, event log output, configuration, server wiring, graceful shutdown, and forced close. Swagger tests exercise the application router.

To regenerate the included mocks after changing interfaces:

```bash
go generate ./...
```

The runner pins `moq` to v0.4.0 and Swagger generation to v1.16.6. These tools do not add runtime dependencies to the app.

### Coverage and concurrency checks

```bash
go test -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out
go tool cover -html=coverage.out
go test -race ./...
```

The unit suite covers 100% of statements in each `internal` package. Coverage scope excludes the process entry point and generated Swagger registration; the application wiring and lifecycle are tested in `internal/app`. Statement coverage does not prove correctness for every possible input.

CI runs vet, build, tests with race detection, and mock/Swagger regeneration checks on Linux.

## Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/orders` | Create an order |
| GET | `/orders` | Get all orders |
| GET | `/orders/{id}` | Get one order |
| PUT | `/orders/{id}` | Replace/edit an order |
| DELETE | `/orders/{id}` | Delete an order |

## Validation and error responses

Create and update requests accept `customerId` and `items`. Each item accepts `productId`, `name`, `quantity`, and `price`.

- `customerId` and each `productId` must be non-empty after trimming whitespace.
- `items` must contain at least one item.
- `quantity` must be a positive integer.
- `price` must be finite and non-negative; zero is allowed. The calculated total must also be finite.
- `name` is optional. Customer IDs, product IDs, and item names are trimmed.
- Unknown JSON fields, malformed JSON, and multiple JSON values are rejected.
- Request bodies must be JSON objects of at most 1 MiB; `null` and arrays are rejected.

The service calculates `total` and assigns the order ID, status, and timestamps. Clients cannot supply those fields in create or update requests.

| Status | Meaning |
|---|---|
| `400` | Invalid JSON, invalid order input, or an invalid order ID path |
| `404` | Order does not exist |
| `405` | Unsupported method; the `Allow` header lists supported methods |
| `500` | Repository or event publishing failure; details are logged on the server |

Order endpoint errors use a JSON body such as:

```json
{"error": "customerId is required"}
```

## PowerShell examples

These examples are recommended when running the lecture demo on Windows PowerShell.

### Health

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
```

### Create an order

```powershell
$body = @'
{
  "customerId": "CUS-1001",
  "items": [
    {
      "productId": "P-100",
      "name": "Mechanical Keyboard",
      "quantity": 1,
      "price": 129.99
    },
    {
      "productId": "P-200",
      "name": "Wireless Mouse",
      "quantity": 2,
      "price": 39.99
    }
  ]
}
'@

Invoke-RestMethod `
  -Uri "http://localhost:8080/orders" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body
```

The server logs an event similar to:

```text
EVENT OrderCreated orderID=ORD-000001 customerID=CUS-1001 total=209.97
```

### Get all orders

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/orders" -Method Get
```

### Get one order

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/orders/ORD-000001" -Method Get
```

### Edit an order

`PUT` replaces the editable order data: `customerId` and the complete `items` array. The order ID and `createdAt` remain unchanged, while `updatedAt` is added.

```powershell
$body = @'
{
  "customerId": "CUS-2002",
  "items": [
    {
      "productId": "P-300",
      "name": "27-inch Monitor",
      "quantity": 2,
      "price": 249.99
    }
  ]
}
'@

Invoke-RestMethod `
  -Uri "http://localhost:8080/orders/ORD-000001" `
  -Method Put `
  -ContentType "application/json" `
  -Body $body
```

The total is recalculated by the service rather than accepted from the client.

The server logs:

```text
EVENT OrderUpdated orderID=ORD-000001 customerID=CUS-2002 total=499.98
```

### Delete an order

```powershell
Invoke-RestMethod `
  -Uri "http://localhost:8080/orders/ORD-000001" `
  -Method Delete
```

A successful delete returns HTTP **204 No Content**.

The server logs:

```text
EVENT OrderDeleted orderID=ORD-000001
```

If you try to get the same order afterward, the API returns HTTP **404 Not Found**.

## curl examples for Bash / WSL

### Create

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customerId":"CUS-1001","items":[{"productId":"P-100","name":"Mechanical Keyboard","quantity":1,"price":129.99}]}'
```

### Edit

```bash
curl -X PUT http://localhost:8080/orders/ORD-000001 \
  -H "Content-Type: application/json" \
  -d '{"customerId":"CUS-2002","items":[{"productId":"P-300","name":"Monitor","quantity":2,"price":249.99}]}'
```

### Delete

```bash
curl -X DELETE http://localhost:8080/orders/ORD-000001
```

## Where is the data stored?

Orders are stored in this in-memory map:

```go
type InMemoryOrderRepository struct {
    mu     sync.RWMutex
    orders map[string]Order
}
```

The repository protects the map with a mutex and copies item slices and update timestamps on reads and writes, so callers cannot mutate stored orders through returned values.

That means the data exists only while the Go process is running. Restart the application and all orders disappear.

This is deliberate for the lecture. It gives us a natural progression:

```text
In-memory map
      ↓
PostgreSQL
      ↓
Event streaming
      ↓
Distributed services
      ↓
Cloud deployment
```

## CRUD flow

```text
POST /orders
    ↓
CreateOrder
    ↓
repository.Create
    ↓
OrderCreated event

PUT /orders/{id}
    ↓
UpdateOrder
    ↓
validate + recalculate total
    ↓
repository.Update
    ↓
OrderUpdated event

DELETE /orders/{id}
    ↓
DeleteOrder
    ↓
repository.Delete
    ↓
OrderDeleted event
```

## Teaching points

This project demonstrates:

- HTTP and REST
- CRUD operations
- JSON request/response handling
- HTTP status codes such as `200`, `201`, `204`, `400`, `404`, `405`, and `500`
- validation
- separation of concerns
- handler/service/repository layers
- dependency inversion through interfaces
- concurrency-safe in-memory storage
- event-driven thinking
- health checks
- middleware
- unit testing
- how a small application evolves toward production architecture

## Production discussion

Persistence and event publishing are separate operations. If publishing fails after a write, the API returns `500` even though the order was already created, updated, or deleted. A transactional outbox would address this gap when adding durable storage and messaging.

For a real production system, the next steps would include:

- PostgreSQL instead of memory
- integer cents or a decimal type instead of `float64` for money
- durable event publishing using a transactional outbox
- Kafka/SNS/SQS/EventBridge instead of log output
- authentication and authorization
- structured logging, metrics, and tracing
- Docker/container deployment
- CI/CD
- cloud infrastructure
