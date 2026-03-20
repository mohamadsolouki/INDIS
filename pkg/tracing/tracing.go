// Package tracing provides OpenTelemetry distributed tracing setup for INDIS services.
//
// Each service calls Init at startup to install a global TracerProvider that exports
// spans to an OTLP collector (Jaeger/Grafana Tempo in dev, production collector in prod).
// If OTEL_EXPORTER_OTLP_ENDPOINT is unset or empty, tracing is disabled and a no-op
// provider is installed so service code does not need conditional checks.
//
// Usage:
//
//	shutdown, err := tracing.Init(ctx, "identity")
//	if err != nil {
//	    log.Fatalf("tracing: %v", err)
//	}
//	defer shutdown(context.Background())
package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ShutdownFunc flushes and shuts down the TracerProvider.
// Call it (with a timeout context) during service shutdown.
type ShutdownFunc func(ctx context.Context) error

// Init configures the global OpenTelemetry TracerProvider for the named service.
//
// The OTLP endpoint is read from the OTEL_EXPORTER_OTLP_ENDPOINT environment variable
// (e.g. "localhost:4317").  If the variable is absent or empty, a no-op provider is
// installed and Init returns immediately without opening a network connection.
//
// Returns a ShutdownFunc that must be called on service exit to flush pending spans.
func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Tracing disabled — install a no-op provider so Tracer() calls are safe.
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(_ context.Context) error { return nil }, nil
	}

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: dial OTLP endpoint %q: %w", endpoint, err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("tracing: create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceNamespace("indis"),
		),
	)
	if err != nil {
		// Resource errors are non-fatal; fall back to the default resource.
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	shutdown := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("tracing: shutdown: %w", err)
		}
		return conn.Close()
	}
	return shutdown, nil
}

// Tracer returns a named tracer from the global provider.
// Equivalent to otel.Tracer(name) but provided here for import locality.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
