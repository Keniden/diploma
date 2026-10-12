package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	cargo_outbox "last-mile-task/internal/platform/outbox/cargo"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
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
	return nil, errors.New("not expected in handler tests")
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

type fakeRoutePointStore struct {
	byID map[values.RoutePointID]dto.RoutePoint
}

func (f *fakeRoutePointStore) RoutePointByID(
	ctx context.Context,
	routePointID values.RoutePointID,
) (dto.RoutePoint, error) {
	rp, ok := f.byID[routePointID]
	if !ok {
		return dto.RoutePoint{}, errors.New("route point not found")
	}

	return rp, nil
}

func (f *fakeRoutePointStore) PointUUID(
	ctx context.Context,
	pointID values.PointID,
) (uuid.UUID, error) {
	return uuid.MustParse("11111111-2222-3333-4444-555555555555"), nil
}

type fakeStore struct {
	destRP dto.RoutePoint
}

func (f *fakeStore) ShipmentByID(ctx context.Context, shipmentID values.ShipmentID) (dto.Shipment, error) {
	return dto.Shipment{ID: shipmentID, DriverID: values.DriverID(uuid.New())}, nil
}

func (f *fakeStore) PackageDestinationRoutePoint(
	ctx context.Context,
	packageID values.PackageID,
) (dto.RoutePoint, error) {
	return f.destRP, nil
}

type fakeOffices struct{}

func (fakeOffices) GetOfficeByPointID(ctx context.Context, pointID int64) (warehouse.Office, error) {
	return warehouse.Office{ID: int32(pointID), CountryID: 643}, nil
}

type fakeUsers struct{}

func (fakeUsers) UserByID(ctx context.Context, id values.DriverID) (*warehouse.User, error) {
	return &warehouse.User{FirstName: "Ivan", LastName: "Petrov", Phone: "+7-900"}, nil
}

func newHandler(q *fakeQuerier) *handlers.PackageEventHandler {
	routePoints := &fakeRoutePointStore{byID: map[values.RoutePointID]dto.RoutePoint{
		values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")): {PointID: 100},
		values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")): {PointID: 200},
	}}

	warehousePublisher := warehouse.NewPublisher(
		&fakeStore{destRP: dto.RoutePoint{PointID: 300}},
		wms_outbox.NewWriter(q),
		fakeOffices{},
		fakeUsers{},
	)

	return handlers.NewPackageEventHandler(
		routePoints,
		warehousePublisher,
		cargo.NewPublisher(q),
	)
}

func messageTypes(t *testing.T, q *fakeQuerier) map[string]int {
	t.Helper()

	types := make(map[string]int)

	for _, call := range q.execs {
		if !strings.Contains(call.sql, "INSERT INTO lmt.outbox") {
			continue
		}

		types[call.args[1].(string)]++
	}

	return types
}

func TestPackageScannedEventProducesCargoAndWMSMessages(t *testing.T) {
	q := &fakeQuerier{}
	h := newHandler(q)

	shipmentID := values.ShipmentID(uuid.New())
	rpLoad := values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"))
	rpUnload := values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002"))
	scannedAt := time.Now().UTC().Truncate(time.Microsecond)

	event := events.PackageScannedEvent{
		Packages: []events.ScannedPackage{
			// Load at the first route point -> wms.tares.load only.
			{
				PackageID:    values.PackageID(uuid.New()),
				TareID:       11,
				TareType:     "MPB",
				TareSticker:  "MPB-L",
				RoutePointID: rpLoad,
				PointID:      100,
				ShipmentID:   shipmentID,
				ActionType:   values.RoutePointActionTypeLoad,
				Timestamp:    scannedAt,
			},
			// Unload at the second route point -> wms.tares.unload + cargo.packages.scanned.
			{
				PackageID:    values.PackageID(uuid.New()),
				TareID:       12,
				TareType:     "MPB",
				TareSticker:  "MPB-U",
				RoutePointID: rpUnload,
				PointID:      200,
				ShipmentID:   shipmentID,
				ActionType:   values.RoutePointActionTypeUnload,
				Timestamp:    scannedAt,
			},
		},
	}

	if err := h.HandlePackageScannedEvent(context.Background(), event); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	types := messageTypes(t, q)
	if types["wms.tares.load"] != 1 {
		t.Fatalf("expected 1 wms.tares.load message, got %d", types["wms.tares.load"])
	}

	if types["wms.tares.unload"] != 1 {
		t.Fatalf("expected 1 wms.tares.unload message, got %d", types["wms.tares.unload"])
	}

	if types["cargo.packages.scanned"] != 1 {
		t.Fatalf("expected 1 cargo.packages.scanned message, got %d", types["cargo.packages.scanned"])
	}

	if types["cargo.packages.status.updated"] != 0 {
		t.Fatalf("unexpected cargo.packages.status.updated messages: %d", types["cargo.packages.status.updated"])
	}

	// Cargo payload: only the unloaded package, pointed at the route point uuid.
	var scanned cargo_outbox.ScannedPackagesMessage
	for _, call := range q.execs {
		if call.args[1].(string) != "cargo.packages.scanned" {
			continue
		}

		if err := json.Unmarshal(call.args[2].(outbox.Payload), &scanned); err != nil {
			t.Fatalf("cargo payload is not JSON: %v", err)
		}
	}

	if len(scanned.Packages) != 1 {
		t.Fatalf("expected 1 scanned package, got %d", len(scanned.Packages))
	}

	pkg := scanned.Packages[0]
	if pkg.Action != cargo_outbox.UpdatePackagesStatusActionTypeUnload {
		t.Fatalf("unexpected action: %q", pkg.Action)
	}

	if pkg.CurrentPointUUID.String() != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("unexpected current point uuid: %v", pkg.CurrentPointUUID)
	}

	if !pkg.Dt.Equal(scannedAt) {
		t.Fatalf("unexpected dt: %v", pkg.Dt)
	}

	// WMS payload: the load tare carries the destination office id.
	for _, call := range q.execs {
		if call.args[1].(string) != "wms.tares.load" {
			continue
		}

		var loadMsg wms_outbox.Message
		if err := json.Unmarshal(call.args[2].(outbox.Payload), &loadMsg); err != nil {
			t.Fatalf("wms payload is not JSON: %v", err)
		}

		if len(loadMsg.Tares) != 1 {
			t.Fatalf("expected 1 load tare, got %d", len(loadMsg.Tares))
		}

		if loadMsg.Tares[0].DestinationOfficeID != 300 {
			t.Fatalf("unexpected destination office: %d", loadMsg.Tares[0].DestinationOfficeID)
		}
	}
}

func TestPackageScannedEventGroupsPackagesByShipmentAndRoutePoint(t *testing.T) {
	q := &fakeQuerier{}
	h := newHandler(q)

	shipmentA := values.ShipmentID(uuid.New())
	shipmentB := values.ShipmentID(uuid.New())
	rp := values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"))

	event := events.PackageScannedEvent{
		Packages: []events.ScannedPackage{
			{PackageID: values.PackageID(uuid.New()), ShipmentID: shipmentA, RoutePointID: rp, ActionType: values.RoutePointActionTypeUnload},
			{PackageID: values.PackageID(uuid.New()), ShipmentID: shipmentA, RoutePointID: rp, ActionType: values.RoutePointActionTypeUnload},
			{PackageID: values.PackageID(uuid.New()), ShipmentID: shipmentB, RoutePointID: rp, ActionType: values.RoutePointActionTypeUnload},
		},
	}

	if err := h.HandlePackageScannedEvent(context.Background(), event); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	types := messageTypes(t, q)
	// Two (shipment, route point) groups -> two wms unload messages.
	if types["wms.tares.unload"] != 2 {
		t.Fatalf("expected 2 wms.tares.unload messages, got %d", types["wms.tares.unload"])
	}

	// Unload packages are cargo-scanned per group as well.
	if types["cargo.packages.scanned"] != 2 {
		t.Fatalf("expected 2 cargo.packages.scanned messages, got %d", types["cargo.packages.scanned"])
	}
}

func TestPackageInTransitEventProducesCargoStatusMessage(t *testing.T) {
	q := &fakeQuerier{}
	h := newHandler(q)

	rp := values.RoutePointID(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"))
	transitionedAt := time.Now().UTC().Truncate(time.Microsecond)

	event := events.PackageInTransitEvent{
		RoutePointID:   rp,
		PackageIDs:     []values.PackageID{values.PackageID(uuid.New())},
		TransitionedAt: transitionedAt,
	}

	if err := h.HandlePackageInTransitEvent(context.Background(), event); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	types := messageTypes(t, q)
	if types["cargo.packages.status.updated"] != 1 {
		t.Fatalf("expected 1 cargo.packages.status.updated message, got %d", types["cargo.packages.status.updated"])
	}

	if types["wms.tares.load"] != 0 || types["wms.tares.unload"] != 0 {
		t.Fatalf("in transit must not touch wms messages: %v", types)
	}

	var msg cargo_outbox.PackagesStatusMessage
	for _, call := range q.execs {
		if call.args[1].(string) != "cargo.packages.status.updated" {
			continue
		}

		if err := json.Unmarshal(call.args[2].(outbox.Payload), &msg); err != nil {
			t.Fatalf("cargo payload is not JSON: %v", err)
		}
	}

	if len(msg.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(msg.Packages))
	}

	if msg.Packages[0].Action != cargo_outbox.UpdatePackagesStatusActionTypeDepart {
		t.Fatalf("unexpected action: %q", msg.Packages[0].Action)
	}

	if !msg.Packages[0].Dt.Equal(transitionedAt) {
		t.Fatalf("unexpected dt: %v", msg.Packages[0].Dt)
	}
}

func TestPackageInTransitEventSkipsEmptyPackageIDs(t *testing.T) {
	q := &fakeQuerier{}
	h := newHandler(q)

	if err := h.HandlePackageInTransitEvent(context.Background(), events.PackageInTransitEvent{}); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(q.execs) != 0 {
		t.Fatalf("expected no outbox writes, got %d", len(q.execs))
	}
}
