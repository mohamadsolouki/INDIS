// INDIS — Biometric management — capture, deduplication, template storage
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	biometricv1 "github.com/IranProsperityProject/INDIS/api/gen/go/biometric/v1"
	indiscrypto "github.com/IranProsperityProject/INDIS/pkg/crypto"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/biometric/internal/config"
	"github.com/IranProsperityProject/INDIS/services/biometric/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/biometric/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/biometric/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS biometric service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("biometric")
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

	// In production the AES key is loaded from the HSM (FIPS 140-2 Level 3).
	encryptKey, err := indiscrypto.GenerateRandomKey(32)
	if err != nil {
		log.Fatalf("key generation: %v", err)
	}

	repo := repository.New(pool)
	svc := service.New(repo, encryptKey, cfg.AIServiceURL)
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
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("biometric")))...)
	biometricv1.RegisterBiometricServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS biometric service...")
	grpcServer.GracefulStop()
}
