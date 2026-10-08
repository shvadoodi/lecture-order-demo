package order

import "time"

// OrderStatus represents the order lifecycle state. This demo uses CREATED only.
type OrderStatus string

const (
	OrderStatusCreated OrderStatus = "CREATED"
)

// OrderItem describes a product and its unit price. Production money handling
// should use integer cents or a decimal type instead of float64.
type OrderItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// Order contains server-managed identity, totals, and timestamps.
type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customerId"`
	Items      []OrderItem `json:"items"`
	Total      float64     `json:"total"`
	Status     OrderStatus `json:"status"`
	CreatedAt  time.Time   `json:"createdAt"`
	// A nil pointer means the order has never been updated and omits the JSON field.
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// CreateOrderRequest includes only fields a client is allowed to supply.
type CreateOrderRequest struct {
	CustomerID string      `json:"customerId"`
	Items      []OrderItem `json:"items"`
}

// UpdateOrderRequest replaces all editable fields (PUT), rather than patching them.
type UpdateOrderRequest struct {
	CustomerID string      `json:"customerId"`
	Items      []OrderItem `json:"items"`
}
