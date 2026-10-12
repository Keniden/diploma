package application

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

// AssemblerService builds shipments: driver preconditions and the shipment
// aggregate (shipment + route from template) inside one transaction.
type AssemblerService struct {
	shipments              ports.ShipmentRepository
	shipmentPolicy         *policies.ShipmentPolicy
	routes                 ports.RouteRepository
	packages               ports.PackageRepository
	shipmentEventPublisher ports.ShipmentEventPublisher
	tx                     ports.Transactional
}

// NewAssembler builds the shipment assembler.
func NewAssembler(
	shipments ports.ShipmentRepository,
	shipmentPolicy *policies.ShipmentPolicy,
	routes ports.RouteRepository,
	packages ports.PackageRepository,
	shipmentEventPublisher ports.ShipmentEventPublisher,
	tx ports.Transactional,
) *AssemblerService {
	return &AssemblerService{
		shipments:              shipments,
		shipmentPolicy:         shipmentPolicy,
		routes:                 routes,
		packages:               packages,
		shipmentEventPublisher: shipmentEventPublisher,
		tx:                     tx,
	}
}

// DriverCanCreateShipment checks that the driver is allowed to create a new
// shipment (no active shipments yet).
func (s *AssemblerService) DriverCanCreateShipment(
	ctx context.Context,
	driver values.Driver,
) error {
	if err := s.shipmentPolicy.CanCreateShipmentByDriverID(ctx, driver.ID); err != nil {
		return fmt.Errorf("can create shipment by driver id: %w", err)
	}

	return nil
}

// CreateShipmentFromRouteTemplate builds a new shipment with a route
// assembled from the template, persists both aggregates and publishes the
// ShipmentCreated event, all inside one transaction.
func (s *AssemblerService) CreateShipmentFromRouteTemplate(
	ctx context.Context,
	template values.RouteTemplate,
) (values.ShipmentID, error) {
	shipment := entities.NewShipment()

	route, err := entities.NewRouteFromTemplate(template)
	if err != nil {
		return values.NilID, fmt.Errorf("new route from template: %w", err)
	}

	if err = s.shipmentPolicy.CanCreateShipment(ctx, route); err != nil {
		return values.NilID, fmt.Errorf("can create shipment: %w", err)
	}

	route.AssignToShipment(shipment.ID)

	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if saveErr := s.shipments.SaveShipment(txCtx, shipment); saveErr != nil {
			return fmt.Errorf("save shipment: %w", saveErr)
		}

		if saveErr := s.routes.SaveRoute(txCtx, route); saveErr != nil {
			return fmt.Errorf("save route: %w", saveErr)
		}

		ev, evErr := events.NewShipmentCreatedEvent(shipment, route)
		if evErr != nil {
			return fmt.Errorf("new shipment created event: %w", evErr)
		}

		return s.shipmentEventPublisher.PublishShipmentEvent(txCtx, ev)
	})
	if err != nil {
		return values.NilID, err
	}

	return shipment.ID, nil
}
