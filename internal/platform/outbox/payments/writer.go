package payments

import (
	"context"
	"encoding/json"

	paymentsclient "last-mile-task/internal/clients/payments"
	"last-mile-task/internal/primitives"
	"last-mile-task/pkg/outbox"
)

// Writer writes payments.last_mile_shipment_accrual messages into lmt.outbox
// inside the current transaction.
type Writer struct {
	writer *outbox.Writer
}

// NewWriter builds the payments outbox writer over a shared outbox querier.
func NewWriter(querier outbox.Querier) *Writer {
	return &Writer{
		writer: outbox.NewWriter(
			outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		),
	}
}

// WriteLastMileShipmentAccrual marshals the accrual and appends a
// payments.last_mile_shipment_accrual message to lmt.outbox.
func (w *Writer) WriteLastMileShipmentAccrual(
	ctx context.Context,
	accrual paymentsclient.LastMileShipmentAccrual,
) error {
	data, err := json.Marshal(accrual)
	if err != nil {
		return err
	}

	return w.writer.WriteMessage(ctx, outbox.NewMessage(
		data,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(MessageTypeLastMileShipmentAccrual),
	))
}