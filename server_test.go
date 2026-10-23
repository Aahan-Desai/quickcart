package quickcart

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func testServer(products []Product) (*Store, http.Handler) {
	store := NewStore(products)
	return store, NewServer(store).Handler()
}

func request(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestHealth(t *testing.T) {
	_, handler := testServer(nil)
	response := request(t, handler, http.MethodGet, "/health", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", contentType)
	}
}

func TestProductsAndProductLookup(t *testing.T) {
	_, handler := testServer([]Product{{ID: "p-1", Name: "Tea", Price: 125, AvailableStock: 3}})
	list := request(t, handler, http.MethodGet, "/products", "")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("p-1")) {
		t.Fatalf("unexpected product list: status=%d body=%s", list.Code, list.Body.String())
	}
	item := request(t, handler, http.MethodGet, "/products/p-1", "")
	if item.Code != http.StatusOK || !bytes.Contains(item.Body.Bytes(), []byte("Tea")) {
		t.Fatalf("unexpected product: status=%d body=%s", item.Code, item.Body.String())
	}
	missing := request(t, handler, http.MethodGet, "/products/missing", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing product status = %d", missing.Code)
	}
}

func TestCreateAndGetOrder(t *testing.T) {
	store, handler := testServer([]Product{{ID: "p-1", Name: "Tea", Price: 125, AvailableStock: 3}})
	response := request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"p-1","quantity":2,"price":1}]}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown client price field should be rejected, got %d", response.Code)
	}

	response = request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"p-1","quantity":2}]}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var order Order
	if err := json.NewDecoder(response.Body).Decode(&order); err != nil {
		t.Fatal(err)
	}
	if order.Total != 250 || order.Items[0].UnitPrice != 125 {
		t.Fatalf("server total/item price = %d/%d, want 250/125", order.Total, order.Items[0].UnitPrice)
	}
	if product, _ := store.Product("p-1"); product.AvailableStock != 1 {
		t.Fatalf("stock = %d, want 1", product.AvailableStock)
	}
	get := request(t, handler, http.MethodGet, "/orders/"+order.ID, "")
	if get.Code != http.StatusOK {
		t.Fatalf("get order status = %d", get.Code)
	}
}

func TestInvalidQuantity(t *testing.T) {
	_, handler := testServer([]Product{{ID: "p-1", AvailableStock: 1}})
	for _, body := range []string{
		`{"items":[]}`,
		`{"items":[{"product_id":"p-1","quantity":0}]}`,
		`{"items":[{"product_id":"p-1","quantity":-1}]}`,
	} {
		if response := request(t, handler, http.MethodPost, "/orders", body); response.Code != http.StatusBadRequest {
			t.Fatalf("body %s returned %d", body, response.Code)
		}
	}
}

func TestMalformedJSONAndUnsupportedMethod(t *testing.T) {
	_, handler := testServer([]Product{{ID: "p-1", AvailableStock: 1}})
	malformed := request(t, handler, http.MethodPost, "/orders", `{"items":`)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON status = %d", malformed.Code)
	}
	method := request(t, handler, http.MethodDelete, "/products", "")
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method status = %d", method.Code)
	}
}

func TestInsufficientInventoryAndUnknownProduct(t *testing.T) {
	_, handler := testServer([]Product{{ID: "p-1", AvailableStock: 1}})
	insufficient := request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"p-1","quantity":2}]}`)
	if insufficient.Code != http.StatusBadRequest {
		t.Fatalf("insufficient stock status = %d", insufficient.Code)
	}
	unknown := request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"missing","quantity":1}]}`)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown product status = %d", unknown.Code)
	}
}

func TestConcurrentOrdersDoNotOversell(t *testing.T) {
	store, handler := testServer([]Product{{ID: "p-1", AvailableStock: 1}})
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			statuses <- request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"p-1","quantity":1}]}`).Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	created := 0
	for status := range statuses {
		if status == http.StatusCreated {
			created++
		} else if status != http.StatusBadRequest {
			t.Fatalf("unexpected concurrent status %d", status)
		}
	}
	if created != 1 {
		t.Fatalf("created orders = %d, want 1", created)
	}
	if product, _ := store.Product("p-1"); product.AvailableStock != 0 {
		t.Fatalf("stock = %d, want 0", product.AvailableStock)
	}
}

func TestFailedMultiItemOrderDoesNotChangeInventory(t *testing.T) {
	store, handler := testServer([]Product{
		{ID: "p-a", AvailableStock: 2},
		{ID: "p-b", AvailableStock: 1},
	})
	response := request(t, handler, http.MethodPost, "/orders", `{"items":[{"product_id":"p-a","quantity":1},{"product_id":"p-b","quantity":2}]}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
	if product, _ := store.Product("p-a"); product.AvailableStock != 2 {
		t.Fatalf("p-a stock = %d, want 2", product.AvailableStock)
	}
	if product, _ := store.Product("p-b"); product.AvailableStock != 1 {
		t.Fatalf("p-b stock = %d, want 1", product.AvailableStock)
	}
}
