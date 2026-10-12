package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestPackageStatus_ToString(t *testing.T) {
	tests := []struct {
		status   values.PackageStatus
		expected string
	}{
		{values.PackageStatusUnspecified, "UNSPECIFIED"},
		{values.PackageStatusReadyToLoad, "READY_TO_LOAD"},
		{values.PackageStatusLoaded, "LOADED"},
		{values.PackageStatusInTransit, "IN_TRANSIT"},
		{values.PackageStatusUnloading, "UNLOADING"},
		{values.PackageStatusUnloaded, "UNLOADED"},
		{values.PackageStatusCanceled, "CANCELED"},
		{values.PackageStatus(99), "UNSPECIFIED"},
	}

	for _, tt := range tests {
		got := tt.status.ToString()
		if got != tt.expected {
			t.Fatalf("ToString(%d) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}