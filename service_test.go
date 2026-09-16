package main

import (
	"context"
	"errors"
	"math"
	"testing"
)

type testPublisher struct {
	created []Order
	updated []Order
	deleted []string
}

func (p *testPublisher) PublishOrderCreated(_ context.Context, order Order) error {
	p.created = append(p.created, order)
	return nil
}

func (p *testPublisher) PublishOrderUpdated(_ context.Context, order Order) error {
	p.updated = append(p.updated, order)
	return nil
}

func (p *testPublisher) PublishOrderDeleted(_ context.Context, id string) error {
	p.deleted = append(p.deleted, id)
	return nil
}

func TestCreateOrder(t *testing.T) {
	repository := NewInMemoryOrderRepository()
	publisher := &testPublisher{}
	service := NewOrderService(repository, publisher)

	order, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "CUS-1001",
		Items: []OrderItem{
			{ProductID: "P-100", Name: "Keyboard", Quantity: 1, Price: 129.99},
			{ProductID: "P-200", Name: "Mouse", Quantity: 2, Price: 39.99},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if order.ID != "ORD-000001" {
		t.Fatalf("unexpected order id: %s", order.ID)
	}
	if math.Abs(order.Total-209.97) > 0.000001 {
		t.Fatalf("unexpected total: %.10f", order.Total)
	}
	if len(publisher.created) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(publisher.created))
	}
}

func TestUpdateOrder(t *testing.T) {
	repository := NewInMemoryOrderRepository()
	publisher := &testPublisher{}
	service := NewOrderService(repository, publisher)
	created, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "CUS-1001",
		Items:      []OrderItem{{ProductID: "P-100", Name: "Keyboard", Quantity: 1, Price: 100}},
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}

	updated, err := service.UpdateOrder(context.Background(), created.ID, UpdateOrderRequest{
		CustomerID: "CUS-2002",
		Items:      []OrderItem{{ProductID: "P-300", Name: "Monitor", Quantity: 2, Price: 200}},
	})
	if err != nil {
		t.Fatalf("UpdateOrder returned error: %v", err)
	}
	if updated.CustomerID != "CUS-2002" || math.Abs(updated.Total-400) > 0.000001 {
		t.Fatalf("unexpected updated order: %+v", updated)
	}
	if updated.CreatedAt != created.CreatedAt {
		t.Fatal("createdAt should not change on update")
	}
	if updated.UpdatedAt == nil {
		t.Fatal("expected updatedAt to be set")
	}
	if len(publisher.updated) != 1 {
		t.Fatalf("expected 1 updated event, got %d", len(publisher.updated))
	}
}

func TestUpdateOrderReturnsNotFound(t *testing.T) {
	service := NewOrderService(NewInMemoryOrderRepository(), &testPublisher{})
	_, err := service.UpdateOrder(context.Background(), "ORD-999999", UpdateOrderRequest{
		CustomerID: "CUS-1001",
		Items:      []OrderItem{{ProductID: "P-100", Quantity: 1, Price: 10}},
	})
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestDeleteOrder(t *testing.T) {
	repository := NewInMemoryOrderRepository()
	publisher := &testPublisher{}
	service := NewOrderService(repository, publisher)
	created, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		CustomerID: "CUS-1001",
		Items:      []OrderItem{{ProductID: "P-100", Quantity: 1, Price: 10}},
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}

	if err := service.DeleteOrder(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteOrder returned error: %v", err)
	}
	if _, err := service.GetOrder(context.Background(), created.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("expected deleted order to be missing, got %v", err)
	}
	if len(publisher.deleted) != 1 || publisher.deleted[0] != created.ID {
		t.Fatalf("unexpected deleted events: %+v", publisher.deleted)
	}
}

func TestCreateOrderRejectsEmptyCustomer(t *testing.T) {
	service := NewOrderService(NewInMemoryOrderRepository(), &testPublisher{})
	_, err := service.CreateOrder(context.Background(), CreateOrderRequest{
		Items: []OrderItem{{ProductID: "P-100", Quantity: 1, Price: 10}},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
