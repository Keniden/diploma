package application_test

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

func testRouteTemplate() values.RouteTemplate {
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

func TestDriverCanCreateShipment_Allowed(t *testing.T) {
	policyRepo := &mockShipmentPolicyRepository{
		getShipmentCountByDriverFn: func(_ context.Context, _ values.DriverID, _ ...values.ShipmentStatus) (int, error) {
			return 0, nil
		},
	}
	assembler := application.NewAssembler(
		&mockShipmentRepository{},
		policies.NewShipmentPolicy(policyRepo),
		&mockRouteRepository{},
		&mockPackageRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	if err := assembler.DriverCanCreateShipment(context.Background(), driver); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDriverCanCreateShipment_Blocked(t *testing.T) {
	policyRepo := &mockShipmentPolicyRepository{
		getShipmentCountByDriverFn: func(_ context.Context, _ values.DriverID, _ ...values.ShipmentStatus) (int, error) {
			return 1, nil
		},
	}
	assembler := application.NewAssembler(
		&mockShipmentRepository{},
		policies.NewShipmentPolicy(policyRepo),
		&mockRouteRepository{},
		&mockPackageRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	if err := assembler.DriverCanCreateShipment(context.Background(), driver); err == nil {
		t.Fatal("expected active shipment error")
	}
}

func TestCreateShipmentFromRouteTemplate_Success(t *testing.T) {
	var savedShipmentID values.ShipmentID
	var savedRouteShipmentID values.ShipmentID
	var publishedEvent events.ShipmentEvent

	shipmentRepo := &mockShipmentRepository{
		saveShipmentFn: func(_ context.Context, shipment *entities.Shipment) error {
			savedShipmentID = shipment.ID
			return nil
		},
	}
	routeRepo := &mockRouteRepository{
		saveRouteFn: func(_ context.Context, route *entities.Route) error {
			savedRouteShipmentID = route.ShipmentID
			return nil
		},
	}
	publisher := &mockShipmentEventPublisher{
		publishFn: func(_ context.Context, event events.ShipmentEvent) error {
			publishedEvent = event
			return nil
		},
	}
	policyRepo := &mockShipmentPolicyRepository{
		hasSourcePackagesFn: func(_ context.Context, _ *entities.Route) (bool, error) {
			return true, nil
		},
	}

	assembler := application.NewAssembler(
		shipmentRepo,
		policies.NewShipmentPolicy(policyRepo),
		routeRepo,
		&mockPackageRepository{},
		publisher,
		newMockTransactional(),
	)

	id, err := assembler.CreateShipmentFromRouteTemplate(context.Background(), testRouteTemplate())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != savedShipmentID {
		t.Fatal("saved shipment id mismatch")
	}
	if savedRouteShipmentID != id {
		t.Fatal("route not assigned to the created shipment")
	}

	created, ok := publishedEvent.(events.ShipmentCreatedEvent)
	if !ok {
		t.Fatalf("expected ShipmentCreatedEvent, got %T", publishedEvent)
	}
	if created.ShipmentID != id {
		t.Fatal("event shipment id mismatch")
	}
	if created.SourcePointID != values.PointID(101) {
		t.Fatal("event source point mismatch")
	}
}

func TestCreateShipmentFromRouteTemplate_NoSourcePackages(t *testing.T) {
	policyRepo := &mockShipmentPolicyRepository{
		hasSourcePackagesFn: func(_ context.Context, _ *entities.Route) (bool, error) {
			return false, nil
		},
	}
	assembler := application.NewAssembler(
		&mockShipmentRepository{},
		policies.NewShipmentPolicy(policyRepo),
		&mockRouteRepository{},
		&mockPackageRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	_, err := assembler.CreateShipmentFromRouteTemplate(context.Background(), testRouteTemplate())
	if err == nil {
		t.Fatal("expected no source packages error")
	}
}

func TestCreateShipmentFromRouteTemplate_TooShortRoute(t *testing.T) {
	shortTemplate := values.RouteTemplate{
		RouteTemplateID: values.RouteTemplateID(2),
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: values.PointID(101), SeqNum: 0},
		},
	}

	assembler := application.NewAssembler(
		&mockShipmentRepository{},
		policies.NewShipmentPolicy(&mockShipmentPolicyRepository{}),
		&mockRouteRepository{},
		&mockPackageRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	_, err := assembler.CreateShipmentFromRouteTemplate(context.Background(), shortTemplate)
	if err == nil {
		t.Fatal("expected route too short error")
	}
}

func TestCreateShipmentFromRouteTemplate_SaveError(t *testing.T) {
	shipmentRepo := &mockShipmentRepository{
		saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error {
			return errors.New("db down")
		},
	}
	assembler := application.NewAssembler(
		shipmentRepo,
		policies.NewShipmentPolicy(&mockShipmentPolicyRepository{}),
		&mockRouteRepository{},
		&mockPackageRepository{},
		&mockShipmentEventPublisher{},
		newMockTransactional(),
	)

	if _, err := assembler.CreateShipmentFromRouteTemplate(context.Background(), testRouteTemplate()); err == nil {
		t.Fatal("expected save error")
	}
}