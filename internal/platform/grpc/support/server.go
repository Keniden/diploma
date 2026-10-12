// Package support is the gRPC server for the support (cancel) endpoints.
// The skeleton ships a stub that registers the service and returns
// codes.Unimplemented for every method.
//
// TODO(LASTMILE-095): port the real handlers behind internal/application
// once support is ported.
package support

import (
	"google.golang.org/grpc"

	pb "last-mile-task/pkg/gen/api/proto/v1/internals/support"
)

// Server is a stub implementation of the support service.
type Server struct {
	pb.UnimplementedSupportServer
}

// New builds the stub server.
func New() *Server {
	return &Server{}
}

// RegisterGRPC registers the service on the gRPC server.
func (s *Server) RegisterGRPC(srv *grpc.Server) {
	pb.RegisterSupportServer(srv, s)
}