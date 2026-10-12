package drivergrpc

import (
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"last-mile-task/internal/shipping/domain/values"
	shipmentq "last-mile-task/internal/shipping/queries/driver/shipment"
	routepoint "last-mile-task/pkg/gen/api/proto/v1/public/driver/route_point"
	shipment "last-mile-task/pkg/gen/api/proto/v1/public/driver/shipment"
)

// mapDetailedToShipment converts a detailed shipment read model into its
// gRPC representation.
//
// TODO(LASTMILE-086): the real mapping also attaches the office week schedule
// per route point (SchedulesByPointID → Shipment_Route_WorkSchedule) and maps
// extra fields not present in the skeleton contract — skipped here.
func mapDetailedToShipment(ship *shipmentq.DetailedShipment) *shipment.Shipment {
	loadDonePackageIDs := ship.LoadDonePackageIDs()

	out := &shipment.Shipment{
		Id:      ship.ID.String(),
		Status:  shipment.Status(ship.Status),
		ShortId: ship.ShortID,
		Route: &shipment.Shipment_Route{
			Id: ship.Route.ID.String(),
			Points: mapDetailedRoutePoints(
				ship.RoutePoints,
				loadDonePackageIDs,
			),
		},
	}

	if ship.Reserve != nil {
		reserve := &shipment.Shipment_Reserve{
			CreatedAt: timestamppb.New(ship.Reserve.UpdatedAt),
		}
		if ship.Reserve.ExpiresAt != nil {
			reserve.ExpiresAt = timestamppb.New(*ship.Reserve.ExpiresAt)
		}
		out.Reserve = reserve
	}

	return out
}

func mapDetailedRoutePoints(
	routePoints []shipmentq.DetailedRoutePoint,
	loadDonePackageIDs map[uuid.UUID]struct{},
) []*shipment.Shipment_Route_Point {
	points := make([]*shipment.Shipment_Route_Point, 0, len(routePoints))
	for _, rp := range routePoints {
		points = append(points, &shipment.Shipment_Route_Point{
			Id:           rp.ID.String(),
			Address:      rp.Address,
			SeqNum:       rp.SeqNum,
			PackageCount: rp.UnloadPackageCount(loadDonePackageIDs),
			ShortId:      strconv.FormatInt(rp.PointID, 10),
			Status:       mapRoutePointStatus(values.RoutePointStatus(rp.Status)),
		})
	}
	return points
}

// mapRoutePointStatus converts the domain route point status into the proto
// enum.
//
// NOTE: the skeleton proto enum reserves value 0 for UNSPECIFIED while the
// domain value starts counting from WAITING_ARRIVAL (0). The real contract has
// no UNSPECIFIED member, so there the statuses are cast 1:1; here an explicit
// mapping is required and represented via switch.
func mapRoutePointStatus(status values.RoutePointStatus) routepoint.RoutePointStatus {
	switch status {
	case values.RoutePointStatusWaitingArrival:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_WAITING_ARRIVAL
	case values.RoutePointStatusDriverArrived:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_DRIVER_ARRIVED
	case values.RoutePointStatusPackagesNotUnloaded:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_PACKAGES_NOT_UNLOADED
	case values.RoutePointStatusPackagesNotLoaded:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_PACKAGES_NOT_LOADED
	case values.RoutePointStatusCompleted:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_COMPLETED
	default:
		return routepoint.RoutePointStatus_ROUTE_POINT_STATUS_WAITING_ARRIVAL
	}
}
