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
	service := order.NewOrderService(memory.NewInMemoryOrderRepository(), eventlog.NewLogEventPublisher(logger))
	return &http.Server{Addr: cfg.Address, Handler: httpapi.NewRouter(httpapi.NewOrderHandler(service)), ReadHeaderTimeout: cfg.ReadHeaderTimeout}
}

//go:generate go run ../../tools/generate.go moq -out server_moq_test.go . Server

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
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			closeErr := server.Close()
			<-result
			return errors.Join(fmt.Errorf("shutdown server: %w", err), closeErr)
		}
		err := <-result
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
