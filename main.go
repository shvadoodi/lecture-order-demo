package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	repository := NewInMemoryOrderRepository()
	publisher := NewLogEventPublisher()
	service := NewOrderService(repository, publisher)
	handler := NewOrderHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", method(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
	}))
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateOrder(w, r)
		case http.MethodGet:
			handler.GetOrders(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})
	mux.HandleFunc("/orders/", method(http.MethodGet, handler.GetOrderByID))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Order API running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func method(allowed string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowed {
			w.Header().Set("Allow", allowed)
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
			return
		}
		next(w, r)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
