// INDIS — Audit logging — append-only, cryptographically signed, 10-year retention
//
// Exposes two ports:
//
//	gRPC on GRPC_PORT (default :50055) — AppendEvent, QueryEvents, GetEventByID
//	HTTP on HTTP_PORT (default :9200)  — GET /v1/audit/events (query/search)
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
	"strconv"
	"syscall"
	"time"

	auditv1 "github.com/mohamadsolouki/INDIS/api/gen/go/audit/v1"
	"github.com/mohamadsolouki/INDIS/pkg/blockchain"
	indismetrics "github.com/mohamadsolouki/INDIS/pkg/metrics"
	indistrace "github.com/mohamadsolouki/INDIS/pkg/tracing"
	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
	indistls "github.com/mohamadsolouki/INDIS/pkg/tls"
	"github.com/mohamadsolouki/INDIS/services/audit/internal/config"
	"github.com/mohamadsolouki/INDIS/services/audit/internal/handler"
	"github.com/mohamadsolouki/INDIS/services/audit/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/audit/internal/service"
	"google.golang.org/grpc"
)

func main() {
	log.Printf("Starting INDIS audit service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("audit")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := indistrace.Init(ctx, "audit")
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

	chain := blockchain.NewMockAdapter()
	log.Printf("Blockchain backend: %s", cfg.BlockchainType)

	repo := repository.New(pool)
	svc := service.New(repo, chain)
	h := handler.New(svc)

	go func() {
		if err := runCredentialRevokedConsumer(ctx, cfg.KafkaBrokers, cfg.KafkaGroupID, svc); err != nil && ctx.Err() == nil {
			log.Printf("credential revoked consumer stopped: %v", err)
		}
	}()

	// ── gRPC server ────────────────────────────────────────────────────────────

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcOpts, err := indistls.ServerOptionsFromEnv()
	if err != nil {
		log.Fatalf("grpc transport options: %v", err)
	}
	grpcServer := grpc.NewServer(append(grpcOpts, grpc.UnaryInterceptor(indismetrics.UnaryServerInterceptor("audit")))...)
	auditv1.RegisterAuditServiceServer(grpcServer, h)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// ── HTTP query server ──────────────────────────────────────────────────────
	// GET /v1/audit/events?actor_did=...&subject_did=...&action=...&from=...&to=...&limit=50
	//
	// This endpoint lets the API gateway (and internal tooling) query audit events
	// without going through gRPC, which simplifies scripting and debugging.
	// In production it should be firewalled to internal networks only.

	if cfg.HTTPPort > 0 {
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/audit/events", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				writeJSONError(w, http.StatusMethodNotAllowed, "GET required")
				return
			}
			q := r.URL.Query()

			var limit int32 = 50
			if v := q.Get("limit"); v != "" {
				n, err := strconv.Atoi(v)
				if err == nil && n > 0 {
					limit = int32(n)
				}
			}

			req := &auditv1.QueryEventsRequest{
				ActorDid:   q.Get("actor_did"),
				SubjectDid: q.Get("subject_did"),
				FromTime:   q.Get("from"),
				ToTime:     q.Get("to"),
				Limit:      limit,
				PageToken:  q.Get("page_token"),
			}

			recs, nextToken, err := svc.QueryEvents(r.Context(), req)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}

			type eventOut struct {
				EventID    string `json:"event_id"`
				Category   int32  `json:"category"`
				Action     string `json:"action"`
				ActorDID   string `json:"actor_did"`
				SubjectDID string `json:"subject_did"`
				ResourceID string `json:"resource_id"`
				ServiceID  string `json:"service_id"`
				PrevHash   string `json:"prev_hash"`
				EntryHash  string `json:"entry_hash"`
				Timestamp  string `json:"timestamp"`
			}
			events := make([]eventOut, len(recs))
			for i, rec := range recs {
				events[i] = eventOut{
					EventID:    rec.EventID,
					Category:   rec.Category,
					Action:     rec.Action,
					ActorDID:   rec.ActorDID,
					SubjectDID: rec.SubjectDID,
					ResourceID: rec.ResourceID,
					ServiceID:  rec.ServiceID,
					PrevHash:   rec.PrevHash,
					EntryHash:  rec.EntryHash,
					Timestamp:  rec.Timestamp.UTC().Format(time.RFC3339),
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"events":          events,
				"next_page_token": nextToken,
			})
		})

		httpSrv := &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			log.Printf("Audit HTTP query endpoint listening on :%d", cfg.HTTPPort)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("audit HTTP server error: %v", err)
			}
		}()

		// Ensure the HTTP server also shuts down on signal.
		go func() {
			<-ctx.Done()
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutCancel()
			_ = httpSrv.Shutdown(shutCtx)
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS audit service...")
	cancel()
	grpcServer.GracefulStop()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
