package quickcart

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Store is an in-memory catalogue and order store. It is safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	products map[string]Product
	orders   map[string]Order
	nextID   uint64
}

func NewStore(products []Product) *Store {
	byID := make(map[string]Product, len(products))
	for _, product := range products {
		byID[product.ID] = product
	}
	return &Store{products: byID, orders: make(map[string]Order)}
}

func SeededStore() *Store {
	return NewStore([]Product{
		{ID: "p-apple", Name: "Apples (1 kg)", Price: 350, AvailableStock: 20},
		{ID: "p-coffee", Name: "Coffee beans (250 g)", Price: 899, AvailableStock: 10},
		{ID: "p-notebook", Name: "Notebook", Price: 249, AvailableStock: 30},
	})
}

func (s *Store) ListProducts() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	products := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	sort.Slice(products, func(i, j int) bool { return products[i].ID < products[j].ID })
	return products
}

func (s *Store) Product(id string) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	product, ok := s.products[id]
	return product, ok
}

func (s *Store) Order(id string) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	return order, ok
}

// CreateOrder validates every line before changing inventory. A single write
// lock makes validation, stock decrement, and order creation one atomic action.
func (s *Store) CreateOrder(requested []requestedItem) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(requested) == 0 {
		return Order{}, fmt.Errorf("order must contain at least one item")
	}

	quantities := make(map[string]int, len(requested))
	for _, item := range requested {
		if item.ProductID == "" || item.Quantity <= 0 {
			return Order{}, fmt.Errorf("each item requires a product_id and positive quantity")
		}
		product, exists := s.products[item.ProductID]
		if !exists {
			return Order{}, fmt.Errorf("product %q was not found", item.ProductID)
		}
		quantities[item.ProductID] += item.Quantity
		if quantities[item.ProductID] > product.AvailableStock {
			return Order{}, fmt.Errorf("insufficient stock for product %q", item.ProductID)
		}
	}

	items := make([]OrderItem, 0, len(quantities))
	var total int64
	for productID, quantity := range quantities {
		product := s.products[productID]
		items = append(items, OrderItem{ProductID: product.ID, Name: product.Name, Quantity: quantity, UnitPrice: product.Price})
		total += product.Price * int64(quantity)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProductID < items[j].ProductID })
	for productID, quantity := range quantities {
		product := s.products[productID]
		product.AvailableStock -= quantity
		s.products[productID] = product
	}

	s.nextID++
	order := Order{
		ID:        fmt.Sprintf("o-%d", s.nextID),
		Items:     items,
		Total:     total,
		CreatedAt: time.Now().UTC(),
	}
	s.orders[order.ID] = order
	return order, nil
}
