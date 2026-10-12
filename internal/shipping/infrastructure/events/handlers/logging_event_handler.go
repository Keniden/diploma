package handlers

import (
	"context"
	"log/slog"

	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
)

func TypedLoggedHandler[E ~string, T any](
	event E,
	handler func(ctx context.Context, event T) error,
) (E, inmem.EventHandler) {
	return event,
		inmem.TypedHandler(func(ctx context.Context, data T) error {
			slog.InfoContext(ctx,
				"handle event",
				"event", event,
			)

			slog.DebugContext(ctx,
				"handle event",
				"event", event,
				"data", data,
			)

			err := handler(ctx, data)
			if err != nil {
				slog.ErrorContext(ctx,
					"event handled",
					"event", event,
					"error", err,
					"data", data,
				)
			}

			return err
		})
}
