package inmem

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/domain/events"
)

func TestSyncEventBusPublishesToSubscription(t *testing.T) {
	bus := NewSyncEventBus()

	var got *events.PackageScannedEvent
	bus.SubscribePackageEvent(events.PackageScanned, func(ctx context.Context, event any) error {
		e, ok := event.(events.PackageScannedEvent)
		if !ok {
			t.Fatalf("unexpected event type %T", event)
		}

		got = &e

		return nil
	})

	expected := events.PackageScannedEvent{
		Packages: []events.ScannedPackage{{TareSticker: "MPB-1"}},
	}

	if err := bus.PublishPackageEvent(context.Background(), expected); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if got == nil {
		t.Fatal("handler must be called")
	}

	if len(got.Packages) != 1 || got.Packages[0].TareSticker != "MPB-1" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestSyncEventBusStopsOnHandlerError(t *testing.T) {
	bus := NewSyncEventBus()

	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, _ any) error {
		return errors.New("boom")
	})

	err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{})
	if err == nil {
		t.Fatal("publish must fail when a handler fails")
	}
}

func TestSyncEventBusDoesNotCallOtherSubscriptions(t *testing.T) {
	bus := NewSyncEventBus()

	called := false
	bus.SubscribeShipmentEvent(events.ShipmentCreated, func(_ context.Context, _ any) error {
		called = true

		return nil
	})

	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if called {
		t.Fatal("shipment handler must not be called for a package event")
	}
}

func TestSyncEventBusPublishAfterCloseIsNoop(t *testing.T) {
	bus := NewSyncEventBus()

	called := false
	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, _ any) error {
		called = true

		return nil
	})

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{}); err != nil {
		t.Fatalf("publish after close must not fail: %v", err)
	}

	if called {
		t.Fatal("handler must not be called after close")
	}
}
