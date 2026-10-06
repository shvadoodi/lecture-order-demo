package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

const validBody = "{\"customerId\":\"CUS-1\",\"items\":[{\"productId\":\"P-1\",\"quantity\":1,\"price\":10}]}"

func TestOrderEndpoints(t *testing.T) {
	for _, operation := range []string{"create", "list", "get", "update", "delete"} {
		for _, outcome := range []string{"success", "internal", "not found", "validation"} {
			if outcome == "not found" && (operation == "create" || operation == "list") {
				continue
			}
			if outcome == "validation" && operation != "create" && operation != "update" {
				continue
			}
			t.Run(operation+"/"+outcome, func(t *testing.T) {
				ctx := context.WithValue(context.Background(), struct{}{}, "request")
				var failure error
				status := http.StatusOK
				switch outcome {
				case "internal":
					failure = errors.New("private database error")
					status = http.StatusInternalServerError
				case "not found":
					failure = fmt.Errorf("wrapped: %w", domain.ErrOrderNotFound)
					status = http.StatusNotFound
				case "validation":
					failure = &domain.ValidationError{Err: domain.ErrInvalidCustomer}
					status = http.StatusBadRequest
				}
				expected := domain.Order{ID: "ORD-1", CustomerID: "CUS-1", Items: []domain.OrderItem{{ProductID: "P-1", Quantity: 1, Price: 10}}, Total: 10}
				checkContext := func(got context.Context) {
					t.Helper()
					if got != ctx {
						t.Fatal("context not forwarded")
					}
				}
				checkID := func(id string) {
					t.Helper()
					if id != expected.ID {
						t.Fatalf("id = %s", id)
					}
				}
				checkInput := func(customer string, items []domain.OrderItem) {
					t.Helper()
					if customer != expected.CustomerID || !reflect.DeepEqual(items, expected.Items) {
						t.Fatal("incorrect input passed to service")
					}
				}
				mock := &OrderServiceMock{}
				method, path, body := http.MethodGet, "/orders", ""
				switch operation {
				case "create":
					method, body = http.MethodPost, validBody
					if outcome == "success" {
						status = http.StatusCreated
					}
					mock.CreateOrderFunc = func(got context.Context, request domain.CreateOrderRequest) (domain.Order, error) {
						checkContext(got)
						checkInput(request.CustomerID, request.Items)
						return expected, failure
					}
				case "list":
					mock.GetOrdersFunc = func(got context.Context) ([]domain.Order, error) {
						checkContext(got)
						return []domain.Order{expected}, failure
					}
				case "get":
					path += "/ORD-1"
					mock.GetOrderFunc = func(got context.Context, id string) (domain.Order, error) {
						checkContext(got)
						checkID(id)
						return expected, failure
					}
				case "update":
					method, path, body = http.MethodPut, "/orders/ORD-1", validBody
					mock.UpdateOrderFunc = func(got context.Context, id string, request domain.UpdateOrderRequest) (domain.Order, error) {
						checkContext(got)
						checkID(id)
						checkInput(request.CustomerID, request.Items)
						return expected, failure
					}
				case "delete":
					method, path = http.MethodDelete, "/orders/ORD-1"
					if outcome == "success" {
						status = http.StatusNoContent
					}
					mock.DeleteOrderFunc = func(got context.Context, id string) error { checkContext(got); checkID(id); return failure }
				}
				response := httptest.NewRecorder()
				NewRouter(NewOrderHandler(mock)).ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx))
				if response.Code != status {
					t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
				}
				count := len(mock.CreateOrderCalls()) + len(mock.GetOrdersCalls()) + len(mock.GetOrderCalls()) + len(mock.UpdateOrderCalls()) + len(mock.DeleteOrderCalls())
				if count != 1 {
					t.Fatalf("service called %d times", count)
				}
				if status == http.StatusNoContent {
					if response.Body.Len() != 0 {
						t.Fatal("204 response contains a body")
					}
					return
				}
				if response.Header().Get("Content-Type") != "application/json" {
					t.Fatal("missing JSON content type")
				}
				if failure != nil {
					var got ErrorResponse
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					want := "internal server error"
					if outcome == "not found" {
						want = "order not found"
					}
					if outcome == "validation" {
						want = "customerId is required"
					}
					if got.Error != want {
						t.Fatalf("error=%q want=%q", got.Error, want)
					}
				} else if operation == "list" {
					var got []domain.Order
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, []domain.Order{expected}) {
						t.Fatalf("body=%s", response.Body.String())
					}
				} else {
					var got domain.Order
					if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, expected) {
						t.Fatalf("body=%s", response.Body.String())
					}
				}
			})
		}
	}
}

func TestInvalidRequestsDoNotCallService(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, body := range []string{"", "{", "[]", "null", "{} {}", "{} garbage", "{\"unexpected\":true}", "{\"items\":[{\"quantity\":1.5}]}"} {
			t.Run(method+"/"+body, func(t *testing.T) {
				path := "/orders"
				if method == http.MethodPut {
					path += "/ORD-1"
				}
				recorder := httptest.NewRecorder()
				NewRouter(NewOrderHandler(&OrderServiceMock{})).ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
				}
			})
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		for _, path := range []string{"/orders/", "/orders/one/two", "/orders/%20"} {
			t.Run(method+path, func(t *testing.T) {
				response := httptest.NewRecorder()
				NewRouter(NewOrderHandler(&OrderServiceMock{})).ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(validBody)))
				if response.Code != http.StatusBadRequest {
					t.Fatalf("status=%d", response.Code)
				}
			})
		}
	}
}

func TestRoutesAndMethods(t *testing.T) {
	router := NewRouter(NewOrderHandler(&OrderServiceMock{}))
	for _, tc := range []struct {
		method, path, allow string
		status              int
	}{
		{http.MethodGet, "/health", "", http.StatusOK},
		{http.MethodPost, "/health", "GET", http.StatusMethodNotAllowed},
		{http.MethodPatch, "/orders", "GET, POST", http.StatusMethodNotAllowed},
		{http.MethodPost, "/orders/ORD-1", "GET, PUT, DELETE", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing", "", http.StatusNotFound},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status || response.Header().Get("Allow") != tc.allow {
				t.Fatalf("status=%d allow=%q", response.Code, response.Header().Get("Allow"))
			}
			if tc.path == "/health" && tc.status == http.StatusOK && strings.TrimSpace(response.Body.String()) != "{\"status\":\"UP\"}" {
				t.Fatalf("health=%s", response.Body.String())
			}
		})
	}
}
