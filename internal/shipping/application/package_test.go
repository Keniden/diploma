package application_test

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

type noopPackageEventPublisher struct{}

func (noopPackageEventPublisher) PublishPackageEvent(context.Context, events.PackageEvent) error {
	return nil
}

func TestDeletePackage_Success(t *testing.T) {
	shipmentID := values.NewID()
	pkg := entities.NewPackage()
	pkg.AssignToShipment(shipmentID)

	var deletedID values.PackageID
	packageRepo := &mockPackageRepository{
		getPackageByIDFn: func(ctx context.Context, _ values.ShipmentID, id values.PackageID) (*entities.Package, error) {
			return pkg, nil
		},
		deletePackageFn: func(ctx context.Context, _ values.ShipmentID, id values.PackageID) error {
			deletedID = id
			return nil
		},
	}

	svc := application.NewPackageService(packageRepo, &mockRouteRepository{}, newMockTransactional(), noopPackageEventPublisher{})

	err := svc.DeletePackage(context.Background(), shipmentID, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deletedID != pkg.ID {
		t.Fatal("wrong package deleted")
	}
}

func TestDeletePackage_NotFound(t *testing.T) {
	packageRepo := &mockPackageRepository{
		getPackageByIDFn: func(ctx context.Context, _ values.ShipmentID, _ values.PackageID) (*entities.Package, error) {
			return nil, errors.New("not found")
		},
	}

	svc := application.NewPackageService(packageRepo, &mockRouteRepository{}, newMockTransactional(), noopPackageEventPublisher{})

	err := svc.DeletePackage(context.Background(), values.NewID(), values.NewID())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeletePackage_ShipmentMismatch(t *testing.T) {
	pkg := entities.NewPackage()
	pkg.AssignToShipment(values.NewID())

	packageRepo := &mockPackageRepository{
		getPackageByIDFn: func(ctx context.Context, _ values.ShipmentID, _ values.PackageID) (*entities.Package, error) {
			return nil, errors.New("not found")
		},
	}

	svc := application.NewPackageService(packageRepo, &mockRouteRepository{}, newMockTransactional(), noopPackageEventPublisher{})

	err := svc.DeletePackage(context.Background(), values.NewID(), pkg.ID)
	if err == nil {
		t.Fatal("expected shipment mismatch error")
	}
}

func TestSyncPackagesAtRoutePoint_Success(t *testing.T) {
	shipmentID := values.NewID()
	targetID := values.NewID()
	rpID := values.NewID()

	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: rpID, PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: targetID, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	var savedPkgs []*entities.Package
	packageRepo := &mockPackageRepository{
		getPackagesByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) ([]*entities.Package, error) {
			return nil, nil
		},
		savePackageFn: func(ctx context.Context, pkg *entities.Package) error {
			savedPkgs = append(savedPkgs, pkg)
			return nil
		},
		deletePackageFn: func(ctx context.Context, _ values.ShipmentID, _ values.PackageID) error {
			return nil
		},
	}

	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
	}

	returnProvider := &mockPackageProvider{
		listReturnPackagesFn: func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ReturnPackage, error) {
			return nil, nil
		},
	}

	svc := application.NewPackageSyncService(
		&mockShipmentRepository{},
		routeRepo,
		packageRepo,
		returnProvider,
		newMockTransactional(),
	)

	_, err := svc.SyncPackagesAtRoutePoint(context.Background(), shipmentID, rpID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSyncPackagesAtRoutePoint_EndPoint(t *testing.T) {
	shipmentID := values.NewID()
	targetID := values.NewID()

	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: targetID, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	pkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: targetID,
		Direction:     values.PackageDemandDirectionForward,
	})
	_ = pkg.Loaded()
	_ = pkg.MarkInTransit()

	packageRepo := &mockPackageRepository{
		getPackagesByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{pkg}, nil
		},
		savePackageFn: func(ctx context.Context, p *entities.Package) error {
			return nil
		},
	}

	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
	}

	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()
	_ = shipment.Start()
	_ = shipment.ReserveUnloading()
	_ = shipment.Unloading()
	shipmentRepo := &mockShipmentRepository{
		getShipmentByIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		saveShipmentFn: func(ctx context.Context, s *entities.Shipment) error {
			return nil
		},
	}

	svc := application.NewPackageSyncService(
		shipmentRepo,
		routeRepo,
		packageRepo,
		&mockPackageProvider{},
		newMockTransactional(),
	)

	result, err := svc.SyncPackagesAtRoutePoint(context.Background(), shipmentID, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 package at end point, got %d", len(result))
	}
	if pkg.Status != values.PackageStatusUnloading {
		t.Fatalf("expected unloading status, got %v", pkg.Status)
	}
}

func TestSyncPackagesAtRoutePoint_MidPoint_NewReturns(t *testing.T) {
	shipmentID := values.NewID()
	targetID := values.NewID()
	midID := values.NewID()

	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: midID, PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: targetID, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	var savedPkgs []*entities.Package
	packageRepo := &mockPackageRepository{
		getPackagesByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{}, nil
		},
		savePackageFn: func(ctx context.Context, pkg *entities.Package) error {
			savedPkgs = append(savedPkgs, pkg)
			return nil
		},
	}

	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
	}

	returnID := values.NewID()
	returnProvider := &mockPackageProvider{
		listReturnPackagesFn: func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ReturnPackage, error) {
			return []ports.ReturnPackage{
				{ID: returnID, TareID: 1},
			}, nil
		},
	}

	svc := application.NewPackageSyncService(
		&mockShipmentRepository{},
		routeRepo,
		packageRepo,
		returnProvider,
		newMockTransactional(),
	)

	_, err := svc.SyncPackagesAtRoutePoint(context.Background(), shipmentID, midID)
	if err != nil {
		t.Fatal(err)
	}
	if len(savedPkgs) != 1 {
		t.Fatalf("expected 1 new return package saved, got %d", len(savedPkgs))
	}
	if savedPkgs[0].ID != returnID {
		t.Fatal("wrong return package ID")
	}
	if !savedPkgs[0].IsReturn {
		t.Fatal("expected return package")
	}
}

func TestSyncPackagesAtRoutePoint_RouteNotFound(t *testing.T) {
	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return nil, errors.New("route not found")
		},
	}

	svc := application.NewPackageSyncService(
		&mockShipmentRepository{},
		routeRepo,
		&mockPackageRepository{},
		&mockPackageProvider{},
		newMockTransactional(),
	)

	_, err := svc.SyncPackagesAtRoutePoint(context.Background(), values.NewID(), values.NewID())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSyncPackagesAtRoutePoint_StartPoint_SkipsDeleteWhenNotReadyToLoad(t *testing.T) {
	shipmentID := values.NewID()
	sourceID := values.NewID()
	midID := values.NewID()
	targetID := values.NewID()

	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID, PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: midID, PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: targetID, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	loadedPkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   sourceID,
		UnloadPointID: midID,
		Direction:     values.PackageDemandDirectionForward,
	})
	_ = loadedPkg.Loaded()

	var deleted bool
	packageRepo := &mockPackageRepository{
		getPackagesByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{loadedPkg}, nil
		},
		deletePackageFn: func(ctx context.Context, _ values.ShipmentID, _ values.PackageID) error {
			deleted = true
			return nil
		},
	}

	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
	}

	returnProvider := &mockPackageProvider{
		listForwardPackagesFn: func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ForwardPackage, error) {
			return nil, nil
		},
	}

	svc := application.NewPackageSyncService(
		&mockShipmentRepository{},
		routeRepo,
		packageRepo,
		returnProvider,
		newMockTransactional(),
	)

	_, err := svc.SyncPackagesAtRoutePoint(context.Background(), shipmentID, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected loaded package not to be deleted")
	}
}

func TestSyncPackagesAtRoutePoint_MidPoint_SkipsDeleteWhenNotReadyToLoad(t *testing.T) {
	shipmentID := values.NewID()
	targetID := values.NewID()
	midID := values.NewID()

	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: midID, PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: targetID, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	loadedReturn := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   midID,
		UnloadPointID: targetID,
		Direction:     values.PackageDemandDirectionBackward,
	})
	_ = loadedReturn.Loaded()

	var deleted bool
	packageRepo := &mockPackageRepository{
		getPackagesByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{loadedReturn}, nil
		},
		deletePackageFn: func(ctx context.Context, _ values.ShipmentID, _ values.PackageID) error {
			deleted = true
			return nil
		},
	}

	routeRepo := &mockRouteRepository{
		getRouteByShipmentIDFn: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
	}

	returnProvider := &mockPackageProvider{
		listReturnPackagesFn: func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ReturnPackage, error) {
			return nil, nil
		},
	}

	svc := application.NewPackageSyncService(
		&mockShipmentRepository{},
		routeRepo,
		packageRepo,
		returnProvider,
		newMockTransactional(),
	)

	_, err := svc.SyncPackagesAtRoutePoint(context.Background(), shipmentID, midID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected loaded return package not to be deleted")
	}
}
