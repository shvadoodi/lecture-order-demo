package eventlog

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

func TestPublisher(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		publish       func(*LogEventPublisher, context.Context) error
	}{
		{"created", "EVENT OrderCreated orderID=ORD-1 customerID=CUS-1 total=12.50", func(p *LogEventPublisher, ctx context.Context) error {
			return p.PublishOrderCreated(ctx, domain.Order{ID: "ORD-1", CustomerID: "CUS-1", Total: 12.5})
		}},
		{"updated", "EVENT OrderUpdated orderID=ORD-1 customerID=CUS-1 total=12.50", func(p *LogEventPublisher, ctx context.Context) error {
			return p.PublishOrderUpdated(ctx, domain.Order{ID: "ORD-1", CustomerID: "CUS-1", Total: 12.5})
		}},
		{"deleted", "EVENT OrderDeleted orderID=ORD-1", func(p *LogEventPublisher, ctx context.Context) error { return p.PublishOrderDeleted(ctx, "ORD-1") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			publisher := NewLogEventPublisher(log.New(&output, "", 0))
			if err := tc.publish(publisher, context.Background()); err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(output.String()) != tc.message {
				t.Fatalf("log=%q", output.String())
			}
			output.Reset()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := tc.publish(publisher, ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("error=%v", err)
			}
			if output.Len() != 0 {
				t.Fatal("canceled operation published event")
			}
		})
	}
}
