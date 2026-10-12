package application

import (
	platformgrpc "last-mile-task/internal/platform/grpc"
	grpcmaintenance "last-mile-task/internal/platform/grpc/maintenance"
)

// registerMaintenance returns the maintenance gRPC service.
//
// TODO(LASTMILE-095): replace the stub with the real server backed by
// internal/maintenance (sample data generation) once it is ported.
func (a *Application) registerMaintenance() platformgrpc.Service {
	return grpcmaintenance.New()
}
