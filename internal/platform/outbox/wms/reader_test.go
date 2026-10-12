package wms_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
	"last-mile-task/pkg/wms/wmsmock"
)

// fakeRows supports a single outbox.Query result message.
type fakeRows struct {
	msg   *outbox.Message
	scans int
}

func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.msg.ID
	*(dest[1].(*string)) = r.msg.Type
	*(dest[2].(*outbox.Payload)) = r.msg.Payload
	*(dest[3].(*outbox.Metadata)) = r.msg.Metadata
	*(dest[4].(*int)) = 0
	*(dest[5].(*time.Time)) = time.Now().UTC().Add(-time.Hour)
	*(dest[6].(*outbox.Status)) = outbox.MessageStatusTODO
	*(dest[7].(*string)) = ""

	return nil
}

func (r *fakeRows) Next() bool {
	r.scans++

	return r.scans <= 1
}

func (r *fakeRows) Close() {}

func (r *fakeRows) Err() error { return nil }

type readerQuerier struct {
	mu      sync.Mutex
	message *outbox.Message
	execs   []execCall
}

func (f *readerQuerier) Exec(ctx context.Context, sql string, arguments ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.execs = append(f.execs, execCall{sql: sql, args: arguments})

	return nil
}

func (f *readerQuerier) Query(ctx context.Context, sql string, args ...any) (outbox.Rows, error) {
	return &fakeRows{msg: f.message}, nil
}

func (f *readerQuerier) execCalls() []execCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := make([]execCall, len(f.execs))
	copy(calls, f.execs)

	return calls
}

func (f *readerQuerier) BeginTx(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (f *readerQuerier) Commit(ctx context.Context) error {
	return nil
}

func (f *readerQuerier) Rollback(ctx context.Context) error {
	return nil
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

func newWMSClient(t *testing.T, server *wmsmock.Server) *wms.Client {
	t.Helper()

	client, err := wms.New(wms.Config{
		Endpoints: []wms.Endpoint{
			{Country: "russia", Enabled: true, URI: server.URL(), Auth: wms.Auth{Username: "u", Password: "p"}},
		},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("wms client: %v", err)
	}

	return client
}

func wmsMessagePayload(t *testing.T, msg wms_outbox.Message) outbox.Payload {
	t.Helper()

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return data
}

func TestUnloadReaderExportsTaresToWMS(t *testing.T) {
	server := wmsmock.New()
	defer func() { _ = server.Close() }()

	payload := wmsMessagePayload(t, wms_outbox.Message{
		Shipment: wms.Shipment{Driver: wms.Driver{Name: "Ivan Petrov", Phone: "+7-900"}},
		Tares: []tares.Tare{{
			Sticker:   tares.NewSticker("MPB-200"),
			OfficeID:  11,
			CountryID: 643,
			Timestamp: time.Now().UTC().Add(-time.Minute),
		}},
	})

	q := &readerQuerier{
		message: outbox.NewMessage(
			payload,
			outbox.MessageWithID(uuid.New()),
			outbox.MessageWithType(wms_outbox.MessageTypeTaresUnload),
			outbox.MessageWithScheduledAt(time.Now().UTC().Add(-time.Hour)),
		),
	}

	reader := wms_outbox.NewUnloadReader(q, newWMSClient(t, server))
	reader.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func() { _ = reader.Stop(ctx) }()

	waitFor(t, time.Second, func() bool {
		return len(server.Events()) == 1
	}, "wms mock must receive the unload request")

	event := server.Events()[0]
	if event.Kind != "unload" {
		t.Fatalf("expected unload request, got %q", event.Kind)
	}

	var tares []wms.UnloadedTare
	if err := json.Unmarshal(event.Body, &tares); err != nil {
		t.Fatalf("unmarshal tares: %v", err)
	}

	if len(tares) != 1 {
		t.Fatalf("expected 1 tare, got %d", len(tares))
	}

	if tares[0].Sticker != "MPB-200" || tares[0].SourceOfficeID != 11 || tares[0].CountryID != 643 {
		t.Fatalf("unexpected tare: %+v", tares[0])
	}

	// At-least-once: successful export completes the row.
	waitFor(t, time.Second, func() bool {
		for _, call := range q.execCalls() {
			if strings.Contains(call.sql, "SET status = $1") {
				return true
			}
		}

		return false
	}, "message must be completed in the outbox")
}

func TestLoadReaderExportsTaresToWMS(t *testing.T) {
	server := wmsmock.New()
	defer func() { _ = server.Close() }()

	payload := wmsMessagePayload(t, wms_outbox.Message{
		Shipment: wms.Shipment{Driver: wms.Driver{Name: "Ivan Petrov", Phone: "+7-900"}},
		Tares: []tares.Tare{{
			Sticker:             tares.NewSticker("MPB-201"),
			OfficeID:            11,
			CountryID:           643,
			Timestamp:           time.Now().UTC().Add(-time.Minute),
			DestinationOfficeID: 22,
		}},
	})

	q := &readerQuerier{
		message: outbox.NewMessage(
			payload,
			outbox.MessageWithID(uuid.New()),
			outbox.MessageWithType(wms_outbox.MessageTypeTaresLoad),
			outbox.MessageWithScheduledAt(time.Now().UTC().Add(-time.Hour)),
		),
	}

	reader := wms_outbox.NewLoadReader(q, newWMSClient(t, server))
	reader.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func() { _ = reader.Stop(ctx) }()

	waitFor(t, time.Second, func() bool {
		return len(server.Events()) == 1
	}, "wms mock must receive the load request")

	event := server.Events()[0]
	if event.Kind != "load" {
		t.Fatalf("expected load request, got %q", event.Kind)
	}

	var tares []wms.LoadedTare
	if err := json.Unmarshal(event.Body, &tares); err != nil {
		t.Fatalf("unmarshal tares: %v", err)
	}

	if len(tares) != 1 {
		t.Fatalf("expected 1 tare, got %d", len(tares))
	}

	if tares[0].DriverName != "Ivan Petrov" || tares[0].DriverPhone != "+7-900" {
		t.Fatalf("driver data must be exported: %+v", tares[0])
	}

	if tares[0].Sticker != "MPB-201" || tares[0].SourceOfficeID != 11 || tares[0].CountryID != 643 {
		t.Fatalf("unexpected tare: %+v", tares[0])
	}
}
