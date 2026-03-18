// INDIS — Electoral module — STARK-ZK voter verification, referendum support
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/config"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS electoral service...")

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

	repo := repository.New(pool)
	svc := service.New(repo)
	h := handler.New(svc)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcOpts, err := serverTransportOptionsFromEnv()
	if err != nil {
		log.Fatalf("grpc transport options: %v", err)
	}
	grpcServer := grpc.NewServer(grpcOpts...)
	electoralv1.RegisterElectoralServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS electoral service...")
	grpcServer.GracefulStop()
}

func serverTransportOptionsFromEnv() ([]grpc.ServerOption, error) {
	mode := os.Getenv("GRPC_TLS_MODE")
	if mode == "" || mode == "plaintext" {
		return nil, nil
	}
	if mode != "tls" {
		return nil, fmt.Errorf("GRPC_TLS_MODE must be plaintext or tls, got %q", mode)
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	caFile := os.Getenv("TLS_CA_FILE")
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE are required when GRPC_TLS_MODE=tls")
	}

	creds, err := indistls.LoadServerTLS(certFile, keyFile, caFile)
	if err != nil {
		return nil, err
	}
	return []grpc.ServerOption{grpc.Creds(creds)}, nil
}
