package cargo_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	cargo_outbox "last-mile-task/internal/platform/outbox/cargo"
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

func TestWriteScannedPackagesPersistsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	w := cargo_outbox.NewWriter(q)

	scannedAt := time.Now().UTC().Truncate(time.Microsecond)
	pointUUID := uuid.New()
	pkgID := uuid.New()

	msg := cargo_outbox.ScannedPackagesMessage{
		Packages: []cargo_outbox.UpdatePackagesStatusPackage{
			{
				ID:               pkgID,
				Action:           cargo_outbox.UpdatePackagesStatusActionTypeUnload,
				CurrentPointUUID: pointUUID,
				Dt:               scannedAt,
			},
		},
	}

	if err := w.WriteScannedPackages(context.Background(), msg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if len(q.execs) != 1 {
		t.Fatalf("expected exactly one write, got %d", len(q.execs))
	}

	call := lastExec(t, q)
	assertInsert(t, call)

	if id := call.args[0].(uuid.UUID); id == uuid.Nil {
		t.Fatal("message id must not be nil")
	}

	if msgType := call.args[1].(string); msgType != "cargo.packages.scanned" {
		t.Fatalf("expected type cargo.packages.scanned, got %q", msgType)
	}

	// Payload must be {"packages":[...]} with the exact cargo-core wire shape.
	var got cargo_outbox.ScannedPackagesMessage
	if err := json.Unmarshal(call.args[2].(outbox.Payload), &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}

	if len(got.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(got.Packages))
	}

	pkg := got.Packages[0]
	if pkg.ID != pkgID {
		t.Fatalf("unexpected package id: %v", pkg.ID)
	}

	if pkg.Action != cargo_outbox.UpdatePackagesStatusActionTypeUnload {
		t.Fatalf("unexpected action: %q", pkg.Action)
	}

	if pkg.CurrentPointUUID != pointUUID {
		t.Fatalf("unexpected current point uuid: %v", pkg.CurrentPointUUID)
	}

	if !pkg.Dt.Equal(scannedAt) {
		t.Fatalf("unexpected dt: %v", pkg.Dt)
	}
}

func TestWritePackagesStatusPersistsOutboxMessage(t *testing.T) {
	q := &fakeQuerier{}
	w := cargo_outbox.NewWriter(q)

	if err := w.WritePackagesStatus(context.Background(), cargo_outbox.PackagesStatusMessage{
		Packages: []cargo_outbox.UpdatePackagesStatusPackage{
			{ID: uuid.New(), Action: cargo_outbox.UpdatePackagesStatusActionTypeDepart},
		},
	}); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	call := lastExec(t, q)
	assertInsert(t, call)

	if msgType := call.args[1].(string); msgType != "cargo.packages.status.updated" {
		t.Fatalf("expected type cargo.packages.status.updated, got %q", msgType)
	}
}

func TestWriteReturnsErrorOnOutboxFailure(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	w := cargo_outbox.NewWriter(q)

	err := w.WriteScannedPackages(context.Background(), cargo_outbox.ScannedPackagesMessage{})
	if err == nil {
		t.Fatal("write must fail when the outbox write fails")
	}
}
