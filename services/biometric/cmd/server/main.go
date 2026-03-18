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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// In production the AES key is loaded from the HSM (FIPS 140-2 Level 3).
	encryptKey, err := indiscrypto.GenerateRandomKey(32)
	if err != nil {
		log.Fatalf("key generation: %v", err)
	}

	repo := repository.New(pool)
	svc := service.New(repo, encryptKey)
	h := handler.New(svc)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
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
