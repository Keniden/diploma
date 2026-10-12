package events

import (
	"log/slog"
	"time"

	"last-mile-task/internal/shipping/domain/values"
)

type ReserveEventType string

type ReserveEvent interface {
	event[ReserveEventType]
}

const (
	ReserveQueued   ReserveEventType = "reserve.queued"
	ReserveApproved ReserveEventType = "reserve.approved"
	ReserveReleased ReserveEventType = "reserve.released"
)

type ReserveQueuedEvent struct {
	ID         values.ReserveID
	ShipmentID values.ShipmentID
	Driver     values.Driver
}

func (e ReserveQueuedEvent) Type() ReserveEventType {
	return ReserveQueued
}

func (e ReserveQueuedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("reserve_id", e.ID.String()),
		slog.String("shipment_id", e.ShipmentID.String()),
	)
}

type ReserveApprovedEvent struct {
	ID              values.ReserveID
	ShipmentID      values.ShipmentID
	Task            values.ReserveTask
	Timestamp       time.Time
	IsFirstApproval bool
	IsFake          bool
}

func (e ReserveApprovedEvent) Type() ReserveEventType {
	return ReserveApproved
}

func (e ReserveApprovedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("reserve_id", e.ID.String()),
		slog.String("shipment_id", e.ShipmentID.String()),
		slog.Bool("is_first_approval", e.IsFirstApproval),
		slog.Bool("is_fake", e.IsFake),
	)
}

type ReserveReleasedEvent struct {
	ID        values.ReserveID
	Task      values.ReserveTask
	Timestamp time.Time
}

func (e ReserveReleasedEvent) Type() ReserveEventType {
	return ReserveReleased
}

func (e ReserveReleasedEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("reserve_id", e.ID.String()),
	)
}
