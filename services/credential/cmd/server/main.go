// INDIS — Credential issuance, verification, revocation, and selective disclosure
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/config"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS credential service...")

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

	// In production the signing key is loaded from HSM (FIPS 140-2 Level 3).
	// For development a fresh ephemeral key is generated on startup.
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("key generation: %v", err)
	}

	repo := repository.New(pool)
	chain := blockchain.NewMockAdapter()
	svc := service.New(repo, chain, cfg.IssuerDID, privateKey)

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

	go func() {
		if err := runEnrollmentCompletedConsumer(ctx, cfg.KafkaBrokers, cfg.KafkaGroupID, svc); err != nil && ctx.Err() == nil {
			log.Printf("enrollment consumer stopped: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	credentialv1.RegisterCredentialServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS credential service...")
	cancel()
	grpcServer.GracefulStop()
}
