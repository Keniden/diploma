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
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
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
	return nil, errors.New("not expected in publisher tests")
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

func TestPublishWritesOnlyUnloadedPackages(t *testing.T) {
	q := &fakeQuerier{}
	p := cargo.NewPublisher(q)

	pointUUID := uuid.New()
	loadID := uuid.New()
	unloadID1 := uuid.New()
	unloadID2 := uuid.New()

	routePoint := dto.RoutePoint{ID: values.RoutePointID(uuid.New()), PointID: 1, PointUUID: pointUUID}
	scannedAt := time.Now().UTC().Truncate(time.Microsecond)

	err := p.Publish(context.Background(), routePoint, []dto.Package{
		{ID: loadID, ActionType: values.RoutePointActionTypeLoad, ScannedAt: scannedAt},
		{ID: unloadID1, ActionType: values.RoutePointActionTypeUnload, ScannedAt: scannedAt},
		{ID: unloadID2, ActionType: values.RoutePointActionTypeUnload, ScannedAt: scannedAt},
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execs) != 1 {
		t.Fatalf("expected exactly one outbox write, got %d", len(q.execs))
	}

	call := q.execs[0]
	if !strings.Contains(call.sql, "INSERT INTO lmt.outbox") {
		t.Fatalf("expected insert, got %q", call.sql)
	}

	if msgType := call.args[1].(string); msgType != "cargo.packages.scanned" {
		t.Fatalf("expected cargo.packages.scanned, got %q", msgType)
	}

	var msg cargo_outbox.ScannedPackagesMessage
	if err := json.Unmarshal(call.args[2].(outbox.Payload), &msg); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}

	if len(msg.Packages) != 2 {
		t.Fatalf("expected 2 unloaded packages, got %d", len(msg.Packages))
	}

	for _, pkg := range msg.Packages {
		if pkg.Action != cargo_outbox.UpdatePackagesStatusActionTypeUnload {
			t.Fatalf("unexpected action: %q", pkg.Action)
		}

		if pkg.CurrentPointUUID != pointUUID {
			t.Fatalf("unexpected point uuid: %v", pkg.CurrentPointUUID)
		}

		if !pkg.Dt.Equal(scannedAt) {
			t.Fatalf("unexpected dt: %v", pkg.Dt)
		}
	}
}

func TestPublishSkipsWriteWithoutUnloadedPackages(t *testing.T) {
	q := &fakeQuerier{}
	p := cargo.NewPublisher(q)

	err := p.Publish(context.Background(), dto.RoutePoint{PointUUID: uuid.New()}, []dto.Package{
		{ID: uuid.New(), ActionType: values.RoutePointActionTypeLoad},
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execs) != 0 {
		t.Fatalf("expected no outbox writes, got %d", len(q.execs))
	}
}

func TestPublishInTransitStatusWritesDepartPackages(t *testing.T) {
	q := &fakeQuerier{}
	p := cargo.NewPublisher(q)

	pointUUID := uuid.New()
	pkgID := uuid.New()
	transitionedAt := time.Now().UTC().Truncate(time.Microsecond)

	err := p.PublishInTransitStatus(
		context.Background(),
		dto.RoutePoint{PointUUID: pointUUID},
		[]values.PackageID{pkgID},
		transitionedAt,
	)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execs) != 1 {
		t.Fatalf("expected exactly one outbox write, got %d", len(q.execs))
	}

	call := q.execs[0]
	if msgType := call.args[1].(string); msgType != "cargo.packages.status.updated" {
		t.Fatalf("expected cargo.packages.status.updated, got %q", msgType)
	}

	var msg cargo_outbox.PackagesStatusMessage
	if err := json.Unmarshal(call.args[2].(outbox.Payload), &msg); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}

	if len(msg.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(msg.Packages))
	}

	pkg := msg.Packages[0]
	if pkg.ID != pkgID {
		t.Fatalf("unexpected package id: %v", pkg.ID)
	}

	if pkg.Action != cargo_outbox.UpdatePackagesStatusActionTypeDepart {
		t.Fatalf("unexpected action: %q", pkg.Action)
	}

	if !pkg.Dt.Equal(transitionedAt) {
		t.Fatalf("unexpected dt: %v", pkg.Dt)
	}
}

func TestPublishReturnsErrorWhenWriteFails(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	p := cargo.NewPublisher(q)

	err := p.Publish(context.Background(), dto.RoutePoint{PointUUID: uuid.New()}, []dto.Package{
		{ID: uuid.New(), ActionType: values.RoutePointActionTypeUnload},
	})
	if err == nil {
		t.Fatal("publish must fail when the outbox write fails")
	}
}
