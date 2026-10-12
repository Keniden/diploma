package drivergrpc

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	shipmentq "last-mile-task/internal/shipping/queries/driver/shipment"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
	routepoint "last-mile-task/pkg/gen/api/proto/v1/public/driver/route_point"
	shipment "last-mile-task/pkg/gen/api/proto/v1/public/driver/shipment"
)

func TestMapDetailedToShipment_RoutePointShortID(t *testing.T) {
	routePointID := uuid.New()
	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{
			ID:      uuid.New(),
			ShortID: 42,
		},
		DetailedRoute: shipmentq.DetailedRoute{
			Route: readmodels.Route{ID: uuid.New()},
			RoutePoints: []shipmentq.DetailedRoutePoint{
				{
					RoutePoint: readmodels.RoutePoint{
						ID:      routePointID,
						PointID: 12345,
						Address: "test address",
						SeqNum:  1,
					},
				},
			},
		},
	}

	out := mapDetailedToShipment(ship)
	if len(out.Route.Points) != 1 {
		t.Fatalf("expected 1 route point, got %d", len(out.Route.Points))
	}
	if out.Route.Points[0].ShortId != "12345" {
		t.Fatalf("expected short_id 12345, got %q", out.Route.Points[0].ShortId)
	}
}

func TestMapDetailedToShipment_RoutePointStatus(t *testing.T) {
	// NOTE: the skeleton proto enum has UNSPECIFIED=0 while the domain values
	// start from WAITING_ARRIVAL=0 (the real contract has no UNSPECIFIED
	// member) — the mapping must shift the values.
	cases := []struct {
		domain values.RoutePointStatus
		want   routepoint.RoutePointStatus
	}{
		{values.RoutePointStatusWaitingArrival, routepoint.RoutePointStatus_ROUTE_POINT_STATUS_WAITING_ARRIVAL},
		{values.RoutePointStatusDriverArrived, routepoint.RoutePointStatus_ROUTE_POINT_STATUS_DRIVER_ARRIVED},
		{values.RoutePointStatusPackagesNotUnloaded, routepoint.RoutePointStatus_ROUTE_POINT_STATUS_PACKAGES_NOT_UNLOADED},
		{values.RoutePointStatusPackagesNotLoaded, routepoint.RoutePointStatus_ROUTE_POINT_STATUS_PACKAGES_NOT_LOADED},
		{values.RoutePointStatusCompleted, routepoint.RoutePointStatus_ROUTE_POINT_STATUS_COMPLETED},
	}

	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{ID: uuid.New()},
		DetailedRoute: shipmentq.DetailedRoute{
			Route: readmodels.Route{ID: uuid.New()},
		},
	}

	for _, tc := range cases {
		ship.RoutePoints = []shipmentq.DetailedRoutePoint{
			{
				RoutePoint: readmodels.RoutePoint{
					ID:      uuid.New(),
					PointID: 1,
					Status:  int(tc.domain),
				},
			},
		}

		out := mapDetailedToShipment(ship)
		if got := out.Route.Points[0].Status; got != tc.want {
			t.Fatalf("expected status %s, got %s", tc.want, got)
		}
	}
}

func TestMapDetailedToShipment_UnknownRoutePointStatus(t *testing.T) {
	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{ID: uuid.New()},
		DetailedRoute: shipmentq.DetailedRoute{
			Route: readmodels.Route{ID: uuid.New()},
			RoutePoints: []shipmentq.DetailedRoutePoint{
				{
					RoutePoint: readmodels.RoutePoint{
						ID:      uuid.New(),
						PointID: 1,
						Status:  int(values.RoutePointStatus(99)),
					},
				},
			},
		},
	}

	out := mapDetailedToShipment(ship)
	if got := out.Route.Points[0].Status; got != routepoint.RoutePointStatus_ROUTE_POINT_STATUS_WAITING_ARRIVAL {
		t.Fatalf("expected fallback WAITING_ARRIVAL, got %s", got)
	}
}

func TestMapDetailedToShipment_ShipmentStatus(t *testing.T) {
	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{
			ID:     uuid.New(),
			Status: int(values.ShipmentStatusCompleted),
		},
	}

	out := mapDetailedToShipment(ship)
	if got := out.Status; got != shipment.Status_STATUS_COMPLETED {
		t.Fatalf("expected STATUS_COMPLETED, got %s", got)
	}
}

func TestMapDetailedToShipment_Reserve(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC()
	updatedAt := time.Now().Add(-time.Hour).UTC()

	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{ID: uuid.New(), UpdatedAt: updatedAt},
		Reserve: &readmodels.ShipmentReserve{
			ExpiresAt: &expiresAt,
			UpdatedAt: updatedAt,
		},
	}

	out := mapDetailedToShipment(ship)
	if out.Reserve == nil {
		t.Fatal("expected reserve to be mapped")
	}
	if out.Reserve.CreatedAt.AsTime().Unix() != updatedAt.Unix() {
		t.Fatalf("expected created_at %s, got %s", updatedAt, out.Reserve.CreatedAt.AsTime())
	}
	if out.Reserve.ExpiresAt.AsTime().Unix() != expiresAt.Unix() {
		t.Fatalf("expected expires_at %s, got %s", expiresAt, out.Reserve.ExpiresAt.AsTime())
	}
}

func TestMapDetailedToShipment_PackageCount(t *testing.T) {
	pkgID := uuid.New()
	sourcePointID := uuid.New()
	targetPointID := uuid.New()

	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{ID: uuid.New()},
		DetailedRoute: shipmentq.DetailedRoute{
			Route: readmodels.Route{ID: uuid.New()},
			RoutePoints: []shipmentq.DetailedRoutePoint{
				{
					RoutePoint: readmodels.RoutePoint{ID: sourcePointID, PointID: 1, SeqNum: 1},
					Packages: []shipmentq.DetailedPackage{
						{
							Package: readmodels.Package{ID: pkgID},
							PackageAction: readmodels.PackageAction{
								PackageID:    pkgID,
								RoutePointID: sourcePointID,
								Action:       int(values.RoutePointActionTypeLoad),
								Status:       int(values.RoutePointPackageStatusDone),
							},
						},
					},
				},
				{
					RoutePoint: readmodels.RoutePoint{ID: targetPointID, PointID: 2, SeqNum: 2},
					Packages: []shipmentq.DetailedPackage{
						{
							Package: readmodels.Package{ID: pkgID},
							PackageAction: readmodels.PackageAction{
								PackageID:    pkgID,
								RoutePointID: targetPointID,
								Action:       int(values.RoutePointActionTypeUnload),
								Status:       int(values.RoutePointPackageStatusTODO),
							},
						},
					},
				},
			},
		},
	}

	out := mapDetailedToShipment(ship)
	if got := out.Route.Points[1].PackageCount; got != 1 {
		t.Fatalf("expected package count 1, got %d", got)
	}
	if got := out.Route.Points[0].PackageCount; got != 0 {
		t.Fatalf("expected package count 0 on source point, got %d", got)
	}
}

func TestMapDetailedToShipment_NilReserve(t *testing.T) {
	ship := &shipmentq.DetailedShipment{
		Shipment: readmodels.Shipment{ID: uuid.New()},
	}

	out := mapDetailedToShipment(ship)
	if out.Reserve != nil {
		t.Fatalf("expected nil reserve, got %v", out.Reserve)
	}
}
