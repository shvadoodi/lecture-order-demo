package main

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository interface {
	Create(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, order Order) error
	Delete(ctx context.Context, id string) error
}

type InMemoryOrderRepository struct {
	mu     sync.RWMutex
	orders map[string]Order
}

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{orders: make(map[string]Order)}
}

func (r *InMemoryOrderRepository) Create(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryOrderRepository) GetByID(ctx context.Context, id string) (Order, error) {
	if err := ctx.Err(); err != nil {
		return Order{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, exists := r.orders[id]
	if !exists {
		return Order{}, ErrOrderNotFound
	}
	return order, nil
}

func (r *InMemoryOrderRepository) GetAll(ctx context.Context) ([]Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	orders := make([]Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.Before(orders[j].CreatedAt) })
	return orders, nil
}

func (r *InMemoryOrderRepository) Update(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orders[order.ID]; !exists {
		return ErrOrderNotFound
	}
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryOrderRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orders[id]; !exists {
		return ErrOrderNotFound
	}
	delete(r.orders, id)
	return nil
}
