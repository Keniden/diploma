package documents

import (
	"context"
	"encoding/json"

	"last-mile-task/internal/clients/cargo-documents"
	"last-mile-task/pkg/outbox"
)

// Reader delivers documents.publish_after_start_shipment messages to
// cargo-documents CreateEventByType. The at-least-once delivery mechanics
// (complete on success, reschedule with backoff on error) live in
// pkg/outbox.Reader.
func NewReader(querier outbox.Querier, client cargodocuments.DocumentsClient) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var event cargodocuments.Event
			if err := json.Unmarshal(message.Payload, &event); err != nil {
				return err
			}

			return client.CreateEventByType(ctx, event)
		}),
		outbox.ReaderWithMessageType(MessageTypePublishAfterStartShipment),
	)
}