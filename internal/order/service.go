package order

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidCustomer = errors.New("customerId is required")
	ErrEmptyOrder      = errors.New("order must contain at least one item")
)

// OrderService owns business rules and coordinates storage and events.
// Dependencies are interfaces so tests and future adapters can replace them.
type OrderService struct {
	repository OrderRepository
	publisher  EventPublisher
	counter    uint64
	// A clock function makes timestamp assertions deterministic in tests.
	now func() time.Time
}

// NewOrderService receives dependencies rather than constructing adapters.
func NewOrderService(repository OrderRepository, publisher EventPublisher) *OrderService {
	return &OrderService{repository: repository, publisher: publisher, now: time.Now}
}

// CreateOrder validates input, saves the order, then publishes its creation.
func (s *OrderService) CreateOrder(ctx context.Context, request CreateOrderRequest) (Order, error) {
	customerID, items, total, err := validateOrderInput(request.CustomerID, request.Items)
	if err != nil {
		return Order{}, err
	}

	// HTTP requests run concurrently. Atomic increment prevents duplicate IDs
	// within this process; durable storage would need a persistent ID strategy.
	id := atomic.AddUint64(&s.counter, 1)
	order := Order{
		ID:         fmt.Sprintf("ORD-%06d", id),
		CustomerID: customerID,
		Items:      items,
		Total:      total,
		Status:     OrderStatusCreated,
		CreatedAt:  s.now().UTC(),
	}

	if err := s.repository.Create(ctx, order); err != nil {
		return Order{}, fmt.Errorf("save order: %w", err)
	}
	// Storage and publishing are separate operations: a publishing error does
	// not undo the write. A durable system can address this with an outbox.
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		return Order{}, fmt.Errorf("publish order-created event: %w", err)
	}
	return order, nil
}

// GetOrder retrieves one order without involving HTTP details.
func (s *OrderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return s.repository.GetByID(ctx, id)
}

// GetOrders retrieves all stored orders.
func (s *OrderService) GetOrders(ctx context.Context) ([]Order, error) {
	return s.repository.GetAll(ctx)
}

// UpdateOrder replaces editable data while preserving identity and creation time.
func (s *OrderService) UpdateOrder(ctx context.Context, id string, request UpdateOrderRequest) (Order, error) {
	existing, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Order{}, err
	}

	customerID, items, total, err := validateOrderInput(request.CustomerID, request.Items)
	if err != nil {
		return Order{}, err
	}

	now := s.now().UTC()
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

// DeleteOrder checks existence, removes the order, then publishes its deletion.
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

// ValidationError identifies invalid order input while preserving the underlying error.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

func validateOrderInput(customerID string, items []OrderItem) (string, []OrderItem, float64, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return "", nil, 0, &ValidationError{Err: ErrInvalidCustomer}
	}
	if len(items) == 0 {
		return "", nil, 0, &ValidationError{Err: ErrEmptyOrder}
	}

	var total float64
	// Copy before trimming so validation never mutates the caller's slice.
	validatedItems := make([]OrderItem, len(items))
	copy(validatedItems, items)

	for i := range validatedItems {
		item := &validatedItems[i]
		item.ProductID = strings.TrimSpace(item.ProductID)
		item.Name = strings.TrimSpace(item.Name)
		if item.ProductID == "" {
			return "", nil, 0, &ValidationError{Err: fmt.Errorf("item %d: productId is required", i+1)}
		}
		if item.Quantity <= 0 {
			return "", nil, 0, &ValidationError{Err: fmt.Errorf("item %d: quantity must be greater than zero", i+1)}
		}
		if math.IsNaN(item.Price) || math.IsInf(item.Price, 0) {
			return "", nil, 0, &ValidationError{Err: fmt.Errorf("item %d: price must be finite", i+1)}
		}
		if item.Price < 0 {
			return "", nil, 0, &ValidationError{Err: fmt.Errorf("item %d: price cannot be negative", i+1)}
		}
		// The server calculates totals instead of trusting a client-supplied amount.
		total += float64(item.Quantity) * item.Price
		if math.IsInf(total, 0) {
			return "", nil, 0, &ValidationError{Err: errors.New("order total is too large")}
		}
	}
	return customerID, validatedItems, total, nil
}
