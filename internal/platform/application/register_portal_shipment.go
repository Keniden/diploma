package application

import (
	platformgrpc "last-mile-task/internal/platform/grpc"
	grpcportalshipment "last-mile-task/internal/platform/grpc/portal_shipment"
)

// registerPortalShipment returns the internal portal gRPC service.
//
// TODO(LASTMILE-095): replace the stub with the real server backed by
// internal/application/portal and the portal queries once they are ported.
func (a *Application) registerPortalShipment() platformgrpc.Service {
	return grpcportalshipment.New()
}
