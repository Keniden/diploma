package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	docs_outbox "last-mile-task/internal/platform/outbox/documents"
	"last-mile-task/pkg/outbox"
)

// documentsExporter delivers documents.publish_after_start_shipment outbox
// messages to the cargo-documents CreateEventByType service. The at-least-once
// delivery semantics (complete on success, reschedule with backoff on failure)
// live in pkg/outbox.Reader; the exporter starts the reader, surfaces its
// errors into the log and stops it on shutdown.
type documentsExporter struct {
	logger  *slog.Logger
	readers []outboxReader
}

func newDocumentsExporter(logger *slog.Logger, readers []outboxReader) *documentsExporter {
	return &documentsExporter{
		logger:  logger,
		readers: readers,
	}
}

// Start launches all readers.
func (d *documentsExporter) Start() {
	for _, r := range d.readers {
		r.Start()
	}
}

// Run pumps reader errors into the log until the context is canceled.
func (d *documentsExporter) Run(ctx context.Context) {
	for _, r := range d.readers {
		r := r

		go func() {
			for err := range r.Errors() {
				if err != nil {
					d.logger.Error("documents exporter", "error", err)
				}
			}
		}()
	}

	<-ctx.Done()
}

// Stop gracefully stops all readers and returns the joined stop errors.
func (d *documentsExporter) Stop(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, r := range d.readers {
		wg.Add(1)

		go func(r outboxReader) {
			defer wg.Done()

			if err := r.Stop(ctx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(r)
	}

	wg.Wait()

	return errors.Join(errs...)
}

// newDocumentsReader builds the documents outbox reader over the shared
// querier and the cargo-documents client.
func newDocumentsReader(
	querier outbox.Querier,
	client cargodocuments.DocumentsClient,
) outboxReader {
	return docs_outbox.NewReader(querier, client)
}