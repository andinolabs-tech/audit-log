package grpcserver

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// authUnary is a placeholder for future mTLS / JWT validation.
func authUnary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(ctx, req)
}

// New returns a gRPC server with tracing/metrics hooks. Callers register services on the returned instance.
func New() *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(authUnary),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
}
