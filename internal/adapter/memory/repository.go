package memory

import (
	"context"
	"sort"
	"sync"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

type InMemoryOrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{orders: make(map[string]domain.Order)}
}

func (r *InMemoryOrderRepository) Create(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = cloneOrder(order)
	return nil
}

func (r *InMemoryOrderRepository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	if err := ctx.Err(); err != nil {
		return domain.Order{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, exists := r.orders[id]
	if !exists {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

func (r *InMemoryOrderRepository) GetAll(ctx context.Context) ([]domain.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	orders := make([]domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, cloneOrder(order))
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.Before(orders[j].CreatedAt) })
	return orders, nil
}

func (r *InMemoryOrderRepository) Update(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orders[order.ID]; !exists {
		return domain.ErrOrderNotFound
	}
	r.orders[order.ID] = cloneOrder(order)
	return nil
}

func (r *InMemoryOrderRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orders[id]; !exists {
		return domain.ErrOrderNotFound
	}
	delete(r.orders, id)
	return nil
}

// cloneOrder keeps mutable slices and timestamps private to each caller.
func cloneOrder(order domain.Order) domain.Order {
	if order.Items != nil {
		order.Items = append([]domain.OrderItem{}, order.Items...)
	}
	if order.UpdatedAt != nil {
		updatedAt := *order.UpdatedAt
		order.UpdatedAt = &updatedAt
	}
	return order
}

var _ domain.OrderRepository = (*InMemoryOrderRepository)(nil)
