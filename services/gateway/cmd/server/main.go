// Command server is the INDIS API gateway.
//
// It exposes an HTTP/1.1 REST API on port 8080 and proxies requests to the
// gRPC backend services (identity:50051 … justice:50058) using generated
// protobuf client stubs.  Per-IP token-bucket rate limiting is enforced
// before any request reaches a backend.
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

	"github.com/IranProsperityProject/INDIS/services/gateway/internal/config"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/proxy"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/ratelimit"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	clients, err := proxy.New(
		cfg.IdentityAddr,
		cfg.CredentialAddr,
		cfg.EnrollmentAddr,
		cfg.BiometricAddr,
		cfg.AuditAddr,
		cfg.NotificationAddr,
		cfg.ElectoralAddr,
		cfg.JusticeAddr,
	)
	if err != nil {
		log.Fatalf("proxy: %v", err)
	}
	defer clients.Close()

	limiter := ratelimit.New(cfg.RateLimitRPS)
	gw := handler.New(clients, limiter)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      gw,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("INDIS gateway listening on :%d", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway…")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Gateway stopped.")
}
