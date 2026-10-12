//nolint:unused // scaffolding (mocks/helpers) kept for unported services/suites.
package application_test

import (
	"context"
	"time"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

// ==== Transactional ====

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

// ==== ShipmentRepository ====

type mockShipmentRepository struct {
	getShipmentByIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	saveShipmentFn    func(ctx context.Context, shipment *entities.Shipment) error
}

func (m *mockShipmentRepository) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	return m.getShipmentByIDFn(ctx, id)
}

func (m *mockShipmentRepository) SaveShipment(ctx context.Context, shipment *entities.Shipment) error {
	return m.saveShipmentFn(ctx, shipment)
}

// ==== RouteRepository ====

type mockRouteRepository struct {
	getRouteByIDFn         func(ctx context.Context, id values.RouteID) (*entities.Route, error)
	getRouteByShipmentIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	saveRouteFn            func(ctx context.Context, route *entities.Route) error
}

func (m *mockRouteRepository) GetRouteByID(ctx context.Context, id values.RouteID) (*entities.Route, error) {
	return m.getRouteByIDFn(ctx, id)
}

func (m *mockRouteRepository) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	return m.getRouteByShipmentIDFn(ctx, id)
}

func (m *mockRouteRepository) SaveRoute(ctx context.Context, route *entities.Route) error {
	return m.saveRouteFn(ctx, route)
}

// ==== PackageRepository ====

type mockPackageRepository struct {
	getPackageByIDFn                func(ctx context.Context, shipmentID values.ShipmentID, id values.PackageID) (*entities.Package, error)
	getPackagesByShipmentIDFn       func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getLoadedPackagesByShipmentIDFn func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getPackagesByIDsFn              func(ctx context.Context, shipmentID values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error)
	savePackageFn                   func(ctx context.Context, pkg *entities.Package) error
	deletePackageFn                 func(ctx context.Context, shipmentID values.ShipmentID, id values.PackageID) error
}

func (m *mockPackageRepository) GetPackageByID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) (*entities.Package, error) {
	return m.getPackageByIDFn(ctx, shipmentID, id)
}

func (m *mockPackageRepository) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	return m.getPackagesByShipmentIDFn(ctx, shipmentID)
}

func (m *mockPackageRepository) GetLoadedPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	return m.getLoadedPackagesByShipmentIDFn(ctx, shipmentID)
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
	return nil
}

func (m *mockPackageRepository) DeletePackage(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) error {
	return m.deletePackageFn(ctx, shipmentID, id)
}

// ==== ReserveRepository ====

type mockReserveRepository struct {
	getReserveByIDFn                     func(ctx context.Context, id values.ReserveID) (*entities.Reserve, error)
	getReserveByShipmentIDFn             func(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error)
	getReserveByRoutePointIDFn           func(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error)
	listExpiredWaitingApprovalReservesFn func(ctx context.Context, limit int) ([]*entities.Reserve, error)
	listExpiredActiveReservesFn          func(ctx context.Context, limit int) ([]*entities.Reserve, error)
	saveReserveFn                        func(ctx context.Context, reserve *entities.Reserve) error
}

func (m *mockReserveRepository) GetReserveByID(ctx context.Context, id values.ReserveID) (*entities.Reserve, error) {
	if m.getReserveByIDFn != nil {
		return m.getReserveByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReserveRepository) GetReserveByShipmentID(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error) {
	return m.getReserveByShipmentIDFn(ctx, shipmentID)
}

func (m *mockReserveRepository) GetReserveByRoutePointID(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error) {
	return m.getReserveByRoutePointIDFn(ctx, routePointID)
}

func (m *mockReserveRepository) ListExpiredWaitingApprovalReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	if m.listExpiredWaitingApprovalReservesFn != nil {
		return m.listExpiredWaitingApprovalReservesFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockReserveRepository) ListExpiredActiveReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	if m.listExpiredActiveReservesFn != nil {
		return m.listExpiredActiveReservesFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockReserveRepository) SaveReserve(ctx context.Context, reserve *entities.Reserve) error {
	return m.saveReserveFn(ctx, reserve)
}

// ==== Event Publishers ====

type mockShipmentEventPublisher struct {
	publishFn func(ctx context.Context, event events.ShipmentEvent) error
}

func (m *mockShipmentEventPublisher) PublishShipmentEvent(ctx context.Context, event events.ShipmentEvent) error {
	return m.publishFn(ctx, event)
}

type mockReserveEventPublisher struct {
	publishFn func(ctx context.Context, event events.ReserveEvent) error
}

func (m *mockReserveEventPublisher) PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error {
	return m.publishFn(ctx, event)
}

type mockRouteEventPublisher struct {
	routeEvents []events.RouteEvent
	err         error
}

func (m *mockRouteEventPublisher) PublishRouteEvent(ctx context.Context, event events.RouteEvent) error {
	if m.err != nil {
		return m.err
	}
	m.routeEvents = append(m.routeEvents, event)
	return nil
}

// mockShipmentCompleter records CompleteShipmentByID calls.
type mockShipmentCompleter struct {
	calls []values.ShipmentID
	err   error
}

func (m *mockShipmentCompleter) CompleteShipmentByID(_ context.Context, id values.ShipmentID) error {
	if m.err != nil {
		return m.err
	}
	m.calls = append(m.calls, id)
	return nil
}

// ==== ReserveManager ====

type mockReserveManager struct {
	reserveOrQueueFn     func(ctx context.Context, shipment *entities.Shipment, source *entities.RoutePoint, targets []*entities.RoutePoint) (*entities.Reserve, error)
	cancelReserveFn      func(ctx context.Context, reserve *entities.Reserve, pointID values.PointID) error
	approveReserveTaskFn func(ctx context.Context, task values.ReserveTask) (time.Time, error)
}

func (m *mockReserveManager) ReserveOrQueue(ctx context.Context, shipment *entities.Shipment, source *entities.RoutePoint, targets []*entities.RoutePoint) (*entities.Reserve, error) {
	return m.reserveOrQueueFn(ctx, shipment, source, targets)
}

func (m *mockReserveManager) CancelReserve(ctx context.Context, reserve *entities.Reserve, pointID values.PointID) error {
	if m.cancelReserveFn != nil {
		return m.cancelReserveFn(ctx, reserve, pointID)
	}
	return nil
}

func (m *mockReserveManager) ApproveReserveTask(ctx context.Context, task values.ReserveTask) (time.Time, error) {
	if m.approveReserveTaskFn != nil {
		return m.approveReserveTaskFn(ctx, task)
	}
	return time.Now().Add(time.Hour), nil
}

// ==== PackageProvider ====

type mockPackageProvider struct {
	listReturnPackagesFn  func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ReturnPackage, error)
	listForwardPackagesFn func(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ForwardPackage, error)
	getTareStickerFn      func(ctx context.Context, packageID values.PackageID) (string, error)
}

func (m *mockPackageProvider) ListReturnPackages(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ReturnPackage, error) {
	if m.listReturnPackagesFn != nil {
		return m.listReturnPackagesFn(ctx, srcPointID, dstPointIDs)
	}
	return nil, nil
}

func (m *mockPackageProvider) ListForwardPackages(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ports.ForwardPackage, error) {
	if m.listForwardPackagesFn != nil {
		return m.listForwardPackagesFn(ctx, srcPointID, dstPointIDs)
	}
	return nil, nil
}

func (m *mockPackageProvider) GetTareSticker(ctx context.Context, packageID values.PackageID) (string, error) {
	if m.getTareStickerFn != nil {
		return m.getTareStickerFn(ctx, packageID)
	}
	return "", nil
}

// ==== ShipmentPolicyRepository (for wiring up policies.ShipmentPolicy) ====

type mockShipmentPolicyRepository struct {
	getShipmentByIDFn           func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	getRouteByShipmentIDFn      func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	getShipmentCountByDriverFn  func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error)
	getPackageCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error)
	getPackagesByShipmentIDFn   func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getReserveCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error)
	hasSourcePackagesFn         func(ctx context.Context, route *entities.Route) (bool, error)
}

func (m *mockShipmentPolicyRepository) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	return m.getShipmentByIDFn(ctx, id)
}

func (m *mockShipmentPolicyRepository) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	return m.getRouteByShipmentIDFn(ctx, id)
}

func (m *mockShipmentPolicyRepository) GetShipmentCountByDriverID(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
	return m.getShipmentCountByDriverFn(ctx, driverID, statuses...)
}

func (m *mockShipmentPolicyRepository) GetPackageCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
	return m.getPackageCountByShipmentFn(ctx, shipmentID, statuses...)
}

func (m *mockShipmentPolicyRepository) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	if m.getPackagesByShipmentIDFn == nil {
		return nil, nil
	}
	return m.getPackagesByShipmentIDFn(ctx, shipmentID)
}

func (m *mockShipmentPolicyRepository) GetReserveCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
	return m.getReserveCountByShipmentFn(ctx, shipmentID, statuses...)
}

func (m *mockShipmentPolicyRepository) HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error) {
	if m.hasSourcePackagesFn == nil {
		return true, nil
	}
	return m.hasSourcePackagesFn(ctx, route)
}

func newShipmentPolicy(repo policies.ShipmentPolicyRepository) *policies.ShipmentPolicy {
	return policies.NewShipmentPolicy(repo)
}
