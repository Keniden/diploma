package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	publicdriver "last-mile-task/internal/application/public_driver"
	"last-mile-task/internal/shipping/domain/values"
)

// DriverIDHeader is the metadata key carrying the driver identity. The real
// proxy (client-proxy-gateway) injects the driver id; in the skeleton the
// e2e/mock client sends it explicitly.
const DriverIDHeader = "x-driver-id"

// DriverID resolves the driver id from the incoming gRPC metadata into the
// context consumed by the drivers provider.
//
// TODO(LASTMILE-086): replace with the real auth/gateway identity once the
// client-proxy integration is ported.
func DriverID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(DriverIDHeader); len(vals) > 0 && vals[0] != "" {
				if id, err := uuid.Parse(vals[0]); err == nil {
					ctx = context.WithValue(ctx, publicdriver.ContextDriverIDKey{}, values.DriverID(id))
				}
			}
		}
		return handler(ctx, req)
	}
}