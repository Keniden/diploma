package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestShipmentStatus_ToString(t *testing.T) {
	tests := []struct {
		status   values.ShipmentStatus
		expected string
	}{
		{values.ShipmentStatusNew, "NEW"},
		{values.ShipmentStatusAssigned, "ASSIGNED"},
		{values.ShipmentStatusLoading, "LOADING"},
		{values.ShipmentStatusInProgress, "IN_PROGRESS"},
		{values.ShipmentStatusCompleted, "COMPLETED"},
		{values.ShipmentStatusCanceled, "CANCELED"},
		{values.ShipmentStatusUnspecified, "UNSPECIFIED"},
	}

	for _, tt := range tests {
		got := tt.status.ToString()
		if got != tt.expected {
			t.Fatalf("ToString(%d) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}
