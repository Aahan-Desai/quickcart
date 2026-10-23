# QuickCart

QuickCart is a small Go REST API for learning the backend fundamentals behind an e-commerce catalogue and order flow. It deliberately uses only the Go standard library and stores all data in memory, so the important ideas are visible without setup overhead.

## Implemented features

- `net/http` server and Go 1.22 method/path routing.
- JSON request and response handling with consistent JSON errors.
- Seeded product catalogue, product listing, and product lookup.
- Server-calculated order totals: clients submit product IDs and quantities only.
- Whole-order validation before inventory changes.
- A `sync.RWMutex` protecting the shared maps and an exclusive lock around order creation.
- A 1 MiB JSON request-body limit, malformed JSON handling, and automatic `405 Method Not Allowed` responses from `http.ServeMux`.
- Graceful shutdown on Ctrl+C / `SIGTERM`.
- Standard-library HTTP tests, including a concurrent final-stock scenario.

Money values are integer minor units (for example, `350` can represent 350 paise) to avoid floating-point rounding in this learning project.

## Requirements

Install Go **1.22 or later**. The standard-library route patterns such as `GET /products/{id}` require Go 1.22.

Confirm your installation:

```powershell
go version
```

## Run

From this directory:

```powershell
go run ./cmd/quickcart
```

The server listens at `http://localhost:8080`. Press Ctrl+C to shut it down gracefully.

## Test

```powershell
go test ./...
```

For additional data-race detection during learning:

```powershell
go test -race ./...
```

## Verification status

Formatted with Go 1.27.3 and verified with `go test ./...` on Windows. The
live server also started and listened on port 8080. This managed workspace
blocks local HTTP connections, so endpoint requests must be tried in a normal
local terminal. `go test -race ./...` could not run here because CGO is
unavailable, which the Windows race detector requires.

## API

| Method | Path | Success |
| --- | --- | --- |
| GET | `/health` | `200 OK` |
| GET | `/products` | `200 OK` |
| GET | `/products/{id}` | `200 OK` or `404 Not Found` |
| POST | `/orders` | `201 Created` or `400 Bad Request` |
| GET | `/orders/{id}` | `200 OK` or `404 Not Found` |

### Health

```powershell
curl http://localhost:8080/health
```

```json
{"status":"ok"}
```

### List products

```powershell
curl http://localhost:8080/products
```

Example response:

```json
[{"id":"p-apple","name":"Apples (1 kg)","price":350,"available_stock":20}]
```

### Create an order

```powershell
curl -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d '{"items":[{"product_id":"p-apple","quantity":2}]}'
```

Example response:

```json
{"id":"o-1","items":[{"product_id":"p-apple","name":"Apples (1 kg)","quantity":2,"unit_price":350}],"total":700,"created_at":"2026-10-23T00:00:00Z"}
```

Price and total are intentionally absent from the request. The server finds the product price, calculates the total, and records that price on the order. Unknown JSON fields are rejected.

### Read an order

```powershell
curl http://localhost:8080/orders/o-1
```

### Error shape

Invalid input, unknown products, or insufficient stock return `400` with a safe message:

```json
{"error":"insufficient stock for product \"p-apple\""}
```

## How the mutex makes ordering atomic

The store uses `RLock` for reads. `CreateOrder` takes the write lock for the short critical section that:

1. validates every item and combines duplicate product lines;
2. checks stock for the whole request;
3. decrements stock and saves the new order.

No inventory is changed during validation. If any line fails, the method returns before the decrement loop, so a multi-item order cannot leave partial stock changes. While one order holds the write lock, another order cannot validate against the same old stock level. The concurrency test starts two orders together for a single remaining unit and asserts that exactly one gets `201 Created`.

## Current limitations

- Data is lost when the process stops; there is no database or persistence.
- There is no authentication, payment processing, idempotency key, pagination, product administration, or reservation expiry.
- IDs are in-process counters, not globally distributed IDs.
- Error messages are suitable for this learning API, not a complete production error taxonomy.

These are intentional omissions for a small, explainable first version.
