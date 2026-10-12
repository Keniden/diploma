package entities_test

import (
	"testing"
	"time"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func TestNewQueuedReserve(t *testing.T) {
	shipmentID := values.NewID()
	routePointID := values.NewID()
	gate := values.ReserveGate{ID: 1, Name: "Gate 1", PositionNumber: 1}

	r := entities.NewQueuedReserve(shipmentID, routePointID, gate)

	if r.ID == values.NilID {
		t.Fatal("expected non-nil reserve ID")
	}
	if r.ShipmentID != shipmentID {
		t.Fatalf("expected shipment ID %v, got %v", shipmentID, r.ShipmentID)
	}
	if r.RoutePointID != routePointID {
		t.Fatalf("expected route point ID %v, got %v", routePointID, r.RoutePointID)
	}
	if r.Status != values.ReserveStatusQueued {
		t.Fatalf("expected queued status, got %v", r.Status)
	}
	if r.Gate.ID != gate.ID {
		t.Fatalf("expected gate ID %d, got %d", gate.ID, r.Gate.ID)
	}
}

func TestNewReserveForApproval(t *testing.T) {
	shipmentID := values.NewID()
	routePointID := values.NewID()
	gate := values.ReserveGate{ID: 2, Name: "Gate 2", PositionNumber: 2}
	task := values.ReserveTask{ID: "task-1"}
	expiresAt := time.Now().Add(time.Hour)

	r := entities.NewReserveForApproval(shipmentID, routePointID, gate, task, expiresAt)

	if r.Status != values.ReserveStatusWaitingApproval {
		t.Fatalf("expected waiting approval status, got %v", r.Status)
	}
	if r.Task.ID != task.ID {
		t.Fatalf("expected task ID %s, got %s", task.ID, r.Task.ID)
	}
	if !r.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiresAt %v, got %v", expiresAt, r.ExpiresAt)
	}
}

func TestNewFakeReserveForTargetApproval(t *testing.T) {
	shipmentID := values.NewID()
	routePointID := values.NewID()
	before := values.Now()

	r := entities.NewFakeReserveForTargetApproval(shipmentID, routePointID)

	if r.Status != values.ReserveStatusWaitingApproval {
		t.Fatalf("expected waiting approval status, got %v", r.Status)
	}
	if r.ShipmentID != shipmentID {
		t.Fatalf("expected shipment ID %v, got %v", shipmentID, r.ShipmentID)
	}
	if r.RoutePointID != routePointID {
		t.Fatalf("expected route point ID %v, got %v", routePointID, r.RoutePointID)
	}
	if r.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if duration := r.ExpiresAt.Sub(before); duration < 20*time.Minute || duration >= 20*time.Minute+time.Second {
		t.Fatalf("expected ExpiresAt about 20 minutes after creation, got %v", duration)
	}
	if !r.IsFake() {
		t.Fatal("expected fake reserve")
	}
}

func TestReserve_IsFake(t *testing.T) {
	fakeReserve := entities.NewFakeReserveForTargetApproval(values.NewID(), values.NewID())
	if !fakeReserve.IsFake() {
		t.Fatal("expected fake reserve")
	}

	realReserve := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)
	if realReserve.IsFake() {
		t.Fatal("expected real reserve")
	}
}

func TestReserve_Approve(t *testing.T) {
	initialExpiresAt := values.Now().Add(time.Hour)
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		initialExpiresAt,
	)

	err := r.Approve()
	if err != nil {
		t.Fatal(err)
	}

	if r.Status != values.ReserveStatusActive {
		t.Fatalf("expected active status, got %v", r.Status)
	}
	if r.ScannedAt == nil {
		t.Fatal("expected ScannedAt to be set")
	}
	if r.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if !r.ExpiresAt.Equal(initialExpiresAt) {
		t.Fatalf("expected ExpiresAt to be preserved, got %v, want %v", r.ExpiresAt, initialExpiresAt)
	}
}

func TestReserve_Approve_NotWaitingApproval(t *testing.T) {
	r := entities.NewQueuedReserve(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
	)

	err := r.Approve()
	if err == nil {
		t.Fatal("expected error when approving queued reserve")
	}
	if r.Status != values.ReserveStatusQueued {
		t.Fatal("status should not change after failed approve")
	}
}

func TestReserve_Approve_AlreadyActive(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)
	if err := r.Approve(); err != nil {
		t.Fatal(err)
	}

	err := r.Approve()
	if err == nil {
		t.Fatal("expected error when approving already active reserve")
	}
}

func TestReserve_Release(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)
	if err := r.Approve(); err != nil {
		t.Fatal(err)
	}

	err := r.Release()
	if err != nil {
		t.Fatal(err)
	}

	if r.Status != values.ReserveStatusReleased {
		t.Fatalf("expected released status, got %v", r.Status)
	}
}

func TestReserve_Release_NotActive(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)

	err := r.Release()
	if err == nil {
		t.Fatal("expected error when releasing waiting approval reserve")
	}
	if r.Status != values.ReserveStatusWaitingApproval {
		t.Fatal("status should not change after failed release")
	}
}

func TestReserve_Cancel_Queued(t *testing.T) {
	r := entities.NewQueuedReserve(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
	)

	err := r.Cancel()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != values.ReserveStatusCanceled {
		t.Fatalf("expected canceled status after cancel, got %v", r.Status)
	}
}

func TestReserve_Cancel_WaitingApproval(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)

	err := r.Cancel()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != values.ReserveStatusCanceled {
		t.Fatalf("expected canceled status after cancel, got %v", r.Status)
	}
}

func TestReserve_Cancel_Active(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)
	_ = r.Approve()

	err := r.Cancel()
	if err == nil {
		t.Fatal("expected error when canceling active reserve")
	}
	if r.Status != values.ReserveStatusActive {
		t.Fatal("status should not change after failed cancel")
	}
}

func TestReserve_ExpireWaitingApproval(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)

	err := r.ExpireWaitingApproval()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != values.ReserveStatusExpiredWaitingApproval {
		t.Fatalf("expected expired waiting approval status, got %v", r.Status)
	}
}

func TestReserve_ExpireWaitingApproval_NotWaitingApproval(t *testing.T) {
	r := entities.NewQueuedReserve(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
	)

	err := r.ExpireWaitingApproval()
	if err == nil {
		t.Fatal("expected error when expiring queued reserve")
	}
	if r.Status != values.ReserveStatusQueued {
		t.Fatal("status should not change after failed expire")
	}
}

func TestReserve_ExpireActive(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)
	if err := r.Approve(); err != nil {
		t.Fatal(err)
	}

	err := r.ExpireActive()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != values.ReserveStatusExpiredActive {
		t.Fatalf("expected expired active status, got %v", r.Status)
	}
}

func TestReserve_ExpireActive_NotActive(t *testing.T) {
	r := entities.NewReserveForApproval(
		values.NewID(),
		values.NewID(),
		values.ReserveGate{ID: 1},
		values.ReserveTask{ID: "task-1"},
		time.Now().Add(time.Hour),
	)

	err := r.ExpireActive()
	if err == nil {
		t.Fatal("expected error when expiring waiting approval reserve")
	}
	if r.Status != values.ReserveStatusWaitingApproval {
		t.Fatal("status should not change after failed expire")
	}
}
