package quickcart

import "time"

// Product is an item that can be purchased from the catalogue.
type Product struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Price          int64  `json:"price"`
	AvailableStock int    `json:"available_stock"`
}

// OrderItem is a product snapshot stored on an order. Price is always chosen
// by the server from the current product catalogue.
type OrderItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// Order records a successful inventory reservation and purchase.
type Order struct {
	ID        string      `json:"id"`
	Items     []OrderItem `json:"items"`
	Total     int64       `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
}

type createOrderRequest struct {
	Items []requestedItem `json:"items"`
}

type requestedItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}
