# Lecture Order Demo

A small Go REST API for the lecture **From Classroom Code to Production Software**.

The project intentionally starts with an **in-memory repository** so students can see the architecture without needing Docker, PostgreSQL, Kafka, or cloud credentials.

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
Order stored

Service
    ↓
EventPublisher → log output (Kafka stand-in)
```

The interfaces make it possible to later replace the in-memory repository with PostgreSQL and the log publisher with Kafka/SNS/SQS/EventBridge.

## Requirements

- Go 1.22+

## Run

```bash
go run .
```

The API starts at `http://localhost:8080`.

## Test

```bash
go test ./...
```

## Endpoints

### Health

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"UP"}
```

### Create an order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

Example response:

```json
{
  "id": "ORD-000001",
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
  ],
  "total": 209.97,
  "status": "CREATED",
  "createdAt": "2026-09-16T17:00:00Z"
}
```

The server also logs an event similar to:

```text
EVENT OrderCreated orderID=ORD-000001 customerID=CUS-1001 total=209.97
```

### Get all orders

```bash
curl http://localhost:8080/orders
```

### Get one order

```bash
curl http://localhost:8080/orders/ORD-000001
```

## Where is the data stored?

Orders are stored in this in-memory map:

```go
type InMemoryOrderRepository struct {
    mu     sync.RWMutex
    orders map[string]Order
}
```

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

## Teaching points

This project is designed to demonstrate:

- HTTP and REST
- JSON request/response handling
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
