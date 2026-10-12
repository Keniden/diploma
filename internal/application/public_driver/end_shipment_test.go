package publicdriver_test

import (
	"context"
	"testing"

	publicdriver "last-mile-task/internal/application/public_driver"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/domain/entities"
	shippingErrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

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

func newProgressService(
	shipment *entities.Shipment,
	eventsSink *[]events.ShipmentEvent,
) *shipping_application.ShipmentProgressService {
	return shipping_application.NewShipmentProgressService(
		policies.NewShipmentPolicy(&mockShipmentPolicyRepository{
			getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
				return shipment, nil
			},
		}),
		&mockShipmentRepository{
			getShipmentByIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
				return shipment, nil
			},
			saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error {
				return nil
			},
		},
		&mockShipmentEventPublisher{events: eventsSink},
		inlineTx{},
	)
}

func TestEndShipment_Success(t *testing.T) {
	shipment := newInProgressShipment(t)
	eventsSink := &[]events.ShipmentEvent{}
	app := publicdriver.New(publicdriver.Config{
		ProgressService: newProgressService(shipment, eventsSink),
	})

	if err := app.EndShipment(context.Background(), shipment.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}
	if len(*eventsSink) != 1 {
		t.Fatalf("expected one published event, got %d", len(*eventsSink))
	}
	completed, ok := (*eventsSink)[0].(events.ShipmentCompletedEvent)
	if !ok {
		t.Fatalf("expected ShipmentCompletedEvent, got %T", (*eventsSink)[0])
	}
	if completed.ID != shipment.ID {
		t.Fatal("event shipment id mismatch")
	}
}

func TestEndShipment_NotReadyToComplete(t *testing.T) {
	shipment := entities.NewShipment() // NEW status: not completable.
	eventsSink := &[]events.ShipmentEvent{}
	app := publicdriver.New(publicdriver.Config{
		ProgressService: newProgressService(shipment, eventsSink),
	})

	err := app.EndShipment(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected error")
	}
	if !shippingErrors.Is(err, shippingErrors.InvalidShipmentCannotCompleteCode) {
		t.Fatalf("expected shipment cannot complete code, got %v", err)
	}
	if shipment.Status != values.ShipmentStatusNew {
		t.Fatalf("shipment status must not change, got %v", shipment.Status)
	}
	if len(*eventsSink) != 0 {
		t.Fatal("no event must be published on failure")
	}
}