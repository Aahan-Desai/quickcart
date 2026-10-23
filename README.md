# QuickCart

A lightweight e-commerce REST API built with Go, focusing on backend development, inventory management, concurrency safety, and reliable order processing.

## Overview

QuickCart is a Go-based backend application that provides RESTful endpoints for product discovery, order creation, inventory validation, and order retrieval. It uses Go's standard library to implement HTTP routing, JSON handling, concurrent access control, and automated testing.

The project demonstrates core backend engineering principles through a minimal e-commerce workflow, with particular emphasis on data consistency, input validation, and thread-safe order processing.

## Features

* **RESTful API:** HTTP endpoints for health checks, product discovery, and order management.
* **Product Catalogue:** List available products and retrieve individual product details.
* **Order Management:** Create orders and retrieve order information.
* **Inventory Validation:** Verify stock availability before processing orders.
* **Atomic Order Processing:** Validate the complete order before modifying inventory to prevent partial stock updates.
* **Concurrency Safety:** Use `sync.RWMutex` to protect shared state during concurrent operations.
* **Server-Side Pricing:** Calculate order totals using catalogue prices rather than client-supplied values.
* **Request Validation:** Handle malformed JSON, reject unknown fields, and enforce request-body size limits.
* **Graceful Shutdown:** Handle termination signals to shut down the HTTP server cleanly.
* **Automated Testing:** Test HTTP behaviour, order validation, and concurrent inventory operations.

## Technology Stack

| Technology      | Purpose                                |
| --------------- | -------------------------------------- |
| Go              | Backend application development        |
| `net/http`      | HTTP server and request routing        |
| `encoding/json` | JSON serialization and deserialization |
| `sync.RWMutex`  | Thread-safe access to shared state     |
| Go Testing      | Unit and HTTP handler testing          |

## Project Structure

```text
QuickCart/
├── cmd/
│   └── quickcart/
│       └── main.go
├── internal/
│   └── ...
├── go.mod
└── README.md
```

The application entry point is located in `cmd/quickcart/`. Internal packages are organised under `internal/`, where applicable.

## Getting Started

### Prerequisites

* Go 1.22 or later

### Installation

Clone the repository and navigate to the project directory:

```bash
git clone <your-repository-url>
cd QuickCart
```

### Run the Application

Start the API server:

```bash
go run ./cmd/quickcart
```

The server listens on `http://localhost:8080`.

Press `Ctrl+C` to shut down the server.

## API Reference

| Method | Endpoint         | Description                    |
| ------ | ---------------- | ------------------------------ |
| `GET`  | `/health`        | Check server health            |
| `GET`  | `/products`      | Retrieve the product catalogue |
| `GET`  | `/products/{id}` | Retrieve a product by ID       |
| `POST` | `/orders`        | Validate and create an order   |
| `GET`  | `/orders/{id}`   | Retrieve an order by ID        |

### Health Check

**Request**

```http
GET /health
```

**Response — `200 OK`**

```json
{
  "status": "ok"
}
```

### List Products

**Request**

```http
GET /products
```

**Example Response**

```json
[
  {
    "id": "p-apple",
    "name": "Apples (1 kg)",
    "price": 350,
    "available_stock": 20
  }
]
```

### Create an Order

**Request**

```http
POST /orders
Content-Type: application/json
```

**Request Body**

```json
{
  "items": [
    {
      "product_id": "p-apple",
      "quantity": 2
    }
  ]
}
```

**Example Response — `201 Created`**

```json
{
  "id": "o-1",
  "items": [
    {
      "product_id": "p-apple",
      "name": "Apples (1 kg)",
      "quantity": 2,
      "unit_price": 350
    }
  ],
  "total": 700,
  "created_at": "2026-10-23T00:00:00Z"
}
```

The server retrieves prices from the product catalogue, calculates the total, validates inventory, and records the order. Product prices and order totals are not accepted from the client.

### Retrieve an Order

**Request**

```http
GET /orders/o-1
```

Returns the order details when the specified order exists.

### Error Responses

Errors are returned as JSON with an explanatory message.

**Example**

```json
{
  "error": "insufficient stock for product \"p-apple\""
}
```

Invalid requests, unknown products, and insufficient inventory are rejected without partially updating stock.

## Order Processing and Concurrency

QuickCart uses Go's `sync.RWMutex` to protect shared product and order data.

Order creation follows these steps:

1. Validate the requested products and quantities.
2. Combine duplicate product entries where applicable.
3. Verify that sufficient inventory is available for every item.
4. Calculate the order total using catalogue prices.
5. Update inventory and save the order.

Inventory changes occur only after the complete order has passed validation. An exclusive lock prevents concurrent order requests from validating and modifying the same inventory against conflicting stock values.

A concurrency test simulates two orders competing for the final available unit and verifies that only one succeeds.

## Monetary Representation

Product prices and order totals use integer minor units instead of floating-point values.

For example, `350` represents ₹3.50 when prices are denominated in paise. This approach avoids floating-point rounding errors in monetary calculations.

## Testing

Run the test suite:

```bash
go test ./...
```

Run the tests with race detection where supported:

```bash
go test -race ./...
```

The test suite includes HTTP behaviour, request validation, order processing, and concurrent inventory access.

## Future Enhancements

Potential extensions include:

* PostgreSQL integration for persistent storage.
* Authentication and authorisation.
* Idempotency keys for duplicate order prevention.
* Pagination for product listings.
* Background workers for asynchronous order processing.
* Structured logging and expanded integration testing.

## License

Add a license if you intend to distribute the project for reuse. Otherwise, remove this section until a license has been selected.
