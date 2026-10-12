// External integration stubs of the shipping context: the real adapters are
// backed by gRPC/HTTP clients (route, office, cargo-core, cargo-shipment),
// which are not ported in the skeleton. Each stub keeps the exact adapter
// shape of the real repository and returns empty/no-result semantics so the
// slice stays green until the client-backed adapters land.
package infrastructure

import (
	"context"
	"log/slog"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
)

// RouteTemplatesProvider is the slice-scoped stand-in for the route template
// provider (route client GetRoutecar + office points mapping).
//
// TODO(LASTMILE-110): replace with the client-backed provider once the
// internal/clients/route adapter is ported. Every template lookup fails with
// ErrRouteTemplateNotFound until then.
type RouteTemplatesProvider struct{}

// NewRouteTemplatesProvider builds the stub provider.
func NewRouteTemplatesProvider() ports.RouteTemplateProvider { return &RouteTemplatesProvider{} }

// GetRouteTemplateByID always fails: the route client is not ported.
func (p *RouteTemplatesProvider) GetRouteTemplateByID(
	_ context.Context, _ values.RouteTemplateID,
) (values.RouteTemplate, error) {
	return values.RouteTemplate{}, ports.ErrRouteTemplateNotFound
}

// DemandProvider is the slice-scoped stand-in for the package demand provider
// (cargo-core ListPackagesToLoad + tare sticker splitting).
//
// TODO(LASTMILE-111): replace with the client-backed provider once the
// cargo-core client is ported. Empty demand keeps the demand planning no-op.
type DemandProvider struct{}

// NewDemandProvider builds the stub provider (also used as the policy
// repository's SourcePackagesProvider).
func NewDemandProvider() *DemandProvider { return &DemandProvider{} }

// GetPackageDemandByRoute returns no demand (cargo-core not ported).
func (p *DemandProvider) GetPackageDemandByRoute(
	_ context.Context, _ *entities.Route,
) ([]values.PackageDemand, error) {
	return nil, nil
}

// HasSourcePackages reports no source packages (cargo-core not ported).
func (p *DemandProvider) HasSourcePackages(_ context.Context, _ *entities.Route) (bool, error) {
	return false, nil
}

// CargoShipment is the slice-scoped stand-in for the cargo-shipment service
// receiving shipment lifecycle events (CreateShipment, SetTransport,
// DriveShipment).
//
// TODO(LASTMILE-112): replace with the client-backed handler once the
// internal/clients/cargo-shipment adapter is ported. Calls are logged and
// discarded.
type CargoShipment struct{}

// NewCargoShipment builds the stub cargo-shipment listener.
func NewCargoShipment() handlers.CargoShipment { return &CargoShipment{} }

// CreateShipment logs the outbound call (client not ported).
func (c *CargoShipment) CreateShipment(_ context.Context, req handlers.CargoShipmentCreateRequest) error {
	slog.Debug("stub cargo-shipment: CreateShipment",
		"shipment_id", req.ShipmentID.String(),
		"route_id", req.RouteID.String(),
		"source_route_point_id", req.SourceRoutePointID.String(),
		"source_point_id", req.SourcePointID,
	)

	return nil
}

// SetTransport logs the outbound call (client not ported).
func (c *CargoShipment) SetTransport(_ context.Context, req handlers.CargoShipmentSetTransportRequest) error {
	slog.Debug("stub cargo-shipment: SetTransport",
		"shipment_id", req.ShipmentID.String(),
		"driver_id", req.DriverID,
		"carrier_id", req.CarrierID,
		"vehicle_id", req.VehicleID,
		"vehicle_number", req.VehicleNumber,
	)

	return nil
}

// DriveShipment logs the outbound call (client not ported).
func (c *CargoShipment) DriveShipment(_ context.Context, req handlers.CargoShipmentDriveShipmentRequest) error {
	slog.Debug("stub cargo-shipment: DriveShipment",
		"shipment_id", req.ShipmentID.String(),
		"trigger", req.Trigger,
	)

	return nil
}
