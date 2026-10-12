package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	paymentsclient "last-mile-task/internal/clients/payments"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/pkg/outbox"
)

type fakeQuerier struct {
	execErr error
	execs   []execCall
}

type execCall struct {
	sql  string
	args []any
}

func (f *fakeQuerier) Exec(ctx context.Context, sql string, arguments ...any) error {
	f.execs = append(f.execs, execCall{sql: sql, args: arguments})
	return f.execErr
}

func (f *fakeQuerier) Query(context.Context, string, ...any) (outbox.Rows, error) {
	return nil, errors.New("not expected in writer tests")
}

func (f *fakeQuerier) BeginTx(context.Context) (context.Context, error) { return context.Background(), nil }
func (f *fakeQuerier) Commit(context.Context) error                      { return nil }
func (f *fakeQuerier) Rollback(context.Context) error                    { return nil }

func lastExec(t *testing.T, f *fakeQuerier) execCall {
	t.Helper()
	if len(f.execs) == 0 {
		t.Fatal("expected at least one Exec call")
	}
	return f.execs[len(f.execs)-1]
}

func TestWriteLastMileShipmentAccrualPersistsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	w := payments_outbox.NewWriter(q)

	accrual := paymentsclient.LastMileShipmentAccrual{
		EventID:    "evt-1",
		ShipmentID: uuid.New(),
	}

	if err := w.WriteLastMileShipmentAccrual(context.Background(), accrual); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if len(q.execs) != 1 {
		t.Fatalf("expected exactly one write, got %d", len(q.execs))
	}

	call := lastExec(t, q)
	if !strings.Contains(call.sql, "INSERT INTO lmt.outbox") {
		t.Fatalf("expected insert into lmt.outbox, got %q", call.sql)
	}
	if len(call.args) != 8 {
		t.Fatalf("expected 8 insert args, got %d", len(call.args))
	}
	if id := call.args[0].(uuid.UUID); id == uuid.Nil {
		t.Fatal("message id must not be nil")
	}
	if msgType := call.args[1].(string); msgType != payments_outbox.MessageTypeLastMileShipmentAccrual {
		t.Fatalf("expected type %q, got %q", payments_outbox.MessageTypeLastMileShipmentAccrual, msgType)
	}

	var got paymentsclient.LastMileShipmentAccrual
	if err := json.Unmarshal(call.args[2].(outbox.Payload), &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if got.EventID != "evt-1" || got.ShipmentID != accrual.ShipmentID {
		t.Fatalf("unexpected accrual payload: %+v", got)
	}
}

func TestWriteReturnsErrorOnOutboxFailure(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	w := payments_outbox.NewWriter(q)

	err := w.WriteLastMileShipmentAccrual(context.Background(), paymentsclient.LastMileShipmentAccrual{})
	if err == nil {
		t.Fatal("write must fail when the outbox write fails")
	}
}