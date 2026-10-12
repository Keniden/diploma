package application_test

import (
	"context"
	"testing"

	"last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	shippingErrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

func newProgressService(
	shipment *entities.Shipment,
	publisher *mockShipmentEventPublisher,
) (*application.ShipmentProgressService, *mockShipmentRepository) {
	shipmentRepo := &mockShipmentRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error {
			return nil
		},
	}

	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(&mockShipmentPolicyRepository{}),
		shipmentRepo,
		publisher,
		newMockTransactional(),
	)

	return svc, shipmentRepo
}

func TestAssignDriverByShipmentID_Success(t *testing.T) {
	shipment := entities.NewShipment()
	var publishedEvent events.ShipmentEvent

	publisher := &mockShipmentEventPublisher{
		publishFn: func(_ context.Context, event events.ShipmentEvent) error {
			publishedEvent = event
			return nil
		},
	}
	svc, _ := newProgressService(shipment, publisher)

	driver := values.Driver{ID: values.NewID(), SupplierID: 17}
	if err := svc.AssignDriverByShipmentID(context.Background(), shipment.ID, driver); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !shipment.IsAssigned() {
		t.Fatal("shipment must be ASSIGNED")
	}
	if shipment.DriverID != driver.ID || shipment.SupplierID != driver.SupplierID {
		t.Fatal("driver/supplier not assigned")
	}

	assigned, ok := publishedEvent.(events.ShipmentDriverAssignEvent)
	if !ok {
		t.Fatalf("expected ShipmentDriverAssignEvent, got %T", publishedEvent)
	}
	if assigned.ID != shipment.ID || assigned.Driver.ID != driver.ID {
		t.Fatal("event payload mismatch")
	}
}

func TestAssignDriverByShipmentID_InvalidTransition(t *testing.T) {
	shipment := entities.NewShipment()
	if err := shipment.AssignDriver(values.NewID(), 1); err != nil {
		t.Fatal(err)
	}

	svc, _ := newProgressService(shipment, &mockShipmentEventPublisher{})

	// Assigning twice is a prohibited transition: NEW→ASSIGNED only.
	if err := svc.AssignDriverByShipmentID(context.Background(), shipment.ID, values.Driver{ID: values.NewID()}); err == nil {
		t.Fatal("expected invalid transition error")
	}
}

func TestAssignTransportByShipmentID_Success(t *testing.T) {
	shipment := entities.NewShipment()
	var publishedEvent events.ShipmentEvent

	publisher := &mockShipmentEventPublisher{
		publishFn: func(_ context.Context, event events.ShipmentEvent) error {
			publishedEvent = event
			return nil
		},
	}
	svc, _ := newProgressService(shipment, publisher)

	vehicle := values.Vehicle{ID: 42, LicensePlate: values.NewLicensePlate("E777AA77")}
	if err := svc.AssignTransportByShipmentID(context.Background(), shipment.ID, vehicle); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if shipment.LicensePlate != vehicle.LicensePlate {
		t.Fatal("license plate not set")
	}

	assigned, ok := publishedEvent.(events.ShipmentTransportAssignEvent)
	if !ok {
		t.Fatalf("expected ShipmentTransportAssignEvent, got %T", publishedEvent)
	}
	if assigned.ID != shipment.ID || assigned.Vehicle.ID != vehicle.ID {
		t.Fatal("event payload mismatch")
	}
}

func TestCancelShipmentByID_Success(t *testing.T) {
	shipment := entities.NewShipment()
	var publishedEvent events.ShipmentEvent

	publisher := &mockShipmentEventPublisher{
		publishFn: func(_ context.Context, event events.ShipmentEvent) error {
			publishedEvent = event
			return nil
		},
	}
	svc, _ := newProgressService(shipment, publisher)

	if err := svc.CancelShipmentByID(context.Background(), shipment.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if shipment.Status != values.ShipmentStatusCanceled {
		t.Fatal("shipment must be CANCELED")
	}

	canceled, ok := publishedEvent.(events.ShipmentCanceledEvent)
	if !ok {
		t.Fatalf("expected ShipmentCanceledEvent, got %T", publishedEvent)
	}
	if canceled.ID != shipment.ID {
		t.Fatal("event payload mismatch")
	}
}

func TestCancelShipmentByID_NotFound(t *testing.T) {
	shipmentRepo := &mockShipmentRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return nil, ports.ErrShipmentNotFound
		},
	}
	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(&mockShipmentPolicyRepository{}),
		shipmentRepo,
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	if err := svc.CancelShipmentByID(context.Background(), values.NewID()); err == nil {
		t.Fatal("expected not found error")
	}
}

func newInProgressShipment(t *testing.T) *entities.Shipment {
	t.Helper()

	shipment := entities.NewShipment()
	if err := shipment.AssignDriver(values.NewID(), 42); err != nil {
		t.Fatal(err)
	}
	if err := shipment.Loading(); err != nil {
		t.Fatal(err)
	}
	if err := shipment.Start(); err != nil {
		t.Fatal(err)
	}
	return shipment
}

func TestCompleteShipmentByID_Success(t *testing.T) {
	shipment := newInProgressShipment(t)

	policyRepo := &mockShipmentPolicyRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
	}
	var publishedEvent events.ShipmentEvent
	publisher := &mockShipmentEventPublisher{
		publishFn: func(_ context.Context, event events.ShipmentEvent) error {
			publishedEvent = event
			return nil
		},
	}

	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(policyRepo),
		&mockShipmentRepository{
			getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
				return shipment, nil
			},
			saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error {
				return nil
			},
		},
		publisher,
		newMockTransactional(),
	)

	if err := svc.CompleteShipmentByID(context.Background(), shipment.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}

	completed, ok := publishedEvent.(events.ShipmentCompletedEvent)
	if !ok {
		t.Fatalf("expected ShipmentCompletedEvent, got %T", publishedEvent)
	}
	if completed.ID != shipment.ID {
		t.Fatal("event payload mismatch")
	}
}

func TestCompleteShipmentByID_UnloadingSuccess(t *testing.T) {
	shipment := newInProgressShipment(t)
	if err := shipment.Unloading(); err != nil {
		t.Fatal(err)
	}

	policyRepo := &mockShipmentPolicyRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
	}
	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(policyRepo),
		&mockShipmentRepository{
			getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
				return shipment, nil
			},
			saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error {
				return nil
			},
		},
		&mockShipmentEventPublisher{
			publishFn: func(_ context.Context, _ events.ShipmentEvent) error { return nil },
		},
		newMockTransactional(),
	)

	if err := svc.CompleteShipmentByID(context.Background(), shipment.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}
}

func TestCompleteShipmentByID_InvalidStatus(t *testing.T) {
	shipment := entities.NewShipment() // NEW status: not completable.

	policyRepo := &mockShipmentPolicyRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
	}
	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(policyRepo),
		&mockShipmentRepository{
			getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
				return shipment, nil
			},
		},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	err := svc.CompleteShipmentByID(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected a compliance error")
	}
	if !shippingErrors.Is(err, shippingErrors.InvalidShipmentCannotCompleteCode) {
		t.Fatalf("expected shipment cannot complete code, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusNew {
		t.Fatalf("shipment status must not change, got %v", shipment.Status)
	}
}

func TestCompleteShipmentByID_PendingUnload(t *testing.T) {
	shipment := newInProgressShipment(t)

	policyRepo := &mockShipmentPolicyRepository{
		getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getPackagesByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{
				{
					ID: values.NewID(),
					Actions: []*entities.PackageAction{
						{
							Type:   values.RoutePointActionTypeLoad,
							Status: values.RoutePointPackageStatusDone,
						},
						{
							Type:   values.RoutePointActionTypeUnload,
							Status: values.RoutePointPackageStatusTODO,
						},
					},
				},
			}, nil
		},
	}
	svc := application.NewShipmentProgressService(
		policies.NewShipmentPolicy(policyRepo),
		&mockShipmentRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	if err := svc.CompleteShipmentByID(context.Background(), shipment.ID); err == nil {
		t.Fatal("expected pending unload error")
	}
}