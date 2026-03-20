// INDIS — Verifier Service — ZK-proof-based credential verification and verifier org management.
// Implements PRD FR-012 (verifier registration) and FR-013 (credential verification).
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	verifierv1 "github.com/IranProsperityProject/INDIS/api/gen/go/verifier/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indistrace "github.com/IranProsperityProject/INDIS/pkg/tracing"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/verifier/internal/config"
	"github.com/IranProsperityProject/INDIS/services/verifier/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/verifier/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/verifier/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS verifier service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("verifier")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "verifier")
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
		log.Fatalf("migrations apply: %v", err)
	}

	repo := repository.New(pool)
	svc := service.New(repo, cfg.ZKProofURL)
	h := handler.New(svc)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcOpts, err := indistls.ServerOptionsFromEnv()
	if err != nil {
		log.Fatalf("grpc transport options: %v", err)
	}
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("verifier")))...)
	verifierv1.RegisterVerifierServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS verifier service...")
	grpcServer.GracefulStop()
}
