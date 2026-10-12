package outbox

import "context"

type Writer struct {
	outbox OutboxWriter
}

type WriterOption func(wr *Writer)

func NewWriter(outbox OutboxWriter, opts ...WriterOption) *Writer {
	wr := &Writer{
		outbox: outbox,
	}

	for _, opt := range opts {
		opt(wr)
	}

	return wr
}

func (w *Writer) WriteMessage(ctx context.Context, message *Message) error {
	return w.outbox.WriteMessage(ctx, message)
}
