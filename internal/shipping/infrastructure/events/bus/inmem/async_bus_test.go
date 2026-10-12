//nolint:unused // scaffolding (mocks/helpers) kept for unported services/suites.
package inmem

import (
	"context"
	"errors"
	"testing"
	"time"

	"last-mile-task/internal/shipping/domain/events"
)

func waitFor(t *testing.T, timeout time.Duration, fn func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal(msg)
}

func TestAsyncEventBusDeliversAsynchronously(t *testing.T) {
	bus := NewAsyncEventBus(8)
	defer func() { _ = bus.Close(context.Background()) }()

	delivered := make(chan struct{}, 1)

	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, event any) error {
		e, ok := event.(events.PackageScannedEvent)
		if !ok {
			t.Fatalf("unexpected event type %T", event)
		}

		if len(e.Packages) != 1 {
			t.Fatalf("unexpected event: %+v", e)
		}

		delivered <- struct{}{}

		return nil
	})

	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{
		Packages: []events.ScannedPackage{{TareSticker: "MPB-2"}},
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("handler must be called asynchronously")
	}
}

func TestAsyncEventBusReportsHandlerError(t *testing.T) {
	bus := NewAsyncEventBus(8)
	defer func() { _ = bus.Close(context.Background()) }()

	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, _ any) error {
		return errors.New("boom")
	})

	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{}); err != nil {
		t.Fatalf("async publish must not fail: %v", err)
	}

	select {
	case err := <-bus.Errors():
		if err == nil || err.Error() != "boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("handler error must be reported on Errors()")
	}
}

func TestAsyncEventBusCloseFlushesPendingEvents(t *testing.T) {
	bus := NewAsyncEventBus(1)

	delivered := make(chan struct{}, 1)

	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, _ any) error {
		delivered <- struct{}{}

		return nil
	})

	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	select {
	case <-delivered:
	default:
		t.Fatal("pending event must be flushed on Close")
	}
}

func TestAsyncEventBusPassesPublishContextToHandler(t *testing.T) {
	bus := NewAsyncEventBus(1)
	defer func() { _ = bus.Close(context.Background()) }()

	delivered := make(chan context.Context, 1)

	bus.SubscribePackageEvent(events.PackageScanned, func(ctx context.Context, _ any) error {
		delivered <- ctx

		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bus.PublishPackageEvent(ctx, events.PackageScannedEvent{}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-delivered:
		if got != ctx {
			t.Fatal("handler must receive the publish context")
		}
	case <-time.After(time.Second):
		t.Fatal("handler must be called")
	}
}
