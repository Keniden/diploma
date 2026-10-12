package wms

import (
	"context"

	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
)

type exporter func(ctx context.Context, msg *outbox.Message) error

func (e exporter) HandleMessage(ctx context.Context, msg *outbox.Message) error {
	return e(ctx, msg)
}

func NewLoadReader(querier outbox.Querier, client *wms.Client) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		exporter(func(ctx context.Context, msg *outbox.Message) error {
			decoded, err := decodeMessage(msg.Payload)
			if err != nil {
				return err
			}

			return client.Load(ctx, decoded.Shipment, decoded.Tares)
		}),
		outbox.ReaderWithMessageType(MessageTypeTaresLoad),
	)
}

func NewUnloadReader(querier outbox.Querier, client *wms.Client) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		exporter(func(ctx context.Context, msg *outbox.Message) error {
			decoded, err := decodeMessage(msg.Payload)
			if err != nil {
				return err
			}

			return client.Unload(ctx, decoded.Shipment, decoded.Tares)
		}),
		outbox.ReaderWithMessageType(MessageTypeTaresUnload),
	)
}
