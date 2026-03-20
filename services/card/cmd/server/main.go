// Command server is the INDIS card service.
//
// It exposes an HTTP service on port 8400 (default) that manages physical
// identity card data in ICAO 9303 / ISO 7816 format per PRD FR-016.
// No biometric raw data is ever stored; chip data contains only the DID
// document reference and the issuer public key.
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

	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	"github.com/IranProsperityProject/INDIS/services/card/internal/config"
	"github.com/IranProsperityProject/INDIS/services/card/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/card/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/card/internal/service"
)

func main() {
	log.Printf("Starting INDIS card service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "card")
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
	svc, err := service.New(repo, cfg.CardIssuerSeed)
	if err != nil {
		log.Fatalf("service init: %v", err)
	}

	h := handler.New(svc)

	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("INDIS card service listening on %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS card service...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	fmt.Println("Card service stopped.")
}
