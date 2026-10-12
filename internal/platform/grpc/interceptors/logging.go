package interceptors

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
)

// Logging traces every unary call with the full method name and the error
// (when any).
func Logging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		slog.InfoContext(ctx,
			"grpc: handle",
			"rpc", info.FullMethod,
		)

		resp, err = handler(ctx, req)
		if err != nil {
			slog.ErrorContext(ctx,
				"grpc: handled",
				"rpc", info.FullMethod,
				"error", err,
			)
		}

		return resp, err
	}
}