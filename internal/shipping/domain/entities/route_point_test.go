package entities_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

func newRoutePoint(status values.RoutePointStatus) *entities.RoutePoint {
	return &entities.RoutePoint{
		ID:     values.NewID(),
		PointID: values.PointID(101),
		SeqNum:  1,
		Status:  status,
	}
}

func TestRoutePoint_ValidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		status values.RoutePointStatus
		action func(rp *entities.RoutePoint) error
		want   values.RoutePointStatus
	}{
		{
			name:   "WaitingArrival -> DriverArrived",
			status: values.RoutePointStatusWaitingArrival,
			action: func(rp *entities.RoutePoint) error { return rp.DriverArrived() },
			want:   values.RoutePointStatusDriverArrived,
		},
		{
			name:   "WaitingArrival -> Completed",
			status: values.RoutePointStatusWaitingArrival,
			action: func(rp *entities.RoutePoint) error { return rp.Completed() },
			want:   values.RoutePointStatusCompleted,
		},
		{
			name:   "DriverArrived -> Completed",
			status: values.RoutePointStatusDriverArrived,
			action: func(rp *entities.RoutePoint) error { return rp.Completed() },
			want:   values.RoutePointStatusCompleted,
		},
		{
			name:   "DriverArrived -> PackagesNotUnloaded",
			status: values.RoutePointStatusDriverArrived,
			action: func(rp *entities.RoutePoint) error { return rp.PackagesNotUnloaded() },
			want:   values.RoutePointStatusPackagesNotUnloaded,
		},
		{
			name:   "DriverArrived -> PackagesNotLoaded",
			status: values.RoutePointStatusDriverArrived,
			action: func(rp *entities.RoutePoint) error { return rp.PackagesNotLoaded() },
			want:   values.RoutePointStatusPackagesNotLoaded,
		},
		{
			name:   "PackagesNotUnloaded -> DriverArrived",
			status: values.RoutePointStatusPackagesNotUnloaded,
			action: func(rp *entities.RoutePoint) error { return rp.DriverArrived() },
			want:   values.RoutePointStatusDriverArrived,
		},
		{
			name:   "PackagesNotLoaded -> DriverArrived",
			status: values.RoutePointStatusPackagesNotLoaded,
			action: func(rp *entities.RoutePoint) error { return rp.DriverArrived() },
			want:   values.RoutePointStatusDriverArrived,
		},
		{
			name:   "PackagesNotLoaded -> Completed",
			status: values.RoutePointStatusPackagesNotLoaded,
			action: func(rp *entities.RoutePoint) error { return rp.Completed() },
			want:   values.RoutePointStatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := newRoutePoint(tt.status)
			if err := tt.action(rp); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if rp.Status != tt.want {
				t.Fatalf("expected status %v, got %v", tt.want, rp.Status)
			}
		})
	}
}

func TestRoutePoint_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		status values.RoutePointStatus
		action func(rp *entities.RoutePoint) error
	}{
		{
			name:   "Completed -> DriverArrived",
			status: values.RoutePointStatusCompleted,
			action: func(rp *entities.RoutePoint) error { return rp.DriverArrived() },
		},
		{
			name:   "Completed -> PackagesNotUnloaded",
			status: values.RoutePointStatusCompleted,
			action: func(rp *entities.RoutePoint) error { return rp.PackagesNotUnloaded() },
		},
		{
			name:   "WaitingArrival -> PackagesNotUnloaded",
			status: values.RoutePointStatusWaitingArrival,
			action: func(rp *entities.RoutePoint) error { return rp.PackagesNotUnloaded() },
		},
		{
			name:   "WaitingArrival -> PackagesNotLoaded",
			status: values.RoutePointStatusWaitingArrival,
			action: func(rp *entities.RoutePoint) error { return rp.PackagesNotLoaded() },
		},
		{
			name:   "PackagesNotUnloaded -> Completed",
			status: values.RoutePointStatusPackagesNotUnloaded,
			action: func(rp *entities.RoutePoint) error { return rp.Completed() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := newRoutePoint(tt.status)
			err := tt.action(rp)
			if err == nil {
				t.Fatal("expected an error for the prohibited transition")
			}
			if !lmterrors.Is(err, lmterrors.InvalidRoutePointStatusTransitionCode) {
				t.Fatalf("expected invalid route point status transition error, got %v", err)
			}
			if rp.Status != tt.status {
				t.Fatalf("status must not change on prohibited transition, got %v, want %v", rp.Status, tt.status)
			}
		})
	}
}