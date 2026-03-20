// Command server is the INDIS API gateway.
//
// It exposes an HTTP/1.1 REST API on port 8080 and proxies requests to the
// gRPC backend services (identity:50051 … justice:50058) using generated
// protobuf client stubs.  Per-IP token-bucket rate limiting is enforced
// before any request reaches a backend.
//
// Additional responsibilities:
//   - JWT (HS256) and API-key authentication via auth middleware
//   - CORS headers via cors middleware
//   - Security headers (X-Content-Type-Options, etc.) on every response
//   - Privacy Control Center API backed by gateway's own Postgres tables
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	indismetrics "github.com/IranProsperityProject/INDIS/pkg/metrics"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/auth"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/config"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/cors"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/handler"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/proxy"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/ratelimit"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	indismetrics.Init("gateway")
	if err := indismetrics.ServeMetrics(fmt.Sprintf(":%d", cfg.MetricsPort)); err != nil {
		log.Fatalf("metrics: %v", err)
	}
	log.Printf("Metrics endpoint listening on :%d/metrics", cfg.MetricsPort)

	clients, err := proxy.New(
		cfg.IdentityAddr,
		cfg.CredentialAddr,
		cfg.EnrollmentAddr,
		cfg.BiometricAddr,
		cfg.AuditAddr,
		cfg.NotificationAddr,
		cfg.ElectoralAddr,
		cfg.JusticeAddr,
		proxy.TransportConfig{
			Mode:           cfg.BackendTLSMode,
			CAFile:         cfg.BackendCAFile,
			ClientCertFile: cfg.BackendClientCertFile,
			ClientKeyFile:  cfg.BackendClientKeyFile,
		},
	)
	if err != nil {
		log.Fatalf("proxy: %v", err)
	}
	defer clients.Close()

	// Gateway's own Postgres pool for consent rules and data-export requests.
	// If DATABASE_URL is unset or the DB is unreachable, the gateway degrades
	// gracefully: privacy APIs return 503, all other routes remain functional.
	var repo *repository.Repository
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, poolErr := repository.NewPool(ctx, cfg.DatabaseURL)
		cancel()
		if poolErr != nil {
			log.Printf("WARNING: gateway DB unavailable (%v); privacy APIs disabled", poolErr)
		} else {
			migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if migrateErr := repository.Migrate(migrateCtx, pool); migrateErr != nil {
				log.Printf("WARNING: gateway DB migration failed (%v); privacy APIs disabled", migrateErr)
			} else {
				repo = repository.New(pool)
				log.Printf("Gateway DB connected and migrated")
			}
			migrateCancel()
			defer pool.Close()
		}
	}

	// Parse API keys from config.
	apiKeys := auth.ParseAPIKeysEnv(cfg.APIKeys)
	if len(apiKeys) == 0 {
		log.Printf("WARNING: no API_KEYS configured; service-to-service API key auth disabled")
	}

	limiter := ratelimit.New(cfg.RateLimitRPS)
	gw := handler.New(clients, limiter, repo, cfg.VerifierHTTPURL, cfg.CardHTTPURL)

	// Nonce cache for JWT jti replay protection.
	nonceCache := auth.NewNonceCache()

	// Build middleware chain: CORS → Auth → Gateway handler.
	var h http.Handler = gw
	h = auth.Middleware(cfg.JWTSecret, apiKeys, nonceCache)(h)
	h = cors.Middleware(cfg.CORSAllowedOrigins)(h)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("INDIS gateway listening on :%d", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway…")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Gateway stopped.")
}
