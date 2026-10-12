package publicdriver

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func TestCancelReserve_RevertsShipment(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	// Move the shipment into RESERVE_LOADING so the cancel can revert it.
	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	reserve := &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: route.Points[0].ID,
		Status:       values.ReserveStatusQueued,
	}
	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return reserve, nil
	}

	err := h.app.CancelReserve(ctx, uuid.UUID(shipment.ID))
	if err != nil {
		t.Fatalf("CancelReserve: %v", err)
	}
	if reserve.Status != values.ReserveStatusCanceled {
		t.Fatalf("expected reserve CANCELED, got %v", reserve.Status)
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("expected shipment back to ASSIGNED, got %v", shipment.Status)
	}
}

func TestCancelReserve_Active_Error(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	// Already-active reserve cannot be canceled.
	reserve := &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: route.Points[0].ID,
		Status:       values.ReserveStatusActive,
	}
	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return reserve, nil
	}

	err := h.app.CancelReserve(ctx, uuid.UUID(shipment.ID))
	if err == nil {
		t.Fatal("expected an error when canceling an active reserve")
	}
	if reserve.Status != values.ReserveStatusActive {
		t.Fatalf("reserve must stay ACTIVE on failure, got %v", reserve.Status)
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("shipment must not change on failure, got %v", shipment.Status)
	}
}