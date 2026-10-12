package demand_test

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

// stubTx runs the callback inline (no real transaction).
type stubTx struct{}

func (stubTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// stubRouteRepository implements the subset of [ports.RouteRepository] used by
// DemandService.
type stubRouteRepository struct {
	route *entities.Route
	err   error
}

func (r *stubRouteRepository) GetRouteByID(context.Context, values.RouteID) (*entities.Route, error) {
	return nil, errors.New("not implemented")
}

func (r *stubRouteRepository) GetRouteByShipmentID(context.Context, values.ShipmentID) (*entities.Route, error) {
	return r.route, r.err
}

func (r *stubRouteRepository) SaveRoute(context.Context, *entities.Route) error {
	return nil
}

// stubPackageRepository implements the subset of [ports.PackageRepository] used
// by DemandService.
type stubPackageRepository struct {
	saveFn func(ctx context.Context, pkg *entities.Package) error
}

func (r *stubPackageRepository) GetPackageByID(context.Context, values.ShipmentID, values.PackageID) (*entities.Package, error) {
	return nil, errors.New("not implemented")
}

func (r *stubPackageRepository) GetPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return nil, errors.New("not implemented")
}

func (r *stubPackageRepository) GetLoadedPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return nil, errors.New("not implemented")
}

func (r *stubPackageRepository) GetPackagesByIDs(context.Context, values.ShipmentID, ...values.PackageID) ([]*entities.Package, error) {
	return nil, errors.New("not implemented")
}

func (r *stubPackageRepository) SavePackage(ctx context.Context, pkg *entities.Package) error {
	if r.saveFn != nil {
		return r.saveFn(ctx, pkg)
	}
	return nil
}

func (r *stubPackageRepository) SavePackageScan(context.Context, *entities.Package, *entities.PackageAction) error {
	return nil
}

func (r *stubPackageRepository) DeletePackage(context.Context, values.ShipmentID, values.PackageID) error {
	return nil
}

// routeWithPoints builds a route with assigned route point IDs (loaded).
func routeWithPoints(t *testing.T) *entities.Route {
	t.Helper()

	template := values.RouteTemplate{
		RouteTemplateID: values.RouteTemplateID(1),
		Distance:        250,
		Duration:        300,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: values.PointID(101), Distance: 0, SeqNum: 0},
			{RoutePointTemplateID: values.PointID(102), Distance: 250, SeqNum: 1},
		},
	}

	route, err := entities.NewRouteFromTemplate(template)
	if err != nil {
		t.Fatalf("new route from template: %v", err)
	}
	return route
}

type fakeDemandProvider struct {
	demands []values.PackageDemand
	err     error
}

func (f *fakeDemandProvider) GetPackageDemandByRoute(
	_ context.Context, _ *entities.Route,
) ([]values.PackageDemand, error) {
	return f.demands, f.err
}

func TestPlanPackagesToShipmentByID_Success(t *testing.T) {
	route := routeWithPoints(t)
	source, _ := route.Source()
	target, _ := route.Target()
	shipmentID := values.NewID()

	var saved []*entities.Package
	routeRepo := &stubRouteRepository{route: route}
	pkgRepo := &stubPackageRepository{
		saveFn: func(_ context.Context, pkg *entities.Package) error {
			saved = append(saved, pkg)
			return nil
		},
	}

	demandID := values.NewID()
	provider := &fakeDemandProvider{demands: []values.PackageDemand{
		{
			PackageID:     demandID,
			TareType:      "BOX",
			TareID:        1,
			LoadPointID:   source.ID,
			UnloadPointID: target.ID,
			Direction:     values.PackageDemandDirectionForward,
		},
	}}

	svc := demand.NewDemandService(routeRepo, pkgRepo, provider, &stubTx{})
	if err := svc.PlanPackagesToShipmentByID(context.Background(), shipmentID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(saved) != 1 {
		t.Fatalf("expected 1 saved package, got %d", len(saved))
	}
	pkg := saved[0]
	if pkg.ID != demandID || pkg.ShipmentID != shipmentID {
		t.Fatal("package fields mismatch")
	}
	if !pkg.IsReadyToLoad() || pkg.IsReturn {
		t.Fatal("package must be ready to load and direct")
	}
	if len(pkg.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(pkg.Actions))
	}
}

func TestPlanPackagesToShipmentByID_ProviderError(t *testing.T) {
	route := routeWithPoints(t)

	var saved int
	svc := demand.NewDemandService(
		&stubRouteRepository{route: route},
		&stubPackageRepository{saveFn: func(_ context.Context, _ *entities.Package) error {
			saved++
			return nil
		}},
		&fakeDemandProvider{err: errors.New("cargo unavailable")},
		&stubTx{},
	)

	if err := svc.PlanPackagesToShipmentByID(context.Background(), values.NewID()); err == nil {
		t.Fatal("expected provider error")
	}
	if saved != 0 {
		t.Fatal("no package must be saved on provider error")
	}
}

func TestPlanPackagesToShipmentByID_RouteNotFound(t *testing.T) {
	svc := demand.NewDemandService(
		&stubRouteRepository{err: errors.New("route not found")},
		&stubPackageRepository{},
		&fakeDemandProvider{},
		&stubTx{},
	)

	if err := svc.PlanPackagesToShipmentByID(context.Background(), values.NewID()); err == nil {
		t.Fatal("expected route error")
	}
}