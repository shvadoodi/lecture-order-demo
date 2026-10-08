package httpapi

import (
	"context"
	"log"
	"net/http"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

//go:generate go run ../../../tools/generate.go moq -out service_moq_test.go . OrderService

// OrderService describes what HTTP handlers need from the business layer.
// Defining it here keeps handlers independent of the concrete service.
type OrderService interface {
	CreateOrder(ctx context.Context, request domain.CreateOrderRequest) (domain.Order, error)
	GetOrders(ctx context.Context) ([]domain.Order, error)
	GetOrder(ctx context.Context, id string) (domain.Order, error)
	UpdateOrder(ctx context.Context, id string, request domain.UpdateOrderRequest) (domain.Order, error)
	DeleteOrder(ctx context.Context, id string) error
}

// ErrorResponse is the public JSON shape for API failures.
type ErrorResponse struct {
	Error string `json:"error"`
}

// OrderHandler translates HTTP input and output; business rules stay in order.
type OrderHandler struct {
	service OrderService
}

// NewOrderHandler injects the service used for every order request.
func NewOrderHandler(service OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// CreateOrder handles POST /orders.
// @Summary Create an order
// @Tags orders
// @Produce json
// @Accept json
// @Param request body domain.CreateOrderRequest true "Order data"
// @Success 201 {object} domain.Order
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var request domain.CreateOrderRequest
	if err := decodeJSONBody(r, &request); err != nil {
		log.Printf("order request rejected: method=%s path=%q stage=decoding error=%v", r.Method, r.URL.Path, err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	// Forward request cancellation to storage and event publishing.
	order, err := h.service.CreateOrder(r.Context(), request)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

// GetOrders handles GET /orders.
// @Summary Get all orders
// @Tags orders
// @Produce json
// @Success 200 {array} domain.Order
// @Failure 500 {object} ErrorResponse
// @Router /orders [get]
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.GetOrders(r.Context())
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// GetOrderByID handles GET /orders/{id}.
// @Summary Get one order
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	order, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// UpdateOrder handles PUT /orders/{id}.
// @Summary Replace an order
// @Tags orders
// @Produce json
// @Accept json
// @Param id path string true "Order ID"
// @Param request body domain.UpdateOrderRequest true "Order data"
// @Success 200 {object} domain.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{id} [put]
func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	var request domain.UpdateOrderRequest
	if err := decodeJSONBody(r, &request); err != nil {
		log.Printf("order request rejected: method=%s path=%q stage=decoding error=%v", r.Method, r.URL.Path, err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	order, err := h.service.UpdateOrder(r.Context(), id, request)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// DeleteOrder handles DELETE /orders/{id}.
// @Summary Delete an order
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{id} [delete]
func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	err := h.service.DeleteOrder(r.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	// A successful delete has no response body.
	w.WriteHeader(http.StatusNoContent)
}
