package reserves

import (
	"testing"
	"time"

	"last-mile-task/internal/shipping/domain/values"
)

func TestReserveFromRecord(t *testing.T) {
	t.Run("maps all fields from record", func(t *testing.T) {
		now := time.Now()
		rec := reserveRecord{
			ID:         [16]byte{1, 2, 3, 4},
			RoutePoint: [16]byte{5, 6, 7, 8},
			ShipmentID: [16]byte{9, 10, 11, 12},
			Status:     int16(values.ReserveStatusActive),
			TaskID:     "task-123",
			GateID:     42,
			GateName:   "Gate A",
			ExpiresAt:  &now,
			ScannedAt:  &now,
		}

		res := reserveFromRecord(rec)
		if res.ID != rec.ID {
			t.Errorf("ID: got %v, want %v", res.ID, rec.ID)
		}
		if res.RoutePointID != rec.RoutePoint {
			t.Errorf("RoutePointID: got %v, want %v", res.RoutePointID, rec.RoutePoint)
		}
		if res.ShipmentID != rec.ShipmentID {
			t.Errorf("ShipmentID: got %v, want %v", res.ShipmentID, rec.ShipmentID)
		}
		if res.Status != values.ReserveStatusActive {
			t.Errorf("Status: got %v, want %v", res.Status, values.ReserveStatusActive)
		}
		if res.Gate.ID != values.GateID(rec.GateID) {
			t.Errorf("Gate.ID: got %v, want %v", res.Gate.ID, rec.GateID)
		}
		if res.Gate.Name != rec.GateName {
			t.Errorf("Gate.Name: got %v, want %v", res.Gate.Name, rec.GateName)
		}
		if res.Task.ID != values.ReserveTaskID(rec.TaskID) {
			t.Errorf("Task.ID: got %v, want %v", res.Task.ID, rec.TaskID)
		}
		if res.ExpiresAt != rec.ExpiresAt {
			t.Errorf("ExpiresAt: got %v, want %v", res.ExpiresAt, rec.ExpiresAt)
		}
		if res.ScannedAt != rec.ScannedAt {
			t.Errorf("ScannedAt: got %v, want %v", res.ScannedAt, rec.ScannedAt)
		}
	})

	t.Run("handles nil expires_at", func(t *testing.T) {
		rec := reserveRecord{
			ID:        [16]byte{1},
			ExpiresAt: nil,
		}

		res := reserveFromRecord(rec)
		if res.ExpiresAt != nil {
			t.Errorf("ExpiresAt should be nil for queued reserve")
		}
	})
}

func TestNullableTimeToDB(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if got := nullableTimeToDB(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("zero time returns nil", func(t *testing.T) {
		zero := time.Time{}
		if got := nullableTimeToDB(&zero); got != nil {
			t.Errorf("expected nil for zero time, got %v", got)
		}
	})

	t.Run("valid time returns pointer", func(t *testing.T) {
		now := time.Now()
		got := nullableTimeToDB(&now)
		if got == nil || !got.Equal(now) {
			t.Errorf("got %v, want %v", got, now)
		}
	})
}
