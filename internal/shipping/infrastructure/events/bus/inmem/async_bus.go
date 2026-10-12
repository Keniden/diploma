package inmem

import (
	"context"
	"sync"

	"last-mile-task/internal/shipping/domain/events"
)

// The real repository asserts at compile time that both buses implement the
// ports.ShipmentEventPublisher / ports.PackageEventPublisher /
// ports.ReserveEventPublisher / ports.RouteEventPublisher interfaces (see
// internal/shipping/application/ports/events.go). The skeleton's application
// layer is ported in parallel, so the assertions are added back once that
// package lands. TODO(LASTMILE-050): restore the compile-time interface
// assertions against ports.
type asyncEvent struct {
	ctx       context.Context
	eventType string
	event     any
}

type AsyncEventBus struct {
	mu     sync.RWMutex
	subs   map[string][]EventHandler
	errCh  chan error
	events chan asyncEvent
	wg     sync.WaitGroup
}

func NewAsyncEventBus(bufferSize int) *AsyncEventBus {
	b := &AsyncEventBus{
		subs:   make(map[string][]EventHandler),
		errCh:  make(chan error, 64),
		events: make(chan asyncEvent, bufferSize),
	}
	b.wg.Add(1)
	go b.worker()
	return b
}

func (b *AsyncEventBus) worker() {
	defer b.wg.Done()
	for evt := range b.events {
		b.dispatch(evt)
	}
}

func (b *AsyncEventBus) dispatch(evt asyncEvent) {
	b.mu.RLock()
	handlers := b.subs[evt.eventType]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(evt.ctx, evt.event); err != nil {
			select {
			case b.errCh <- err:
			default:
			}
		}
	}
}

func (b *AsyncEventBus) SubscribeShipmentEvent(eventType events.ShipmentEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *AsyncEventBus) SubscribeReserveEvent(eventType events.ReserveEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *AsyncEventBus) SubscribePackageEvent(eventType events.PackageEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *AsyncEventBus) SubscribeRouteEvent(eventType events.RouteEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *AsyncEventBus) PublishShipmentEvent(ctx context.Context, event events.ShipmentEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *AsyncEventBus) PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *AsyncEventBus) PublishPackageEvent(ctx context.Context, event events.PackageEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *AsyncEventBus) PublishRouteEvent(ctx context.Context, event events.RouteEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *AsyncEventBus) publish(ctx context.Context, eventType string, event any) error {
	select {
	case b.events <- asyncEvent{ctx: ctx, eventType: eventType, event: event}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *AsyncEventBus) Close(_ context.Context) error {
	close(b.events)
	b.wg.Wait()
	return nil
}

func (b *AsyncEventBus) Errors() <-chan error {
	return b.errCh
}
