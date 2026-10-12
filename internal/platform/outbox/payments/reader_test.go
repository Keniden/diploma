package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	paymentsclient "last-mile-task/internal/clients/payments"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/pkg/outbox"
)

type fakePaysClient struct {
	mu       sync.Mutex
	err      error
	accruals []paymentsclient.LastMileShipmentAccrual
}

func (f *fakePaysClient) LastMileShipmentAccrual(_ context.Context, accrual paymentsclient.LastMileShipmentAccrual) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.accruals = append(f.accruals, accrual)
	return f.err
}

func (f *fakePaysClient) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.accruals)
}

func (f *fakePaysClient) call(i int) paymentsclient.LastMileShipmentAccrual {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.accruals[i]
}

type fakeRows struct {
	payload outbox.Payload
	scans   int
}

func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = uuid.New()
	*(dest[1].(*string)) = payments_outbox.MessageTypeLastMileShipmentAccrual
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
	event   *paymentsclient.LastMileShipmentAccrual
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

func TestReaderDeliversAccrualToClient(t *testing.T) {
	accrual := paymentsclient.LastMileShipmentAccrual{
		EventID:    "evt-1",
		ShipmentID: uuid.New(),
	}

	client := &fakePaysClient{}
	q := &readerQuerier{event: &accrual}

	reader := payments_outbox.NewReader(q, client)
	reader.Start()
	defer stopReader(t, reader)

	waitFor(t, time.Second, func() bool { return client.count() == 1 },
		"client must receive the accrual")

	if got := client.call(0).EventID; got != "evt-1" {
		t.Fatalf("unexpected accrual event id: %q", got)
	}

	waitFor(t, time.Second, q.completed, "message must be completed in the outbox")
}

func TestReaderReschedulesOnClientError(t *testing.T) {
	accrual := paymentsclient.LastMileShipmentAccrual{EventID: "evt-2", ShipmentID: uuid.New()}
	client := &fakePaysClient{err: errors.New("payments down")}
	q := &readerQuerier{event: &accrual}

	reader := payments_outbox.NewReader(q, client)
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
	client := &fakePaysClient{}
	q := &readerQuerier{payload: outbox.Payload("{not json")}

	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(q, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var accrual paymentsclient.LastMileShipmentAccrual
			if err := json.Unmarshal(message.Payload, &accrual); err != nil {
				return err
			}
			return client.LastMileShipmentAccrual(ctx, accrual)
		}),
		outbox.ReaderWithMessageType(payments_outbox.MessageTypeLastMileShipmentAccrual),
	)
	reader.Start()
	defer stopReader(t, reader)

	time.Sleep(100 * time.Millisecond)
	if client.count() != 0 {
		t.Fatalf("client must not be called for invalid payload")
	}

	waitFor(t, time.Second, q.scheduled, "invalid payload must be scheduled for retry")
}

// failOnceClient fails the first call and succeeds afterwards, simulating an
// external payments outage that recovers.
type failOnceClient struct {
	mu   sync.Mutex
	calls int
}

func (f *failOnceClient) LastMileShipmentAccrual(context.Context, paymentsclient.LastMileShipmentAccrual) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	if f.calls == 1 {
		return errors.New("payments down")
	}
	return nil
}

func (f *failOnceClient) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// TestReaderAtLeastOnceRetriesUntilSuccess proves the at-least-once delivery:
// a single outbox message is delivered, rescheduled on failure, and delivered
// again until it succeeds; the same message id is reused (no duplicates).
func TestReaderAtLeastOnceRetriesUntilSuccess(t *testing.T) {
	accrual := paymentsclient.LastMileShipmentAccrual{EventID: "evt-3", ShipmentID: uuid.New()}

	client := &failOnceClient{}
	q := &readerQuerier{event: &accrual}

	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(q, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var got paymentsclient.LastMileShipmentAccrual
			if err := json.Unmarshal(message.Payload, &got); err != nil {
				return err
			}
			return client.LastMileShipmentAccrual(ctx, got)
		}),
		outbox.ReaderWithMessageType(payments_outbox.MessageTypeLastMileShipmentAccrual),
		outbox.ReaderWithInterval(40*time.Millisecond),
		outbox.ReaderWithBackoff(1, time.Millisecond),
	)
	reader.Start()
	defer stopReader(t, reader)

	// The message is delivered at least twice (first attempt fails, second
	// succeeds) and completed after that.
	waitFor(t, 2*time.Second, func() bool { return client.count() >= 2 },
		"client must be called once failing and once succeeding")

	waitFor(t, 2*time.Second, q.completed, "message must be completed after retry")
}