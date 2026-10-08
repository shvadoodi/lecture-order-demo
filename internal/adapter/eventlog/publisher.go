package eventlog

import (
	"context"
	"log"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

// LogEventPublisher demonstrates the event boundary by writing to a log.
// It does not provide durable delivery like a message broker.
type LogEventPublisher struct {
	logger *log.Logger
}

// NewLogEventPublisher receives a logger so tests can capture event output.
func NewLogEventPublisher(logger *log.Logger) *LogEventPublisher {
	return &LogEventPublisher{logger: logger}
}

// PublishOrderCreated logs the order after it has been stored.
func (p *LogEventPublisher) PublishOrderCreated(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderCreated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}

// PublishOrderUpdated logs the current data after a successful update.
func (p *LogEventPublisher) PublishOrderUpdated(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderUpdated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}

// PublishOrderDeleted needs only the ID because the order has been removed.
func (p *LogEventPublisher) PublishOrderDeleted(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderDeleted orderID=%s", id)
	return nil
}

// Compile-time check that the adapter implements every publishing method.
var _ domain.EventPublisher = (*LogEventPublisher)(nil)
