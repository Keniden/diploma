package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestRoutePointStatus_ToString(t *testing.T) {
	tests := []struct {
		status   values.RoutePointStatus
		expected string
	}{
		{values.RoutePointStatusWaitingArrival, "WAITING_ARRIVAL"},
		{values.RoutePointStatusDriverArrived, "DRIVER_ARRIVED"},
		{values.RoutePointStatusPackagesNotUnloaded, "PACKAGES_NOT_UNLOADED"},
		{values.RoutePointStatusPackagesNotLoaded, "PACKAGES_NOT_LOADED"},
		{values.RoutePointStatusCompleted, "COMPLETED"},
		{values.RoutePointStatus(99), "UNSPECIFIED"},
	}

	for _, tt := range tests {
		got := tt.status.ToString()
		if got != tt.expected {
			t.Fatalf("ToString(%d) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}