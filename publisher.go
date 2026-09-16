package main

import (
	"context"
	"log"
)

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order Order) error
}

type LogEventPublisher struct{}

func NewLogEventPublisher() *LogEventPublisher { return &LogEventPublisher{} }

func (p *LogEventPublisher) PublishOrderCreated(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log.Printf("EVENT OrderCreated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}
