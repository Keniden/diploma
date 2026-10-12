package primitives_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/primitives"
)

func TestNow_UTCTruncatedToMicroseconds(t *testing.T) {
	now := primitives.Now()

	require.Equal(t, time.UTC, now.Location(), "primitives.Now() must return UTC")
	require.Equal(t, 0, now.Nanosecond()%int(time.Microsecond),
		"primitives.Now() must be truncated to microseconds")
}