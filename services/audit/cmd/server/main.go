// INDIS — Audit logging — append-only, cryptographically signed, 10-year retention
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/config"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS audit service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("audit")
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

	repo := repository.New(pool)
	svc := service.New(repo)
	h := handler.New(svc)

	go func() {
		if err := runCredentialRevokedConsumer(ctx, cfg.KafkaBrokers, cfg.KafkaGroupID, svc); err != nil && ctx.Err() == nil {
			log.Printf("credential revoked consumer stopped: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcOpts, err := indistls.ServerOptionsFromEnv()
	if err != nil {
		log.Fatalf("grpc transport options: %v", err)
	}
	grpcServer := grpc.NewServer(grpcOpts...)
	auditv1.RegisterAuditServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS audit service...")
	cancel()
	grpcServer.GracefulStop()
}
