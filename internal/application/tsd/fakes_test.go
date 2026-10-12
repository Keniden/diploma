package tsd

import (
	"context"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
	"last-mile-task/pkg/qr"
)

// fakeHandshake is a QRHandshake stub that returns a fixed session or error.
type fakeHandshake struct {
	session qr.Session
	err     error
}

func (f *fakeHandshake) DecodeQR(_ string) (qr.Session, error) {
	return f.session, f.err
}

// fakeScanner is a Scanner stub that records calls and returns a scripted error
// per package id (errByID) or a global error (err).
type fakeScanner struct {
	shipmentID values.ShipmentID
	packages   []scan.Package
	err        error
	errByID    map[string]error
	calls      int
	calledFor  []values.PackageID
}

func (f *fakeScanner) ScanPackages(_ context.Context, shipmentID values.ShipmentID, pkgs []scan.Package) error {
	f.calls++
	f.shipmentID = shipmentID
	f.packages = pkgs
	f.calledFor = append(f.calledFor, scan.PackageIDs(pkgs)...)
	if len(pkgs) == 1 {
		if err, ok := f.errByID[pkgs[0].ID.String()]; ok {
			return err
		}
	}
	return f.err
}

// fakeReserveApprover is a ReserveApprover stub.
type fakeReserveApprover struct {
	err                error
	approvedShipment   values.ShipmentID
	approvedRoutePoint values.RoutePointID
}

func (f *fakeReserveApprover) ApproveReserveByRoutePointID(
	_ context.Context, shipmentID values.ShipmentID, routePointID values.RoutePointID,
) error {
	f.approvedShipment = shipmentID
	f.approvedRoutePoint = routePointID
	return f.err
}

// fakeRouteRepository is a RouteRepository stub.
type fakeRouteRepository struct {
	route   *entities.Route
	errLoad error
	errSave error
	saved   *entities.Route
}

func (f *fakeRouteRepository) GetRouteByShipmentID(_ context.Context, shipmentID values.ShipmentID) (*entities.Route, error) {
	return f.route, f.errLoad
}

func (f *fakeRouteRepository) SaveRoute(_ context.Context, route *entities.Route) error {
	f.saved = route
	return f.errSave
}

// fakePackagesToScan is a PackagesToScanLister stub.
type fakePackagesToScan struct {
	packages        []PackageToScan
	err             error
	askedShipment   values.ShipmentID
	askedRoutePoint values.RoutePointID
}

func (f *fakePackagesToScan) ListPackagesToScan(
	_ context.Context, shipmentID values.ShipmentID, routePointID values.RoutePointID,
) ([]PackageToScan, error) {
	f.askedShipment = shipmentID
	f.askedRoutePoint = routePointID
	return f.packages, f.err
}

func testShipmentID() values.ShipmentID {
	return values.ShipmentID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
}

func testRoutePointID() values.RoutePointID {
	return values.RoutePointID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
}

func testPackageID() values.PackageID {
	return values.PackageID(uuid.MustParse("33333333-3333-3333-3333-333333333333"))
}
