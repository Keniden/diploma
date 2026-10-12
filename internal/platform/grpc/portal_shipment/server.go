// Package portal_shipment is the gRPC server for the internal carrier
// portal. The skeleton ships a stub that registers the service and returns
// codes.Unimplemented for every method.
//
// TODO(LASTMILE-095): port the real handlers (ListShipments,
// GetShipmentByID, export endpoints, ...) behind internal/application/portal
// and the portal queries once they are ported.
package portal_shipment

import (
	"google.golang.org/grpc"

	pb "last-mile-task/pkg/gen/api/proto/v1/portal_internal"
)

// Server is a stub implementation of the internal portal service.
type Server struct {
	pb.UnimplementedInternalPortalServer
}

// New builds the stub server.
func New() *Server {
	return &Server{}
}

// RegisterGRPC registers the service on the gRPC server.
func (s *Server) RegisterGRPC(srv *grpc.Server) {
	pb.RegisterInternalPortalServer(srv, s)
}