package outbox

import (
	"context"
	"encoding/json"
)

func DecorateMessageHandlerJSON[T any](
	decoratee func(ctx context.Context, event T) error,
) MessageHandlerFunc {
	return func(ctx context.Context, message *Message) error {
		var event T

		if err := json.Unmarshal(message.Payload, &event); err != nil {
			return err
		}

		return decoratee(ctx, event)
	}
}
