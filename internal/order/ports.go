package order

import (
	"context"
	"errors"
)

//go:generate go run ../../tools/generate.go moq -out ports_moq_test.go . OrderRepository EventPublisher

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository interface {
	Create(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, order Order) error
	Delete(ctx context.Context, id string) error
}

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order Order) error
	PublishOrderUpdated(ctx context.Context, order Order) error
	PublishOrderDeleted(ctx context.Context, id string) error
}
