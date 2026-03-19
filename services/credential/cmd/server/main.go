// INDIS — Credential issuance, verification, revocation, and selective disclosure
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/cache"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
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

	indismetrics.Init("credential")
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

	// In production the signing key is loaded from HSM (FIPS 140-2 Level 3).
	// For development a fresh ephemeral key is generated on startup.
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("key generation: %v", err)
	}

	repo := repository.New(pool)
	chain := blockchain.NewMockAdapter()
	svc := service.New(repo, chain, cfg.IssuerDID, privateKey)

	// Wire zkproof service for ZK proof verification (PRD §FR-002).
	if cfg.ZKProofURL != "" {
		svc.SetZKProofURL(cfg.ZKProofURL)
		log.Printf("ZK proof verification enabled: %s", cfg.ZKProofURL)
	}

	if redisAddr := redisAddrFromConfig(cfg.RedisURL); redisAddr != "" {
		revocationCache, cacheErr := cache.NewRedisRevocationCache(redisAddr)
		if cacheErr != nil {
			log.Printf("revocation cache disabled: %v", cacheErr)
		} else {
			svc.SetRevocationCache(revocationCache)
			defer func() {
				if cerr := revocationCache.Close(); cerr != nil {
					log.Printf("revocation cache close: %v", cerr)
				}
			}()
		}
	}

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

	grpcOpts, err := indistls.ServerOptionsFromEnv()
	if err != nil {
		log.Fatalf("grpc transport options: %v", err)
	}
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("credential")))...)
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

func redisAddrFromConfig(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return u.Host
	}
	return raw
}
