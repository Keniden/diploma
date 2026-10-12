package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestPackageActionType_ToString(t *testing.T) {
	tests := []struct {
		action   values.PackageActionType
		expected string
	}{
		{values.RoutePointActionTypeUnspecified, "UNSPECIFIED"},
		{values.RoutePointActionTypeLoad, "LOAD"},
		{values.RoutePointActionTypeUnload, "UNLOAD"},
		{values.RoutePointActionTypeReserve, "RESERVE"},
		{values.PackageActionType(99), "UNSPECIFIED"},
	}

	for _, tt := range tests {
		got := tt.action.ToString()
		if got != tt.expected {
			t.Fatalf("ToString(%d) = %q, want %q", tt.action, got, tt.expected)
		}
	}
}

func TestPackageActionType_ProtoDomainMapping(t *testing.T) {
	// The domain RESERVE action must not collide with the proto enum values
	// (UNSPECIFIED/LOAD/UNLOAD only, see TODO(LASTMILE-008)).
	if values.RoutePointActionTypeReserve == values.RoutePointActionTypeUnload {
		t.Fatal("RESERVE must not reuse the UNLOAD value")
	}
}