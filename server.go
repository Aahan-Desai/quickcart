package quickcart

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxRequestBodyBytes = 1 << 20

type Server struct {
	store *Store
}

func NewServer(store *Store) *Server { return &Server{store: store} }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /products", s.products)
	mux.HandleFunc("GET /products/{id}", s.product)
	mux.HandleFunc("POST /orders", s.createOrder)
	mux.HandleFunc("GET /orders/{id}", s.order)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) products(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListProducts())
}

func (s *Server) product(w http.ResponseWriter, r *http.Request) {
	product, ok := s.store.Product(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()

	var request createOrderRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
		}
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON value")
		return
	}

	order, err := s.store.CreateOrder(request.Items)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) order(w http.ResponseWriter, r *http.Request) {
	order, ok := s.store.Order(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}
