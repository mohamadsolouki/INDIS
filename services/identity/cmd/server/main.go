// INDIS — Core identity management — DID generation, resolution, lifecycle
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/config"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS identity service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("identity")
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

	migrationsDir, err := indismigrate.ResolveMigrationsDir("")
	if err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := indismigrate.Migrate(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("migrations apply: %v", err)
	}

	repo := repository.New(pool)
	chain := blockchain.NewMockAdapter()
	svc := service.New(repo, chain)

	producer, err := events.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		log.Printf("events producer disabled: %v", err)
	} else {
		svc.SetEventPublisher(producer)
		defer func() {
			if cerr := producer.Close(); cerr != nil {
				log.Printf("events producer close: %v", cerr)
			}
		}()
	}

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
	identityv1.RegisterIdentityServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS identity service...")
	grpcServer.GracefulStop()
}
