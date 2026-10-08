package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shvadoodi/lecture-order-demo/internal/app"
	"github.com/shvadoodi/lecture-order-demo/internal/config"
)

// @title Lecture Order Demo API
// @version 1.0
// @description In-memory order API for the lecture From Classroom Code to Production Software.
// @BasePath /
func main() {
	// Fail fast on invalid settings before opening a listening socket.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	logger := log.New(os.Stdout, "", log.LstdFlags)
	// Ctrl+C and SIGTERM cancel this context and trigger graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Keep process concerns here; app owns dependency wiring and lifecycle.
	server := app.NewServer(cfg, logger)
	logger.Printf("Order API listening on %s", cfg.Address)
	if err := app.Run(ctx, server, cfg.ShutdownTimeout); err != nil {
		logger.Fatal(err)
	}
}
