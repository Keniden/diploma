package reserves

import (
	"fmt"
	"time"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func reserveStatusFromDB(status int16) values.ReserveStatus {
	switch values.ReserveStatus(status) {
	case values.ReserveStatusQueued,
		values.ReserveStatusWaitingApproval,
		values.ReserveStatusActive,
		values.ReserveStatusExpiredWaitingApproval,
		values.ReserveStatusExpiredActive,
		values.ReserveStatusReleased,
		values.ReserveStatusCanceled:
		return values.ReserveStatus(status)
	default:
		return values.ReserveStatusQueued
	}
}

func reserveStatusToDB(status values.ReserveStatus) (int16, error) {
	switch status {
	case values.ReserveStatusQueued,
		values.ReserveStatusWaitingApproval,
		values.ReserveStatusActive,
		values.ReserveStatusExpiredWaitingApproval,
		values.ReserveStatusExpiredActive,
		values.ReserveStatusReleased,
		values.ReserveStatusCanceled:
		return int16(status), nil
	default:
		return 0, fmt.Errorf("unknown reserve status: %d", int(status))
	}
}

func reserveFromRecord(rec reserveRecord) *entities.Reserve {
	return &entities.Reserve{
		ID:           rec.ID,
		ShipmentID:   rec.ShipmentID,
		RoutePointID: rec.RoutePoint,
		Status:       reserveStatusFromDB(rec.Status),
		Gate: values.ReserveGate{
			ID:   values.GateID(rec.GateID),
			Name: rec.GateName,
		},
		Task: values.ReserveTask{
			ID: values.ReserveTaskID(rec.TaskID),
		},
		ScannedAt: rec.ScannedAt,
		ExpiresAt: rec.ExpiresAt,
	}
}

func nullableTimeToDB(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}

	v := *t
	return &v
}
