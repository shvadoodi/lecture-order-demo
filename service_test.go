package main

import (
	"context"
	"math"
	"testing"
)

type testPublisher struct{ published []Order }

func (p *testPublisher) PublishOrderCreated(_ context.Context, order Order) error {
	p.published = append(p.published, order)
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
	if len(publisher.published) != 1 {
		t.Fatalf("expected 1 event, got %d", len(publisher.published))
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
