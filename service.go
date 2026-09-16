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
	customerID, items, total, err := validateOrderInput(request.CustomerID, request.Items)
	if err != nil {
		return Order{}, err
	}

	id := atomic.AddUint64(&s.counter, 1)
	order := Order{
		ID:         fmt.Sprintf("ORD-%06d", id),
		CustomerID: customerID,
		Items:      items,
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

func (s *OrderService) UpdateOrder(ctx context.Context, id string, request UpdateOrderRequest) (Order, error) {
	existing, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Order{}, err
	}

	customerID, items, total, err := validateOrderInput(request.CustomerID, request.Items)
	if err != nil {
		return Order{}, err
	}

	now := time.Now().UTC()
	existing.CustomerID = customerID
	existing.Items = items
	existing.Total = total
	existing.UpdatedAt = &now

	if err := s.repository.Update(ctx, existing); err != nil {
		return Order{}, fmt.Errorf("update order: %w", err)
	}
	if err := s.publisher.PublishOrderUpdated(ctx, existing); err != nil {
		return Order{}, fmt.Errorf("publish order-updated event: %w", err)
	}
	return existing, nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	if err := s.publisher.PublishOrderDeleted(ctx, id); err != nil {
		return fmt.Errorf("publish order-deleted event: %w", err)
	}
	return nil
}

func validateOrderInput(customerID string, items []OrderItem) (string, []OrderItem, float64, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return "", nil, 0, ErrInvalidCustomer
	}
	if len(items) == 0 {
		return "", nil, 0, ErrEmptyOrder
	}

	var total float64
	validatedItems := make([]OrderItem, len(items))
	copy(validatedItems, items)

	for i := range validatedItems {
		validatedItems[i].ProductID = strings.TrimSpace(validatedItems[i].ProductID)
		validatedItems[i].Name = strings.TrimSpace(validatedItems[i].Name)
		if validatedItems[i].ProductID == "" {
			return "", nil, 0, fmt.Errorf("item %d: productId is required", i+1)
		}
		if validatedItems[i].Quantity <= 0 {
			return "", nil, 0, fmt.Errorf("item %d: quantity must be greater than zero", i+1)
		}
		if validatedItems[i].Price < 0 {
			return "", nil, 0, fmt.Errorf("item %d: price cannot be negative", i+1)
		}
		total += float64(validatedItems[i].Quantity) * validatedItems[i].Price
	}
	return customerID, validatedItems, total, nil
}
