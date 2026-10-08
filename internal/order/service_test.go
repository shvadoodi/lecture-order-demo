package order

import (
	"bytes"
	"context"
	"errors"
	"log"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func validRequest() CreateOrderRequest {
	return CreateOrderRequest{CustomerID: " CUS-1 ", Items: []OrderItem{{ProductID: " P-1 ", Name: " Keyboard ", Quantity: 2, Price: 12.5}}}
}

func TestCreateOrder(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var sequence []string
	repo := &OrderRepositoryMock{CreateFunc: func(got context.Context, order Order) error {
		if got != ctx {
			t.Fatal("context was not forwarded")
		}
		sequence = append(sequence, "save")
		return nil
	}}
	pub := &EventPublisherMock{PublishOrderCreatedFunc: func(got context.Context, order Order) error {
		if got != ctx {
			t.Fatal("context was not forwarded")
		}
		sequence = append(sequence, "publish")
		return nil
	}}
	service := NewOrderService(repo, pub)
	service.now = func() time.Time { return now }
	request := validRequest()
	got, err := service.CreateOrder(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	want := Order{ID: "ORD-000001", CustomerID: "CUS-1", Items: []OrderItem{{ProductID: "P-1", Name: "Keyboard", Quantity: 2, Price: 12.5}}, Total: 25, Status: OrderStatusCreated, CreatedAt: now}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(sequence, []string{"save", "publish"}) {
		t.Fatalf("sequence = %v", sequence)
	}
	if len(repo.CreateCalls()) != 1 || len(pub.PublishOrderCreatedCalls()) != 1 {
		t.Fatal("unexpected dependency call count")
	}
	if !reflect.DeepEqual(repo.CreateCalls()[0].Order, got) || !reflect.DeepEqual(pub.PublishOrderCreatedCalls()[0].Order, got) {
		t.Fatal("dependencies received a different order")
	}
	if request.CustomerID != " CUS-1 " || request.Items[0].ProductID != " P-1 " {
		t.Fatal("request mutated")
	}
}

func TestValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func(*CreateOrderRequest)
		sentinel error
	}{
		{"customer", func(r *CreateOrderRequest) { r.CustomerID = " " }, ErrInvalidCustomer},
		{"items", func(r *CreateOrderRequest) { r.Items = nil }, ErrEmptyOrder},
		{"product", func(r *CreateOrderRequest) { r.Items[0].ProductID = " " }, nil},
		{"zero quantity", func(r *CreateOrderRequest) { r.Items[0].Quantity = 0 }, nil},
		{"negative quantity", func(r *CreateOrderRequest) { r.Items[0].Quantity = -1 }, nil},
		{"negative price", func(r *CreateOrderRequest) { r.Items[0].Price = -1 }, nil},
		{"NaN", func(r *CreateOrderRequest) { r.Items[0].Price = math.NaN() }, nil},
		{"infinity", func(r *CreateOrderRequest) { r.Items[0].Price = math.Inf(1) }, nil},
		{"negative infinity", func(r *CreateOrderRequest) { r.Items[0].Price = math.Inf(-1) }, nil},
		{"overflow", func(r *CreateOrderRequest) { r.Items[0].Price = math.MaxFloat64 }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			tc.change(&r)
			repo, pub := &OrderRepositoryMock{}, &EventPublisherMock{}
			_, err := NewOrderService(repo, pub).CreateOrder(context.Background(), r)
			var validation *ValidationError
			if err == nil || err.Error() == "" {
				t.Fatal("validation error must explain invalid input")
			}
			if !errors.As(err, &validation) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if tc.sentinel != nil && !errors.Is(err, tc.sentinel) {
				t.Fatalf("underlying error lost: %v", err)
			}
			if len(repo.CreateCalls()) != 0 || len(pub.PublishOrderCreatedCalls()) != 0 {
				t.Fatal("invalid order caused side effects")
			}
		})
	}
}

func TestValidationAllowsFreeItemsAndCalculatesMultipleItems(t *testing.T) {
	r := validRequest()
	r.Items = append(r.Items, OrderItem{ProductID: "free", Quantity: 3, Price: 0}, OrderItem{ProductID: "P-2", Quantity: 2, Price: 5})
	_, items, total, err := validateOrderInput(r.CustomerID, r.Items)
	if err != nil || total != 35 || len(items) != 3 {
		t.Fatalf("items=%v total=%v err=%v", items, total, err)
	}
}

func TestCreateFailures(t *testing.T) {
	for _, stage := range []string{"save", "publish"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New(stage)
			repo := &OrderRepositoryMock{CreateFunc: func(context.Context, Order) error {
				if stage == "save" {
					return failure
				}
				return nil
			}}
			pub := &EventPublisherMock{PublishOrderCreatedFunc: func(context.Context, Order) error { return failure }}
			got, err := NewOrderService(repo, pub).CreateOrder(context.Background(), validRequest())
			if !errors.Is(err, failure) || got.ID != "" {
				t.Fatalf("order=%v err=%v", got, err)
			}
			want := 0
			if stage == "publish" {
				want = 1
			}
			if len(pub.PublishOrderCreatedCalls()) != want {
				t.Fatal("published after failed save")
			}
		})
	}
}

func TestReadsForwardResultsAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, failure := range []error{nil, ErrOrderNotFound, context.Canceled} {
		expected := Order{ID: "ORD-1"}
		repo := &OrderRepositoryMock{
			GetByIDFunc: func(got context.Context, id string) (Order, error) {
				if got != ctx || id != expected.ID {
					t.Fatal("incorrect get arguments")
				}
				return expected, failure
			},
			GetAllFunc: func(got context.Context) ([]Order, error) {
				if got != ctx {
					t.Fatal("incorrect context")
				}
				return []Order{expected}, failure
			},
		}
		service := NewOrderService(repo, &EventPublisherMock{})
		got, err := service.GetOrder(ctx, expected.ID)
		if got.ID != expected.ID || !errors.Is(err, failure) {
			t.Fatalf("get = %v, %v", got, err)
		}
		all, err := service.GetOrders(ctx)
		if len(all) != 1 || all[0].ID != expected.ID || !errors.Is(err, failure) {
			t.Fatalf("list = %v, %v", all, err)
		}
	}
}

func TestUpdateOrder(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := created.Add(time.Hour)
	existing := Order{ID: "ORD-1", CreatedAt: created, Status: OrderStatusCreated}
	var sequence []string
	repo := &OrderRepositoryMock{
		GetByIDFunc: func(got context.Context, id string) (Order, error) {
			if got != ctx || id != existing.ID {
				t.Fatal("incorrect lookup")
			}
			sequence = append(sequence, "get")
			return existing, nil
		},
		UpdateFunc: func(context.Context, Order) error { sequence = append(sequence, "update"); return nil },
	}
	pub := &EventPublisherMock{PublishOrderUpdatedFunc: func(context.Context, Order) error { sequence = append(sequence, "publish"); return nil }}
	service := NewOrderService(repo, pub)
	service.now = func() time.Time { return now }
	r := validRequest()
	got, err := service.UpdateOrder(ctx, existing.ID, UpdateOrderRequest(r))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != existing.ID || got.CreatedAt != created || got.Status != existing.Status || got.UpdatedAt == nil || *got.UpdatedAt != now || got.Total != 25 || got.CustomerID != "CUS-1" {
		t.Fatalf("unexpected update: %+v", got)
	}
	if !reflect.DeepEqual(sequence, []string{"get", "update", "publish"}) {
		t.Fatalf("sequence = %v", sequence)
	}
	if !reflect.DeepEqual(repo.UpdateCalls()[0].Order, got) || !reflect.DeepEqual(pub.PublishOrderUpdatedCalls()[0].Order, got) {
		t.Fatal("incorrect update/event payload")
	}
}

func TestUpdateFailures(t *testing.T) {
	for _, stage := range []string{"get", "validate", "update", "publish"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New(stage)
			repo := &OrderRepositoryMock{
				GetByIDFunc: func(context.Context, string) (Order, error) {
					if stage == "get" {
						return Order{}, failure
					}
					return Order{ID: "ORD-1"}, nil
				},
				UpdateFunc: func(context.Context, Order) error {
					if stage == "update" {
						return failure
					}
					return nil
				},
			}
			pub := &EventPublisherMock{PublishOrderUpdatedFunc: func(context.Context, Order) error { return failure }}
			r := UpdateOrderRequest(validRequest())
			if stage == "validate" {
				r.CustomerID = ""
			}
			_, err := NewOrderService(repo, pub).UpdateOrder(context.Background(), "ORD-1", r)
			if stage == "validate" {
				if !errors.Is(err, ErrInvalidCustomer) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatal(err)
			}
			wantUpdate := 0
			if stage == "update" || stage == "publish" {
				wantUpdate = 1
			}
			wantPublish := 0
			if stage == "publish" {
				wantPublish = 1
			}
			if len(repo.UpdateCalls()) != wantUpdate || len(pub.PublishOrderUpdatedCalls()) != wantPublish {
				t.Fatal("unexpected downstream calls")
			}
		})
	}
}

func TestDeleteOrder(t *testing.T) {
	for _, stage := range []string{"success", "get", "delete", "publish"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			failure := errors.New(stage)
			var sequence []string
			repo := &OrderRepositoryMock{
				GetByIDFunc: func(got context.Context, id string) (Order, error) {
					if got != ctx || id != "ORD-1" {
						t.Fatal("incorrect lookup")
					}
					sequence = append(sequence, "get")
					if stage == "get" {
						return Order{}, failure
					}
					return Order{ID: id}, nil
				},
				DeleteFunc: func(got context.Context, id string) error {
					if got != ctx || id != "ORD-1" {
						t.Fatal("incorrect deletion")
					}
					sequence = append(sequence, "delete")
					if stage == "delete" {
						return failure
					}
					return nil
				},
			}
			pub := &EventPublisherMock{PublishOrderDeletedFunc: func(got context.Context, id string) error {
				if got != ctx || id != "ORD-1" {
					t.Fatal("incorrect event")
				}
				sequence = append(sequence, "publish")
				if stage == "publish" {
					return failure
				}
				return nil
			}}
			err := NewOrderService(repo, pub).DeleteOrder(ctx, "ORD-1")
			if stage == "success" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatal(err)
			}
			want := []string{"get", "delete", "publish"}
			if stage == "get" {
				want = want[:1]
			}
			if stage == "delete" {
				want = want[:2]
			}
			if !reflect.DeepEqual(sequence, want) {
				t.Fatalf("sequence = %v", sequence)
			}
		})
	}
}

func TestConcurrentCreatesHaveUniqueIDs(t *testing.T) {
	repo := &OrderRepositoryMock{CreateFunc: func(context.Context, Order) error { return nil }}
	pub := &EventPublisherMock{PublishOrderCreatedFunc: func(context.Context, Order) error { return nil }}
	service := NewOrderService(repo, pub)
	ids := make(chan string, 100)
	var wg sync.WaitGroup
	for i := 0; i < cap(ids); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			order, err := service.CreateOrder(context.Background(), validRequest())
			if err != nil {
				t.Error(err)
				return
			}
			ids <- order.ID
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
	if len(seen) != cap(ids) {
		t.Fatalf("created %d orders", len(seen))
	}
}

// Verify each creation failure is observable even when the service is called directly.
func TestCreateOrderFailureLogs(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	failure := errors.New("dependency unavailable")
	for _, stage := range []string{"validation", "storage", "publishing"} {
		t.Run(stage, func(t *testing.T) {
			output.Reset()
			repo := &OrderRepositoryMock{CreateFunc: func(context.Context, Order) error {
				if stage == "storage" {
					return failure
				}
				return nil
			}}
			pub := &EventPublisherMock{PublishOrderCreatedFunc: func(context.Context, Order) error {
				if stage == "publishing" {
					return failure
				}
				return nil
			}}
			request := validRequest()
			wantErr := failure
			if stage == "validation" {
				request.CustomerID = ""
				wantErr = ErrInvalidCustomer
			}
			_, err := NewOrderServiceWithLogger(repo, pub, logger).CreateOrder(context.Background(), request)
			if !errors.Is(err, wantErr) {
				t.Fatalf("error=%v want=%v", err, wantErr)
			}
			message := output.String()
			if !strings.Contains(message, "stage="+stage) || !strings.Contains(message, wantErr.Error()) {
				t.Fatalf("missing failure details: %s", message)
			}
			if stage == "publishing" && !strings.Contains(message, "saved=true") {
				t.Fatalf("log must explain that the order was already saved: %s", message)
			}
		})
	}
	output.Reset()
	repo := &OrderRepositoryMock{CreateFunc: func(context.Context, Order) error { return nil }}
	pub := &EventPublisherMock{PublishOrderCreatedFunc: func(context.Context, Order) error { return nil }}
	if _, err := NewOrderServiceWithLogger(repo, pub, logger).CreateOrder(context.Background(), validRequest()); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("successful creation logged a failure: %s", output.String())
	}
}

func TestOtherOperationFailureLogs(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	failure := errors.New("dependency unavailable")
	for _, tc := range []struct{ operation, stage string }{
		{"get", "storage"}, {"list", "storage"},
		{"update", "lookup"}, {"update", "validation"}, {"update", "storage"}, {"update", "publishing"},
		{"delete", "lookup"}, {"delete", "storage"}, {"delete", "publishing"},
	} {
		t.Run(tc.operation+"/"+tc.stage, func(t *testing.T) {
			output.Reset()
			repo := &OrderRepositoryMock{
				GetByIDFunc: func(context.Context, string) (Order, error) {
					if tc.operation == "get" || tc.stage == "lookup" {
						return Order{}, failure
					}
					return Order{ID: "ORD-1"}, nil
				},
				GetAllFunc: func(context.Context) ([]Order, error) { return nil, failure },
				UpdateFunc: func(context.Context, Order) error {
					if tc.stage == "storage" {
						return failure
					}
					return nil
				},
				DeleteFunc: func(context.Context, string) error {
					if tc.stage == "storage" {
						return failure
					}
					return nil
				},
			}
			pub := &EventPublisherMock{
				PublishOrderUpdatedFunc: func(context.Context, Order) error { return failure },
				PublishOrderDeletedFunc: func(context.Context, string) error { return failure },
			}
			service := NewOrderServiceWithLogger(repo, pub, logger)
			ctx := context.Background()
			var err error
			want := failure
			switch tc.operation {
			case "get":
				_, err = service.GetOrder(ctx, "ORD-1")
			case "list":
				_, err = service.GetOrders(ctx)
			case "update":
				request := validRequest()
				if tc.stage == "validation" {
					request.CustomerID = ""
					want = ErrInvalidCustomer
				}
				_, err = service.UpdateOrder(ctx, "ORD-1", UpdateOrderRequest{CustomerID: request.CustomerID, Items: request.Items})
			case "delete":
				err = service.DeleteOrder(ctx, "ORD-1")
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			message := output.String()
			if !strings.Contains(message, tc.operation+" order") || !strings.Contains(message, "stage="+tc.stage) || !strings.Contains(message, want.Error()) {
				t.Fatalf("missing failure details: %s", message)
			}
			if tc.operation != "list" && !strings.Contains(message, "orderID=ORD-1") {
				t.Fatalf("missing ID: %s", message)
			}
			if tc.stage == "publishing" {
				state := "saved=true"
				if tc.operation == "delete" {
					state = "deleted=true"
				}
				if !strings.Contains(message, state) {
					t.Fatalf("missing storage outcome: %s", message)
				}
			}
		})
	}
}
