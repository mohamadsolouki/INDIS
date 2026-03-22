// INDIS — Notification service — SMS, Push, Email for credential expiry and verification alerts
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

	notificationv1 "github.com/mohamadsolouki/INDIS/api/gen/go/notification/v1"
	indismetrics "github.com/mohamadsolouki/INDIS/pkg/metrics"
	indistrace "github.com/mohamadsolouki/INDIS/pkg/tracing"
	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
	indistls "github.com/mohamadsolouki/INDIS/pkg/tls"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/config"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/handler"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS notification service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("notification")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "notification")
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
	svc := service.New(repo)
	h := handler.New(svc)

	// Start background dispatcher that delivers queued notifications.
	go svc.RunDispatcher(ctx, 30*time.Second)

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
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("notification")))...)
	notificationv1.RegisterNotificationServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS notification service...")
	cancel()
	grpcServer.GracefulStop()
}
