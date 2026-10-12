package publicdriver

import (
	"context"
	"errors"

	"last-mile-task/internal/shipping/domain/values"
)

// ContextDriverIDKey is the context key under which the gRPC layer stores the
// driver ID resolved from the incoming request metadata (x-driver-id header).
type ContextDriverIDKey struct{}

// ContextDriversProvider resolves the current driver from a context value set
// by the request-id/driver-id gRPC interceptors.
//
// TODO(LASTMILE-081): the real provider returns a full values.Driver with
// roles/supplier resolved through users/client-proxy plus gateway metadata.
// The skeleton resolves only the driver ID.
type ContextDriversProvider struct{}

// NewContextDriversProvider builds the context-backed drivers provider.
func NewContextDriversProvider() *ContextDriversProvider {
	return &ContextDriversProvider{}
}

// GetDriverFromContext returns the driver ID stored in the context by the
// interceptor as a minimal Driver (supplier unresolved to 0), or an error
// when the request carries no driver identity.
func (p *ContextDriversProvider) GetDriverFromContext(ctx context.Context) (values.Driver, error) {
	id, ok := ctx.Value(ContextDriverIDKey{}).(values.DriverID)
	if !ok {
		return values.Driver{}, errors.New("driver id missing in request context")
	}

	return values.Driver{
		ID:         id,
		SupplierID: 0,
	}, nil
}