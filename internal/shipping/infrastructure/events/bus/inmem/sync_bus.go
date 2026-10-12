package inmem

import (
	"context"
	"sync"

	"last-mile-task/internal/shipping/domain/events"
)

// The real repository asserts at compile time that both buses implement
// ports.ShipmentEventPublisher / ports.PackageEventPublisher /
// ports.ReserveEventPublisher / ports.RouteEventPublisher (see
// internal/shipping/application/ports/events.go). The skeleton's application
// layer is ported in parallel, so the assertions are added back once that
// package lands. TODO(LASTMILE-050): restore the compile-time interface
// assertions against ports.
type SyncEventBus struct {
	mu   sync.RWMutex
	subs map[string][]EventHandler
}

func NewSyncEventBus() *SyncEventBus {
	return &SyncEventBus{
		subs: make(map[string][]EventHandler),
	}
}

func (b *SyncEventBus) SubscribeShipmentEvent(eventType events.ShipmentEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *SyncEventBus) SubscribeReserveEvent(eventType events.ReserveEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *SyncEventBus) SubscribePackageEvent(eventType events.PackageEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *SyncEventBus) SubscribeRouteEvent(eventType events.RouteEventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[string(eventType)] = append(b.subs[string(eventType)], handler)
}

func (b *SyncEventBus) PublishShipmentEvent(ctx context.Context, event events.ShipmentEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *SyncEventBus) PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *SyncEventBus) PublishPackageEvent(ctx context.Context, event events.PackageEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *SyncEventBus) PublishRouteEvent(ctx context.Context, event events.RouteEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *SyncEventBus) Close(_ context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = nil
	return nil
}

func (b *SyncEventBus) publish(ctx context.Context, eventType string, event any) error {
	b.mu.RLock()
	handlers := b.subs[eventType]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
