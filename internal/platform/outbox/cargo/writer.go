package cargo

import (
	"context"
	"encoding/json"

	"last-mile-task/internal/primitives"
	"last-mile-task/pkg/outbox"
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

func (w *Writer) WriteScannedPackages(
	ctx context.Context,
	msg ScannedPackagesMessage,
) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return w.writer.WriteMessage(ctx, outbox.NewMessage(
		data,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(MessageTypeScannedPackages),
	))
}

func (w *Writer) WritePackagesStatus(
	ctx context.Context,
	msg PackagesStatusMessage,
) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return w.writer.WriteMessage(ctx, outbox.NewMessage(
		data,
		outbox.MessageWithID(primitives.NewID()),
		outbox.MessageWithType(MessageTypePackagesStatusUpdated),
	))
}
