package primitives_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/primitives"
)

func TestNewCoordinates_Valid(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{"zero", 0, 0},
		{"moscow", 55.7558, 37.6173},
		{"lat boundaries", -90, 180},
		{"lon boundaries", 90, -180},
	}

	for _, tt := range tests {
		coords, err := primitives.NewCoordinates(tt.latitude, tt.longitude)
		require.NoError(t, err, tt.name)
		require.Equal(t, tt.latitude, coords.Latitude, tt.name)
		require.Equal(t, tt.longitude, coords.Longitude, tt.name)
	}
}

func TestNewCoordinates_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{"latitude above range", 90.1, 0},
		{"latitude below range", -90.1, 0},
		{"longitude above range", 0, 180.1},
		{"longitude below range", 0, -180.1},
	}

	for _, tt := range tests {
		_, err := primitives.NewCoordinates(tt.latitude, tt.longitude)
		require.Error(t, err, tt.name)
	}
}