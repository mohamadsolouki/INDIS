// INDIS — Transitional justice — anonymous testimony, conditional amnesty workflows
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/config"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS justice service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("justice")
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
	justicev1.RegisterJusticeServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS justice service...")
	grpcServer.GracefulStop()
}
