package outbox

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewMessageDefaults(t *testing.T) {
	msg := NewMessage(Payload(`{"a":1}`))

	if msg.ID == uuid.Nil {
		t.Fatal("ID must be generated")
	}

	if msg.Type != "" {
		t.Fatalf("Type must be empty, got %q", msg.Type)
	}

	if string(msg.Payload) != `{"a":1}` {
		t.Fatalf("Payload mismatch: %s", msg.Payload)
	}

	if msg.Metadata == nil {
		t.Fatal("Metadata must be initialized")
	}

	if msg.status != MessageStatusTODO {
		t.Fatalf("status must be TODO, got %q", msg.status)
	}

	if msg.attempt != 0 {
		t.Fatalf("attempt must be 0, got %d", msg.attempt)
	}

	if !msg.scheduledAt.Before(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("scheduledAt must be near now, got %s", msg.scheduledAt)
	}
}

func TestNewMessageWithOptions(t *testing.T) {
	id := uuid.New()
	scheduledAt := time.Now().UTC().Add(time.Hour)
	md := NewMetadata()
	md.Set("k", "v")

	msg := NewMessage(
		Payload("payload"),
		MessageWithID(id),
		MessageWithType("shipment.events"),
		MessageWithScheduledAt(scheduledAt),
		MessageWithMetadata(md),
	)

	if msg.ID != id {
		t.Fatalf("ID mismatch: %s != %s", msg.ID, id)
	}

	if msg.Type != "shipment.events" {
		t.Fatalf("Type mismatch: %q", msg.Type)
	}

	if !msg.scheduledAt.Equal(scheduledAt) {
		t.Fatalf("scheduledAt mismatch: %s != %s", msg.scheduledAt, scheduledAt)
	}

	got, ok := msg.Metadata.Get("k")
	if !ok || got != "v" {
		t.Fatalf("Metadata.Get mismatch: %v, %v", got, ok)
	}
}

func TestMessageStateTransitions(t *testing.T) {
	msg := NewMessage(Payload("p"))

	msg.complete()
	if msg.status != MessageStatusCompleted {
		t.Fatalf("status must be COMPLETED, got %q", msg.status)
	}

	msg.incAttempt()
	if msg.attempt != 1 {
		t.Fatalf("attempt must be 1, got %d", msg.attempt)
	}

	msg.schedule(time.Minute, "boom")
	if msg.status != MessageStatusTODO {
		t.Fatalf("status must be TODO after schedule, got %q", msg.status)
	}
	if msg.err != "boom" {
		t.Fatalf("err must be boom, got %q", msg.err)
	}
	if !msg.scheduledAt.After(time.Now().UTC().Add(30 * time.Second)) {
		t.Fatalf("scheduledAt must be in the future, got %s", msg.scheduledAt)
	}

	msg.discard()
	if msg.status != MessageStatusDiscarded {
		t.Fatalf("status must be DISCARDED, got %q", msg.status)
	}
}

func TestMetadataOnNilMap(t *testing.T) {
	var md Metadata

	if _, ok := md.Get("x"); ok {
		t.Fatal("Get on nil metadata must return false")
	}

	md.Set("x", 1) // must not panic
}

func TestStatusConstants(t *testing.T) {
	if MessageStatusTODO != "TODO" ||
		MessageStatusCompleted != "COMPLETED" ||
		MessageStatusDiscarded != "DISCARDED" {
		t.Fatal("status constants changed")
	}
}
