package warehouse_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/internal/shipping/domain/values"
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

type fakeStore struct {
	shipment dto.Shipment
	destRP   dto.RoutePoint

	shipmentErr error
	destRPErr   error
}

func (f *fakeStore) ShipmentByID(ctx context.Context, shipmentID values.ShipmentID) (dto.Shipment, error) {
	if f.shipmentErr != nil {
		return dto.Shipment{}, f.shipmentErr
	}

	f.shipment.ID = shipmentID

	return f.shipment, nil
}

func (f *fakeStore) PackageDestinationRoutePoint(
	ctx context.Context,
	packageID values.PackageID,
) (dto.RoutePoint, error) {
	if f.destRPErr != nil {
		return dto.RoutePoint{}, f.destRPErr
	}

	return f.destRP, nil
}

type fakeOffices struct {
	byPointID map[int64]warehouse.Office
	officeErr error
}

func (f *fakeOffices) GetOfficeByPointID(ctx context.Context, pointID int64) (warehouse.Office, error) {
	if f.officeErr != nil {
		return warehouse.Office{}, f.officeErr
	}

	office, ok := f.byPointID[pointID]
	if !ok {
		return warehouse.Office{}, errors.New("office not found")
	}

	return office, nil
}

type fakeUsers struct {
	user     *warehouse.User
	userErr  error
	driverID values.DriverID
}

func (f *fakeUsers) UserByID(ctx context.Context, id values.DriverID) (*warehouse.User, error) {
	if f.userErr != nil {
		return nil, f.userErr
	}

	f.driverID = id

	return f.user, nil
}

func TestPublishLoadAndUnloadTares(t *testing.T) {
	q := &fakeQuerier{}
	driverID := values.DriverID(uuid.New())

	store := &fakeStore{
		shipment: dto.Shipment{ID: uuid.New(), DriverID: driverID},
		destRP:   dto.RoutePoint{ID: values.RoutePointID(uuid.New()), PointID: 200},
	}
	offices := &fakeOffices{byPointID: map[int64]warehouse.Office{
		100: {ID: 11, CountryID: 643},
		200: {ID: 22, CountryID: 643},
	}}
	users := &fakeUsers{user: &warehouse.User{
		FirstName:  "Ivan",
		MiddleName: "Ivanovich",
		LastName:   "Petrov",
		Phone:      "+7-900",
	}}

	p := warehouse.NewPublisher(store, wms_outbox.NewWriter(q), offices, users)

	shipmentID := values.ShipmentID(uuid.New())
	loadID := values.PackageID(uuid.New())
	unloadID := values.PackageID(uuid.New())

	err := p.Publish(context.Background(), shipmentID, dto.RoutePoint{PointID: 100}, []dto.Package{
		{ID: loadID, TareSticker: "MPB-L1", ActionType: values.RoutePointActionTypeLoad},
		{ID: unloadID, TareSticker: "MPB-U1", ActionType: values.RoutePointActionTypeUnload},
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execs) != 2 {
		t.Fatalf("expected 2 outbox writes (load + unload), got %d", len(q.execs))
	}

	types := map[string]bool{}
	var loaded *wms_outbox.Message
	var unloaded *wms_outbox.Message

	for _, call := range q.execs {
		msgType := call.args[1].(string)
		types[msgType] = true

		var msg wms_outbox.Message
		if err := json.Unmarshal(call.args[2].(outbox.Payload), &msg); err != nil {
			t.Fatalf("payload is not JSON: %v", err)
		}

		switch msgType {
		case wms_outbox.MessageTypeTaresLoad:
			loaded = &msg
		case wms_outbox.MessageTypeTaresUnload:
			unloaded = &msg
		}
	}

	if !types[wms_outbox.MessageTypeTaresLoad] || !types[wms_outbox.MessageTypeTaresUnload] {
		t.Fatalf("expected both load and unload messages, got %v", types)
	}

	// Driver data comes from the users client.
	for name, msg := range map[string]*wms_outbox.Message{"load": loaded, "unload": unloaded} {
		if msg == nil {
			t.Fatalf("missing %s message", name)
		}

		if msg.Shipment.Driver.Name != "Ivan Ivanovich Petrov" {
			t.Fatalf("unexpected driver name: %q", msg.Shipment.Driver.Name)
		}

		if msg.Shipment.Driver.Phone != "+7-900" {
			t.Fatalf("unexpected driver phone: %q", msg.Shipment.Driver.Phone)
		}
	}

	if len(loaded.Tares) != 1 || loaded.Tares[0].Sticker.String() != "MPB-L1" {
		t.Fatalf("unexpected load tares: %+v", loaded.Tares)
	}

	tare := loaded.Tares[0]
	if tare.OfficeID != 11 || tare.CountryID != 643 || tare.DestinationOfficeID != 22 {
		t.Fatalf("unexpected load tare: %+v", tare)
	}

	if len(unloaded.Tares) != 1 || unloaded.Tares[0].Sticker.String() != "MPB-U1" {
		t.Fatalf("unexpected unload tares: %+v", unloaded.Tares)
	}

	if users.driverID != driverID {
		t.Fatalf("users client must receive the shipment driver id, got %v", users.driverID)
	}
}

func TestPublishSkipsTaresWithUnknownAction(t *testing.T) {
	q := &fakeQuerier{}
	store := &fakeStore{shipment: dto.Shipment{ID: uuid.New()}}
	offices := &fakeOffices{byPointID: map[int64]warehouse.Office{100: {ID: 11, CountryID: 643}}}
	users := &fakeUsers{user: &warehouse.User{FirstName: "Ivan"}}

	p := warehouse.NewPublisher(store, wms_outbox.NewWriter(q), offices, users)

	err := p.Publish(context.Background(), values.ShipmentID(uuid.New()), dto.RoutePoint{PointID: 100}, []dto.Package{
		{ID: uuid.New(), TareSticker: "MPB-X", ActionType: values.RoutePointActionTypeUnspecified},
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if len(q.execs) != 0 {
		t.Fatalf("expected no outbox writes, got %d", len(q.execs))
	}
}

func TestPublishFailsWhenDestinationRoutePointMissing(t *testing.T) {
	q := &fakeQuerier{}
	store := &fakeStore{
		shipment:  dto.Shipment{ID: uuid.New(), DriverID: values.DriverID(uuid.New())},
		destRPErr: errors.New("destination route point not found"),
	}
	offices := &fakeOffices{byPointID: map[int64]warehouse.Office{100: {ID: 11, CountryID: 643}}}
	users := &fakeUsers{user: &warehouse.User{FirstName: "Ivan"}}

	p := warehouse.NewPublisher(store, wms_outbox.NewWriter(q), offices, users)

	err := p.Publish(context.Background(), values.ShipmentID(uuid.New()), dto.RoutePoint{PointID: 100}, []dto.Package{
		{ID: uuid.New(), TareSticker: "MPB-L2", ActionType: values.RoutePointActionTypeLoad},
	})
	if err == nil {
		t.Fatal("publish must fail when the destination route point can't be resolved")
	}

	if len(q.execs) != 0 {
		t.Fatalf("no outbox write must happen on failure, got %d", len(q.execs))
	}
}

func TestPublishReturnsErrorWhenWriteFails(t *testing.T) {
	q := &fakeQuerier{execErr: errors.New("insert failed")}
	store := &fakeStore{
		shipment: dto.Shipment{ID: uuid.New(), DriverID: values.DriverID(uuid.New())},
		destRP:   dto.RoutePoint{PointID: 200},
	}
	offices := &fakeOffices{byPointID: map[int64]warehouse.Office{
		100: {ID: 11, CountryID: 643},
		200: {ID: 22, CountryID: 643},
	}}
	users := &fakeUsers{user: &warehouse.User{FirstName: "Ivan"}}

	p := warehouse.NewPublisher(store, wms_outbox.NewWriter(q), offices, users)

	err := p.Publish(context.Background(), values.ShipmentID(uuid.New()), dto.RoutePoint{PointID: 100}, []dto.Package{
		{ID: uuid.New(), TareSticker: "MPB-L3", ActionType: values.RoutePointActionTypeLoad},
	})
	if err == nil {
		t.Fatal("publish must fail when the outbox write fails")
	}
}
