package application

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

// ShipmentProgressService advances the shipment aggregate: driver/transport
// assignment and cancellation, each transition inside its own transaction
// with the corresponding domain event.
type ShipmentProgressService struct {
	shipmentPolicy         *policies.ShipmentPolicy
	shipments              ports.ShipmentRepository
	shipmentEventPublisher ports.ShipmentEventPublisher
	tx                     ports.Transactional
}

// NewShipmentProgressService builds the shipment progress service.
func NewShipmentProgressService(
	shipmentPolicy *policies.ShipmentPolicy,
	shipments ports.ShipmentRepository,
	shipmentEventPublisher ports.ShipmentEventPublisher,
	tx ports.Transactional,
) *ShipmentProgressService {
	return &ShipmentProgressService{
		shipmentPolicy:         shipmentPolicy,
		shipments:              shipments,
		shipmentEventPublisher: shipmentEventPublisher,
		tx:                     tx,
	}
}

// AssignDriverByShipmentID assigns the driver (and the supplier from the
// driver profile) to the shipment and publishes the ShipmentDriverAssign
// event inside one transaction.
func (s *ShipmentProgressService) AssignDriverByShipmentID(
	ctx context.Context,
	id values.ShipmentID,
	driver values.Driver,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		shipment, err := s.shipments.GetShipmentByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		if err := shipment.AssignDriver(driver.ID, driver.SupplierID); err != nil {
			return fmt.Errorf("assign driver: %w", err)
		}

		if err := s.shipments.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		return s.shipmentEventPublisher.PublishShipmentEvent(
			txCtx,
			events.ShipmentDriverAssignEvent{
				ID:     shipment.ID,
				Driver: driver,
			},
		)
	})
}

// AssignTransportByShipmentID sets the vehicle license plate on the shipment
// and publishes the ShipmentTransportAssign event inside one transaction.
func (s *ShipmentProgressService) AssignTransportByShipmentID(
	ctx context.Context,
	id values.ShipmentID,
	vehicle values.Vehicle,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		shipment, err := s.shipments.GetShipmentByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		if err := shipment.SetLicensePLate(vehicle.LicensePlate); err != nil {
			return fmt.Errorf("set license plate: %w", err)
		}

		if err := s.shipments.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		return s.shipmentEventPublisher.PublishShipmentEvent(
			txCtx,
			events.ShipmentTransportAssignEvent{
				ID:      shipment.ID,
				Vehicle: vehicle,
			},
		)
	})
}

// CompleteShipmentByID completes a shipment: it loads the aggregate, verifies
// the completion policy (no pending unload actions, shipment in an
// allowed status), moves the shipment to COMPLETED and publishes the
// ShipmentCompleted event inside one transaction.
func (s *ShipmentProgressService) CompleteShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.shipmentPolicy.CanCompleteShipmentByID(txCtx, id); err != nil {
			return fmt.Errorf("can complete shipment: %w", err)
		}

		shipment, err := s.shipments.GetShipmentByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		if err := shipment.Complete(); err != nil {
			return fmt.Errorf("complete shipment: %w", err)
		}

		if err := s.shipments.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		return s.shipmentEventPublisher.PublishShipmentEvent(
			txCtx,
			events.ShipmentCompletedEvent{ID: shipment.ID},
		)
	})
}

// CancelShipmentByID cancels the shipment inside a transaction and publishes
// the ShipmentCanceled event afterwards.
func (s *ShipmentProgressService) CancelShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	if err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		shipment, err := s.shipments.GetShipmentByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		if err := shipment.Cancel(); err != nil {
			return fmt.Errorf("cancel shipment: %w", err)
		}

		if err := s.shipments.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return s.shipmentEventPublisher.PublishShipmentEvent(ctx, events.ShipmentCanceledEvent{ID: id})
}
