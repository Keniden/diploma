package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/clients/payments"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

// CargoShipment is the seam to the cargo-shipment service: shipment lifecycle
// events (created, driver/transport assigned, completed) are propagated
// synchronously inside the request transaction.
//
// TODO(LASTMILE-112): the real implementation is the cargo-shipment gRPC
// client (internal/clients/cargo-shipment); the skeleton wires a logging stub
// until the client is ported.
type CargoShipment interface {
	CreateShipment(ctx context.Context, req CargoShipmentCreateRequest) error
	SetTransport(ctx context.Context, req CargoShipmentSetTransportRequest) error
	DriveShipment(ctx context.Context, req CargoShipmentDriveShipmentRequest) error
}

// CargoShipmentCreateRequest mirrors the cargo-shipment CreateShipment call.
type CargoShipmentCreateRequest struct {
	ShipmentID         values.ShipmentID
	RouteID            values.RouteID
	SourceRoutePointID values.RoutePointID
	SourcePointID      int64
}

// CargoShipmentSetTransportRequest mirrors the cargo-shipment SetTransport
// call for both driver and vehicle assignment (only one of the two pairs is
// set per call).
type CargoShipmentSetTransportRequest struct {
	ShipmentID    values.ShipmentID
	DriverID      string
	CarrierID     string
	VehicleID     int64
	VehicleNumber string
}

// CargoShipmentDriveShipmentRequest mirrors the cargo-shipment DriveShipment
// call, carrying the trigger to run (START / FINISH).
type CargoShipmentDriveShipmentRequest struct {
	ShipmentID values.ShipmentID
	Trigger    string
}

// DriveShipment triggers, matching the cargo-shipment contract.
const (
	CargoShipmentDriveShipmentTriggerStart  = "START"
	CargoShipmentDriveShipmentTriggerFinish = "FINISH"
)

// ShipmentEventHandler propagates shipment domain events to the
// cargo-shipment service and writes the payments accrual outbox message.
type ShipmentEventHandler struct {
	cargoShipment  CargoShipment
	paymentsWriter *payments_outbox.Writer
}

// NewShipmentEventHandler builds the shipment event handler. The payments
// writer is optional (nil keeps the handler payments-agnostic for the
// non-end-shipment lifecycle events).
func NewShipmentEventHandler(
	cargoShipment CargoShipment,
	paymentsWriters ...*payments_outbox.Writer,
) *ShipmentEventHandler {
	var paymentsWriter *payments_outbox.Writer
	if len(paymentsWriters) > 0 {
		paymentsWriter = paymentsWriters[0]
	}

	return &ShipmentEventHandler{
		cargoShipment:  cargoShipment,
		paymentsWriter: paymentsWriter,
	}
}

// HandleShipmentCreatedEvent notifies cargo-shipment about a new shipment.
func (h *ShipmentEventHandler) HandleShipmentCreatedEvent(
	ctx context.Context,
	event events.ShipmentCreatedEvent,
) error {
	err := h.cargoShipment.CreateShipment(
		ctx,
		CargoShipmentCreateRequest{
			ShipmentID:         event.ShipmentID,
			RouteID:            event.RouteID,
			SourceRoutePointID: event.SourceRoutePointID,
			SourcePointID:      int64(event.SourcePointID),
		},
	)
	if err != nil {
		return fmt.Errorf("create shipment with carrier: %w", err)
	}

	return nil
}

// HandleShipmentDriverAssignEvent propagates the assigned driver to
// cargo-shipment.
func (h *ShipmentEventHandler) HandleShipmentDriverAssignEvent(
	ctx context.Context,
	event events.ShipmentDriverAssignEvent,
) error {
	if err := h.cargoShipment.SetTransport(
		ctx,
		CargoShipmentSetTransportRequest{
			ShipmentID: event.ID,
			DriverID:   event.Driver.ID.String(),
			CarrierID:  fmt.Sprintf("%d", event.Driver.SupplierID),
		},
	); err != nil {
		return fmt.Errorf("set transport: %w", err)
	}

	return nil
}

// HandleShipmentTransportAssignEvent propagates the assigned vehicle to
// cargo-shipment.
func (h *ShipmentEventHandler) HandleShipmentTransportAssignEvent(
	ctx context.Context,
	event events.ShipmentTransportAssignEvent,
) error {
	if err := h.cargoShipment.SetTransport(
		ctx,
		CargoShipmentSetTransportRequest{
			ShipmentID:    event.ID,
			VehicleID:     event.Vehicle.ID,
			VehicleNumber: string(event.Vehicle.LicensePlate),
		},
	); err != nil {
		return fmt.Errorf("set transport: %w", err)
	}

	return nil
}

// HandleShipmentCompletedEvent propagates the completed shipment to
// cargo-shipment (DriveShipment FINISH) and writes the
// payments.last_mile_shipment_accrual outbox message, both inside the same
// transaction as the shipment completion.
func (h *ShipmentEventHandler) HandleShipmentCompletedEvent(
	ctx context.Context,
	event events.ShipmentCompletedEvent,
) error {
	if h.paymentsWriter != nil {
		if err := h.paymentsWriter.WriteLastMileShipmentAccrual(
			ctx,
			payments.LastMileShipmentAccrual{
				EventID:    event.ID.String(),
				ShipmentID: uuid.UUID(event.ID),
			},
		); err != nil {
			return fmt.Errorf("write payments accrual: %w", err)
		}
	}

	if err := h.cargoShipment.DriveShipment(
		ctx,
		CargoShipmentDriveShipmentRequest{
			ShipmentID: event.ID,
			Trigger:    CargoShipmentDriveShipmentTriggerFinish,
		},
	); err != nil {
		return fmt.Errorf("drive shipment: %w", err)
	}

	return nil
}
