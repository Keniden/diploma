package publicdriver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	publicdriver "last-mile-task/internal/application/public_driver"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

// ==== stubs (mirror internal/shipping/application mocks_test.go style) ====

// inlineTx runs the transaction callback inline; it satisfies both
// publicdriver.Transactional and ports.Transactional.
type inlineTx struct{}

func (inlineTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type mockDriversProvider struct {
	driver values.Driver
	err    error
}

func (m *mockDriversProvider) GetDriverFromContext(context.Context) (values.Driver, error) {
	return m.driver, m.err
}

type mockRouteTemplateProvider struct {
	tpl values.RouteTemplate
	err error
}

func (m *mockRouteTemplateProvider) GetRouteTemplateByID(
	context.Context, values.RouteTemplateID,
) (values.RouteTemplate, error) {
	return m.tpl, m.err
}

type mockShipmentRepository struct {
	getShipmentByIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	saveShipmentFn    func(ctx context.Context, shipment *entities.Shipment) error
}

func (m *mockShipmentRepository) GetShipmentByID(
	ctx context.Context, id values.ShipmentID,
) (*entities.Shipment, error) {
	if m.getShipmentByIDFn != nil {
		return m.getShipmentByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockShipmentRepository) SaveShipment(
	ctx context.Context, shipment *entities.Shipment,
) error {
	if m.saveShipmentFn != nil {
		return m.saveShipmentFn(ctx, shipment)
	}
	return nil
}

type mockRouteRepository struct {
	getRouteByIDFn         func(ctx context.Context, id values.RouteID) (*entities.Route, error)
	getRouteByShipmentIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	saveRouteFn            func(ctx context.Context, route *entities.Route) error
}

func (m *mockRouteRepository) GetRouteByID(
	ctx context.Context, id values.RouteID,
) (*entities.Route, error) {
	if m.getRouteByIDFn != nil {
		return m.getRouteByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRouteRepository) GetRouteByShipmentID(
	ctx context.Context, id values.ShipmentID,
) (*entities.Route, error) {
	if m.getRouteByShipmentIDFn != nil {
		return m.getRouteByShipmentIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRouteRepository) SaveRoute(ctx context.Context, route *entities.Route) error {
	if m.saveRouteFn != nil {
		return m.saveRouteFn(ctx, route)
	}
	return nil
}

type mockPackageRepository struct {
	savePackageFn func(ctx context.Context, pkg *entities.Package) error
}

func (m *mockPackageRepository) GetPackageByID(
	context.Context, values.ShipmentID, values.PackageID,
) (*entities.Package, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPackageRepository) GetPackagesByShipmentID(
	context.Context, values.ShipmentID,
) ([]*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) GetLoadedPackagesByShipmentID(
	context.Context, values.ShipmentID,
) ([]*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) GetPackagesByIDs(
	context.Context, values.ShipmentID, ...values.PackageID,
) ([]*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) SavePackage(ctx context.Context, pkg *entities.Package) error {
	if m.savePackageFn != nil {
		return m.savePackageFn(ctx, pkg)
	}
	return nil
}

func (m *mockPackageRepository) SavePackageScan(
	context.Context, *entities.Package, *entities.PackageAction,
) error {
	return nil
}

func (m *mockPackageRepository) DeletePackage(
	context.Context, values.ShipmentID, values.PackageID,
) error {
	return nil
}

type mockShipmentEventPublisher struct {
	events *[]events.ShipmentEvent
}

func (m *mockShipmentEventPublisher) PublishShipmentEvent(
	_ context.Context, event events.ShipmentEvent,
) error {
	*m.events = append(*m.events, event)
	return nil
}

type mockShipmentPolicyRepository struct {
	getShipmentCountByDriverFn  func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error)
	getShipmentByIDFn           func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	getRouteByShipmentIDFn      func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	getPackageCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error)
	getPackagesByShipmentIDFn   func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getReserveCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error)
	hasSourcePackagesFn         func(ctx context.Context, route *entities.Route) (bool, error)
}

func (m *mockShipmentPolicyRepository) GetShipmentByID(
	ctx context.Context, id values.ShipmentID,
) (*entities.Shipment, error) {
	if m.getShipmentByIDFn != nil {
		return m.getShipmentByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockShipmentPolicyRepository) GetRouteByShipmentID(
	ctx context.Context, id values.ShipmentID,
) (*entities.Route, error) {
	if m.getRouteByShipmentIDFn != nil {
		return m.getRouteByShipmentIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockShipmentPolicyRepository) GetShipmentCountByDriverID(
	ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus,
) (int, error) {
	if m.getShipmentCountByDriverFn != nil {
		return m.getShipmentCountByDriverFn(ctx, driverID, statuses...)
	}
	return 0, nil
}

func (m *mockShipmentPolicyRepository) GetPackageCountByShipmentID(
	ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus,
) (int, error) {
	if m.getPackageCountByShipmentFn != nil {
		return m.getPackageCountByShipmentFn(ctx, shipmentID, statuses...)
	}
	return 0, nil
}

func (m *mockShipmentPolicyRepository) GetPackagesByShipmentID(
	ctx context.Context, shipmentID values.ShipmentID,
) ([]*entities.Package, error) {
	if m.getPackagesByShipmentIDFn != nil {
		return m.getPackagesByShipmentIDFn(ctx, shipmentID)
	}
	return nil, nil
}

func (m *mockShipmentPolicyRepository) GetReserveCountByShipmentID(
	ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus,
) (int, error) {
	if m.getReserveCountByShipmentFn != nil {
		return m.getReserveCountByShipmentFn(ctx, shipmentID, statuses...)
	}
	return 0, nil
}

func (m *mockShipmentPolicyRepository) HasSourcePackages(
	ctx context.Context, route *entities.Route,
) (bool, error) {
	if m.hasSourcePackagesFn != nil {
		return m.hasSourcePackagesFn(ctx, route)
	}
	return true, nil
}

type mockDemandProvider struct {
	demands      []values.PackageDemand
	err          error
	demandsCount int
}

func (m *mockDemandProvider) GetPackageDemandByRoute(
	context.Context, *entities.Route,
) ([]values.PackageDemand, error) {
	m.demandsCount++
	return m.demands, m.err
}

// ==== harness ====

func csTemplate() values.RouteTemplate {
	return values.RouteTemplate{
		RouteTemplateID: values.RouteTemplateID(1),
		Distance:        250,
		Duration:        300,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: values.PointID(101), Distance: 0, SeqNum: 0},
			{RoutePointTemplateID: values.PointID(102), Distance: 250, SeqNum: 1},
		},
	}
}

// newCreateShipmentApp builds a publicdriver.Application wired with the known
// stubs. record holds the shipment events published by the assembler and
// progress services.
func newCreateShipmentApp(
	shipmentRepo *mockShipmentRepository,
	record *[]events.ShipmentEvent,
	demandProvider *mockDemandProvider,
) *publicdriver.Application {
	routeRepo := &mockRouteRepository{
		saveRouteFn: func(context.Context, *entities.Route) error { return nil },
	}
	// Demand planning re-fetches the route by the created shipment id.
	route, err := entities.NewRouteFromTemplate(csTemplate())
	if err != nil {
		panic(err)
	}
	routeRepo.getRouteByShipmentIDFn = func(context.Context, values.ShipmentID) (*entities.Route, error) {
		return route, nil
	}

	policy := policies.NewShipmentPolicy(&mockShipmentPolicyRepository{})
	assembler := shipping_application.NewAssembler(
		shipmentRepo,
		policy,
		routeRepo,
		&mockPackageRepository{},
		&mockShipmentEventPublisher{events: record},
		inlineTx{},
	)
	progress := shipping_application.NewShipmentProgressService(
		policy,
		shipmentRepo,
		&mockShipmentEventPublisher{events: record},
		inlineTx{},
	)
	demandService := demand.NewDemandService(routeRepo, &mockPackageRepository{}, demandProvider, inlineTx{})

	driverProvider := &mockDriversProvider{
		driver: values.Driver{ID: values.NewID(), SupplierID: 7},
	}
	templateProvider := &mockRouteTemplateProvider{tpl: csTemplate()}

	return publicdriver.NewForTesting(
		driverProvider,
		nil, // shippingQueries: not used by CreateShipment.
		nil, // scannerService: not used by CreateShipment.
		assembler,
		progress,
		nil,
		demandService,
		templateProvider,
		inlineTx{},
	)
}

// TestCreateShipment_Success proves the happy path: the shipment is created,
// the driver assigned, demand planned, and the created shipment id is returned.
func TestCreateShipment_Success(t *testing.T) {
	var createdShipmentID values.ShipmentID
	shipmentRepo := &mockShipmentRepository{
		saveShipmentFn: func(_ context.Context, shipment *entities.Shipment) error {
			createdShipmentID = shipment.ID
			return nil
		},
		getShipmentByIDFn: func(_ context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			s := entities.NewShipment()
			s.ID = id
			return s, nil
		},
	}

	var record []events.ShipmentEvent
	demandProvider := &mockDemandProvider{}
	app := newCreateShipmentApp(shipmentRepo, &record, demandProvider)

	id, err := app.CreateShipment(context.Background(), publicdriver.CreateShipmentRequest{RouteTemplateID: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != uuid.UUID(createdShipmentID) {
		t.Fatalf("expected created shipment id %s, got %s", createdShipmentID, id)
	}
	if demandProvider.demandsCount != 1 {
		t.Fatalf("expected demand to be planned once, got %d", demandProvider.demandsCount)
	}

	// The driver must have been assigned on the created shipment.
	if len(record) < 2 {
		t.Fatalf("expected ShipmentCreated and ShipmentDriverAssign events, got %d", len(record))
	}
	if _, ok := record[0].(events.ShipmentCreatedEvent); !ok {
		t.Fatalf("expected ShipmentCreatedEvent, got %T", record[0])
	}
	drv, ok := record[1].(events.ShipmentDriverAssignEvent)
	if !ok {
		t.Fatalf("expected ShipmentDriverAssignEvent, got %T", record[1])
	}
	if drv.ID != createdShipmentID {
		t.Fatalf("expected driver assign on %s, got %s", createdShipmentID, drv.ID)
	}
}

// TestCreateShipment_MidFlowFailure_CancelsShipment proves the atomicity fix:
// when driver assignment fails after the shipment has been created, the created
// shipment is canceled via CancelShipmentByID and the original cause is
// returned (wrapped).
func TestCreateShipment_MidFlowFailure_CancelsShipment(t *testing.T) {
	assignErr := errors.New("db down")

	var createdShipmentID values.ShipmentID
	getShipmentByIDCalls := 0
	shipmentRepo := &mockShipmentRepository{
		saveShipmentFn: func(_ context.Context, shipment *entities.Shipment) error {
			createdShipmentID = shipment.ID
			return nil
		},
		getShipmentByIDFn: func(_ context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			getShipmentByIDCalls++
			// First call is from AssignDriverByShipmentID: fail mid-flow.
			if getShipmentByIDCalls == 1 {
				return nil, assignErr
			}
			// Subsequent calls (CancelShipmentByID) resolve the created shipment.
			s := entities.NewShipment()
			s.ID = id
			return s, nil
		},
	}

	var record []events.ShipmentEvent
	app := newCreateShipmentApp(shipmentRepo, &record, &mockDemandProvider{})

	id, err := app.CreateShipment(context.Background(), publicdriver.CreateShipmentRequest{RouteTemplateID: 1})
	if err == nil {
		t.Fatal("expected an error when driver assignment fails")
	}
	if id != uuid.Nil {
		t.Fatalf("expected nil id on failure, got %s", id)
	}
	if !errors.Is(err, assignErr) {
		t.Fatalf("expected the original cause to be wrapped, got %v", err)
	}

	// CancelShipmentByID must have been issued for the created shipment id.
	if getShipmentByIDCalls < 2 {
		t.Fatalf("expected a second GetShipmentByID (cancel) call, got %d", getShipmentByIDCalls)
	}
	foundCancel := false
	for _, ev := range record {
		if canceled, ok := ev.(events.ShipmentCanceledEvent); ok && canceled.ID == createdShipmentID {
			foundCancel = true
		}
	}
	if !foundCancel {
		t.Fatalf("expected ShipmentCanceledEvent for %s, got events %+v", createdShipmentID, record)
	}
}