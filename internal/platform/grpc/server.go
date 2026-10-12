// Package grpc bundles the gRPC server shared across services: it builds the
// server with the interceptor chain and registers the services from the
// subpackages.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"last-mile-task/internal/platform/grpc/interceptors"
)

// ServerConfig configures the gRPC server.
type ServerConfig struct {
	Addr string
}

// Server is the shared gRPC server.
type Server struct {
	config   ServerConfig
	logger   *slog.Logger
	server   *grpc.Server
	listener net.Listener
}

// NewServer builds the gRPC server with the interceptors available in the
// skeleton (recovery, request-id, logging).
func NewServer(
	logger *slog.Logger,
	cfg ServerConfig,
) *Server {
	interceptorChain := []grpc.UnaryServerInterceptor{
		interceptors.Recovery(),
		interceptors.RequestID(),
		interceptors.DriverID(),
		interceptors.Logging(),
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptorChain...),
	)

	return &Server{
		config: cfg,
		logger: logger,
		server: srv,
	}
}

// Service is any gRPC server that knows how to register itself.
type Service interface {
	RegisterGRPC(srv *grpc.Server)
}

// RegisterServices registers the given services on the underlying gRPC
// server.
func (s *Server) RegisterServices(services ...Service) {
	for _, service := range services {
		service.RegisterGRPC(s.server)
	}
}

// ListenAndServe binds the configured address and starts serving. It blocks
// until the server is stopped.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("grpc: listen to tcp@%s: %w", s.config.Addr, err)
	}

	s.listener = listener

	s.logger.Info(
		"grpc: grpc server starting",
		"addr", s.config.Addr,
	)

	return s.server.Serve(listener)
}

// Shutdown gracefully stops the server; it is forced to stop when the
// context expires.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info(
		"grpc: grpc server shutting down",
		"addr", s.config.Addr,
	)

	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.server.Stop()
		return errors.New("grpc: server force shutdown")
	case <-done:
	}

	return nil
}