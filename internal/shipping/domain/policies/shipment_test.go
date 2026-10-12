package policies_test

import (
	"context"
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

type mockRepo struct {
	getShipmentByID           func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	getRouteByShipmentID      func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	getShipmentCountByDriver  func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error)
	getPackageCountByShipment func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error)
	getPackagesByShipmentID   func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getReserveCountByShipment func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error)
	hasSourcePackages         func(ctx context.Context, route *entities.Route) (bool, error)
}

func (m *mockRepo) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	return m.getShipmentByID(ctx, id)
}

func (m *mockRepo) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	return m.getRouteByShipmentID(ctx, id)
}

func (m *mockRepo) GetShipmentCountByDriverID(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
	return m.getShipmentCountByDriver(ctx, driverID, statuses...)
}

func (m *mockRepo) GetPackageCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
	return m.getPackageCountByShipment(ctx, shipmentID, statuses...)
}

func (m *mockRepo) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	if m.getPackagesByShipmentID == nil {
		return nil, nil
	}
	return m.getPackagesByShipmentID(ctx, shipmentID)
}

func (m *mockRepo) GetReserveCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
	return m.getReserveCountByShipment(ctx, shipmentID, statuses...)
}

func (m *mockRepo) HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error) {
	if m.hasSourcePackages == nil {
		return true, nil
	}
	return m.hasSourcePackages(ctx, route)
}

func testRoute() *entities.Route {
	route, err := entities.NewRouteFromTemplate(values.RouteTemplate{
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: 1, SeqNum: 1},
			{RoutePointTemplateID: 2, SeqNum: 2},
			{RoutePointTemplateID: 3, SeqNum: 3},
		},
	})
	if err != nil {
		panic(err)
	}
	return route
}

func TestCanCreateShipment_HasSourcePackages(t *testing.T) {
	repo := &mockRepo{
		hasSourcePackages: func(ctx context.Context, route *entities.Route) (bool, error) {
			return true, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCreateShipment(context.Background(), testRoute())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanCreateShipment_NoSourcePackages(t *testing.T) {
	repo := &mockRepo{
		hasSourcePackages: func(ctx context.Context, route *entities.Route) (bool, error) {
			return false, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCreateShipment(context.Background(), testRoute())
	if err == nil {
		t.Fatal("expected error when route has no source packages")
	}
}

func TestCanCreateShipmentByDriverID_NoActiveShipments(t *testing.T) {
	repo := &mockRepo{
		getShipmentCountByDriver: func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCreateShipmentByDriverID(context.Background(), values.NewID())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanCreateShipmentByDriverID_HasActiveShipments(t *testing.T) {
	repo := &mockRepo{
		getShipmentCountByDriver: func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
			return 2, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCreateShipmentByDriverID(context.Background(), values.NewID())
	if err == nil {
		t.Fatal("expected error for driver with active shipments")
	}
}

func TestCanStartShipmentByID_Success(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return &entities.Route{
				Points: []*entities.RoutePoint{
					{ID: values.NewID()},
					{ID: values.NewID()},
					{ID: values.NewID()},
				},
			}, nil
		},
		getPackageCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
			return 5, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanStartShipmentByID(context.Background(), shipment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanStartShipmentByID_NotLoading(t *testing.T) {
	shipment := entities.NewShipment()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanStartShipmentByID(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected error for shipment not in loading status")
	}
}

func TestCanStartShipmentByID_TooShortRoute(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.Loading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return &entities.Route{
				Points: []*entities.RoutePoint{
					{ID: values.NewID()},
				},
			}, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanStartShipmentByID(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestCanStartShipmentByID_InTransitPackages(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return &entities.Route{
				Points: []*entities.RoutePoint{
					{ID: values.NewID()},
					{ID: values.NewID()},
					{ID: values.NewID()},
				},
			}, nil
		},
		getPackageCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
			if len(statuses) != 2 ||
				statuses[0] != values.PackageStatusLoaded ||
				statuses[1] != values.PackageStatusInTransit {
				t.Fatalf("unexpected statuses: %v", statuses)
			}
			return 1, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanStartShipmentByID(context.Background(), shipment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanStartShipmentByID_NoPackages(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return &entities.Route{
				Points: []*entities.RoutePoint{
					{ID: values.NewID()},
					{ID: values.NewID()},
					{ID: values.NewID()},
				},
			}, nil
		},
		getPackageCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanStartShipmentByID(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected error for no loaded packages")
	}
}

func TestCanProceedToLoadingByID_HasActiveReserve(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 1, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanProceedToLoadingByID(context.Background(), values.NewID())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanProceedToLoadingByID_NoActiveReserve(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanProceedToLoadingByID(context.Background(), values.NewID())
	if err == nil {
		t.Fatal("expected error for no active reserve")
	}
}

func TestCanCompleteShipmentByID_Success(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()
	_ = shipment.Start()
	_ = shipment.ReserveUnloading()
	_ = shipment.Unloading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getPackageCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCompleteShipmentByID(context.Background(), shipment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanCompleteShipmentByID_InProgressSuccess(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()
	_ = shipment.Start()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getPackageCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCompleteShipmentByID(context.Background(), shipment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanCompleteShipmentByID_InvalidStatus(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCompleteShipmentByID(context.Background(), shipment.ID)
	if err == nil {
		t.Fatal("expected error for invalid shipment status")
	}
}

func TestCanCompleteShipmentByID_HasPendingUnloadActions(t *testing.T) {
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()
	_ = shipment.Start()

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getPackagesByShipmentID: func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{
				{
					Actions: []*entities.PackageAction{
						{
							Type:   values.RoutePointActionTypeLoad,
							Status: values.RoutePointPackageStatusDone,
						},
						{
							Type:   values.RoutePointActionTypeUnload,
							Status: values.RoutePointPackageStatusTODO,
						},
					},
				},
			}, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanCompleteShipmentByID(context.Background(), shipment.ID)
	if !lmterrors.Is(err, lmterrors.InvalidPackagePendingUnloadCode) {
		t.Fatalf("expected lmt_invalid_package_pending_unload, got %v", err)
	}
}

func TestCanReserveGateByShipmentID_Success(t *testing.T) {
	sourceID := values.NewID()
	targetID := values.NewID()
	shipmentID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: values.NewID()},
			{ID: targetID},
		},
	}
	shipment := entities.NewShipment()
	shipment.ID = shipmentID
	_ = shipment.AssignDriver(values.NewID(), 42)

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanReserveGate(context.Background(), shipmentID, sourceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanReserveGate_SourcePoint_NotAssigned(t *testing.T) {
	sourceID := values.NewID()
	targetID := values.NewID()
	shipmentID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: values.NewID()},
			{ID: targetID},
		},
	}
	shipment := entities.NewShipment()
	shipment.ID = shipmentID

	repo := &mockRepo{
		getShipmentByID: func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			return shipment, nil
		},
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanReserveGate(context.Background(), shipmentID, sourceID)
	if err == nil {
		t.Fatal("expected error for shipment not in assigned status")
	}
}

func TestCanReserveGateByShipmentID_TargetPoint(t *testing.T) {
	sourceID := values.NewID()
	targetID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: values.NewID()},
			{ID: targetID},
		},
	}

	repo := &mockRepo{
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanReserveGate(context.Background(), values.NewID(), targetID)
	if err == nil {
		t.Fatal("expected error for target route point")
	}
}

func TestCanReserveGateByShipmentID_MiddlePoint(t *testing.T) {
	sourceID := values.NewID()
	middleID := values.NewID()
	targetID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: middleID},
			{ID: targetID},
		},
	}

	repo := &mockRepo{
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 0, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanReserveGate(context.Background(), values.NewID(), middleID)
	if err == nil {
		t.Fatal("expected error for middle route point")
	}
}

func TestCanReserveGate_AlreadyHasReserve(t *testing.T) {
	repo := &mockRepo{
		getReserveCountByShipment: func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
			return 1, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanReserveGate(context.Background(), values.NewID(), values.NewID())
	if err == nil {
		t.Fatal("expected error for existing reserve")
	}
}

func TestCanProceedToUnloadingOnArrivalByID_Success(t *testing.T) {
	sourceID := values.NewID()
	targetID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: values.NewID()},
			{ID: targetID},
		},
	}

	repo := &mockRepo{
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getPackagesByShipmentID: func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{
				{
					Actions: []*entities.PackageAction{
						{
							Type:   values.RoutePointActionTypeLoad,
							Status: values.RoutePointPackageStatusDone,
						},
						{
							RoutePointID: targetID,
							Type:         values.RoutePointActionTypeUnload,
							Status:       values.RoutePointPackageStatusTODO,
						},
					},
				},
			}, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanProceedToUnloadingOnArrivalByID(context.Background(), values.NewID())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCanProceedToUnloadingOnArrivalByID_NoLoadedPackageAtTarget(t *testing.T) {
	sourceID := values.NewID()
	middleID := values.NewID()
	targetID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: middleID},
			{ID: targetID},
		},
	}

	repo := &mockRepo{
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getPackagesByShipmentID: func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{
				{
					Actions: []*entities.PackageAction{
						{
							RoutePointID: sourceID,
							Type:         values.RoutePointActionTypeLoad,
							Status:       values.RoutePointPackageStatusDone,
						},
						{
							RoutePointID: middleID,
							Type:         values.RoutePointActionTypeUnload,
							Status:       values.RoutePointPackageStatusDone,
						},
					},
				},
			}, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanProceedToUnloadingOnArrivalByID(context.Background(), values.NewID())
	if err == nil {
		t.Fatal("expected error when target has no loaded packages")
	}
}

func TestCanProceedToUnloadingOnArrivalByID_PendingUnloadAtOtherRoutePoint(t *testing.T) {
	sourceID := values.NewID()
	middleID := values.NewID()
	targetID := values.NewID()
	route := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: sourceID},
			{ID: middleID},
			{ID: targetID},
		},
	}

	repo := &mockRepo{
		getRouteByShipmentID: func(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
			return route, nil
		},
		getPackagesByShipmentID: func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
			return []*entities.Package{
				{
					Actions: []*entities.PackageAction{
						{
							Type:   values.RoutePointActionTypeLoad,
							Status: values.RoutePointPackageStatusDone,
						},
						{
							RoutePointID: targetID,
							Type:         values.RoutePointActionTypeUnload,
							Status:       values.RoutePointPackageStatusTODO,
						},
					},
				},
				{
					Actions: []*entities.PackageAction{
						{
							Type:   values.RoutePointActionTypeLoad,
							Status: values.RoutePointPackageStatusDone,
						},
						{
							RoutePointID: middleID,
							Type:         values.RoutePointActionTypeUnload,
							Status:       values.RoutePointPackageStatusTODO,
						},
					},
				},
			}, nil
		},
	}

	policy := policies.NewShipmentPolicy(repo)
	err := policy.CanProceedToUnloadingOnArrivalByID(context.Background(), values.NewID())
	if err == nil {
		t.Fatal("expected error for pending unload at another route point")
	}
}
