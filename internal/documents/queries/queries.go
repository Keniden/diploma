package queries

import (
	"context"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
)

// DocumentReader is the read surface the document read-models depend on.
// documentspg.DocumentRepository implements it.
type DocumentReader interface {
	GetByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Document, error)
	GetByEventUUID(ctx context.Context, eventUUID string) (*entities.Document, error)
}

// Queries is the document read-model accessor.
type Queries struct {
	reader DocumentReader
}

// New builds the read-model accessor over a DocumentReader.
func New(reader DocumentReader) *Queries {
	return &Queries{reader: reader}
}

// ByShipmentID returns all documents of a shipment.
func (q *Queries) ByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Document, error) {
	return q.reader.GetByShipmentID(ctx, shipmentID)
}

// ByEventUUID returns the document with the given event uuid.
func (q *Queries) ByEventUUID(ctx context.Context, eventUUID string) (*entities.Document, error) {
	return q.reader.GetByEventUUID(ctx, eventUUID)
}