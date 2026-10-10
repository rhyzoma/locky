package grpcserver

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor prevents a handler panic from terminating the server.
func RecoveryUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.ErrorContext(ctx, "panic recovered from gRPC handler", "method", info.FullMethod, "panic", recovered)
			resp = nil
			err = status.Error(codes.Internal, "internal server error")
		}
	}()

	return handler(ctx, req)
}
