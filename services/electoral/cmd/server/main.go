// INDIS — Electoral module — STARK-ZK voter verification, referendum support
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indistrace "github.com/IranProsperityProject/INDIS/pkg/tracing"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
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

	indismetrics.Init("electoral")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "electoral")
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
	svc := service.NewWithNonceReplayWindow(repo, cfg.ZKProofURL, time.Duration(cfg.RemoteNonceWindowMinutes)*time.Minute)
	h := handler.New(svc)

	// Admin HTTP server for election lifecycle management (port 9200).
	// Protected by admin role in gateway; not exposed directly to public.
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/v1/electoral/elections/", func(w http.ResponseWriter, r *http.Request) {
		// Expect: POST /v1/electoral/elections/{id}/finalize
		path := r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Extract election ID from path.
		var electionID string
		var action string
		parts := splitPath(path)
		if len(parts) == 5 && parts[4] == "finalize" {
			electionID = parts[3]
			action = "finalize"
		}
		if electionID == "" || action == "" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		adminDID := r.Header.Get("X-Admin-DID")
		if adminDID == "" {
			adminDID = "admin"
		}
		if err := svc.FinalizeElection(r.Context(), electionID, adminDID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"election_id": electionID, "status": "tallied"})
	})
	go func() {
		log.Printf("Electoral admin HTTP listening on :9200")
		if err := http.ListenAndServe(":9200", adminMux); err != nil {
			log.Printf("electoral admin HTTP error: %v", err)
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
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("electoral")))...)
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

func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
