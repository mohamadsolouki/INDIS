// INDIS — Gov Portal Service — ministry dashboard, bulk operations, and GraphQL API.
// Implements PRD FR-009 (ministry dashboard), FR-010 (bulk operations), FR-011 (audit reports).
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

	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	"github.com/IranProsperityProject/INDIS/services/govportal/internal/config"
	"github.com/IranProsperityProject/INDIS/services/govportal/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/govportal/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/govportal/internal/service"
)

func main() {
	log.Printf("Starting INDIS govportal service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("govportal")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := indismigrate.ApplyStartupMigrations(ctx, pool, ""); err != nil {
		log.Fatalf("migrations apply: %v", err)
	}

	repo := repository.New(pool)
	svc := service.New(repo)
	h := handler.New(svc, cfg.JWTSecret)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS govportal service...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
