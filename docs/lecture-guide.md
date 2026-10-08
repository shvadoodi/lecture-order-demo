# Lecture companion: classroom code to production software

This guide follows Shad_LaSalle_One_Hour_Lecture.pptx. The runnable app stays small: Go's HTTP server, an in-memory order repository, and events written to a log. The slides describe the larger system this example can grow into.

## Slide-to-code map

| Slides | Topic | Code to show | What students can observe |
|---|---|---|---|
| 6–8 | Create an order | [handler.go](../internal/adapter/httpapi/handler.go), [service.go](../internal/order/service.go) | Decode, validate, save, publish, return 201 |
| 9–10 | Failures and data consistency | [order_flow_test.go](../internal/app/order_flow_test.go) | A publishing failure leaves the order saved; repeating POST creates another order |
| 11 | API contracts | [model.go](../internal/order/model.go), [swagger.json](swagger.json) | Request fields, response fields, status codes, error format |
| 12 | Events | [ports.go](../internal/order/ports.go), [publisher.go](../internal/adapter/eventlog/publisher.go) | An interface separates order logic from the publishing adapter |
| 14 | Repeatable delivery | [ci.yml](../.github/workflows/ci.yml) | Vet, build, race tests, and generation checks run in CI |
| 15 | Observability | [service.go](../internal/order/service.go), [router.go](../internal/adapter/httpapi/router.go) | Request logs and failure stages explain what happened |
| 16 | Input boundaries and configuration | [helpers.go](../internal/adapter/httpapi/helpers.go), [config.go](../internal/config/config.go) | Bounded JSON, unknown-field rejection, validated settings |
| 17 | Testing levels | Adjacent unit tests and [order_flow_test.go](../internal/app/order_flow_test.go) | Mocks isolate dependencies; component tests cross HTTP and real memory storage |
| 18–20 | Teamwork and responsible AI | README, tests, and small code changes | Review a change, explain failure paths, then run checks |
| 23, 25–26 | Discussion and prepared demo | The examples below | Investigate duplicate orders and missing downstream actions |

## A short prepared demo

Run go run ./cmd/api from the repository root. Open http://localhost:8080/swagger/index.html. Create an order, inspect the 201 response and OrderCreated log, then retrieve the order by ID.

Send an order with an empty customerId or quantity of zero. Show the 400 response and validation log. The service rejects input before writing or publishing.

For the failure discussion, run this command in another terminal:

```powershell
go test -v ./internal/app -run 'TestPublishingFailureLeavesOrderStored|TestRepeatedPostCreatesSeparateOrders'
```

Read those tests with students. Ask why a failed response may still leave data stored and why blindly retrying POST can duplicate an order. The unavailable broker is simulated in the test; the running app uses a log publisher and has no email or payment service.

Finish by showing dependency wiring in internal/app/app.go and a service unit test. Ask which adapter students would replace to add PostgreSQL and which interface a broker adapter would implement.

## Implemented practices and production extensions

| Implemented in this demo | Additional work for a deployed ordering system |
|---|---|
| Input validation, bounded bodies, stable JSON errors, Swagger | Authentication, authorization, rate limiting, API version policy |
| Mutex-protected storage and defensive copies | Durable database, constraints, migrations, transactions, backup and recovery |
| Unique IDs within one process | IDs safe across restarts and multiple instances |
| Publishing interface and event logs | Transactional outbox, broker, consumers, duplicate detection, delivery retries |
| Tests documenting repeated POST behavior | Idempotency keys with atomic, persistent deduplication and request matching |
| Operation logs and basic health check | Correlation IDs, request latency metrics, tracing, alerts, dependency readiness |
| Environment settings and graceful shutdown | Secret management, deployment configuration, capacity and timeout policies |
| CI build and tests | Deployment automation, security scanning, rollback and monitoring |

The event adapter runs synchronously and logs a message. It demonstrates a boundary rather than asynchronous delivery or real downstream consumers. The health endpoint reports that the process can answer requests; it does not verify external dependencies. Prices use float64 for readability, so a real payment system needs integer cents or a decimal representation.

## Student exercises

1. Add a validation test, then explain why invalid input must not reach storage.
2. Replace the failing publisher in the component test with the log publisher and compare storage, HTTP status, and logs.
3. Sketch a transaction that saves both an order and an outbox record. Explain how a background publisher handles retries.
4. Design an idempotency-key contract. Discuss concurrent requests, changed payloads, retention, and restarts before writing code.
5. Review an AI-generated change: explain it, inspect edge cases, run tests, and decide whether its complexity helps this app.
