package inmem

import "context"

// EventHandler is a generic event handler that receives any event.
type EventHandler func(ctx context.Context, event any) error

// TypedHandler wraps a typed event handler into an EventHandler by performing
// a type assertion on the event. Panics if the event type does not match T.
func TypedHandler[T any](handler func(ctx context.Context, event T) error) EventHandler {
	return func(ctx context.Context, event any) error {
		return handler(ctx, event.(T))
	}
}
