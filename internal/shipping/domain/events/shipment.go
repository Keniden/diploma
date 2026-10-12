package events

import (
	"log/slog"

	"last-mile-task/internal/shipping/domain/values"
)

type ShipmentEventType string

type ShipmentEvent interface {
	event[ShipmentEventType]
}

const (
	ShipmentCreated         ShipmentEventType = "shipment.created"
	ShipmentTransportAssign ShipmentEventType = "shipment.transport.assign"
	ShipmentDriverAssign    ShipmentEventType = "shipment.driver.assign"
	ShipmentLoading         ShipmentEventType = "shipment.loading"
	ShipmentStarted         ShipmentEventType = "shipment.started"
	ShipmentCompleted       ShipmentEventType = "shipment.completed"
	ShipmentCanceled        ShipmentEventType = "shipment.canceled"
)

type ShipmentDriverAssignEvent struct {
	ID     values.ShipmentID
	Driver values.Driver
}

func (e ShipmentDriverAssignEvent) Type() ShipmentEventType {
	return ShipmentDriverAssign
}

func (e ShipmentDriverAssignEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}

type ShipmentTransportAssignEvent struct {
	ID      values.ShipmentID
	Vehicle values.Vehicle
}

func (e ShipmentTransportAssignEvent) Type() ShipmentEventType {
	return ShipmentTransportAssign
}

func (e ShipmentTransportAssignEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}

type ShipmentStartedEvent struct {
	ID values.ShipmentID
}

func (e ShipmentStartedEvent) Type() ShipmentEventType {
	return ShipmentStarted
}

func (e ShipmentStartedEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}

type ShipmentCompletedEvent struct {
	ID values.ShipmentID
}

func (e ShipmentCompletedEvent) Type() ShipmentEventType {
	return ShipmentCompleted
}

func (e ShipmentCompletedEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}

type ShipmentCanceledEvent struct {
	ID values.ShipmentID
}

func (e ShipmentCanceledEvent) Type() ShipmentEventType {
	return ShipmentCanceled
}

func (e ShipmentCanceledEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}

type ShipmentLoadingEvent struct {
	ID values.ShipmentID
}

func (e ShipmentLoadingEvent) Type() ShipmentEventType {
	return ShipmentLoading
}

func (e ShipmentLoadingEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ID.String()))
}
