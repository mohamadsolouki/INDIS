package metrics

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor records request counts, status codes, and method
// latency for gRPC unary handlers.
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		method := shortMethodName(info.FullMethod)
		code := status.Code(err)
		codeLabel := code.String()

		if GRPCRequestsTotal != nil {
			GRPCRequestsTotal.WithLabelValues(serviceName, method, codeLabel).Inc()
		}
		if OperationsTotal != nil {
			statusLabel := "ok"
			if code != codes.OK {
				statusLabel = "error"
			}
			OperationsTotal.WithLabelValues(serviceName, method, statusLabel).Inc()
		}
		if OperationDuration != nil {
			OperationDuration.WithLabelValues(serviceName, method).Observe(time.Since(start).Seconds())
		}

		return resp, err
	}
}

func shortMethodName(fullMethod string) string {
	if fullMethod == "" {
		return "unknown"
	}
	idx := strings.LastIndex(fullMethod, "/")
	if idx == -1 || idx+1 >= len(fullMethod) {
		return fullMethod
	}
	return fullMethod[idx+1:]
}
