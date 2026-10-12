// Package maintenance is the gRPC server for the maintenance (sample data)
// service. The skeleton ships a stub that registers the service and returns
// codes.Unimplemented for every method.
//
// TODO(LASTMILE-095): port the real handlers (GenerateSampleData,
// CreateSampleData, DeleteSampleData) behind internal/maintenance once that
// package is ported.
package maintenance

import (
	"google.golang.org/grpc"

	pb "last-mile-task/pkg/gen/api/proto/v1/private/maintenance"
)

// Server is a stub implementation of the maintenance service.
type Server struct {
	pb.UnimplementedMaintenanceServer
}

// New builds the stub server.
func New() *Server {
	return &Server{}
}

// RegisterGRPC registers the service on the gRPC server.
func (s *Server) RegisterGRPC(srv *grpc.Server) {
	pb.RegisterMaintenanceServer(srv, s)
}