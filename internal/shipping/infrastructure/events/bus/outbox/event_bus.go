package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/pkg/outbox"
)

// EventBus is an event bus that turns domain events into outbox messages
// (one message per published event, keyed by the event type) instead of
// dispatching them to in-memory handlers. Messages are persisted into
// lmt.outbox and are later consumed by outbox.Reader subscriptions started
// with Subscribe*Event.
//
// Delivery semantics: Publish writes exactly one message per call (the write
// is the only step, so a failed write surfaces to the caller and the whole
// domain transaction can be rolled back), and consumption is at-least-once:
// outbox.Reader retries a failed handler with backoff and discards the
// message after ReaderWithMaxAttempts attempts.
type EventHandler = func(ctx context.Context, message *outbox.Message) error

type EventBus struct {
	writer *outbox.Writer
	outbox *outbox.PostgresOutbox

	errCh chan error
	subs  []*outbox.Reader
}

// The real repository accepts *database.Repository and wraps it with
// database.NewOutboxAdapter. The skeleton has no database layer yet.
// TODO(LASTMILE-057): accept *database.Repository once it is ported.
func New(querier outbox.Querier) *EventBus {
	pgOutbox := outbox.NewPostgresOutbox(querier, "lmt.outbox")

	return &EventBus{
		writer: outbox.NewWriter(
			pgOutbox,
			// TODO(LASTMILE-051): relay/serialize options like the real repo.
		),
		outbox: pgOutbox,

		errCh: make(chan error, 1024),
		subs:  make([]*outbox.Reader, 0, 10),
	}
}

func (b *EventBus) SubscribePackageEvent(eventType events.PackageEventType, handler EventHandler) {
	b.subscribe(string(eventType), handler)
}

func (b *EventBus) SubscribeReserveEvent(eventType events.ReserveEventType, handler EventHandler) {
	b.subscribe(string(eventType), handler)
}

func (b *EventBus) SubscribeShipmentEvent(eventType events.ShipmentEventType, handler EventHandler) {
	b.subscribe(string(eventType), handler)
}

func (b *EventBus) SubscribeRouteEvent(eventType events.RouteEventType, handler EventHandler) {
	b.subscribe(string(eventType), handler)
}

func (b *EventBus) PublishShipmentEvent(ctx context.Context, event events.ShipmentEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *EventBus) PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *EventBus) PublishPackageEvent(ctx context.Context, event events.PackageEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *EventBus) PublishRouteEvent(ctx context.Context, event events.RouteEvent) error {
	return b.publish(ctx, string(event.Type()), event)
}

func (b *EventBus) Close(ctx context.Context) error {
	// The real repo stops the readers via errgroup.WithContext; errgroup is
	// only an indirect dependency of this skeleton, so a sync-based join is
	// used instead (same approach as pkg/wms/client.go). TODO(LASTMILE-051).
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, sub := range b.subs {
		wg.Add(1)

		go func(sub *outbox.Reader) {
			defer wg.Done()

			if err := sub.Stop(ctx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(sub)
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (b *EventBus) Errors() <-chan error {
	return b.errCh
}

func (b *EventBus) publish(ctx context.Context, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := outbox.NewMessage(
		payload,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(event),
	)

	return b.writer.WriteMessage(ctx, message)
}

func (b *EventBus) subscribe(event string, handler outbox.MessageHandlerFunc) {
	reader := outbox.NewReader(
		b.outbox,
		handler,
		outbox.ReaderWithMessageType(event),
		// TODO(LASTMILE-051): interval/backoff/timeout options like the real repo.
	)

	reader.Start()

	go func() {
		for err := range reader.Errors() {
			select {
			case b.errCh <- err:
			default:
			}
		}
	}()

	b.subs = append(b.subs, reader)
}
