package application

import (
	platformgrpc "last-mile-task/internal/platform/grpc"
	grpcsupport "last-mile-task/internal/platform/grpc/support"
)

// registerSupport returns the support gRPC service.
//
// TODO(LASTMILE-095): replace the stub with the real server backed by
// internal/support once it is ported.
func (a *Application) registerSupport() platformgrpc.Service {
	return grpcsupport.New()
}
