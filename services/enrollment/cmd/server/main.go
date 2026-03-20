// INDIS — Enrollment processing — standard, enhanced, and social attestation pathways
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

	biometricv1 "github.com/IranProsperityProject/INDIS/api/gen/go/biometric/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indistrace "github.com/IranProsperityProject/INDIS/pkg/tracing"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/config"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Printf("Starting INDIS enrollment service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("enrollment")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "enrollment")
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

	// Connect to biometric service for deduplication (best-effort; non-fatal on failure).
	if cfg.BiometricServiceAddr != "" {
		bioConn, bioErr := grpc.NewClient(cfg.BiometricServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if bioErr != nil {
			log.Printf("biometric service connection skipped: %v", bioErr)
		} else {
			svc.SetBiometricClient(biometricv1.NewBiometricServiceClient(bioConn))
			defer bioConn.Close()
			log.Printf("Biometric service connected: %s", cfg.BiometricServiceAddr)
		}
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
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("enrollment")))...)
	enrollmentv1.RegisterEnrollmentServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS enrollment service...")
	grpcServer.GracefulStop()
}
