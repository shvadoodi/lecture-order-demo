package app

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shvadoodi/lecture-order-demo/internal/config"
)

func TestNewServerWiring(t *testing.T) {
	cfg := config.Config{Address: ":9090", ReadHeaderTimeout: 2 * time.Second, ShutdownTimeout: time.Second}
	server := NewServer(cfg, log.New(io.Discard, "", 0))
	if server.Addr != cfg.Address || server.ReadHeaderTimeout != cfg.ReadHeaderTimeout {
		t.Fatal("server ignores configuration")
	}
	create := httptest.NewRecorder()
	server.Handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("{\"customerId\":\"CUS-1\",\"items\":[{\"productId\":\"P-1\",\"quantity\":2,\"price\":5}]}")))
	if create.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", create.Code, create.Body.String())
	}
	get := httptest.NewRecorder()
	server.Handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/orders/ORD-000001", nil))
	if get.Code != http.StatusOK || get.Body.String() != create.Body.String() {
		t.Fatalf("get=%d %s", get.Code, get.Body.String())
	}
}

func TestRunListenResults(t *testing.T) {
	failure := errors.New("bind failed")
	for _, expected := range []error{nil, http.ErrServerClosed, failure} {
		server := &ServerMock{ListenAndServeFunc: func() error { return expected }}
		err := Run(context.Background(), server, time.Second)
		want := expected
		if errors.Is(expected, http.ErrServerClosed) {
			want = nil
		}
		if !errors.Is(err, want) {
			t.Fatalf("error=%v want=%v", err, want)
		}
		if len(server.ShutdownCalls()) != 0 {
			t.Fatal("unexpected shutdown")
		}
	}
}

func TestRunAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, &ServerMock{}, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRunShutdown(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "graceful", true: "force close"}[fails], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started, stopped := make(chan struct{}), make(chan struct{})
			shutdownFailure, closeFailure := errors.New("shutdown failed"), errors.New("close failed")
			server := &ServerMock{
				ListenAndServeFunc: func() error { close(started); <-stopped; return http.ErrServerClosed },
				ShutdownFunc: func(shutdownCtx context.Context) error {
					deadline, ok := shutdownCtx.Deadline()
					if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > time.Second {
						t.Error("shutdown deadline missing or invalid")
					}
					if shutdownCtx.Err() != nil {
						t.Error("shutdown context inherited cancellation")
					}
					if fails {
						return shutdownFailure
					}
					close(stopped)
					return nil
				},
				CloseFunc: func() error { close(stopped); return closeFailure },
			}
			result := make(chan error, 1)
			go func() { result <- Run(ctx, server, time.Second) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("server did not start")
			}
			cancel()
			select {
			case err := <-result:
				if fails {
					if !errors.Is(err, shutdownFailure) || !errors.Is(err, closeFailure) {
						t.Fatalf("error=%v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown did not finish")
			}
			if len(server.ShutdownCalls()) != 1 {
				t.Fatal("shutdown not called once")
			}
			expected := 0
			if fails {
				expected = 1
			}
			if len(server.CloseCalls()) != expected {
				t.Fatal("incorrect forced-close count")
			}
		})
	}
}

func TestRunPreservesListenFailureDuringShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, stopped := make(chan struct{}), make(chan struct{})
	failure := errors.New("serve failed")
	server := &ServerMock{
		ListenAndServeFunc: func() error { close(started); <-stopped; return failure },
		ShutdownFunc:       func(context.Context) error { close(stopped); return nil },
	}
	result := make(chan error, 1)
	go func() { result <- Run(ctx, server, time.Second) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, failure) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown hung")
	}
}
