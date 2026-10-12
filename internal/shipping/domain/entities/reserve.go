package entities

import (
	"time"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

type Reserve struct {
	ID           values.ReserveID
	ShipmentID   values.ShipmentID
	RoutePointID values.RoutePointID
	Status       values.ReserveStatus
	Gate         values.ReserveGate
	Task         values.ReserveTask
	ScannedAt    *time.Time
	ExpiresAt    *time.Time
}

func NewQueuedReserve(
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
	gate values.ReserveGate,
) *Reserve {
	return &Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipmentID,
		RoutePointID: routePointID,
		Status:       values.ReserveStatusQueued,
		Gate:         gate,
	}
}

func NewReserveForApproval(
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
	gate values.ReserveGate,
	task values.ReserveTask,
	expiresAt time.Time,
) *Reserve {
	return &Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipmentID,
		RoutePointID: routePointID,
		Status:       values.ReserveStatusWaitingApproval,
		Gate:         gate,
		Task:         task,
		ExpiresAt:    &expiresAt,
	}
}

func NewFakeReserveForTargetApproval(
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) *Reserve {
	expiresAt := values.Now().Add(20 * time.Minute)

	return &Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipmentID,
		RoutePointID: routePointID,
		Status:       values.ReserveStatusWaitingApproval,
		ExpiresAt:    &expiresAt,
	}
}

func (r *Reserve) IsFake() bool {
	return r.Task.ID == ""
}

func (r *Reserve) Approve() error {
	if r.Status != values.ReserveStatusWaitingApproval {
		return lmterrors.NewInvalidReserveCannotApproveError("reserve cannot be approved")
	}

	r.Status = values.ReserveStatusActive
	now := values.Now()
	r.ScannedAt = &now

	return nil
}

func (r *Reserve) Release() error {
	if r.Status != values.ReserveStatusActive {
		return lmterrors.NewInvalidReserveCannotReleaseError("attempt to release not active reserve")
	}

	r.Status = values.ReserveStatusReleased

	return nil
}

func (r *Reserve) BelongsToShipment(id values.ShipmentID) bool {
	return r.ShipmentID == id
}

func (r *Reserve) Cancel() error {
	if r.Status != values.ReserveStatusQueued && r.Status != values.ReserveStatusWaitingApproval {
		return lmterrors.NewInvalidReserveCannotCancelError("reserve cannot be canceled")
	}

	r.Status = values.ReserveStatusCanceled

	return nil
}

func (r *Reserve) ExpireWaitingApproval() error {
	if r.Status != values.ReserveStatusWaitingApproval {
		return lmterrors.NewInvalidReserveCannotExpireError("reserve cannot be expired")
	}

	r.Status = values.ReserveStatusExpiredWaitingApproval

	return nil
}

func (r *Reserve) ExpireActive() error {
	if r.Status != values.ReserveStatusActive {
		return lmterrors.NewInvalidReserveCannotExpireError("reserve cannot be expired")
	}

	r.Status = values.ReserveStatusExpiredActive

	return nil
}
