package values_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/shipping/domain/values"
)

func TestNow_UTC(t *testing.T) {
	now := values.Now()
	require.Equal(t, time.UTC, now.Location())
}