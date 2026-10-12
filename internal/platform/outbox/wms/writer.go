package wms

import (
	"context"

	"last-mile-task/internal/primitives"
	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
)

type Writer struct {
	writer *outbox.Writer
}

// The real repository accepts *database.Repository and wraps it with
// database.NewOutboxAdapter. The skeleton has no database layer yet.
// TODO(LASTMILE-057): accept *database.Repository once it is ported.
func NewWriter(querier outbox.Querier) *Writer {
	return &Writer{
		writer: outbox.NewWriter(
			outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		),
	}
}

func (w *Writer) UnloadTares(
	ctx context.Context,
	shipment wms.Shipment,
	tt []tares.Tare,
) error {
	return w.writeMessage(ctx, shipment, tt, MessageTypeTaresUnload)
}

func (w *Writer) LoadTares(
	ctx context.Context,
	shipment wms.Shipment,
	tt []tares.Tare,
) error {
	return w.writeMessage(ctx, shipment, tt, MessageTypeTaresLoad)
}

func (w *Writer) writeMessage(
	ctx context.Context,
	shipment wms.Shipment,
	tt []tares.Tare,
	messageType string,
) error {
	data, err := encodeMessage(Message{
		Shipment: shipment,
		Tares:    tt,
	})
	if err != nil {
		return err
	}

	msg := outbox.NewMessage(
		data,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(messageType),
	)

	return w.writer.WriteMessage(ctx, msg)
}
