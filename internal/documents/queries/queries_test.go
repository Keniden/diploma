package queries_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/documents/queries"
)

type fakeReader struct {
	shipmentDocs []*entities.Document
	eventDoc     *entities.Document
	shipmentErr  error
	eventErr     error
}

func (f *fakeReader) GetByShipmentID(context.Context, values.ShipmentID) ([]*entities.Document, error) {
	return f.shipmentDocs, f.shipmentErr
}

func (f *fakeReader) GetByEventUUID(context.Context, string) (*entities.Document, error) {
	return f.eventDoc, f.eventErr
}

func TestByShipmentID(t *testing.T) {
	doc := &entities.Document{ID: uuid.New()}
	q := queries.New(&fakeReader{shipmentDocs: []*entities.Document{doc}})

	got, err := q.ByShipmentID(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != doc {
		t.Fatal("expected the document from the reader")
	}
}

func TestByEventUUID(t *testing.T) {
	doc := &entities.Document{ID: uuid.New()}
	q := queries.New(&fakeReader{eventDoc: doc})

	got, err := q.ByEventUUID(context.Background(), "evt-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != doc {
		t.Fatal("expected the document from the reader")
	}
}

func TestByShipmentID_PropagatesError(t *testing.T) {
	q := queries.New(&fakeReader{shipmentErr: entities.ErrDocumentNotFound})
	_, err := q.ByShipmentID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}