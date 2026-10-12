package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"last-mile-task/internal/platform/requestid"
)

// RequestID extracts X-Request-Id from the incoming metadata and stores it
// in the request context so the logging layer can attach it.
func RequestID() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		return handler(requestid.FromIncomingContext(ctx), req)
	}
}