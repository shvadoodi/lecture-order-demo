package main

import (
	"context"
	"log"
)

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order Order) error
	PublishOrderUpdated(ctx context.Context, order Order) error
	PublishOrderDeleted(ctx context.Context, id string) error
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

func (p *LogEventPublisher) PublishOrderUpdated(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log.Printf("EVENT OrderUpdated orderID=%s customerID=%s total=%.2f", order.ID, order.CustomerID, order.Total)
	return nil
}

func (p *LogEventPublisher) PublishOrderDeleted(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log.Printf("EVENT OrderDeleted orderID=%s", id)
	return nil
}
