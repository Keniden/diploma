package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	paymentsclient "last-mile-task/internal/clients/payments"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/pkg/outbox"
)

// paymentsExporter delivers payments.last_mile_shipment_accrual outbox
// messages to the payments service. The at-least-once delivery semantics
// (complete on success, reschedule with backoff on failure, idempotent per
// message id) live in pkg/outbox.Reader; the exporter starts the reader,
// surfaces its errors into the log and stops it on shutdown.
type paymentsExporter struct {
	logger  *slog.Logger
	readers []outboxReader
}

func newPaymentsExporter(logger *slog.Logger, readers []outboxReader) *paymentsExporter {
	return &paymentsExporter{
		logger:  logger,
		readers: readers,
	}
}

// Start launches all readers.
func (p *paymentsExporter) Start() {
	for _, r := range p.readers {
		r.Start()
	}
}

// Run pumps reader errors into the log until the context is canceled.
func (p *paymentsExporter) Run(ctx context.Context) {
	for _, r := range p.readers {
		r := r

		go func() {
			for err := range r.Errors() {
				if err != nil {
					p.logger.Error("payments exporter", "error", err)
				}
			}
		}()
	}

	<-ctx.Done()
}

// Stop gracefully stops all readers and returns the joined stop errors.
func (p *paymentsExporter) Stop(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, r := range p.readers {
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

// newPaymentsReader builds the payments outbox reader over the shared querier
// and the payments client.
func newPaymentsReader(
	querier outbox.Querier,
	client paymentsclient.PaymentsClient,
) outboxReader {
	return payments_outbox.NewReader(querier, client)
}