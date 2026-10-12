package handlers_test

import (
	"context"
	"errors"
	"testing"

	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	"last-mile-task/pkg/outbox"
)

// recordingCargoShipment records the cargo-shipment calls made by the handler.
type recordingCargoShipment struct {
	createCalls []handlers.CargoShipmentCreateRequest
	setCalls    []handlers.CargoShipmentSetTransportRequest
	driveCalls  []handlers.CargoShipmentDriveShipmentRequest

	createErr error
	setErr    error
	driveErr  error
}

func (c *recordingCargoShipment) CreateShipment(
	_ context.Context,
	req handlers.CargoShipmentCreateRequest,
) error {
	c.createCalls = append(c.createCalls, req)
	return c.createErr
}

func (c *recordingCargoShipment) SetTransport(
	_ context.Context,
	req handlers.CargoShipmentSetTransportRequest,
) error {
	c.setCalls = append(c.setCalls, req)
	return c.setErr
}

func (c *recordingCargoShipment) DriveShipment(
	_ context.Context,
	req handlers.CargoShipmentDriveShipmentRequest,
) error {
	c.driveCalls = append(c.driveCalls, req)
	return c.driveErr
}

func TestHandleShipmentCreatedEvent_CallsCreateShipment(t *testing.T) {
	cargo := &recordingCargoShipment{}
	h := handlers.NewShipmentEventHandler(cargo)

	shipmentID := values.NewID()
	routeID := values.NewID()
	sourceRoutePointID := values.NewID()

	err := h.HandleShipmentCreatedEvent(context.Background(), events.ShipmentCreatedEvent{
		ShipmentID:         shipmentID,
		RouteID:            routeID,
		SourceRoutePointID: sourceRoutePointID,
		SourcePointID:      values.PointID(101),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cargo.createCalls) != 1 {
		t.Fatalf("expected 1 CreateShipment call, got %d", len(cargo.createCalls))
	}
	got := cargo.createCalls[0]
	if got.ShipmentID != shipmentID || got.RouteID != routeID {
		t.Fatal("shipment/route id mismatch")
	}
	if got.SourceRoutePointID != sourceRoutePointID {
		t.Fatal("source route point id mismatch")
	}
	if got.SourcePointID != 101 {
		t.Fatalf("expected source point id 101, got %d", got.SourcePointID)
	}

	if len(cargo.setCalls) != 0 {
		t.Fatalf("expected no SetTransport calls for a created shipment, got %d", len(cargo.setCalls))
	}
}

func TestHandleShipmentCreatedEvent_PropagatesError(t *testing.T) {
	cargoErr := errors.New("cargo down")
	h := handlers.NewShipmentEventHandler(&recordingCargoShipment{createErr: cargoErr})

	err := h.HandleShipmentCreatedEvent(context.Background(), events.ShipmentCreatedEvent{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cargoErr) {
		t.Fatalf("expected wrapped cargo error, got %v", err)
	}
}

func TestHandleShipmentDriverAssignEvent_CallsSetTransport(t *testing.T) {
	cargo := &recordingCargoShipment{}
	h := handlers.NewShipmentEventHandler(cargo)

	shipmentID := values.NewID()
	driverID := values.NewID()
	err := h.HandleShipmentDriverAssignEvent(context.Background(), events.ShipmentDriverAssignEvent{
		ID: shipmentID,
		Driver: values.Driver{
			ID:         driverID,
			SupplierID: 17,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cargo.setCalls) != 1 {
		t.Fatalf("expected 1 SetTransport call, got %d", len(cargo.setCalls))
	}
	got := cargo.setCalls[0]
	if got.ShipmentID != shipmentID {
		t.Fatal("shipment id mismatch")
	}
	if got.DriverID != driverID.String() {
		t.Fatalf("expected driver id %s, got %q", driverID, got.DriverID)
	}
	if got.CarrierID != "17" {
		t.Fatalf("expected carrier id 17, got %q", got.CarrierID)
	}
	if got.VehicleID != 0 || got.VehicleNumber != "" {
		t.Fatalf("driver assign must not carry vehicle fields, got vehicle_id=%d number=%q", got.VehicleID, got.VehicleNumber)
	}

	if len(cargo.createCalls) != 0 {
		t.Fatalf("expected no CreateShipment calls for a driver assign, got %d", len(cargo.createCalls))
	}
}

func TestHandleShipmentTransportAssignEvent_CallsSetTransport(t *testing.T) {
	cargo := &recordingCargoShipment{}
	h := handlers.NewShipmentEventHandler(cargo)

	shipmentID := values.NewID()
	err := h.HandleShipmentTransportAssignEvent(context.Background(), events.ShipmentTransportAssignEvent{
		ID: shipmentID,
		Vehicle: values.Vehicle{
			ID:           42,
			LicensePlate: values.NewLicensePlate("E777AA77"),
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cargo.setCalls) != 1 {
		t.Fatalf("expected 1 SetTransport call, got %d", len(cargo.setCalls))
	}
	got := cargo.setCalls[0]
	if got.ShipmentID != shipmentID {
		t.Fatal("shipment id mismatch")
	}
	if got.VehicleID != 42 {
		t.Fatalf("expected vehicle id 42, got %d", got.VehicleID)
	}
	if got.VehicleNumber != "E777AA77" {
		t.Fatalf("expected vehicle number E777AA77, got %q", got.VehicleNumber)
	}
	// Vehicle assignment must not carry driver/carrier fields.
	if got.DriverID != "" || got.CarrierID != "" {
		t.Fatalf("transport assign must not carry driver fields, got driver=%q carrier=%q", got.DriverID, got.CarrierID)
	}
}

func TestHandleShipmentTransportAssignEvent_PropagatesError(t *testing.T) {
	cargoErr := errors.New("cargo down")
	h := handlers.NewShipmentEventHandler(&recordingCargoShipment{setErr: cargoErr})

	err := h.HandleShipmentTransportAssignEvent(context.Background(), events.ShipmentTransportAssignEvent{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cargoErr) {
		t.Fatalf("expected wrapped cargo error, got %v", err)
	}
}

func TestHandleShipmentCompletedEvent_CallsDriveShipment(t *testing.T) {
	cargo := &recordingCargoShipment{}
	h := handlers.NewShipmentEventHandler(cargo)

	shipmentID := values.NewID()
	err := h.HandleShipmentCompletedEvent(context.Background(), events.ShipmentCompletedEvent{ID: shipmentID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cargo.driveCalls) != 1 {
		t.Fatalf("expected 1 DriveShipment call, got %d", len(cargo.driveCalls))
	}
	got := cargo.driveCalls[0]
	if got.ShipmentID != shipmentID {
		t.Fatal("shipment id mismatch")
	}
	if got.Trigger != handlers.CargoShipmentDriveShipmentTriggerFinish {
		t.Fatalf("expected FINISH trigger, got %q", got.Trigger)
	}
	if len(cargo.createCalls) != 0 || len(cargo.setCalls) != 0 {
		t.Fatal("completed event must not create/set transport")
	}
}

func TestHandleShipmentCompletedEvent_PropagatesError(t *testing.T) {
	cargoErr := errors.New("cargo down")
	h := handlers.NewShipmentEventHandler(&recordingCargoShipment{driveErr: cargoErr})

	err := h.HandleShipmentCompletedEvent(context.Background(), events.ShipmentCompletedEvent{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cargoErr) {
		t.Fatalf("expected wrapped cargo error, got %v", err)
	}
}

func TestHandleShipmentCompletedEvent_WritesPaymentsAccrual(t *testing.T) {
	cargo := &recordingCargoShipment{}
	querier := &handlerFakeQuerier{}
	h := handlers.NewShipmentEventHandler(cargo, payments_outbox.NewWriter(querier))

	shipmentID := values.NewID()
	err := h.HandleShipmentCompletedEvent(context.Background(), events.ShipmentCompletedEvent{ID: shipmentID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !querier.inserted {
		t.Fatal("expected a payments accrual outbox insert")
	}
	if got := querier.lastType; got != payments_outbox.MessageTypeLastMileShipmentAccrual {
		t.Fatalf("expected type %q, got %q", payments_outbox.MessageTypeLastMileShipmentAccrual, got)
	}
	if len(cargo.driveCalls) != 1 {
		t.Fatalf("expected DriveShipment to run alongside the accrual write, got %d", len(cargo.driveCalls))
	}
}

// handlerFakeQuerier records the outbox insert performed by the payments
// writer used inside the shipment event handler tests.
type handlerFakeQuerier struct {
	inserted bool
	lastType string
}

func (f *handlerFakeQuerier) Exec(_ context.Context, _ string, args ...any) error {
	f.inserted = true
	if len(args) >= 2 {
		f.lastType, _ = args[1].(string)
	}
	return nil
}

func (f *handlerFakeQuerier) Query(context.Context, string, ...any) (outbox.Rows, error) {
	return nil, errors.New("not expected")
}

func (f *handlerFakeQuerier) BeginTx(context.Context) (context.Context, error) { return context.Background(), nil }
func (f *handlerFakeQuerier) Commit(context.Context) error                      { return nil }
func (f *handlerFakeQuerier) Rollback(context.Context) error                    { return nil }