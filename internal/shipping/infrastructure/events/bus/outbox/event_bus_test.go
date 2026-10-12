package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/pkg/outbox"
)

// fakeQuerier is an in-memory outbox.Querier: Exec calls are recorded and
// Query returns a single preconfigured message exactly once per call set.
type fakeQuerier struct {
	mu      sync.Mutex
	execErr error
	execs   []execCall

	message  *outbox.Message
	queryErr error
}

type execCall struct {
	sql  string
	args []any
}

func (f *fakeQuerier) Exec(ctx context.Context, sql string, arguments ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.execs = append(f.execs, execCall{sql: sql, args: arguments})

	return f.execErr
}

func (f *fakeQuerier) execCalls() []execCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := make([]execCall, len(f.execs))
	copy(calls, f.execs)

	return calls
}

func (f *fakeQuerier) Query(ctx context.Context, sql string, args ...any) (outbox.Rows, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}

	return &fakeRows{msg: f.message}, nil
}

func (f *fakeQuerier) BeginTx(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (f *fakeQuerier) Commit(ctx context.Context) error {
	return nil
}

func (f *fakeQuerier) Rollback(ctx context.Context) error {
	return nil
}

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

func lastExec(t *testing.T, f *fakeQuerier) execCall {
	t.Helper()

	if len(f.execs) == 0 {
		t.Fatal("expected at least one Exec call")
	}

	return f.execs[len(f.execs)-1]
}

func assertInsert(t *testing.T, call execCall) {
	t.Helper()

	if !strings.Contains(call.sql, "INSERT INTO lmt.outbox") {
		t.Fatalf("expected insert into lmt.outbox, got %q", call.sql)
	}

	if len(call.args) != 8 {
		t.Fatalf("expected 8 insert args, got %d", len(call.args))
	}
}

func containsSQL(sql, part string) bool {
	return strings.Contains(sql, part)
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

func TestEventBusPublishesPackageScannedEventAsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	bus := New(q)

	event := events.PackageScannedEvent{
		Packages: []events.ScannedPackage{{TareSticker: "MPB-11", PackageID: uuid.New()}},
	}

	if err := bus.PublishPackageEvent(context.Background(), event); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execCalls()) != 1 {
		t.Fatalf("expected exactly one outbox write, got %d", len(q.execCalls()))
	}

	call := lastExec(t, q)
	assertInsert(t, call)

	id := call.args[0].(uuid.UUID)
	if id == uuid.Nil {
		t.Fatal("message id must not be nil")
	}

	msgType := call.args[1].(string)
	if msgType != "package.scanned" {
		t.Fatalf("expected message type package.scanned, got %q", msgType)
	}

	// Payload must be the JSON encoding of the domain event itself.
	payload := call.args[2].(outbox.Payload)
	var got events.PackageScannedEvent
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("payload is not JSON of the event: %v", err)
	}

	if len(got.Packages) != 1 || got.Packages[0].TareSticker != "MPB-11" {
		t.Fatalf("unexpected payload: %+v", got)
	}

	if call.args[4].(int) != 0 {
		t.Fatalf("attempt must start at 0, got %v", call.args[4])
	}

	if call.args[6].(outbox.Status) != outbox.MessageStatusTODO {
		t.Fatalf("status must be TODO, got %v", call.args[6])
	}
}

func TestEventBusPublishesShipmentEventAsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	bus := New(q)

	if err := bus.PublishShipmentEvent(context.Background(), events.ShipmentCreatedEvent{ShipmentID: uuid.New()}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if msgType := lastExec(t, q).args[1].(string); msgType != "shipment.created" {
		t.Fatalf("expected message type shipment.created, got %q", msgType)
	}
}

func TestEventBusConsumesPublishedMessageAtLeastOnce(t *testing.T) {
	// Preconfigure the "database" with the message the publish path would
	// have written, then run a subscription: the reader must pick it up by
	// type, hand the payload to the handler and complete the row. This is the
	// at-least-once delivery leg of outbox mechanics.
	payload := []byte(`{"packages":[{"TareSticker":"MPB-12"}]}`)
	msg := outbox.NewMessage(
		payload,
		outbox.MessageWithID(uuid.New()),
		outbox.MessageWithType("package.scanned"),
		outbox.MessageWithScheduledAt(time.Now().UTC().Add(-time.Hour)),
	)

	q := &fakeQuerier{message: msg}

	bus := New(q)

	handled := make(chan struct{}, 1)
	bus.SubscribePackageEvent(events.PackageScanned, func(_ context.Context, m *outbox.Message) error {
		if m.Type != "package.scanned" {
			t.Fatalf("unexpected message type %q", m.Type)
		}

		var got events.PackageScannedEvent
		if err := json.Unmarshal(m.Payload, &got); err != nil {
			t.Fatalf("payload is not JSON of the event: %v", err)
		}

		handled <- struct{}{}

		return nil
	})

	waitFor(t, time.Second, func() bool {
		select {
		case <-handled:
			return true
		default:
			return false
		}
	}, "subscription handler must be called")

	// The reader completes the processed row (UPDATE ... SET status = $1).
	waitFor(t, time.Second, func() bool {
		for _, call := range q.execCalls() {
			if containsSQL(call.sql, "UPDATE") && len(call.args) > 0 {
				if call.args[0] == outbox.MessageStatusCompleted {
					return true
				}
			}
		}

		return false
	}, "processed message must be completed in the outbox")

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestEventBusPublishReturnsErrorWhenWriteFails(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	bus := New(q)

	// A failed write surfaces to the caller: the surrounding domain
	// transaction is rolled back and the event can be published again,
	// which is what guarantees at-least-once delivery.
	if err := bus.PublishPackageEvent(context.Background(), events.PackageScannedEvent{}); err == nil {
		t.Fatal("publish must fail when the outbox write fails")
	}
}

func TestEventBusCloseIsIdempotentWithoutSubscriptions(t *testing.T) {
	bus := New(&fakeQuerier{})

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}
