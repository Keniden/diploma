package events

import (
	"log/slog"

	"last-mile-task/internal/shipping/domain/values"
)

type RouteEventType string

type RouteEvent interface {
	event[RouteEventType]
}

const (
	RouteOptimized RouteEventType = "route.optimized"
	PointCompleted RouteEventType = "route.point.completed"
)

type RouteOptimizedEvent struct {
	ShipmentID  values.ShipmentID
	RouteID     values.RouteID
	RoutePoints []OptimizedRoutePoint
}

type OptimizedRoutePoint struct {
	ID      values.RoutePointID
	PointID values.PointID
	SeqNum  int32
}

func (e RouteOptimizedEvent) Type() RouteEventType {
	return RouteOptimized
}

func (e RouteOptimizedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("shipment_id", e.ShipmentID.String()),
		slog.String("route_id", e.RouteID.String()),
		slog.Int("point_count", len(e.RoutePoints)),
	)
}

type PointCompletedEvent struct {
	ShipmentID   values.ShipmentID
	RoutePointID values.RoutePointID
}

func (e PointCompletedEvent) Type() RouteEventType {
	return PointCompleted
}

func (e PointCompletedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("shipment_id", e.ShipmentID.String()),
		slog.String("route_point_id", e.RoutePointID.String()),
	)
}
