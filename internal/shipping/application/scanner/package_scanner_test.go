package scanner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	domscan "last-mile-task/internal/shipping/domain/values/scan"
)

// ==== Local mocks ====

type mockPackageRepository struct {
	getPackagesByIDsFn func(ctx context.Context, shipmentID values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error)
	savePackageFn      func(ctx context.Context, pkg *entities.Package) error
	savePackageScanFn  func(ctx context.Context, pkg *entities.Package, action *entities.PackageAction) error
}

func (m *mockPackageRepository) GetPackageByID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) (*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) GetLoadedPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	return nil, nil
}

func (m *mockPackageRepository) GetPackagesByIDs(
	ctx context.Context,
	shipmentID values.ShipmentID,
	ids ...values.PackageID,
) ([]*entities.Package, error) {
	return m.getPackagesByIDsFn(ctx, shipmentID, ids...)
}

func (m *mockPackageRepository) SavePackage(ctx context.Context, pkg *entities.Package) error {
	return m.savePackageFn(ctx, pkg)
}

func (m *mockPackageRepository) SavePackageScan(ctx context.Context, pkg *entities.Package, action *entities.PackageAction) error {
	if m.savePackageScanFn != nil {
		return m.savePackageScanFn(ctx, pkg, action)
	}
	return nil
}

func (m *mockPackageRepository) DeletePackage(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) error {
	return nil
}

type mockPackageEventPublisher struct {
	publishFn func(ctx context.Context, event events.PackageEvent) error
}

func (m *mockPackageEventPublisher) PublishPackageEvent(ctx context.Context, event events.PackageEvent) error {
	return m.publishFn(ctx, event)
}

type mockTransactional struct {
	withTxFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactional) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.withTxFn(ctx, fn)
}

func newMockTransactional() *mockTransactional {
	return &mockTransactional{
		withTxFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}
}

func TestScanPackages_Success(t *testing.T) {
	routePointID := values.NewID()
	shipmentID := values.NewID()

	pkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   routePointID,
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	pkg.AssignToShipment(shipmentID)

	packageRepo := &mockPackageRepository{
		getPackagesByIDsFn: func(ctx context.Context, _ values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
			return []*entities.Package{pkg}, nil
		},
		savePackageScanFn: func(ctx context.Context, p *entities.Package, action *entities.PackageAction) error {
			return nil
		},
	}

	var publishedEvent events.PackageEvent
	eventPub := &mockPackageEventPublisher{
		publishFn: func(ctx context.Context, event events.PackageEvent) error {
			publishedEvent = event
			return nil
		},
	}

	svc := scanner.NewPackageScanner(
		packageRepo,
		eventPub,
		newMockTransactional(),
	)

	scanned := []domscan.Package{
		{
			ID:           pkg.ID,
			RoutePointID: routePointID,
			ShipmentID:   shipmentID,
			ActionType:   values.RoutePointActionTypeLoad,
			Timestamp:    time.Now(),
		},
	}

	err := svc.ScanPackages(context.Background(), shipmentID, scanned)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Status != values.PackageStatusLoaded {
		t.Fatalf("expected loaded status, got %v", pkg.Status)
	}
	scanEv, ok := publishedEvent.(events.PackageScannedEvent)
	if !ok {
		t.Fatalf("expected PackageScannedEvent, got %T", publishedEvent)
	}
	if len(scanEv.Packages) != 1 {
		t.Fatalf("expected 1 scanned package in event, got %d", len(scanEv.Packages))
	}
}

func TestScanPackages_NoPackagesFound(t *testing.T) {
	packageRepo := &mockPackageRepository{
		getPackagesByIDsFn: func(ctx context.Context, _ values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
			return nil, nil
		},
	}

	var publishedEvent events.PackageEvent
	eventPub := &mockPackageEventPublisher{
		publishFn: func(ctx context.Context, event events.PackageEvent) error {
			publishedEvent = event
			return nil
		},
	}

	svc := scanner.NewPackageScanner(
		packageRepo,
		eventPub,
		newMockTransactional(),
	)

	err := svc.ScanPackages(context.Background(), values.NilID, []domscan.Package{
		{ID: values.NewID(), Timestamp: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	scanEv, ok := publishedEvent.(events.PackageScannedEvent)
	if !ok {
		t.Fatalf("expected PackageScannedEvent, got %T", publishedEvent)
	}
	if len(scanEv.Packages) != 0 {
		t.Fatalf("expected 0 scanned packages, got %d", len(scanEv.Packages))
	}
}

func TestScanPackages_GetPackagesError(t *testing.T) {
	packageRepo := &mockPackageRepository{
		getPackagesByIDsFn: func(ctx context.Context, _ values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
			return nil, errors.New("db error")
		},
	}

	svc := scanner.NewPackageScanner(
		packageRepo,
		&mockPackageEventPublisher{},
		newMockTransactional(),
	)

	err := svc.ScanPackages(context.Background(), values.NilID, []domscan.Package{
		{ID: values.NewID(), Timestamp: time.Now()},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanPackages_SaveError(t *testing.T) {
	shipmentID := values.NewID()
	pkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	pkg.AssignToShipment(shipmentID)

	packageRepo := &mockPackageRepository{
		getPackagesByIDsFn: func(ctx context.Context, _ values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
			return []*entities.Package{pkg}, nil
		},
		savePackageScanFn: func(ctx context.Context, p *entities.Package, action *entities.PackageAction) error {
			return errors.New("save error")
		},
	}

	svc := scanner.NewPackageScanner(
		packageRepo,
		&mockPackageEventPublisher{},
		newMockTransactional(),
	)

	err := svc.ScanPackages(context.Background(), shipmentID, []domscan.Package{
		{
			ID:           pkg.ID,
			ShipmentID:   shipmentID,
			RoutePointID: pkg.Actions[0].RoutePointID,
			ActionType:   values.RoutePointActionTypeLoad,
			Timestamp:    time.Now(),
		},
	})
	if err == nil {
		t.Fatal("expected save error")
	}
}

func TestScanPackages_SkipsOtherShipments(t *testing.T) {
	routePointID := values.NewID()
	targetShipmentID := values.NewID()
	otherShipmentID := values.NewID()

	targetPkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   routePointID,
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	targetPkg.AssignToShipment(targetShipmentID)

	otherPkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     targetPkg.ID,
		LoadPointID:   routePointID,
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	otherPkg.AssignToShipment(otherShipmentID)

	saveCalls := 0
	packageRepo := &mockPackageRepository{
		getPackagesByIDsFn: func(ctx context.Context, shipmentID values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error) {
			if shipmentID != targetShipmentID {
				t.Fatalf("expected shipment id %s, got %s", targetShipmentID, shipmentID)
			}
			return []*entities.Package{targetPkg}, nil
		},
		savePackageScanFn: func(ctx context.Context, pkg *entities.Package, action *entities.PackageAction) error {
			saveCalls++
			if pkg.ShipmentID != targetShipmentID {
				t.Fatalf("expected save only for target shipment, got %s", pkg.ShipmentID)
			}
			if action.Type != values.RoutePointActionTypeLoad {
				t.Fatalf("expected load action save, got %v", action.Type)
			}
			return nil
		},
	}

	eventPub := &mockPackageEventPublisher{
		publishFn: func(ctx context.Context, event events.PackageEvent) error {
			return nil
		},
	}

	svc := scanner.NewPackageScanner(
		packageRepo,
		eventPub,
		newMockTransactional(),
	)

	err := svc.ScanPackages(context.Background(), targetShipmentID, []domscan.Package{
		{
			ID:           targetPkg.ID,
			ShipmentID:   targetShipmentID,
			RoutePointID: routePointID,
			ActionType:   values.RoutePointActionTypeLoad,
			Timestamp:    time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saveCalls != 1 {
		t.Fatalf("expected 1 save call, got %d", saveCalls)
	}
	if otherPkg.Status != values.PackageStatusReadyToLoad {
		t.Fatalf("expected other shipment package to stay unchanged, got %v", otherPkg.Status)
	}
	if targetPkg.Actions[1].Status != values.RoutePointPackageStatusTODO {
		t.Fatalf("expected unload action to stay TODO, got %v", targetPkg.Actions[1].Status)
	}
}
