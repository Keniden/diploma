package payments

import (
	"context"
	"encoding/json"

	paymentsclient "last-mile-task/internal/clients/payments"
	"last-mile-task/pkg/outbox"
)

// Reader delivers payments.last_mile_shipment_accrual messages to the
// payments service. The at-least-once delivery mechanics (complete on success,
// reschedule with backoff on error, idempotent per message id) live in
// pkg/outbox.Reader.
func NewReader(
	querier outbox.Querier,
	client paymentsclient.PaymentsClient,
) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var accrual paymentsclient.LastMileShipmentAccrual
			if err := json.Unmarshal(message.Payload, &accrual); err != nil {
				return err
			}

			return client.LastMileShipmentAccrual(ctx, accrual)
		}),
		outbox.ReaderWithMessageType(MessageTypeLastMileShipmentAccrual),
	)
}