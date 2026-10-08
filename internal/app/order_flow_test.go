package app

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shvadoodi/lecture-order-demo/internal/adapter/eventlog"
	"github.com/shvadoodi/lecture-order-demo/internal/adapter/httpapi"
	"github.com/shvadoodi/lecture-order-demo/internal/adapter/memory"
	"github.com/shvadoodi/lecture-order-demo/internal/order"
)

// failingPublisher simulates an unavailable downstream system from slide 23.
type failingPublisher struct{}

func (failingPublisher) PublishOrderCreated(context.Context, order.Order) error {
	return errors.New("message broker unavailable")
}
func (failingPublisher) PublishOrderUpdated(context.Context, order.Order) error {
	return errors.New("message broker unavailable")
}
func (failingPublisher) PublishOrderDeleted(context.Context, string) error {
	return errors.New("message broker unavailable")
}

const lectureOrderJSON = `{"customerId":"CUS-1","items":[{"productId":"P-1","quantity":2,"price":5}]}`

// This component test crosses HTTP, business logic, and real memory storage.
// A 500 response does not necessarily mean nothing was saved.
func TestPublishingFailureLeavesOrderStored(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	repository := memory.NewInMemoryOrderRepository()
	service := order.NewOrderServiceWithLogger(repository, failingPublisher{}, logger)
	router := httpapi.NewRouter(httpapi.NewOrderHandler(service))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(lectureOrderJSON)))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "broker") {
		t.Fatal("HTTP response exposed infrastructure details")
	}
	stored, err := repository.GetAll(context.Background())
	if err != nil || len(stored) != 1 || stored[0].Total != 10 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	if !strings.Contains(logs.String(), "saved=true") || !strings.Contains(logs.String(), "message broker unavailable") {
		t.Fatalf("missing diagnostic log: %s", logs.String())
	}
}

// Slide 10 asks whether retrying is safe. Without idempotency keys, identical
// POST requests create separate orders. This test documents that limitation.
func TestRepeatedPostCreatesSeparateOrders(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	repository := memory.NewInMemoryOrderRepository()
	service := order.NewOrderServiceWithLogger(repository, eventlog.NewLogEventPublisher(logger), logger)
	router := httpapi.NewRouter(httpapi.NewOrderHandler(service))
	for attempt := 0; attempt < 2; attempt++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(lectureOrderJSON)))
		if response.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
	stored, err := repository.GetAll(context.Background())
	if err != nil || len(stored) != 2 || stored[0].ID == stored[1].ID {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}
