package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type jsonEvent struct {
	ID string `json:"id"`
}

func TestDecorateMessageHandlerJSON(t *testing.T) {
	received := make(chan jsonEvent, 1)

	handler := DecorateMessageHandlerJSON(func(ctx context.Context, event jsonEvent) error {
		received <- event

		return nil
	})(context.Background(), &Message{
		Payload: Payload(`{"id":"42"}`),
	})

	if err := handler; err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	select {
	case event := <-received:
		if event.ID != "42" {
			t.Fatalf("decoded id mismatch: %q", event.ID)
		}
	default:
		t.Fatal("decoratee must be called")
	}
}

func TestDecorateMessageHandlerJSONInvalidPayload(t *testing.T) {
	handler := DecorateMessageHandlerJSON(func(ctx context.Context, event jsonEvent) error {
		t.Fatal("decoratee must not be called on invalid payload")

		return nil
	})(context.Background(), &Message{
		Payload: Payload(`{invalid`),
	})

	if handler == nil {
		t.Fatal("invalid payload must produce an error")
	}

	var unmarshalErr *json.SyntaxError
	if !errors.As(handler, &unmarshalErr) {
		t.Fatalf("expected JSON syntax error, got %T: %v", handler, handler)
	}
}
