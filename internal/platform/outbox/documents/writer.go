package documents

import (
	"context"
	"encoding/json"

	"last-mile-task/internal/clients/cargo-documents"
	"last-mile-task/internal/primitives"
	"last-mile-task/pkg/outbox"
)

// Writer writes documents.publish_after_start_shipment messages into
// lmt.outbox inside the current transaction.
type Writer struct {
	writer *outbox.Writer
}

// NewWriter builds the documents outbox writer over a shared outbox querier.
func NewWriter(querier outbox.Querier) *Writer {
	return &Writer{
		writer: outbox.NewWriter(
			outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		),
	}
}

// WritePublishAfterStartShipment marshals the document event and appends a
// documents.publish_after_start_shipment message to lmt.outbox.
func (w *Writer) WritePublishAfterStartShipment(ctx context.Context, event cargodocuments.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return w.writer.WriteMessage(ctx, outbox.NewMessage(
		data,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(MessageTypePublishAfterStartShipment),
	))
}