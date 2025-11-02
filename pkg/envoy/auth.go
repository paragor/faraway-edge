package envoy

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TokenAuthInterceptor creates a gRPC unary interceptor that validates bearer tokens
func TokenAuthInterceptor(expectedToken string, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// If no token is configured, allow all requests
		if expectedToken == "" {
			return handler(ctx, req)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("request without metadata", slog.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Check for authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn("request without authorization header", slog.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Validate token (expecting "Bearer <token>" format)
		token := authHeaders[0]
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		if token != expectedToken {
			logger.Warn("invalid token", slog.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		logger.Debug("request authenticated", slog.String("method", info.FullMethod))
		return handler(ctx, req)
	}
}

// TokenAuthStreamInterceptor creates a gRPC stream interceptor that validates bearer tokens
func TokenAuthStreamInterceptor(expectedToken string, logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// If no token is configured, allow all requests
		if expectedToken == "" {
			return handler(srv, ss)
		}

		// Extract metadata from context
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("stream request without metadata", slog.String("method", info.FullMethod))
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Check for authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn("stream request without authorization header", slog.String("method", info.FullMethod))
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Validate token (expecting "Bearer <token>" format)
		token := authHeaders[0]
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		if token != expectedToken {
			logger.Warn("invalid token for stream", slog.String("method", info.FullMethod))
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		logger.Debug("stream request authenticated", slog.String("method", info.FullMethod))
		return handler(srv, ss)
	}
}