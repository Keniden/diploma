//nolint:unused // scaffolding (mocks/helpers) kept for unported services/suites.
package application

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/config"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
	"last-mile-task/pkg/wms/wmsmock"
)

// ---- fake outbox (pkg/outbox.Querier) backed by in-memory state ----------

type fakeOutboxState struct {
	id          uuid.UUID
	messageType string
	payload     outbox.Payload
	status      outbox.Status
	attempt     int
	scheduledAt time.Time
	lastError   string
}

type fakeOutbox struct {
	mu      sync.Mutex
	msg     *fakeOutboxState
	queried int
}

func (f *fakeOutbox) BeginTx(context.Context) (context.Context, error) {
	return context.Background(), nil
}

func (f *fakeOutbox) Commit(context.Context) error   { return nil }
func (f *fakeOutbox) Rollback(context.Context) error { return nil }

func (f *fakeOutbox) Exec(_ context.Context, sql string, arguments ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case strings.Contains(sql, "SET status = $1, attempt = attempt + 1"):
		// CompleteMessages: status = $1, id IN ($2, ...). The postgres
		// adapter appends ids one by one, so a single id arrives as a
		// plain uuid.UUID.
		if len(arguments) < 2 {
			return nil
		}
		f.msg.status = statusOf(arguments[0])
		if id, ok := arguments[1].(uuid.UUID); ok {
			f.msg.id = id
		}
	case strings.Contains(sql, "scheduled_at = $1"):
		// ScheduleMessage: scheduled_at = $1, status = $2, err = $3, id = $4
		f.msg.scheduledAt = arguments[0].(time.Time)
		f.msg.status = statusOf(arguments[1])
		f.msg.lastError = arguments[2].(string)
		f.msg.attempt++
	}

	return nil
}

func (f *fakeOutbox) Query(_ context.Context, _ string, args ...any) (outbox.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.queried++

	// SELECT ... WHERE type = $1 AND status = $2 AND scheduled_at < NOW() LIMIT $3
	if len(args) < 3 {
		return &emptyRows{}, nil
	}

	if args[0].(string) != f.msg.messageType {
		return &emptyRows{}, nil
	}
	if statusOf(args[1]) != f.msg.status {
		return &emptyRows{}, nil
	}
	if !f.msg.scheduledAt.Before(time.Now().UTC()) {
		return &emptyRows{}, nil
	}

	return &messageRows{msg: f.msg}, nil
}

// rows implementation. ReadMessages scans:
// id, type, payload, metadata, attempt, scheduled_at, status, err.
type messageRows struct {
	msg     *fakeOutboxState
	scanned bool
}

func (r *messageRows) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.msg.id
	*(dest[1].(*string)) = r.msg.messageType
	*(dest[2].(*outbox.Payload)) = r.msg.payload
	*(dest[3].(*outbox.Metadata)) = outbox.NewMetadata()
	*(dest[4].(*int)) = r.msg.attempt
	*(dest[5].(*time.Time)) = r.msg.scheduledAt
	*(dest[6].(*outbox.Status)) = r.msg.status
	*(dest[7].(*string)) = r.msg.lastError

	return nil
}

func (r *messageRows) Next() bool {
	if r.scanned {
		return false
	}
	r.scanned = true

	return true
}

func (r *messageRows) Close()     {}
func (r *messageRows) Err() error { return nil }

type emptyRows struct{}

func (e *emptyRows) Scan(...any) error { return nil }
func (e *emptyRows) Next() bool        { return false }
func (e *emptyRows) Close()            {}
func (e *emptyRows) Err() error        { return nil }

// statusOf normalizes a status argument: pkg/outbox passes the untyped
// MessageStatus* constants into variadic any, so they arrive as string.
func statusOf(v any) outbox.Status {
	switch s := v.(type) {
	case outbox.Status:
		return s
	case string:
		return outbox.Status(s)
	default:
		return ""
	}
}

func (s *fakeOutboxState) snapshot() fakeOutboxState {
	return *s
}

func wmsPayload(t *testing.T) outbox.Payload {
	t.Helper()

	data, err := json.Marshal(wms_outbox.Message{
		Shipment: wms.Shipment{Driver: wms.Driver{Name: "Ivan Petrov", Phone: "+7-900"}},
		Tares: []tares.Tare{{
			Sticker:   tares.NewSticker("MPB-200"),
			OfficeID:  11,
			CountryID: 643,
			Timestamp: time.Now().UTC().Add(-time.Minute),
		}},
	})
	require.NoError(t, err)

	return data
}

func newFakeOutbox(t *testing.T, messageType string) *fakeOutbox {
	t.Helper()

	return &fakeOutbox{
		msg: &fakeOutboxState{
			id:          uuid.New(),
			messageType: messageType,
			payload:     wmsPayload(t),
			status:      outbox.MessageStatusTODO,
			scheduledAt: time.Now().UTC().Add(-time.Hour),
		},
	}
}

func newTestWMSClient(t *testing.T, server *wmsmock.Server) *wms.Client {
	t.Helper()

	client, err := wms.New(wms.Config{
		Endpoints: []wms.Endpoint{
			{Country: "russia", Enabled: true, URI: server.URL(), Auth: wms.Auth{Username: "u", Password: "p"}},
		},
		Timeout: time.Second,
	})
	require.NoError(t, err)

	return client
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal(msg)
}

// ---- tests ----------------------------------------------------------------

// Delivery success: the message is exported to WMS and completed in the
// outbox.
func TestWMSEporterLoadReaderCompletesOnSuccess(t *testing.T) {
	server := wmsmock.New()
	defer func() { _ = server.Close() }()

	q := newFakeOutbox(t, wms_outbox.MessageTypeTaresLoad)

	exporter := newWMSEporter(slog.Default(), newWMSReaders(q, newTestWMSClient(t, server)))
	exporter.Start()

	waitFor(t, time.Second, func() bool {
		return len(server.Events()) == 1
	}, "wms mock must receive the load request")

	waitFor(t, time.Second, func() bool {
		q.mu.Lock()
		defer q.mu.Unlock()
		return q.msg.status == outbox.MessageStatusCompleted
	}, "message must be completed in the outbox")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, exporter.Stop(ctx))
}

// Delivery failure (FailNext): the message is rescheduled, not completed.
func TestWMSEporterLoadReaderReschedulesOnError(t *testing.T) {
	server := wmsmock.New()
	defer func() { _ = server.Close() }()
	server.FailNext(1)

	q := newFakeOutbox(t, wms_outbox.MessageTypeTaresLoad)

	exporter := newWMSEporter(slog.Default(), newWMSReaders(q, newTestWMSClient(t, server)))
	exporter.Start()

	waitFor(t, time.Second, func() bool {
		q.mu.Lock()
		defer q.mu.Unlock()
		return q.msg.attempt == 1 && q.msg.status == outbox.MessageStatusTODO && q.msg.lastError != ""
	}, "message must be rescheduled with the attempt incremented")

	require.Empty(t, server.Events(), "failed request must not be recorded by the mock")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, exporter.Stop(ctx))
}

// FailNext then retry: the next poll delivers the message and completes it.
func TestWMSEporterLoadReaderRetriesAfterFailure(t *testing.T) {
	server := wmsmock.New()
	defer func() { _ = server.Close() }()
	server.FailNext(1)

	q := newFakeOutbox(t, wms_outbox.MessageTypeTaresLoad)

	// Built with short intervals (production readers run with the defaults;
	// interval/backoff are configurable on pkg/outbox.Reader).
	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(q, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, msg *outbox.Message) error {
			var decoded wms_outbox.Message
			if err := json.Unmarshal(msg.Payload, &decoded); err != nil {
				return err
			}
			return newTestWMSClient(t, server).Load(ctx, decoded.Shipment, decoded.Tares)
		}),
		outbox.ReaderWithMessageType(wms_outbox.MessageTypeTaresLoad),
		outbox.ReaderWithInterval(10*time.Millisecond),
		outbox.ReaderWithDelayFunc(func(attempt int) time.Duration {
			return time.Duration(attempt) * time.Millisecond
		}),
	)

	exporter := newWMSEporter(slog.Default(), []outboxReader{reader})
	exporter.Start()

	waitFor(t, 2*time.Second, func() bool {
		q.mu.Lock()
		defer q.mu.Unlock()
		return q.msg.status == outbox.MessageStatusCompleted
	}, "message must be completed after the retry")

	waitFor(t, time.Second, func() bool {
		return len(server.Events()) == 1
	}, "wms mock must receive exactly one successful request")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, exporter.Stop(ctx))
}

// Worker orchestration: Start launches readers, Run drains errors, Stop
// stops them.
type fakeReader struct {
	started chan struct{}
	stopped chan struct{}
	errs    chan error
}

func newFakeReader() *fakeReader {
	return &fakeReader{
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
		errs:    make(chan error, 4),
	}
}

func (f *fakeReader) Start() {
	f.started <- struct{}{}
}

func (f *fakeReader) Stop(context.Context) error {
	close(f.stopped)
	return nil
}

func (f *fakeReader) Errors() <-chan error { return f.errs }

func TestWMSEporterStartStopDrainsErrors(t *testing.T) {
	reader := newFakeReader()
	exporter := newWMSEporter(slog.Default(), []outboxReader{reader})

	// Sanity: Start launches, Run pumps errors until the context ends.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		exporter.Run(ctx)
		close(done)
	}()

	exporter.Start()
	select {
	case <-reader.started:
	case <-time.After(time.Second):
		t.Fatal("reader must be started")
	}

	reader.errs <- context.DeadlineExceeded

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run must exit after the context is canceled")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()

	require.NoError(t, exporter.Stop(stopCtx))
	select {
	case <-reader.stopped:
	case <-time.After(time.Second):
		t.Fatal("reader must be stopped")
	}
}

// config.Mock.Enabled: the client is pointed at the in-process wmsmock.
func TestNewWMSClientMockMode(t *testing.T) {
	client, server, err := newWMSClient(config.WMS{BaseURL: "http://127.0.0.1:9090"}, true)
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() { _ = server.Close() }()

	require.NoError(t, client.Load(context.Background(), wms.Shipment{
		Driver: wms.Driver{Name: "Ivan Petrov", Phone: "+7-900"},
	}, []tares.Tare{{
		Sticker:   tares.NewSticker("MPB-200"),
		OfficeID:  11,
		CountryID: 643,
		Timestamp: time.Now().UTC(),
	}}))

	events := server.Events()
	require.Len(t, events, 1)
	require.Equal(t, "load", events[0].Kind)
}
