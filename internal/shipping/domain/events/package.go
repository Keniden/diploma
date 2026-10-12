package events

import (
	"log/slog"
	"time"

	"last-mile-task/internal/shipping/domain/values"
)

type PackageEventType string

type PackageEvent interface {
	event[PackageEventType]
}

const (
	PackageScanned   PackageEventType = "package.scanned"
	PackageInTransit PackageEventType = "package.in_transit"
)

type PackageScannedEvent struct {
	Packages []ScannedPackage
}

func (e PackageScannedEvent) Type() PackageEventType {
	return PackageScanned
}

func (e PackageScannedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("package_count", len(e.Packages)),
	)
}

type PackageInTransitEvent struct {
	RoutePointID   values.RoutePointID
	PackageIDs     []values.PackageID
	TransitionedAt time.Time
}

func (e PackageInTransitEvent) Type() PackageEventType {
	return PackageInTransit
}

func (e PackageInTransitEvent) LogValue() slog.Value {
	pids := make([]string, len(e.PackageIDs))
	for i, id := range e.PackageIDs {
		pids[i] = id.String()
	}
	return slog.GroupValue(
		slog.String("route_point_id", e.RoutePointID.String()),
		slog.Any("package_ids", pids),
	)
}

type ScannedPackage struct {
	PackageID    values.PackageID
	TareID       values.TareID
	TareType     values.TareType
	RoutePointID values.RoutePointID
	PointID      values.PointID
	ShipmentID   values.ShipmentID
	ActionType   values.PackageActionType
	Timestamp    time.Time
	TareSticker  string
}
