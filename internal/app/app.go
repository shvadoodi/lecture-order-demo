package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shvadoodi/lecture-order-demo/internal/adapter/eventlog"
	"github.com/shvadoodi/lecture-order-demo/internal/adapter/httpapi"
	"github.com/shvadoodi/lecture-order-demo/internal/adapter/memory"
	"github.com/shvadoodi/lecture-order-demo/internal/config"
	"github.com/shvadoodi/lecture-order-demo/internal/order"
)

// NewServer wires the concrete adapters at the application boundary.
func NewServer(cfg config.Config, logger *log.Logger) *http.Server {
	// Build from the inside out: storage and events, business logic, then HTTP.
	repository := memory.NewInMemoryOrderRepository()
	publisher := eventlog.NewLogEventPublisher(logger)
	service := order.NewOrderService(repository, publisher)
	handler := httpapi.NewOrderHandler(service)

	return &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouter(handler),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}
}

//go:generate go run ../../tools/generate.go moq -out server_moq_test.go . Server

// Server is the lifecycle contract used by Run. *http.Server implements it;
// tests supply a mock without opening a network port.
type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

// Run serves until cancellation, then gives active requests time to finish.
func Run(ctx context.Context, server Server, shutdownTimeout time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serving blocks, so run it separately while waiting for a shutdown signal.
	// The buffer lets the serving goroutine report its result without a receiver.
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case err := <-result:
		// ErrServerClosed is the normal result of stopping an HTTP server.
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// The signal context is already canceled. Give shutdown a fresh deadline
		// so active requests have time to finish.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			// If graceful shutdown fails, close connections and wait for serving to exit.
			closeErr := server.Close()
			<-result
			return errors.Join(fmt.Errorf("shutdown server: %w", err), closeErr)
		}
		err := <-result
		// ErrServerClosed is the normal result of stopping an HTTP server.
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
