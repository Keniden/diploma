package cargo_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	cargo_outbox "last-mile-task/internal/platform/outbox/cargo"
	"last-mile-task/pkg/outbox"
)

// fakeCargoClient records calls and simulates failures.
type fakeCargoClient struct {
	mu        sync.Mutex
	updateErr error
	calls     []cargo_outbox.UpdatePackagesStatusPackage
}

func (f *fakeCargoClient) UpdatePackagesStatusByAction(
	ctx context.Context,
	packages []cargo_outbox.UpdatePackagesStatusPackage,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, packages...)

	return f.updateErr
}

func (f *fakeCargoClient) callsCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func (f *fakeCargoClient) call(i int) cargo_outbox.UpdatePackagesStatusPackage {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[i]
}

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

// readerQuerier serves the configured outbox row and records Exec calls so
// the at-least-once mechanics (complete/schedule) can be asserted.
type readerQuerier struct {
	mu      sync.Mutex
	message *outbox.Message
	execErr error
	execs   []execCall
}

func (f *readerQuerier) Exec(ctx context.Context, sql string, arguments ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.execs = append(f.execs, execCall{sql: sql, args: arguments})

	return f.execErr
}

func (f *readerQuerier) execCalls() []execCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := make([]execCall, len(f.execs))
	copy(calls, f.execs)

	return calls
}

func (f *readerQuerier) Query(ctx context.Context, sql string, args ...any) (outbox.Rows, error) {
	return &fakeRows{msg: f.message}, nil
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

func mustPayload(t *testing.T, msg cargo_outbox.ScannedPackagesMessage) outbox.Payload {
	t.Helper()

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return data
}

func pastMessage(t *testing.T, payload outbox.Payload, messageType string) *outbox.Message {
	t.Helper()

	return outbox.NewMessage(
		payload,
		outbox.MessageWithID(uuid.New()),
		outbox.MessageWithType(messageType),
		outbox.MessageWithScheduledAt(time.Now().UTC().Add(-time.Hour)),
	)
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

func stopReader(t *testing.T, reader *outbox.Reader) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := reader.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

func TestScannedPackagesReaderDeliversToClient(t *testing.T) {
	client := &fakeCargoClient{}
	q := &readerQuerier{
		message: pastMessage(t, mustPayload(t, cargo_outbox.ScannedPackagesMessage{
			Packages: []cargo_outbox.UpdatePackagesStatusPackage{
				{ID: uuid.New(), Action: cargo_outbox.UpdatePackagesStatusActionTypeUnload},
			},
		}), cargo_outbox.MessageTypeScannedPackages),
	}

	reader := cargo_outbox.NewScannedPackagesReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.callsCount() == 1 },
		"client must receive the package")

	if action := client.call(0).Action; action != cargo_outbox.UpdatePackagesStatusActionTypeUnload {
		t.Fatalf("unexpected action: %q", action)
	}

	// At-least-once: successful handling completes the row.
	waitFor(t, time.Second, func() bool {
		for _, call := range q.execCalls() {
			if strings.Contains(call.sql, "SET status = $1") {
				return true
			}
		}

		return false
	}, "message must be completed in the outbox")
}

func TestScannedPackagesReaderRetriesOnClientError(t *testing.T) {
	client := &fakeCargoClient{updateErr: errors.New("cargo-core down")}
	q := &readerQuerier{
		message: pastMessage(t, mustPayload(t, cargo_outbox.ScannedPackagesMessage{
			Packages: []cargo_outbox.UpdatePackagesStatusPackage{{ID: uuid.New()}},
		}), cargo_outbox.MessageTypeScannedPackages),
	}

	reader := cargo_outbox.NewScannedPackagesReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.callsCount() == 1 },
		"client must be called once")

	// At-least-once: a handler error reschedules the message for a retry.
	waitFor(t, time.Second, func() bool {
		for _, call := range q.execCalls() {
			if strings.Contains(call.sql, "scheduled_at") {
				return true
			}
		}

		return false
	}, "message must be scheduled for retry")

	waitFor(t, time.Second, func() bool {
		select {
		case <-reader.Errors():
			return true
		default:
			return false
		}
	}, "handler error must be reported on Errors()")
}

func TestPackagesStatusReaderDecodesPayload(t *testing.T) {
	client := &fakeCargoClient{}
	payload, err := json.Marshal(cargo_outbox.PackagesStatusMessage{
		Packages: []cargo_outbox.UpdatePackagesStatusPackage{
			{ID: uuid.New(), Action: cargo_outbox.UpdatePackagesStatusActionTypeDepart},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	q := &readerQuerier{
		message: pastMessage(t, payload, cargo_outbox.MessageTypePackagesStatusUpdated),
	}

	reader := cargo_outbox.NewPackagesStatusReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.callsCount() == 1 },
		"client must receive the package")

	if action := client.call(0).Action; action != cargo_outbox.UpdatePackagesStatusActionTypeDepart {
		t.Fatalf("unexpected action: %q", action)
	}
}

func TestReaderFailsOnInvalidPayload(t *testing.T) {
	client := &fakeCargoClient{}
	q := &readerQuerier{
		message: pastMessage(t, outbox.Payload("{not json"), cargo_outbox.MessageTypeScannedPackages),
	}

	reader := cargo_outbox.NewScannedPackagesReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	time.Sleep(100 * time.Millisecond)

	if client.callsCount() != 0 {
		t.Fatalf("client must not be called for invalid payload")
	}

	// The malformed message must be rescheduled (retried), not completed.
	waitFor(t, time.Second, func() bool {
		for _, call := range q.execCalls() {
			if strings.Contains(call.sql, "scheduled_at") {
				return true
			}
		}

		return false
	}, "invalid payload must be scheduled for retry")
}
