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

## Project structure

```text
lecture-order-demo/
├── .env                             local settings (ignored by Git)
├── .env.example                     tracked configuration template
├── .github/workflows/ci.yml          vet, build, race tests, and generation checks
├── .gitignore
├── cmd/api/main.go                  executable entry point and shutdown signals
├── docs/
│   ├── docs.go                      embeds and registers the Swagger specification
│   ├── swagger.json                 generated API specification
│   └── lecture-guide.md             slide-to-code map and prepared classroom demo
├── internal/
│   ├── adapter/
│   │   ├── eventlog/
│   │   │   ├── publisher.go         log-based order event publisher
│   │   │   └── publisher_test.go
│   │   ├── httpapi/
│   │   │   ├── handler.go           HTTP handlers and service interface
│   │   │   ├── handler_test.go
│   │   │   ├── helpers.go           JSON decoding, responses, and error handling
│   │   │   ├── helpers_test.go
│   │   │   ├── router.go            routes, health, logging, and Swagger UI
│   │   │   ├── service_moq_test.go  generated service mock
│   │   │   └── swagger_test.go
│   │   └── memory/
│   │       ├── repository.go        concurrency-safe in-memory storage
│   │       └── repository_test.go
│   ├── app/
│   │   ├── app.go                   dependency wiring and server lifecycle
│   │   ├── app_test.go
│   │   ├── order_flow_test.go        component tests for publishing failures and retries
│   │   └── server_moq_test.go       generated server mock
│   ├── config/
│   │   ├── config.go                .env loading and environment validation
│   │   └── config_test.go
│   └── order/
│       ├── model.go                 order and request types
│       ├── ports.go                 repository and event publisher interfaces
│       ├── ports_moq_test.go        generated repository and publisher mocks
│       ├── service.go               validation, totals, IDs, and CRUD use cases
│       └── service_test.go
├── tools/generate.go                pinned mock and Swagger generator runner
├── go.mod                           module, Go version, and dependencies
├── go.sum                           dependency checksums
└── README.md
```

Tests live beside the code they exercise in `*_test.go` files. Files ending in `_moq_test.go` are generated mocks used only by tests.

The local workspace also contains `vendor/` for vendored dependencies and `.tmp/` for build outputs and caches. Both are ignored by Git and omitted from the tree above.

### Package boundaries

The `internal/order` package owns the business logic and dependency interfaces without depending on HTTP or storage implementations. The memory and event log adapters implement its `OrderRepository` and `EventPublisher` interfaces. HTTP handlers consume an `OrderService` interface declared in `internal/adapter/httpapi`.

`internal/app` connects the adapters and service, constructs the HTTP server, and manages its lifecycle. `cmd/api` loads configuration and handles operating-system shutdown signals. Go's `internal/` directory keeps these packages private to this module's parent tree.

### Suggested reading order

1. [Order models](internal/order/model.go) and [ports](internal/order/ports.go) define the data and dependencies.
2. [Order service](internal/order/service.go) implements validation and CRUD behavior.
3. [Memory repository](internal/adapter/memory/repository.go) and [event publisher](internal/adapter/eventlog/publisher.go) implement the ports.
4. [HTTP handlers](internal/adapter/httpapi/handler.go), [helpers](internal/adapter/httpapi/helpers.go), and [router](internal/adapter/httpapi/router.go) expose the API.
5. [Application wiring](internal/app/app.go), [configuration](internal/config/config.go), and [entry point](cmd/api/main.go) show how the server starts and stops.

## Requirements

- Go 1.22+ to build, run, and test the app.
- Go downloads module dependencies as needed. If a local `vendor/` directory is present, standard build and test commands use it automatically. You can recreate it with `go mod vendor`.
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

Swagger UI: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).

If you change `HTTP_ADDR`, use the same host and port for the Swagger URL.

Run commands from the repository root. To build a binary:

```bash
go build -o .tmp/order-api ./cmd/api
```

### Configuration

The app reads `.env` from the current working directory when it starts, through [internal/config/config.go](internal/config/config.go). The local `.env` is ignored by Git; [.env.example](.env.example) provides the tracked template. After cloning, create your local file:

```powershell
Copy-Item .env.example .env
```

Edit `.env` to configure the server, then run the app from the repository root. For example:

```dotenv
HTTP_ADDR=:8080
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_SHUTDOWN_TIMEOUT=10s
```

Restart the app after changes.

Configuration precedence is process environment, then `.env`, then the defaults below. A missing `.env` is allowed. The file supports `KEY=value`, optional `export ` prefixes, quoted values, blank lines, and full-line `#` comments. Values are literal; shell expansion and inline comments are unsupported.

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

Open [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) to browse and try the API.
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

Use the coverage commands above to inspect current statement coverage. Coverage scope excludes the process entry point and generated Swagger registration; the application wiring and lifecycle are tested in `internal/app`. Statement coverage does not prove correctness for every possible input.

Component tests in `internal/app/order_flow_test.go` exercise HTTP, the real service, and memory storage. They demonstrate a stored order after publishing fails and separate orders after repeated POST requests. These tests do not connect to a real database or broker.

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

## Logging and troubleshooting

The HTTP middleware logs each request's method and path. HTTP adapters log rejected JSON bodies, invalid order paths, and unsupported methods. The order service logs failures for create, list, get, update, and delete, including the operation, failure stage, reason, and order ID when available.

Successful event methods log `EVENT OrderCreated`, `EVENT OrderUpdated`, or `EVENT OrderDeleted`. When cancellation prevents publishing, the event adapter logs `EVENT_FAILED` with the event name, order ID, and cancellation error. Service publishing failures include `saved=true` or `deleted=true` because the storage change has already happened.

Example messages (timestamps omitted):

```text
create order failed: stage=validation error=customerId is required
get order failed: stage=storage orderID=ORD-000001 error=order not found
EVENT_FAILED OrderUpdated orderID=ORD-000001 error=context canceled
update order failed: stage=publishing orderID=ORD-000001 saved=true error=context canceled
```

Failures still return errors to the caller. HTTP clients receive validation or not-found responses where appropriate and a generic `internal server error` for infrastructure failures. Full request bodies are not logged. The service owns business-failure logs, and a shared HTTP helper maps errors to responses without logging them again. The application injects its logger into both the service and event publisher, while HTTP boundary logs use the standard logger. Event cancellation produces an adapter failure entry and a service entry explaining the stored-data outcome. A larger system should add request correlation IDs.

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

Orders are stored in the map in [internal/adapter/memory/repository.go](internal/adapter/memory/repository.go):

```go
type InMemoryOrderRepository struct {
    mu     sync.RWMutex
    orders map[string]domain.Order
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

## Reading the code in class

Use the [lecture companion](docs/lecture-guide.md) for a slide-to-code map, a prepared demo, and student exercises based on the one-hour PowerPoint. It distinguishes implemented practices from production extensions.

Start with `cmd/api/main.go` and `internal/app/app.go` to see startup and the explicit dependency wiring. Then follow one create request through the HTTP handler, order service, memory repository, and event log publisher. Read the adjacent tests to see success, validation, cancellation, and dependency failures.

Source comments explain why interfaces, request contexts, mutexes, defensive copies, bounded JSON input, and graceful shutdown are used. Constructors receive dependencies, and early returns keep failure paths visible. The configuration parser is a small literal `.env` reader; it is not a full shell parser.

This is a teaching application with production practices, while persistence, reliable messaging, and other deployment requirements remain discussion topics below.

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

This demo has no authentication, payment processing, asynchronous consumers, or idempotency keys. Repeating `POST /orders` creates another order, and `/health` checks process availability only.

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
