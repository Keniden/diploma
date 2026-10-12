// Package drivergrpc is the gRPC transport of the public driver slice.
//
// TODO(LASTMILE-085): the real server exposes RegisterGRPC/RegisterGateway
// (grpc-gateway) and every RPC of the driver service. The skeleton ports the
// SetPackagesStatus, GetActiveShipmentByDriverID and the CreateShipment
// family handlers; the rest of the RPCs fall back to Unimplemented via the
// embedded driver.UnimplementedDriverServer.
package drivergrpc

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	"last-mile-task/internal/errors"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
	"last-mile-task/internal/utils"
	driver "last-mile-task/pkg/gen/api/proto/v1/public/driver"
)

var _ driver.DriverServer = (*Server)(nil)

// Server implements the driver gRPC service for the ported slice.
type Server struct {
	driver.UnimplementedDriverServer

	logger *slog.Logger
	app    *publicdriver.Application
}

// New builds the gRPC server over the public driver application.
func New(
	logger *slog.Logger,
	app *publicdriver.Application,
) *Server {
	return &Server{
		logger: logger,
		app:    app,
	}
}

// RegisterGRPC registers the driver service on the platform gRPC server.
func (s *Server) RegisterGRPC(srv *grpc.Server) {
	driver.RegisterDriverServer(srv, s)
}

// typedErrorMessage extracts the client-facing message of an internal error,
// falling back to the raw error text.
func typedErrorMessage(err error) string {
	var lmtErr *errors.Error
	if stderrors.As(err, &lmtErr) {
		return lmtErr.Message()
	}
	return err.Error()
}

// SetPackagesStatus implements [driver.DriverServer].
func (s *Server) SetPackagesStatus(ctx context.Context, req *driver.SetPackagesStatusRequest) (*driver.SetPackagesStatusResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	shipmentID, err := utils.ParseUUID(req.GetShipmentId())
	if err != nil {
		return nil, fmt.Errorf("parse shipment id: %w", err)
	}

	scannedPackages := make([]scan.Package, 0, len(req.GetPackages()))
	for _, p := range req.GetPackages() {
		pkgID, parseErr := utils.ParseUUID(p.GetId())
		if parseErr != nil {
			return nil, fmt.Errorf("parse package id: %w", parseErr)
		}
		ptID, parseErr := utils.ParseUUID(p.GetPointId())
		if parseErr != nil {
			return nil, fmt.Errorf("parse point id: %w", parseErr)
		}

		var timestamp time.Time
		if ts := p.GetTimestamp(); ts != nil {
			timestamp = ts.AsTime()
		}

		scannedPackages = append(scannedPackages, scan.Package{
			ID:           values.PackageID(pkgID),
			RoutePointID: values.RoutePointID(ptID),
			ShipmentID:   values.ShipmentID(shipmentID),
			ActionType:   values.PackageActionType(p.GetAction()),
			Timestamp:    timestamp,
			TareSticker:  p.GetTareSticker(),
		})
	}

	var failedPackages []*driver.SetPackagesStatusResponse_Data_FailedPackages
	for _, sp := range scannedPackages {
		if err := s.app.ScanPackages(ctx, values.ShipmentID(shipmentID), []scan.Package{sp}); err != nil {
			failedPackages = append(failedPackages, &driver.SetPackagesStatusResponse_Data_FailedPackages{
				Id:               sp.ID.String(),
				ErrorDescription: typedErrorMessage(err),
			})
		}
	}

	return &driver.SetPackagesStatusResponse{
		Success: true,
		Data: &driver.SetPackagesStatusResponse_Data{
			FailedPackages: failedPackages,
		},
		Message: "Packages scanned",
	}, nil
}

// mapErrorToGRPC converts use case errors into gRPC statuses: the resource
// not-found sentinels of the ports become NotFound, typed internal errors
// keep their own code and everything else falls back to Internal.
func mapErrorToGRPC(err error) error {
	switch {
	case stderrors.Is(err, ports.ErrRouteTemplateNotFound),
		stderrors.Is(err, ports.ErrShipmentNotFound),
		stderrors.Is(err, ports.ErrRouteNotFound),
		stderrors.Is(err, ports.ErrPackageNotFound),
		stderrors.Is(err, ports.ErrReserveNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return errors.MapToGRPC(err).Err()
	}
}

// CreateShipment implements [driver.DriverServer].
func (s *Server) CreateShipment(ctx context.Context, req *driver.CreateShipmentRequest) (*driver.CreateShipmentResponse, error) {
	shipmentID, err := s.app.CreateShipment(ctx, publicdriver.CreateShipmentRequest{
		RouteTemplateID: req.GetRouteTemplateId(),
	})
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return s.createShipmentResponse(ctx, shipmentID)
}

// CreateShipmentWithTransport implements [driver.DriverServer].
func (s *Server) CreateShipmentWithTransport(ctx context.Context, req *driver.CreateShipmentWithTransportRequest) (*driver.CreateShipmentWithTransportResponse, error) {
	shipmentID, err := s.app.CreateShipmentWithTransport(ctx, publicdriver.CreateShipmentWithTransportRequest{
		RouteTemplateID: req.GetRouteTemplateId(),
		LicensePlate:    req.GetCarLicensePlate(),
	})
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	resp, err := s.createShipmentResponse(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	return &driver.CreateShipmentWithTransportResponse{
		Success: resp.Success,
		Data: &driver.CreateShipmentWithTransportResponse_Data{
			Shipment: resp.GetData().GetShipment(),
		},
		Message: resp.Message,
	}, nil
}

// UpdateShipmentTransport implements [driver.DriverServer].
func (s *Server) UpdateShipmentTransport(ctx context.Context, req *driver.UpdateShipmentTransportRequest) (*driver.UpdateShipmentTransportResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	shipmentID, err := utils.ParseUUID(req.GetShipmentId())
	if err != nil {
		return nil, fmt.Errorf("parse shipment id: %w", err)
	}

	if err := s.app.AssignTransport(ctx, shipmentID, req.GetCarLicensePlate()); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &driver.UpdateShipmentTransportResponse{
		Success: true,
		Message: "Transport updated",
	}, nil
}

// createShipmentResponse loads the created shipment via the read model and
// builds the shared Success/Data{shipment}/Message response.
func (s *Server) createShipmentResponse(ctx context.Context, shipmentID uuid.UUID) (*driver.CreateShipmentResponse, error) {
	ship, err := s.app.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &driver.CreateShipmentResponse{
		Success: true,
		Data: &driver.CreateShipmentResponse_Data{
			Shipment: mapDetailedToShipment(ship),
		},
		Message: "Shipment created",
	}, nil
}

// GetActiveShipmentByDriverID implements [driver.DriverServer].
func (s *Server) GetActiveShipmentByDriverID(ctx context.Context, req *driver.GetActiveShipmentByDriverIDRequest) (*driver.GetActiveShipmentByDriverIDResponse, error) {
	shipmentID, err := s.app.GetActiveShipmentID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active shipment id: %w", err)
	}

	if shipmentID == uuid.Nil {
		return &driver.GetActiveShipmentByDriverIDResponse{
			Success: true,
			Data:    &driver.GetActiveShipmentByDriverIDResponse_Data{},
		}, nil
	}

	ship, err := s.app.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("get shipment by id: %w", err)
	}

	return &driver.GetActiveShipmentByDriverIDResponse{
		Success: true,
		Data: &driver.GetActiveShipmentByDriverIDResponse_Data{
			Shipment: mapDetailedToShipment(ship),
		},
	}, nil
}

// activeShipmentOrError resolves the driver's active shipment id and errors
// with FailedPrecondition when the driver has none.
func (s *Server) activeShipmentOrError(ctx context.Context) (uuid.UUID, error) {
	shipmentID, err := s.app.GetActiveShipmentID(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	if shipmentID == uuid.Nil {
		return uuid.Nil, status.Error(codes.FailedPrecondition, "no active shipment")
	}

	return shipmentID, nil
}

// ConsiderOptions below: the reserve RPCs are declared with
// google.protobuf.Empty in the skeleton contract, so the handlers resolve the
// active shipment and its source route point from context instead of request
// fields.
//
// TODO(LASTMILE-087): once the proto carries shipment_id/route_point_id for
// the reserve RPCs, pass them straight through instead of resolving the active
// shipment. The public_driver reserve RPCs below are the slice port of the
// gate reserve flow (ARCHITECTURE.md §23 flows 4 and 7).

// CreateReserve implements [driver.DriverServer].
func (s *Server) CreateReserve(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	sourceRoutePointID, err := s.app.GetActiveShipmentSourceRoutePointID(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if _, err := s.app.CreateReserve(ctx, publicdriver.CreateReserveRequest{
		ShipmentID:   shipmentID,
		RoutePointID: sourceRoutePointID,
	}); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// GetReserveQRStatus implements [driver.DriverServer].
func (s *Server) GetReserveQRStatus(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := s.app.GetReserveQRStatus(ctx, shipmentID); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// UpdateReserveQRStatus implements [driver.DriverServer].
func (s *Server) UpdateReserveQRStatus(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	sourceRoutePointID, err := s.app.GetActiveShipmentSourceRoutePointID(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if err := s.app.UpdateReserveQRStatus(ctx, publicdriver.UpdateReserveQRStatusRequest{
		ShipmentID:   shipmentID,
		RoutePointID: sourceRoutePointID,
	}); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// CancelReserve implements [driver.DriverServer].
func (s *Server) CancelReserve(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.app.CancelReserve(ctx, shipmentID); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// EndShipment implements [driver.DriverServer]: the driver has completed the
// route, so the active shipment is moved to COMPLETED (ShipmentCompleted event
// → cargo-shipment FINISH + payments accrual outbox, all in one transaction).
func (s *Server) EndShipment(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.app.EndShipment(ctx, shipmentID); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// UpdateRoutePointDriverDeparted implements [driver.DriverServer]: the driver
// departed the current (DRIVER_ARRIVED) route point; if it is the last one the
// shipment is completed as well.
func (s *Server) UpdateRoutePointDriverDeparted(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	shipmentID, err := s.activeShipmentOrError(ctx)
	if err != nil {
		return nil, err
	}

	routePointID, err := s.app.GetActiveShipmentDriverArrivedRoutePointID(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	if routePointID == uuid.Nil {
		return nil, status.Error(codes.FailedPrecondition, "no driver arrived route point")
	}

	if err := s.app.UpdateRoutePointDriverDeparted(ctx, publicdriver.UpdateRoutePointDriverDepartedRequest{
		ShipmentID:   shipmentID,
		RoutePointID: routePointID,
	}); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}
