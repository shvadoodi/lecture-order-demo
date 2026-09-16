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

## Requirements

- Go 1.22+

## Run

```bash
go run .
```

The API starts at:

```text
http://localhost:8080
```

## Test

```bash
go test ./...
```

## Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/orders` | Create an order |
| GET | `/orders` | Get all orders |
| GET | `/orders/{id}` | Get one order |
| PUT | `/orders/{id}` | Replace/edit an order |
| DELETE | `/orders/{id}` | Delete an order |

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
- HTTP status codes such as `200`, `201`, `204`, `400`, `404`, and `405`
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
