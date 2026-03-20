// INDIS — Transitional justice — anonymous testimony, conditional amnesty workflows
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

	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/config"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS justice service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("justice")
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

	repo := repository.New(pool)
	svc := service.New(repo, cfg.ZKProofURL)
	h := handler.New(svc)

	// Admin HTTP server for justice case management (port 9300).
	// Protected by senior/admin role at gateway; not exposed directly to public.
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/v1/justice/cases/", func(w http.ResponseWriter, r *http.Request) {
		// Expect: POST /v1/justice/cases/{case_id}/advance
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := splitPathJustice(r.URL.Path)
		if len(parts) < 4 || parts[len(parts)-1] != "advance" {
			http.Error(w, "not found — use POST /v1/justice/cases/{id}/advance", http.StatusNotFound)
			return
		}
		caseID := parts[len(parts)-2]
		adminDID := r.Header.Get("X-Admin-DID")
		if adminDID == "" {
			adminDID = "admin"
		}
		outCaseID, newStatus, err := svc.AdvanceCaseStatus(r.Context(), caseID, adminDID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"case_id": outCaseID, "status": newStatus})
	})
	go func() {
		log.Printf("Justice admin HTTP listening on :9300")
		if err := http.ListenAndServe(":9300", adminMux); err != nil {
			log.Printf("justice admin HTTP error: %v", err)
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
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("justice")))...)
	justicev1.RegisterJusticeServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS justice service...")
	grpcServer.GracefulStop()
}

func splitPathJustice(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
