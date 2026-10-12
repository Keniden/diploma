package ports

import (
	"context"

	"last-mile-task/internal/shipping/domain/events"
)

type ShipmentEventPublisher interface {
	PublishShipmentEvent(ctx context.Context, event events.ShipmentEvent) error
}

type ReserveEventPublisher interface {
	PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error
}

type PackageEventPublisher interface {
	PublishPackageEvent(ctx context.Context, event events.PackageEvent) error
}

type RouteEventPublisher interface {
	PublishRouteEvent(ctx context.Context, event events.RouteEvent) error
}
