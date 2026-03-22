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
	"time"

	identityv1 "github.com/mohamadsolouki/INDIS/api/gen/go/identity/v1"
	"github.com/mohamadsolouki/INDIS/pkg/blockchain"
	"github.com/mohamadsolouki/INDIS/pkg/events"
	indismetrics "github.com/mohamadsolouki/INDIS/pkg/metrics"
	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
	indistls "github.com/mohamadsolouki/INDIS/pkg/tls"
	indistrace "github.com/mohamadsolouki/INDIS/pkg/tracing"
	"github.com/mohamadsolouki/INDIS/services/identity/internal/config"
	"github.com/mohamadsolouki/INDIS/services/identity/internal/handler"
	"github.com/mohamadsolouki/INDIS/services/identity/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/identity/internal/service"
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

	tracingShutdown, err := indistrace.Init(ctx, "identity")
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
	grpcServer := grpc.NewServer(append(grpcOpts,
		grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("identity")),
		indistrace.ServerOption(),
	)...)
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
