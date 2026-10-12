package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"last-mile-task/internal/config"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/wmsmock"
)

// outboxReader is the surface of pkg/outbox.Reader the exporter needs.
type outboxReader interface {
	Start()
	Stop(ctx context.Context) error
	Errors() <-chan error
}

// wmsExporter delivers wms.tares.load / wms.tares.unload outbox messages to
// WMS. The delivery semantics (at-least-once: complete on success, reschedule
// with backoff on failure, discard after max attempts) live in pkg/outbox
// Reader; the exporter starts the readers, surfaces their errors into the
// log and stops them on shutdown.
type wmsExporter struct {
	logger  *slog.Logger
	readers []outboxReader
}

func newWMSEporter(logger *slog.Logger, readers []outboxReader) *wmsExporter {
	return &wmsExporter{
		logger:  logger,
		readers: readers,
	}
}

// Start launches all readers.
func (w *wmsExporter) Start() {
	for _, r := range w.readers {
		r.Start()
	}
}

// Run pumps reader errors into the log until the context is canceled.
func (w *wmsExporter) Run(ctx context.Context) {
	for _, r := range w.readers {
		r := r

		go func() {
			for err := range r.Errors() {
				if err != nil {
					w.logger.Error("wms exporter", "error", err)
				}
			}
		}()
	}

	<-ctx.Done()
}

// Stop gracefully stops all readers and returns the joined stop errors.
func (w *wmsExporter) Stop(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, r := range w.readers {
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

// newWMSReaders builds the load/unload readers over the shared outbox
// querier and the WMS client (see platform/outbox/wms).
func newWMSReaders(querier outbox.Querier, client *wms.Client) []outboxReader {
	load := wms_outbox.NewLoadReader(querier, client)
	unload := wms_outbox.NewUnloadReader(querier, client)

	return []outboxReader{load, unload}
}

// newWMSClient builds the WMS client. With mocked=true the client is pointed
// at the in-process wmsmock server instead of the configured endpoint. The
// returned mock (nil when mocked=false) must be closed on shutdown.
func newWMSClient(conf config.WMS, mocked bool) (*wms.Client, *wmsmock.Server, error) {
	endpoint := wms.Endpoint{
		Country: "russia",
		Enabled: true,
		Auth:    wms.Auth{Username: conf.Username, Password: conf.Password},
		URI:     conf.BaseURL,
	}

	if mocked {
		server := wmsmock.New()
		endpoint.URI = server.URL()
		endpoint.Auth = wms.Auth{}

		client, err := wms.New(wms.Config{
			Endpoints: []wms.Endpoint{endpoint},
			Timeout:   10 * time.Second,
		})
		if err != nil {
			_ = server.Close()
			return nil, nil, err
		}

		return client, server, nil
	}

	client, err := wms.New(wms.Config{
		Endpoints: []wms.Endpoint{endpoint},
		Timeout:   10 * time.Second,
	})

	return client, nil, err
}
