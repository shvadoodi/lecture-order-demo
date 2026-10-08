package httpapi

import (
	"log"
	"net/http"

	// The blank import runs docs.init to register the embedded Swagger document.
	_ "github.com/shvadoodi/lecture-order-demo/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter registers the API endpoints and request logging.
func NewRouter(handler *OrderHandler) http.Handler {
	mux := http.NewServeMux()

	// Swagger UI endpoint
	mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	mux.HandleFunc("/health", method(http.MethodGet, health))
	// Collection routes and single-order routes accept different HTTP methods.
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateOrder(w, r)
		case http.MethodGet:
			handler.GetOrders(w, r)
		default:
			log.Printf("request rejected: method=%s path=%q error=method not allowed", r.Method, r.URL.Path)
			w.Header().Set("Allow", "GET, POST")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetOrderByID(w, r)
		case http.MethodPut:
			handler.UpdateOrder(w, r)
		case http.MethodDelete:
			handler.DeleteOrder(w, r)
		default:
			log.Printf("request rejected: method=%s path=%q error=method not allowed", r.Method, r.URL.Path)
			w.Header().Set("Allow", "GET, PUT, DELETE")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		}
	})

	return loggingMiddleware(mux)
}

// health reports API availability.
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}

// method rejects unsupported methods and advertises the accepted one.
func method(allowed string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowed {
			log.Printf("request rejected: method=%s path=%q error=method not allowed", r.Method, r.URL.Path)
			w.Header().Set("Allow", allowed)
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
			return
		}
		next(w, r)
	}
}

// loggingMiddleware wraps the router so every request is logged in one place.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
