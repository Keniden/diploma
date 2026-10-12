// Package tsd is the gRPC server for the TSD (terminal scanning device)
// endpoints. It serves the ScanShipment / ScanPackages / ListPackagesToScan
// use cases over internal/application/tsd (ARCHITECTURE.md §11.2 flows 4-6).
//
// The proto contract is simplified to Empty→Empty (TODO LASTMILE-008): the
// scan parameters are resolved from the incoming gRPC metadata in metadata.go.
package tsd

import (
	"context"
	stderrors "errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	tsdapp "last-mile-task/internal/application/tsd"
	"last-mile-task/internal/errors"
	"last-mile-task/internal/shipping/application/ports"
	pb "last-mile-task/pkg/gen/api/proto/v1/internals/tsd"
	"last-mile-task/pkg/qr"
)

var _ pb.TSDServer = (*Server)(nil)

// Server implements the TSD gRPC service.
type Server struct {
	pb.UnimplementedTSDServer

	logger *slog.Logger
	app    *tsdapp.Application
}

// New builds the gRPC server over the TSD application.
func New(logger *slog.Logger, app *tsdapp.Application) *Server {
	return &Server{
		logger: logger,
		app:    app,
	}
}

// RegisterGRPC registers the TSD service on the platform gRPC server.
func (s *Server) RegisterGRPC(srv *grpc.Server) {
	pb.RegisterTSDServer(srv, s)
}

// ScanShipment confirms the gate reserve carried by the QR token in the
// metadata.
func (s *Server) ScanShipment(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	token, err := resolveScanShipment(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.app.ScanShipment(ctx, token); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ScanPackages scans and verifies packages with partial success: packages that
// fail to apply are logged and discarded (the Empty contract cannot surface
// failed_packages until TODO LASTMILE-008 lands).
func (s *Server) ScanPackages(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, packages, err := resolveScanPackages(ctx)
	if err != nil {
		return nil, err
	}

	failures, err := s.app.ScanPackages(ctx, shipmentID, packages)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	for _, failure := range failures {
		s.logger.Debug("tsd: package scan failed",
			"package_id", failure.PackageID.String(),
			"error", failure.Err,
		)
	}

	return &emptypb.Empty{}, nil
}

// ListPackagesToScan lists the packages awaiting a scan at a route point.
func (s *Server) ListPackagesToScan(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, routePointID, err := resolveListPackagesToScan(ctx)
	if err != nil {
		return nil, err
	}

	packages, err := s.app.ListPackagesToScan(ctx, shipmentID, routePointID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	s.logger.Debug("tsd: listed packages to scan",
		"shipment_id", shipmentID.String(),
		"route_point_id", routePointID.String(),
		"count", len(packages),
	)

	return &emptypb.Empty{}, nil
}

// mapErrorToGRPC converts use case errors into gRPC statuses: a malformed QR
// token becomes InvalidArgument, a missing reserve becomes NotFound and
// typed internal errors keep their own code, everything else falls back to
// Internal.
func mapErrorToGRPC(err error) error {
	switch {
	case stderrors.Is(err, qr.ErrMalformedSession):
		return status.Error(codes.InvalidArgument, err.Error())
	case stderrors.Is(err, ports.ErrReserveNotFound),
		stderrors.Is(err, ports.ErrShipmentNotFound),
		stderrors.Is(err, ports.ErrRouteNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return errors.MapToGRPC(err).Err()
	}
}
