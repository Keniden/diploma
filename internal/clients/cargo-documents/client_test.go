package cargodocuments_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
)

func TestDocumentsClientInterfaceSatisfied(t *testing.T) {
	var _ cargodocuments.DocumentsClient = &cargodocuments.Client{}
}

func TestClientCreateEventByTypeIsNoop(t *testing.T) {
	client := cargodocuments.NewClient()

	err := client.CreateEventByType(context.Background(), cargodocuments.Event{
		EventUUID:    "evt-1",
		ShipmentID:   uuid.New(),
		DocumentType: 1,
	})

	if err != nil {
		t.Fatalf("expected stub to succeed, got %v", err)
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	event := cargodocuments.Event{
		EventUUID:  "evt-1",
		ShipmentID: uuid.New(),
		PackageIDs: []uuid.UUID{uuid.New()},
		Items:      []cargodocuments.Item{{Article: "BOX1", Qty: 1, Unit: "pkg"}},
	}

	// The writer marshals the event; the reader must be able to unmarshal it
	// back into the same shape.
	var decoded cargodocuments.Event
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	if decoded.EventUUID != event.EventUUID || decoded.ShipmentID != event.ShipmentID {
		t.Fatal("event fields must survive JSON round trip")
	}
	if len(decoded.PackageIDs) != 1 || len(decoded.Items) != 1 {
		t.Fatalf("unexpected round trip: %+v", decoded)
	}
}