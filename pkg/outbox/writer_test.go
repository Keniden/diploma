package outbox

import (
	"context"
	"errors"
	"testing"
)

type writerStub struct {
	messages []*Message
	err      error
}

func (w *writerStub) WriteMessage(ctx context.Context, msg *Message) error {
	if w.err != nil {
		return w.err
	}

	w.messages = append(w.messages, msg)

	return nil
}

func TestWriterWriteMessage(t *testing.T) {
	stub := &writerStub{}
	writer := NewWriter(stub)

	msg := NewMessage(Payload("x"))

	if err := writer.WriteMessage(context.Background(), msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if len(stub.messages) != 1 || stub.messages[0] != msg {
		t.Fatal("message must be forwarded to OutboxWriter")
	}
}

func TestWriterPropagatesError(t *testing.T) {
	expected := errors.New("write failed")
	writer := NewWriter(&writerStub{err: expected})

	err := writer.WriteMessage(context.Background(), NewMessage(Payload("x")))
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
