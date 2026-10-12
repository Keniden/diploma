package wms_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/pkg/outbox"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
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

func (f *fakeQuerier) Query(ctx context.Context, sql string, args ...any) (outbox.Rows, error) {
	return nil, errors.New("not expected in writer tests")
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

func TestLoadTaresPersistsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	w := wms_outbox.NewWriter(q)

	shipment := wms.Shipment{Driver: wms.Driver{Name: "Ivan Petrov", Phone: "+7-900"}}
	tares := []tares.Tare{{
		Sticker:             tares.NewSticker("MPB-100"),
		OfficeID:            11,
		CountryID:           643,
		Timestamp:           time.Now().UTC().Truncate(time.Microsecond),
		DestinationOfficeID: 22,
	}}

	if err := w.LoadTares(context.Background(), shipment, tares); err != nil {
		t.Fatalf("load tares failed: %v", err)
	}

	if len(q.execs) != 1 {
		t.Fatalf("expected exactly one write, got %d", len(q.execs))
	}

	call := lastExec(t, q)
	assertInsert(t, call)

	if id := call.args[0].(uuid.UUID); id == uuid.Nil {
		t.Fatal("message id must not be nil")
	}

	if msgType := call.args[1].(string); msgType != "wms.tares.load" {
		t.Fatalf("expected type wms.tares.load, got %q", msgType)
	}

	// Payload must be {"shipment":{...},"tares":[...]}.
	var got wms_outbox.Message
	if err := json.Unmarshal(call.args[2].(outbox.Payload), &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}

	if got.Shipment.Driver.Name != "Ivan Petrov" || got.Shipment.Driver.Phone != "+7-900" {
		t.Fatalf("unexpected shipment: %+v", got.Shipment)
	}

	if len(got.Tares) != 1 {
		t.Fatalf("expected 1 tare, got %d", len(got.Tares))
	}

	tare := got.Tares[0]
	if tare.Sticker.String() != "MPB-100" {
		t.Fatalf("unexpected sticker: %q", tare.Sticker.String())
	}

	if tare.OfficeID != 11 || tare.CountryID != 643 || tare.DestinationOfficeID != 22 {
		t.Fatalf("unexpected tare: %+v", tare)
	}
}

func TestUnloadTaresPersistsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	w := wms_outbox.NewWriter(q)

	if err := w.UnloadTares(context.Background(),
		wms.Shipment{Driver: wms.Driver{Name: "Ivan Petrov"}},
		[]tares.Tare{{Sticker: tares.NewSticker("MPB-101"), OfficeID: 11, CountryID: 643}},
	); err != nil {
		t.Fatalf("unload tares failed: %v", err)
	}

	if msgType := lastExec(t, q).args[1].(string); msgType != "wms.tares.unload" {
		t.Fatalf("expected type wms.tares.unload, got %q", msgType)
	}
}

func TestWriteReturnsErrorOnOutboxFailure(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	w := wms_outbox.NewWriter(q)

	err := w.LoadTares(context.Background(), wms.Shipment{}, nil)
	if err == nil {
		t.Fatal("write must fail when the outbox write fails")
	}
}
