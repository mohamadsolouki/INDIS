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

	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Best-effort wiring to backend gRPC services used for bulk execution.
	// If dialing fails, the gov portal execution endpoints will return errors.
	if cfg.CredentialGRPCAddr != "" {
		conn, dialErr := grpc.NewClient(cfg.CredentialGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			log.Printf("WARNING: credential gRPC dial failed (%s): %v", cfg.CredentialGRPCAddr, dialErr)
		} else {
			defer conn.Close()
			svc.SetCredentialClient(credentialv1.NewCredentialServiceClient(conn))
		}
	}

	if cfg.EnrollmentGRPCAddr != "" {
		conn, dialErr := grpc.NewClient(cfg.EnrollmentGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			log.Printf("WARNING: enrollment gRPC dial failed (%s): %v", cfg.EnrollmentGRPCAddr, dialErr)
		} else {
			defer conn.Close()
			svc.SetEnrollmentClient(enrollmentv1.NewEnrollmentServiceClient(conn))
		}
	}

	if cfg.IdentityGRPCAddr != "" {
		conn, dialErr := grpc.NewClient(cfg.IdentityGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			log.Printf("WARNING: identity gRPC dial failed (%s): %v", cfg.IdentityGRPCAddr, dialErr)
		} else {
			defer conn.Close()
			svc.SetIdentityClient(identityv1.NewIdentityServiceClient(conn))
		}
	}

	if cfg.AuditGRPCAddr != "" {
		conn, dialErr := grpc.NewClient(cfg.AuditGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			log.Printf("WARNING: audit gRPC dial failed (%s): %v", cfg.AuditGRPCAddr, dialErr)
		} else {
			defer conn.Close()
			svc.SetAuditClient(auditv1.NewAuditServiceClient(conn))
		}
	}

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
