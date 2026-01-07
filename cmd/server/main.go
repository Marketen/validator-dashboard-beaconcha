// Package main is the entry point for the validator-dashboard API server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/api"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/beaconcha"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/config"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/ratelimiter"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/service"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("starting validator-dashboard",
		"port", cfg.Port,
		"beaconcha_base_url", cfg.BeaconchainBaseURL,
	)

	// Initialize global rate limiter for Beaconcha API (1 req/sec)
	beaconchainRateLimiter := ratelimiter.NewGlobalRateLimiter(cfg.BeaconchainRateLimit)

	// Initialize Beaconcha client
	beaconchainClient := beaconcha.NewClient(
		cfg.BeaconchainBaseURL,
		cfg.BeaconchainAPIKey,
		beaconchainRateLimiter,
		cfg.BeaconchainTimeout,
	)

	// Initialize validator service
	validatorService := service.NewValidatorService(beaconchainClient)

	// Initialize API handler
	handler := api.NewHandler(validatorService, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Router(),
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
