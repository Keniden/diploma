package drivergrpc

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	publicdriver "last-mile-task/internal/application/public_driver"
	lmterrors "last-mile-task/internal/errors"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
	driver "last-mile-task/pkg/gen/api/proto/v1/public/driver"
	packagev1 "last-mile-task/pkg/gen/api/proto/v1/public/driver/package"
	routepoint "last-mile-task/pkg/gen/api/proto/v1/public/driver/route_point"
	shipmentv1 "last-mile-task/pkg/gen/api/proto/v1/public/driver/shipment"
)

// --- fakes -----------------------------------------------------------------

type fakeDriverProvider struct {
	driverID values.DriverID
	err      error
}

func (f *fakeDriverProvider) GetDriverFromContext(context.Context) (values.Driver, error) {
	if f.err != nil {
		return values.Driver{}, f.err
	}
	return values.Driver{ID: f.driverID}, nil
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type fakePackageRepo struct {
	packages []*entities.Package
}

func (f *fakePackageRepo) GetPackagesByIDs(_ context.Context, shipmentID values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
	var out []*entities.Package
	for _, pkg := range f.packages {
		if pkg.ShipmentID != shipmentID {
			continue
		}
		for _, id := range ids {
			if pkg.ID == id {
				out = append(out, pkg)
				break
			}
		}
	}
	return out, nil
}

func (f *fakePackageRepo) GetPackageByID(context.Context, values.ShipmentID, values.PackageID) (*entities.Package, error) {
	return nil, nil
}

func (f *fakePackageRepo) GetPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return f.packages, nil
}

func (f *fakePackageRepo) GetLoadedPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return nil, nil
}

func (f *fakePackageRepo) SavePackage(context.Context, *entities.Package) error {
	return nil
}

func (f *fakePackageRepo) SavePackageScan(context.Context, *entities.Package, *entities.PackageAction) error {
	return nil
}

func (f *fakePackageRepo) DeletePackage(context.Context, values.ShipmentID, values.PackageID) error {
	return nil
}

type fakePublisher struct {
	published int
}

func (f *fakePublisher) PublishPackageEvent(context.Context, events.PackageEvent) error {
	f.published++
	return nil
}

type fakeShipmentStore struct {
	driverShipments []readmodels.Shipment
	shipment        *readmodels.Shipment
	route           *readmodels.Route
	routePoints     []readmodels.RoutePoint
	packages        []readmodels.Package
	packageActions  []readmodels.PackageAction
	reserve         *readmodels.ShipmentReserve
}

func (f *fakeShipmentStore) GetShipmentByID(_ context.Context, id uuid.UUID) (readmodels.Shipment, error) {
	if f.shipment != nil {
		return *f.shipment, nil
	}
	return readmodels.Shipment{}, stderrors.New("shipment not found")
}

func (f *fakeShipmentStore) GetDriverShipmentsByStatuses(context.Context, uuid.UUID, ...values.ShipmentStatus) ([]readmodels.Shipment, error) {
	return f.driverShipments, nil
}

func (f *fakeShipmentStore) GetRouteByShipmentID(_ context.Context, id uuid.UUID) (readmodels.Route, error) {
	route := readmodels.Route{}
	if f.route != nil {
		route = *f.route
	}
	if route.ShipmentID == uuid.Nil {
		route.ShipmentID = id
	}
	return route, nil
}

func (f *fakeShipmentStore) GetRoutePointsByShipmentID(context.Context, uuid.UUID) ([]readmodels.RoutePoint, error) {
	return f.routePoints, nil
}

func (f *fakeShipmentStore) GetPackageActionsByShipmentID(context.Context, uuid.UUID) ([]readmodels.PackageAction, error) {
	return f.packageActions, nil
}

func (f *fakeShipmentStore) GetPackagesByShipmentID(context.Context, uuid.UUID) ([]readmodels.Package, error) {
	return f.packages, nil
}

func (f *fakeShipmentStore) GetRoutesByShipmentIDs(context.Context, []uuid.UUID) ([]readmodels.Route, error) {
	return nil, nil
}

func (f *fakeShipmentStore) GetRoutePointsByShipmentIDs(context.Context, []uuid.UUID) ([]readmodels.RoutePoint, error) {
	return nil, nil
}

func (f *fakeShipmentStore) GetPackagesByShipmentIDs(context.Context, []uuid.UUID) ([]readmodels.Package, error) {
	return nil, nil
}

func (f *fakeShipmentStore) GetPackageActionsByShipmentIDs(context.Context, []uuid.UUID) ([]readmodels.PackageAction, error) {
	return nil, nil
}

func (f *fakeShipmentStore) GetDstPointByPackageIDs(context.Context, []uuid.UUID, uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeShipmentStore) GetActiveShipmentReserveByShipmentID(context.Context, uuid.UUID) (*readmodels.ShipmentReserve, error) {
	return f.reserve, nil
}

func (f *fakeShipmentStore) GetActiveShipmentReservesByShipmentIDs(context.Context, []uuid.UUID) (map[uuid.UUID]readmodels.ShipmentReserve, error) {
	return nil, nil
}

func (f *fakeShipmentStore) HasLoadedReturnPackagesAtRoutePoint(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

type fakeAddresses struct {
	byPointID map[int64]string
}

func (f *fakeAddresses) AddressesByPointIDs(_ context.Context, pointIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(pointIDs))
	for _, id := range pointIDs {
		if addr, ok := f.byPointID[id]; ok {
			out[id] = addr
		} else {
			out[id] = fmt.Sprintf("address %d", id)
		}
	}
	return out, nil
}

// --- helpers ---------------------------------------------------------------

func newTestServer(
	provider publicdriver.DriversProvider,
	queries *shipment.Queries,
	scannerService *scanner.PackageScanner,
) *Server {
	return New(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		publicdriver.NewForTesting(
			provider,
			queries,
			scannerService,
			nil, nil, nil, nil, // creation services (assembler/progress/route point/demand): wired by newCreateServer.
			nil,
			nil,
		),
	)
}

func newTestQueries(store *fakeShipmentStore) *shipment.Queries {
	return shipment.NewQueries(store, &fakeAddresses{byPointID: map[int64]string{1: "address 1"}})
}

func loadPackageRequest(ids []uuid.UUID, shipmentID uuid.UUID, pointID uuid.UUID) *driver.SetPackagesStatusRequest {
	req := &driver.SetPackagesStatusRequest{ShipmentId: shipmentID.String()}
	for _, id := range ids {
		req.Packages = append(req.Packages, &driver.SetPackagesStatusRequest_PackageAtPoint{
			Id:        id.String(),
			PointId:   pointID.String(),
			Action:    packagev1.PointAction_POINT_ACTION_LOAD,
			Timestamp: timestamppb.New(time.Now()),
		})
	}
	return req
}

// --- SetPackagesStatus tests ------------------------------------------------

func TestSetPackagesStatus_ValidationError(t *testing.T) {
	srv := newTestServer(&fakeDriverProvider{}, newTestQueries(&fakeShipmentStore{}), nil)

	_, err := srv.SetPackagesStatus(context.Background(), &driver.SetPackagesStatusRequest{
		ShipmentId: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var lmtErr *lmterrors.Error
	if !stderrors.As(err, &lmtErr) {
		t.Fatalf("expected internal error, got %T: %v", err, err)
	}
	if lmtErr.ProtoCode() != int32(codes.InvalidArgument) {
		t.Fatalf("expected InvalidArgument code, got %d", lmtErr.ProtoCode())
	}
}

func TestSetPackagesStatus_PartialFailure(t *testing.T) {
	shipmentID := uuid.New()
	pointID := uuid.New()
	badPkgID := uuid.New()
	goodPkgID := uuid.New()

	repo := &fakePackageRepo{
		packages: []*entities.Package{
			{
				ID:         values.PackageID(badPkgID),
				ShipmentID: values.ShipmentID(shipmentID),
				// Already loaded: a second LOAD scan is a prohibited
				// transition and must land in failed_packages.
				Status: values.PackageStatusLoaded,
				Actions: []*entities.PackageAction{
					{
						RoutePointID: values.RoutePointID(pointID),
						Type:         values.RoutePointActionTypeLoad,
						Status:       values.RoutePointPackageStatusDone,
					},
				},
			},
			{
				ID:         values.PackageID(goodPkgID),
				ShipmentID: values.ShipmentID(shipmentID),
				Status:     values.PackageStatusReadyToLoad,
				Actions: []*entities.PackageAction{
					{
						RoutePointID: values.RoutePointID(pointID),
						Type:         values.RoutePointActionTypeLoad,
						Status:       values.RoutePointPackageStatusTODO,
					},
				},
			},
		},
	}
	publisher := &fakePublisher{}
	scannerService := scanner.NewPackageScanner(repo, publisher, fakeTx{})

	srv := newTestServer(&fakeDriverProvider{}, newTestQueries(&fakeShipmentStore{}), scannerService)

	resp, err := srv.SetPackagesStatus(context.Background(), loadPackageRequest([]uuid.UUID{badPkgID, goodPkgID}, shipmentID, pointID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Message != "Packages scanned" {
		t.Fatalf("expected message %q, got %q", "Packages scanned", resp.Message)
	}
	failed := resp.GetData().GetFailedPackages()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed package, got %d", len(failed))
	}
	if failed[0].Id != badPkgID.String() {
		t.Fatalf("expected failed package id %s, got %s", badPkgID, failed[0].Id)
	}
	if failed[0].ErrorDescription == "" {
		t.Fatal("expected non-empty error description")
	}
	if publisher.published != 1 {
		t.Fatalf("expected 1 published event for the successful scan, got %d", publisher.published)
	}
}

func TestSetPackagesStatus_AllSuccess(t *testing.T) {
	shipmentID := uuid.New()
	pointID := uuid.New()
	pkgID := uuid.New()

	repo := &fakePackageRepo{
		packages: []*entities.Package{
			{
				ID:         values.PackageID(pkgID),
				ShipmentID: values.ShipmentID(shipmentID),
				Status:     values.PackageStatusReadyToLoad,
				Actions: []*entities.PackageAction{
					{
						RoutePointID: values.RoutePointID(pointID),
						Type:         values.RoutePointActionTypeLoad,
						Status:       values.RoutePointPackageStatusTODO,
					},
				},
			},
		},
	}

	srv := newTestServer(
		&fakeDriverProvider{},
		newTestQueries(&fakeShipmentStore{}),
		scanner.NewPackageScanner(repo, &fakePublisher{}, fakeTx{}),
	)

	resp, err := srv.SetPackagesStatus(context.Background(), loadPackageRequest([]uuid.UUID{pkgID}, shipmentID, pointID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if got := len(resp.GetData().GetFailedPackages()); got != 0 {
		t.Fatalf("expected no failed packages, got %d", got)
	}
}

func TestSetPackagesStatus_InvalidPackageUUID(t *testing.T) {
	srv := newTestServer(&fakeDriverProvider{}, newTestQueries(&fakeShipmentStore{}), nil)

	_, err := srv.SetPackagesStatus(context.Background(), &driver.SetPackagesStatusRequest{
		ShipmentId: uuid.NewString(),
		Packages: []*driver.SetPackagesStatusRequest_PackageAtPoint{
			{Id: "not-a-uuid", PointId: uuid.NewString()},
		},
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// --- GetActiveShipmentByDriverID tests --------------------------------------

func TestGetActiveShipmentByDriverID_NoActiveShipment(t *testing.T) {
	store := &fakeShipmentStore{}
	srv := newTestServer(&fakeDriverProvider{driverID: values.DriverID(uuid.New())}, newTestQueries(store), nil)

	resp, err := srv.GetActiveShipmentByDriverID(context.Background(), &driver.GetActiveShipmentByDriverIDRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Data == nil {
		t.Fatal("expected non-nil data")
	}
	if resp.Data.Shipment != nil {
		t.Fatalf("expected nil shipment, got %v", resp.Data.Shipment)
	}
}

func TestGetActiveShipmentByDriverID_DriverProviderError(t *testing.T) {
	providerErr := stderrors.New("driver not found in context")
	store := &fakeShipmentStore{}
	srv := newTestServer(&fakeDriverProvider{err: providerErr}, newTestQueries(store), nil)

	_, err := srv.GetActiveShipmentByDriverID(context.Background(), &driver.GetActiveShipmentByDriverIDRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stderrors.Is(err, providerErr) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
}

func TestGetActiveShipmentByDriverID_ActiveShipment(t *testing.T) {
	driverID := uuid.New()
	shipmentID := uuid.New()
	routeID := uuid.New()
	routePointID := uuid.New()
	pointID := int64(12345)
	expiresAt := time.Now().Add(time.Hour).UTC()

	store := &fakeShipmentStore{
		driverShipments: []readmodels.Shipment{
			{ID: shipmentID, ShortID: 7, Status: int(values.ShipmentStatusInProgress)},
		},
		shipment: &readmodels.Shipment{ID: shipmentID, ShortID: 7, Status: int(values.ShipmentStatusInProgress)},
		route:    &readmodels.Route{ID: routeID, ShipmentID: shipmentID},
		routePoints: []readmodels.RoutePoint{
			{
				ID:      routePointID,
				PointID: pointID,
				Address: "address 1",
				SeqNum:  1,
				Status:  int(values.RoutePointStatusDriverArrived),
			},
		},
		reserve: &readmodels.ShipmentReserve{
			UpdatedAt: time.Now().Add(-time.Minute).UTC(),
			ExpiresAt: &expiresAt,
		},
	}

	srv := newTestServer(&fakeDriverProvider{driverID: values.DriverID(driverID)}, newTestQueries(store), nil)

	resp, err := srv.GetActiveShipmentByDriverID(context.Background(), &driver.GetActiveShipmentByDriverIDRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Data.GetShipment() == nil {
		t.Fatal("expected mapped shipment")
	}

	ship := resp.Data.GetShipment()
	if ship.Id != shipmentID.String() {
		t.Fatalf("expected shipment id %s, got %s", shipmentID, ship.Id)
	}
	if ship.ShortId != 7 {
		t.Fatalf("expected short id 7, got %d", ship.ShortId)
	}
	if ship.Status != shipmentv1.Status_STATUS_IN_PROGRESS {
		t.Fatalf("expected IN_PROGRESS status, got %s", ship.Status)
	}
	if ship.Route == nil || ship.Route.Id != routeID.String() {
		t.Fatalf("expected route %s, got %+v", routeID, ship.Route)
	}
	if len(ship.Route.Points) != 1 {
		t.Fatalf("expected 1 route point, got %d", len(ship.Route.Points))
	}
	point := ship.Route.Points[0]
	if point.Id != routePointID.String() {
		t.Fatalf("expected route point id %s, got %s", routePointID, point.Id)
	}
	if point.ShortId != "12345" {
		t.Fatalf("expected short id 12345, got %q", point.ShortId)
	}
	if point.Address != "address 12345" {
		t.Fatalf("expected address %q, got %q", "address 12345", point.Address)
	}
	if point.Status != routepoint.RoutePointStatus_ROUTE_POINT_STATUS_DRIVER_ARRIVED {
		t.Fatalf("expected DRIVER_ARRIVED status, got %s", point.Status)
	}
	if ship.Reserve == nil {
		t.Fatal("expected mapped reserve")
	}
	if ship.Reserve.ExpiresAt.AsTime().Unix() != expiresAt.Unix() {
		t.Fatalf("expected expires_at %s, got %s", expiresAt, ship.Reserve.ExpiresAt.AsTime())
	}
}

// --- CreateShipment family fakes ----------------------------------------------

func testTemplate() values.RouteTemplate {
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

type fakeTemplatesProvider struct {
	template values.RouteTemplate
	err      error
}

func (f *fakeTemplatesProvider) GetRouteTemplateByID(context.Context, values.RouteTemplateID) (values.RouteTemplate, error) {
	if f.err != nil {
		return values.RouteTemplate{}, f.err
	}
	return f.template, nil
}

type fakeDemandProvider struct{}

func (fakeDemandProvider) GetPackageDemandByRoute(context.Context, *entities.Route) ([]values.PackageDemand, error) {
	return nil, nil
}

type noopShipmentPublisher struct{}

func (noopShipmentPublisher) PublishShipmentEvent(context.Context, events.ShipmentEvent) error {
	return nil
}

type memoryShipmentRepo struct {
	shipment  *entities.Shipment
	afterSave func(*entities.Shipment)
}

func (r *memoryShipmentRepo) GetShipmentByID(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
	if r.shipment == nil {
		return nil, ports.ErrShipmentNotFound
	}
	return r.shipment, nil
}

func (r *memoryShipmentRepo) SaveShipment(_ context.Context, shipment *entities.Shipment) error {
	r.shipment = shipment
	if r.afterSave != nil {
		r.afterSave(shipment)
	}
	return nil
}

type memoryRouteRepo struct {
	route     *entities.Route
	afterSave func(*entities.Route)
}

func (r *memoryRouteRepo) GetRouteByID(_ context.Context, _ values.RouteID) (*entities.Route, error) {
	if r.route == nil {
		return nil, ports.ErrRouteNotFound
	}
	return r.route, nil
}

func (r *memoryRouteRepo) GetRouteByShipmentID(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
	if r.route == nil {
		return nil, ports.ErrRouteNotFound
	}
	return r.route, nil
}

func (r *memoryRouteRepo) SaveRoute(_ context.Context, route *entities.Route) error {
	r.route = route
	if r.afterSave != nil {
		r.afterSave(route)
	}
	return nil
}

// policyRepo implements [policies.ShipmentPolicyRepository] for the creation
// tests (no packages/reserves are involved).
type policyRepo struct {
	countFn  func(context.Context, values.DriverID, ...values.ShipmentStatus) (int, error)
	sourceFn func(context.Context, *entities.Route) (bool, error)
}

func (r *policyRepo) GetShipmentByID(context.Context, values.ShipmentID) (*entities.Shipment, error) {
	return nil, nil
}

func (r *policyRepo) GetRouteByShipmentID(context.Context, values.ShipmentID) (*entities.Route, error) {
	return nil, nil
}

func (r *policyRepo) GetShipmentCountByDriverID(ctx context.Context, id values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
	if r.countFn == nil {
		return 0, nil
	}
	return r.countFn(ctx, id, statuses...)
}

func (r *policyRepo) GetPackageCountByShipmentID(context.Context, values.ShipmentID, ...values.PackageStatus) (int, error) {
	return 0, nil
}

func (r *policyRepo) GetPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return nil, nil
}

func (r *policyRepo) GetReserveCountByShipmentID(context.Context, values.ShipmentID, ...values.ReserveStatus) (int, error) {
	return 0, nil
}

func (r *policyRepo) HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error) {
	if r.sourceFn == nil {
		return true, nil
	}
	return r.sourceFn(ctx, route)
}

// newCreationServer wires a real assembler/progress/demand over in-memory
// repositories and seeds the shipment read-model store from the saved
// aggregates, so the handlers can answer with the created shipment.
func newCreationServer(
	provider publicdriver.DriversProvider,
	templates ports.RouteTemplateProvider,
) (*Server, *memoryShipmentRepo) {
	store := &fakeShipmentStore{}

	shipmentRepo := &memoryShipmentRepo{
		afterSave: func(s *entities.Shipment) {
			store.shipment = &readmodels.Shipment{
				ID:           uuid.UUID(s.ID),
				Status:       int(s.Status),
				DriverID:     uuid.UUID(s.DriverID),
				SupplierID:   uint64(s.SupplierID),
				LicensePlate: string(s.LicensePlate),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
		},
	}
	routeRepo := &memoryRouteRepo{
		afterSave: func(route *entities.Route) {
			store.route = &readmodels.Route{
				ID:         route.ID,
				ShipmentID: route.ShipmentID,
				TemplateID: int64(route.TemplateID),
				Distance:   route.Distance,
				Duration:   route.Duration,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			points := make([]readmodels.RoutePoint, 0, len(route.Points))
			for _, p := range route.Points {
				points = append(points, readmodels.RoutePoint{
					ID:         p.ID,
					RouteID:    route.ID,
					ShipmentID: route.ShipmentID,
					PointID:    int64(p.PointID),
					Address:    fmt.Sprintf("address %d", p.PointID),
					SeqNum:     p.SeqNum,
					Distance:   p.Distance,
					Status:     int(p.Status),
				})
			}
			store.routePoints = points
		},
	}

	policy := policies.NewShipmentPolicy(&policyRepo{})
	assembler := shipping_application.NewAssembler(shipmentRepo, policy, routeRepo, &fakePackageRepo{}, noopShipmentPublisher{}, fakeTx{})
	progress := shipping_application.NewShipmentProgressService(policy, shipmentRepo, noopShipmentPublisher{}, fakeTx{})
	demandService := demand.NewDemandService(routeRepo, &fakePackageRepo{}, &fakeDemandProvider{}, fakeTx{})

	app := publicdriver.NewForTesting(
		provider,
		newTestQueries(store),
		nil,
		assembler,
		progress,
		nil,
		demandService,
		templates,
		fakeTx{},
	)

	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), app), shipmentRepo
}

// --- CreateShipment tests ----------------------------------------------------

func TestCreateShipment_Success(t *testing.T) {
	driverID := values.NewID()
	srv, shipmentRepo := newCreationServer(
		&fakeDriverProvider{driverID: driverID},
		&fakeTemplatesProvider{template: testTemplate()},
	)

	resp, err := srv.CreateShipment(context.Background(), &driver.CreateShipmentRequest{RouteTemplateId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Message != "Shipment created" {
		t.Fatalf("expected message %q, got %q", "Shipment created", resp.Message)
	}

	ship := resp.GetData().GetShipment()
	if ship == nil {
		t.Fatal("expected mapped shipment")
	}
	if ship.Id != uuid.UUID(shipmentRepo.shipment.ID).String() {
		t.Fatalf("expected shipment id %s, got %s", shipmentRepo.shipment.ID, ship.Id)
	}
	if ship.Status != shipmentv1.Status_STATUS_ASSIGNED {
		t.Fatalf("expected ASSIGNED status, got %s", ship.Status)
	}
	if shipmentRepo.shipment.DriverID != driverID {
		t.Fatal("driver not assigned to the shipment aggregate")
	}
	if shipmentRepo.shipment.Status != values.ShipmentStatusAssigned {
		t.Fatal("expected ASSIGNED shipment status")
	}
}

func TestCreateShipment_TemplateNotFound(t *testing.T) {
	srv, _ := newCreationServer(
		&fakeDriverProvider{driverID: values.NewID()},
		&fakeTemplatesProvider{err: ports.ErrRouteTemplateNotFound},
	)

	_, err := srv.CreateShipment(context.Background(), &driver.CreateShipmentRequest{RouteTemplateId: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestCreateShipmentWithTransport_Success(t *testing.T) {
	driverID := values.NewID()
	srv, shipmentRepo := newCreationServer(
		&fakeDriverProvider{driverID: driverID},
		&fakeTemplatesProvider{template: testTemplate()},
	)

	resp, err := srv.CreateShipmentWithTransport(
		context.Background(),
		&driver.CreateShipmentWithTransportRequest{RouteTemplateId: 1, CarLicensePlate: "aa 1234"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.GetData().GetShipment() == nil {
		t.Fatal("expected mapped shipment")
	}
	if shipmentRepo.shipment.LicensePlate != values.NewLicensePlate("aa 1234") {
		t.Fatalf("expected normalized plate, got %q", shipmentRepo.shipment.LicensePlate)
	}
	if shipmentRepo.shipment.Status != values.ShipmentStatusAssigned {
		t.Fatal("expected ASSIGNED status")
	}
}

func TestCreateShipmentWithTransport_EmptyPlateCancelsShipment(t *testing.T) {
	srv, shipmentRepo := newCreationServer(
		&fakeDriverProvider{driverID: values.NewID()},
		&fakeTemplatesProvider{template: testTemplate()},
	)

	_, err := srv.CreateShipmentWithTransport(
		context.Background(),
		&driver.CreateShipmentWithTransportRequest{RouteTemplateId: 1},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
	if shipmentRepo.shipment.Status != values.ShipmentStatusCanceled {
		t.Fatal("expected created shipment to be canceled")
	}
}

// --- UpdateShipmentTransport tests -------------------------------------------

func TestUpdateShipmentTransport_Success(t *testing.T) {
	existing := entities.NewShipment()
	existing.DriverID = values.NewID() // no-op; keeps the shipment NEW.

	srv, shipmentRepo := newCreationServer(
		&fakeDriverProvider{driverID: values.NewID()},
		&fakeTemplatesProvider{template: testTemplate()},
	)
	shipmentRepo.shipment = existing

	resp, err := srv.UpdateShipmentTransport(
		context.Background(),
		&driver.UpdateShipmentTransportRequest{
			ShipmentId:      uuid.UUID(existing.ID).String(),
			CarLicensePlate: "E777AA77",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Message != "Transport updated" {
		t.Fatalf("expected message %q, got %q", "Transport updated", resp.Message)
	}
	if shipmentRepo.shipment.LicensePlate != "E777AA77" {
		t.Fatalf("expected plate %q, got %q", "E777AA77", shipmentRepo.shipment.LicensePlate)
	}
}

func TestUpdateShipmentTransport_EmptyPlate(t *testing.T) {
	existing := entities.NewShipment()

	srv, shipmentRepo := newCreationServer(
		&fakeDriverProvider{driverID: values.NewID()},
		&fakeTemplatesProvider{template: testTemplate()},
	)
	shipmentRepo.shipment = existing

	_, err := srv.UpdateShipmentTransport(
		context.Background(),
		&driver.UpdateShipmentTransportRequest{
			ShipmentId:      uuid.UUID(existing.ID).String(),
			CarLicensePlate: "",
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestUpdateShipmentTransport_ValidationError(t *testing.T) {
	srv, _ := newCreationServer(
		&fakeDriverProvider{driverID: values.NewID()},
		&fakeTemplatesProvider{template: testTemplate()},
	)

	_, err := srv.UpdateShipmentTransport(
		context.Background(),
		&driver.UpdateShipmentTransportRequest{ShipmentId: "not-a-uuid", CarLicensePlate: "E777AA77"},
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var lmtErr *lmterrors.Error
	if !stderrors.As(err, &lmtErr) {
		t.Fatalf("expected internal error, got %T: %v", err, err)
	}
	if lmtErr.ProtoCode() != int32(codes.InvalidArgument) {
		t.Fatalf("expected InvalidArgument code, got %d", lmtErr.ProtoCode())
	}
}
