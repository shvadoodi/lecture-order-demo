package eventlog

import (
	"context"
	"log"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

type LogEventPublisher struct{ logger *log.Logger }

func NewLogEventPublisher(logger *log.Logger) *LogEventPublisher {
	return &LogEventPublisher{logger: logger}
}

func (p *LogEventPublisher) PublishOrderCreated(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderCreated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}

func (p *LogEventPublisher) PublishOrderUpdated(ctx context.Context, order domain.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderUpdated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}

func (p *LogEventPublisher) PublishOrderDeleted(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Printf("EVENT OrderDeleted orderID=%s", id)
	return nil
}

var _ domain.EventPublisher = (*LogEventPublisher)(nil)
