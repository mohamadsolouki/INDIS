// Command server is the INDIS USSD/SMS gateway service.
//
// It exposes an HTTP service on port 8300 (default) that handles USSD session
// callbacks from telecom operators and SMS OTP generation/verification.
// All citizen PII is stored only as SHA-256 hashes; session state_data is
// wiped on session end per PRD FR-015.6.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
	indistrace "github.com/mohamadsolouki/INDIS/pkg/tracing"
	"github.com/mohamadsolouki/INDIS/services/ussd/internal/config"
	"github.com/mohamadsolouki/INDIS/services/ussd/internal/handler"
	"github.com/mohamadsolouki/INDIS/services/ussd/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/ussd/internal/service"
)

func main() {
	log.Printf("Starting INDIS USSD/SMS gateway service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "ussd")
	if err != nil {
		log.Fatalf("tracing: %v", err)
	}
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		if err := tracingShutdown(shutCtx); err != nil {
			log.Printf("tracing shutdown: %v", err)
		}
	}()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := indismigrate.ApplyStartupMigrations(ctx, pool, ""); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	repo := repository.New(pool)
	svc := service.New(repo, cfg.GatewayURL)
	h := handler.New(svc)

	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("INDIS USSD service listening on %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS USSD service...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	fmt.Println("USSD service stopped.")
}
