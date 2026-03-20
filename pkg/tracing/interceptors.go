package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// ServerStatsHandler returns an otelgrpc stats handler for gRPC servers.
// Wire it with grpc.StatsHandler(tracing.ServerStatsHandler()) when building
// the grpc.Server.  It automatically propagates the W3C traceparent header
// from incoming metadata and creates server-side spans.
func ServerStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// ClientStatsHandler returns an otelgrpc stats handler for gRPC clients.
// Wire it with grpc.WithStatsHandler(tracing.ClientStatsHandler()) when
// dialling upstream services so outbound calls are traced.
func ClientStatsHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

// ServerOption wraps ServerStatsHandler as a grpc.ServerOption for convenience.
func ServerOption() grpc.ServerOption {
	return grpc.StatsHandler(ServerStatsHandler())
}

// DialOption wraps ClientStatsHandler as a grpc.DialOption for convenience.
func DialOption() grpc.DialOption {
	return grpc.WithStatsHandler(ClientStatsHandler())
}
