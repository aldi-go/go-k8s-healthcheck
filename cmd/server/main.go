package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/plabs/go-k8s-healthcheck/internal/config"
	"github.com/plabs/go-k8s-healthcheck/internal/handler"
)

func main() {
	cfg := config.Load()

	// Structured JSON logger — required for production log aggregation (Loki, CloudWatch, etc.)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	health := handler.NewHealthHandler(cfg.AppName, cfg.Version, cfg.AppEnv)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Liveness)
	mux.HandleFunc("GET /readyz", health.Readiness)
	mux.HandleFunc("GET /", health.Info)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so it doesn't block signal handling.
	go func() {
		slog.Info("server starting",
			"port", cfg.Port,
			"env", cfg.AppEnv,
			"version", cfg.Version,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Mark app as ready after successful startup.
	health.SetReady(true)
	slog.Info("server ready to accept traffic")

	// Block until OS signal received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", "signal", sig.String())

	// Mark not-ready first so load balancer stops sending new requests.
	health.SetReady(false)

	// Give in-flight requests time to complete.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownSec)*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
