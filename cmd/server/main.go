package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/middleware"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/validator"
)

func main() {
	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	mux := http.NewServeMux()

	// Validator handler
	mux.Handle("/validator", http.HandlerFunc(validator.Handler))

	// Set chain from env if provided
	if c := os.Getenv("CHAIN"); c != "" {
		validator.SetChain(c)
	}

	// Wrap with abuse prevention
	handler := middleware.IPRateLimit(mux)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
}
