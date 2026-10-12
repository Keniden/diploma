package publicdriver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func reserveWithStatus(status values.ReserveStatus) *entities.Reserve {
	return &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   values.ShipmentID(uuid.New()),
		RoutePointID: values.RoutePointID(uuid.New()),
		Status:       status,
	}
}

func TestGetReserveQRStatus_Scanned(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	scannedAt := values.Now().Add(-time.Minute)
	expiresAt := values.Now().Add(5 * time.Minute)
	reserve := reserveWithStatus(values.ReserveStatusActive)
	reserve.ShipmentID = shipment.ID
	reserve.RoutePointID = route.Points[0].ID
	reserve.ScannedAt = &scannedAt
	reserve.ExpiresAt = &expiresAt

	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return reserve, nil
	}

	got, err := h.app.GetReserveQRStatus(ctx, uuid.UUID(shipment.ID))
	if err != nil {
		t.Fatalf("GetReserveQRStatus: %v", err)
	}
	if !got.IsScanned {
		t.Fatal("expected is_scanned=true when scanned_at is set")
	}
	if got.ReserveID != uuid.UUID(reserve.ID) {
		t.Fatalf("expected reserve id %s, got %s", reserve.ID, got.ReserveID)
	}
	if got.Status != reserve.Status {
		t.Fatalf("expected status %v, got %v", reserve.Status, got.Status)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(*reserve.ExpiresAt) {
		t.Fatalf("expected expires_at %v, got %v", reserve.ExpiresAt, got.ExpiresAt)
	}
}

func TestGetReserveQRStatus_NotScanned(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	reserve := reserveWithStatus(values.ReserveStatusWaitingApproval)
	reserve.ShipmentID = shipment.ID
	reserve.RoutePointID = route.Points[0].ID

	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return reserve, nil
	}

	got, err := h.app.GetReserveQRStatus(ctx, uuid.UUID(shipment.ID))
	if err != nil {
		t.Fatalf("GetReserveQRStatus: %v", err)
	}
	if got.IsScanned {
		t.Fatal("expected is_scanned=false when scanned_at is nil")
	}
}

func TestGetReserveQRStatus_EmptyReserve_NoError(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return nil, ports.ErrReserveNotFound
	}

	got, err := h.app.GetReserveQRStatus(ctx, uuid.UUID(shipment.ID))
	if err != nil {
		t.Fatalf("expected no error for a shipment without a reserve, got %v", err)
	}
	if got != (ReserveQRStatus{}) {
		t.Fatalf("expected the empty state, got %+v", got)
	}
}

func TestUpdateReserveQRStatus_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	reserve := &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: route.Points[0].ID,
		Status:       values.ReserveStatusWaitingApproval,
		Task:         values.ReserveTask{ID: "task-7"},
	}
	h.reserves.getReserveByRoutePointIDFn = func(_ context.Context, _ values.RoutePointID) (*entities.Reserve, error) {
		return reserve, nil
	}

	err := h.app.UpdateReserveQRStatus(ctx, UpdateReserveQRStatusRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(route.Points[0].ID),
	})
	if err != nil {
		t.Fatalf("UpdateReserveQRStatus: %v", err)
	}
	if reserve.Status != values.ReserveStatusActive {
		t.Fatalf("expected reserve ACTIVE, got %v", reserve.Status)
	}
	if reserve.ScannedAt == nil {
		t.Fatal("expected scanned_at to be set on approval")
	}
	if shipment.Status != values.ShipmentStatusLoading {
		t.Fatalf("expected shipment LOADING, got %v", shipment.Status)
	}
}

func TestUpdateReserveQRStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	h.reserves.getReserveByRoutePointIDFn = func(_ context.Context, _ values.RoutePointID) (*entities.Reserve, error) {
		return nil, ports.ErrReserveNotFound
	}

	err := h.app.UpdateReserveQRStatus(ctx, UpdateReserveQRStatusRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(route.Points[0].ID),
	})
	if err == nil {
		t.Fatal("expected an error when the reserve is not found")
	}
	if !errors.Is(err, ports.ErrReserveNotFound) {
		t.Fatalf("expected the reserve not found to be wrapped, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("shipment must not transition on failure, got %v", shipment.Status)
	}
}