package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

type OrderHandler struct {
	service *OrderService
}

func NewOrderHandler(service *OrderService) *OrderHandler { return &OrderHandler{service: service} }

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var request CreateOrderRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	order, err := h.service.CreateOrder(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.GetOrders(r.Context())
	if err != nil {
		log.Printf("get orders error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	order, err := h.service.GetOrder(r.Context(), id)
	if errors.Is(err, ErrOrderNotFound) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "order not found"})
		return
	}
	if err != nil {
		log.Printf("get order error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	var request UpdateOrderRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	order, err := h.service.UpdateOrder(r.Context(), id, request)
	if errors.Is(err, ErrOrderNotFound) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "order not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteOrder(r.Context(), id); errors.Is(err, ErrOrderNotFound) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "order not found"})
		return
	} else if err != nil {
		log.Printf("delete order error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid request body: " + err.Error())
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func orderIDFromPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/orders/"))
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "a single order id is required"})
		return "", false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write response error: %v", err)
	}
}
