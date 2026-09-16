package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidCustomer = errors.New("customerId is required")
	ErrEmptyOrder      = errors.New("order must contain at least one item")
)

type OrderService struct {
	repository OrderRepository
	publisher  EventPublisher
	counter    uint64
}

func NewOrderService(repository OrderRepository, publisher EventPublisher) *OrderService {
	return &OrderService{repository: repository, publisher: publisher}
}

func (s *OrderService) CreateOrder(ctx context.Context, request CreateOrderRequest) (Order, error) {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	if request.CustomerID == "" {
		return Order{}, ErrInvalidCustomer
	}
	if len(request.Items) == 0 {
		return Order{}, ErrEmptyOrder
	}

	var total float64
	for i, item := range request.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return Order{}, fmt.Errorf("item %d: productId is required", i+1)
		}
		if item.Quantity <= 0 {
			return Order{}, fmt.Errorf("item %d: quantity must be greater than zero", i+1)
		}
		if item.Price < 0 {
			return Order{}, fmt.Errorf("item %d: price cannot be negative", i+1)
		}
		total += float64(item.Quantity) * item.Price
	}

	id := atomic.AddUint64(&s.counter, 1)
	order := Order{
		ID:         fmt.Sprintf("ORD-%06d", id),
		CustomerID: request.CustomerID,
		Items:      request.Items,
		Total:      total,
		Status:     OrderStatusCreated,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repository.Create(ctx, order); err != nil {
		return Order{}, fmt.Errorf("save order: %w", err)
	}
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		return Order{}, fmt.Errorf("publish order-created event: %w", err)
	}
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *OrderService) GetOrders(ctx context.Context) ([]Order, error) {
	return s.repository.GetAll(ctx)
}
