// Package metrics provides standard Prometheus metric definitions for INDIS services.
// All metrics use the "indis_" prefix and are registered with the default Prometheus registry.
// Call Init once at service startup before recording any observations.
package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Standard INDIS metric definitions.
var (
	// OperationsTotal counts service operations by service, operation, and status.
	// Labels: service, operation, status ("ok"|"error").
	OperationsTotal *prometheus.CounterVec

	// OperationDuration measures operation latency in seconds.
	// Labels: service, operation.
	OperationDuration *prometheus.HistogramVec

	// GRPCRequestsTotal counts gRPC requests by service, method, and code.
	// Labels: service, method, code.
	GRPCRequestsTotal *prometheus.CounterVec

	// EnrollmentsTotal counts enrollment attempts by pathway and status.
	// Labels: pathway ("standard"|"enhanced"|"social"), status.
	EnrollmentsTotal *prometheus.CounterVec

	// ZKProofDuration measures ZK proof generation/verification latency in seconds.
	// Labels: circuit, operation ("generate"|"verify").
	ZKProofDuration *prometheus.HistogramVec

	// ActiveConnections tracks active gRPC connections per service.
	// Labels: service.
	ActiveConnections *prometheus.GaugeVec
)

// operationBuckets are the histogram buckets used for operation and ZK proof duration metrics.
var operationBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0}

// Init registers all INDIS metrics with the default Prometheus registry.
// serviceName is embedded in the help strings for clarity.
// This function must be called exactly once at service startup before any
// metric observations are recorded.
func Init(serviceName string) {
	OperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "indis_operations_total",
			Help: fmt.Sprintf("Total number of operations performed by the %s service, partitioned by service, operation, and status.", serviceName),
		},
		[]string{"service", "operation", "status"},
	)

	OperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "indis_operation_duration_seconds",
			Help:    fmt.Sprintf("Latency of operations performed by the %s service in seconds.", serviceName),
			Buckets: operationBuckets,
		},
		[]string{"service", "operation"},
	)

	GRPCRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "indis_grpc_requests_total",
			Help: fmt.Sprintf("Total number of gRPC requests handled by the %s service, partitioned by service, method, and status code.", serviceName),
		},
		[]string{"service", "method", "code"},
	)

	EnrollmentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "indis_enrollments_total",
			Help: "Total number of enrollment attempts, partitioned by pathway and status.",
		},
		[]string{"pathway", "status"},
	)

	ZKProofDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "indis_zk_proof_duration_seconds",
			Help:    "Latency of ZK proof generation and verification operations in seconds.",
			Buckets: operationBuckets,
		},
		[]string{"circuit", "operation"},
	)

	ActiveConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "indis_active_connections",
			Help: "Number of active gRPC connections per service.",
		},
		[]string{"service"},
	)

	prometheus.MustRegister(
		OperationsTotal,
		OperationDuration,
		GRPCRequestsTotal,
		EnrollmentsTotal,
		ZKProofDuration,
		ActiveConnections,
	)
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() http.Handler { return promhttp.Handler() }

// ServeMetrics starts an HTTP server exposing /metrics on the given address.
// It returns immediately; the server runs in a background goroutine.
// The returned error is non-nil only if the listener cannot be bound (e.g. address already in use).
// Example: ServeMetrics(":9090")
func ServeMetrics(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler())
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Probe that the address is bindable before detaching into a goroutine so
	// that callers receive a synchronous error on misconfiguration.
	ln, err := newListener(addr)
	if err != nil {
		return fmt.Errorf("metrics: cannot listen on %s: %w", addr, err)
	}

	go func() {
		// Errors from Serve after a clean shutdown are intentionally ignored.
		_ = server.Serve(ln)
	}()

	return nil
}
