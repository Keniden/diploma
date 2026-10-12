package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestPackageActionStatus_ToString(t *testing.T) {
	tests := []struct {
		status   values.PackageActionStatus
		expected string
	}{
		{values.RoutePointPackageStatusUnspecified, "UNSPECIFIED"},
		{values.RoutePointPackageStatusTODO, "TODO"},
		{values.RoutePointPackageStatusDone, "DONE"},
		{values.RoutePointPackageStatusCanceled, "CANCELED"},
		{values.PackageActionStatus(99), "UNSPECIFIED"},
	}

	for _, tt := range tests {
		got := tt.status.ToString()
		if got != tt.expected {
			t.Fatalf("ToString(%d) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func TestReserveStatus_Values(t *testing.T) {
	// Vote-style enum: every status must have a stable, distinct value.
	statuses := []values.ReserveStatus{
		values.ReserveStatusQueued,
		values.ReserveStatusWaitingApproval,
		values.ReserveStatusActive,
		values.ReserveStatusExpiredWaitingApproval,
		values.ReserveStatusExpiredActive,
		values.ReserveStatusReleased,
		values.ReserveStatusCanceled,
	}

	seen := map[values.ReserveStatus]struct{}{}
	for _, s := range statuses {
		if _, ok := seen[s]; ok {
			t.Fatalf("duplicate reserve status value %d", s)
		}
		seen[s] = struct{}{}
	}
}