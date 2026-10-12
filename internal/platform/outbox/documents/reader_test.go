package documents_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	docs_outbox "last-mile-task/internal/platform/outbox/documents"
	"last-mile-task/pkg/outbox"
)

type fakeClient struct {
	mu     sync.Mutex
	err    error
	events []cargodocuments.Event
}

func (f *fakeClient) CreateEventByType(_ context.Context, event cargodocuments.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.events = append(f.events, event)
	return f.err
}

func (f *fakeClient) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func (f *fakeClient) call(i int) cargodocuments.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.events[i]
}

type fakeRows struct {
	payload outbox.Payload
	scans   int
}

func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = uuid.New()
	*(dest[1].(*string)) = docs_outbox.MessageTypePublishAfterStartShipment
	*(dest[2].(*outbox.Payload)) = r.payload
	*(dest[3].(*outbox.Metadata)) = outbox.NewMetadata()
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

func (r *fakeRows) Close()     {}
func (r *fakeRows) Err() error { return nil }

type readerQuerier struct {
	mu      sync.Mutex
	event   *cargodocuments.Event
	payload outbox.Payload
	execs   []string
}

func (r *readerQuerier) Exec(_ context.Context, sql string, _ ...any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.execs = append(r.execs, sql)
	return nil
}

func (r *readerQuerier) completed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sql := range r.execs {
		if strings.Contains(sql, "SET status = $1, attempt = attempt + 1") {
			return true
		}
	}
	return false
}

func (r *readerQuerier) scheduled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sql := range r.execs {
		if strings.Contains(sql, "scheduled_at") {
			return true
		}
	}
	return false
}

func (r *readerQuerier) Query(context.Context, string, ...any) (outbox.Rows, error) {
	if r.payload != nil {
		return &fakeRows{payload: r.payload}, nil
	}

	data, _ := json.Marshal(r.event)
	return &fakeRows{payload: data}, nil
}

func (r *readerQuerier) BeginTx(context.Context) (context.Context, error) { return context.Background(), nil }
func (r *readerQuerier) Commit(context.Context) error                      { return nil }
func (r *readerQuerier) Rollback(context.Context) error                    { return nil }

func stopReader(t *testing.T, reader *outbox.Reader) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := reader.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
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

func TestReaderDeliversEventToClient(t *testing.T) {
	event := cargodocuments.Event{
		EventUUID:    "evt-1",
		ShipmentID:   uuid.New(),
		DocumentType: 1,
		PackageIDs:   []uuid.UUID{uuid.New()},
	}

	client := &fakeClient{}
	q := &readerQuerier{event: &event}

	reader := docs_outbox.NewReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.count() == 1 },
		"client must receive the event")

	if got := client.call(0).EventUUID; got != "evt-1" {
		t.Fatalf("unexpected event uuid: %q", got)
	}

	waitFor(t, time.Second, q.completed, "message must be completed in the outbox")
}

func TestReaderReschedulesOnClientError(t *testing.T) {
	event := cargodocuments.Event{EventUUID: "evt-2", ShipmentID: uuid.New()}
	client := &fakeClient{err: errors.New("cargo-documents down")}
	q := &readerQuerier{event: &event}

	reader := docs_outbox.NewReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.count() == 1 },
		"client must be called once")

	waitFor(t, time.Second, q.scheduled, "message must be scheduled for retry")

	waitFor(t, time.Second, func() bool {
		select {
		case <-reader.Errors():
			return true
		default:
			return false
		}
	}, "handler error must be reported on Errors()")
}

func TestReaderFailsOnInvalidPayload(t *testing.T) {
	client := &fakeClient{}
	q := &readerQuerier{payload: outbox.Payload("{not json")}

	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(q, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var event cargodocuments.Event
			if err := json.Unmarshal(message.Payload, &event); err != nil {
				return err
			}
			return client.CreateEventByType(ctx, event)
		}),
		outbox.ReaderWithMessageType(docs_outbox.MessageTypePublishAfterStartShipment),
	)
	reader.Start()
	defer stopReader(t, reader)

	time.Sleep(100 * time.Millisecond)
	if client.count() != 0 {
		t.Fatalf("client must not be called for invalid payload")
	}

	waitFor(t, time.Second, q.scheduled, "invalid payload must be scheduled for retry")
}