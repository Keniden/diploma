// Package interceptors provides standard gRPC unary server interceptors:
// panic recovery, request-id extraction and request logging.
//
// The real repository adds tracing, metrics, claims and auth interceptors
// (grpc-middleware v2 + OpenTelemetry + Prometheus); none of those
// dependencies exist in the skeleton. TODO(LASTMILE-091): port the missing
// interceptors once tracing/metrics/auth are allowed in go.mod.
package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery converts panics inside handlers into codes.Internal errors
// instead of crashing the gRPC server.
func Recovery() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = status.Errorf(codes.Internal, "%v", p)

				slog.ErrorContext(ctx,
					"grpc: panic recovered",
					"panic", fmt.Sprint(p),
					"stack", string(debug.Stack()),
				)
			}
		}()

		return handler(ctx, req)
	}
}