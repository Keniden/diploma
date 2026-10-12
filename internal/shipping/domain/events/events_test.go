package events_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/shipping/domain/events"
)

func TestEvent_Types(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{"package.scanned", string(events.PackageScannedEvent{}.Type()), string(events.PackageScanned)},
		{"package.in_transit", string(events.PackageInTransitEvent{}.Type()), string(events.PackageInTransit)},
		{"shipment.created", string(events.ShipmentCreatedEvent{}.Type()), string(events.ShipmentCreated)},
		{"shipment.driver.assign", string(events.ShipmentDriverAssignEvent{}.Type()), string(events.ShipmentDriverAssign)},
		{"shipment.transport.assign", string(events.ShipmentTransportAssignEvent{}.Type()), string(events.ShipmentTransportAssign)},
		{"shipment.loading", string(events.ShipmentLoadingEvent{}.Type()), string(events.ShipmentLoading)},
		{"shipment.started", string(events.ShipmentStartedEvent{}.Type()), string(events.ShipmentStarted)},
		{"shipment.completed", string(events.ShipmentCompletedEvent{}.Type()), string(events.ShipmentCompleted)},
		{"shipment.canceled", string(events.ShipmentCanceledEvent{}.Type()), string(events.ShipmentCanceled)},
		{"route.optimized", string(events.RouteOptimizedEvent{}.Type()), string(events.RouteOptimized)},
		{"route.point.completed", string(events.PointCompletedEvent{}.Type()), string(events.PointCompleted)},
		{"reserve.queued", string(events.ReserveQueuedEvent{}.Type()), string(events.ReserveQueued)},
		{"reserve.approved", string(events.ReserveApprovedEvent{}.Type()), string(events.ReserveApproved)},
		{"reserve.released", string(events.ReserveReleasedEvent{}.Type()), string(events.ReserveReleased)},
	}

	for _, tt := range tests {
		if tt.typ != tt.want {
			t.Fatalf("%s: Type() = %q, want %q", tt.name, tt.typ, tt.want)
		}
	}
}

func TestEvent_LogValues(t *testing.T) {
	tests := []struct {
		name     string
		value    slog.Value
		attrKeys []string
	}{
		{"package.scanned", events.PackageScannedEvent{}.LogValue(), []string{"package_count"}},
		{"package.in_transit", events.PackageInTransitEvent{}.LogValue(), []string{"route_point_id", "package_ids"}},
		{"shipment.created", events.ShipmentCreatedEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.driver.assign", events.ShipmentDriverAssignEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.transport.assign", events.ShipmentTransportAssignEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.loading", events.ShipmentLoadingEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.started", events.ShipmentStartedEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.completed", events.ShipmentCompletedEvent{}.LogValue(), []string{"shipment_id"}},
		{"shipment.canceled", events.ShipmentCanceledEvent{}.LogValue(), []string{"shipment_id"}},
		{"route.optimized", events.RouteOptimizedEvent{}.LogValue(), []string{"shipment_id", "route_id", "point_count"}},
		{"route.point.completed", events.PointCompletedEvent{}.LogValue(), []string{"shipment_id", "route_point_id"}},
		{"reserve.queued", events.ReserveQueuedEvent{}.LogValue(), []string{"reserve_id", "shipment_id"}},
		{"reserve.approved", events.ReserveApprovedEvent{}.LogValue(), []string{"reserve_id", "shipment_id", "is_first_approval", "is_fake"}},
		{"reserve.released", events.ReserveReleasedEvent{}.LogValue(), []string{"reserve_id"}},
	}

	for _, tt := range tests {
		require.Equal(t, slog.KindGroup, tt.value.Kind(), "%s: LogValue() must be a group", tt.name)
		require.Equal(t, tt.attrKeys, groupAttrKeys(tt.value), "%s: attr keys", tt.name)
	}
}

func groupAttrKeys(value slog.Value) []string {
	group := value.Group()
	keys := make([]string, 0, len(group))
	for _, attr := range group {
		keys = append(keys, attr.Key)
	}
	return keys
}