package values_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/shipping/domain/values"
)

func TestNewID_IsV7(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := values.NewID()
		require.Equal(t, uuid.Version(7), id.Version())
		require.NotEqual(t, uuid.Nil, id)
	}
}

func TestNewShipmentShortID_Positive(t *testing.T) {
	for i := 0; i < 20; i++ {
		require.Positive(t, int64(values.NewShipmentShortID()))
	}
}

func TestTareID_StringAndSticker(t *testing.T) {
	id := values.TareID(42)
	require.Equal(t, "42", id.String())
	require.Equal(t, "BOX42", id.Sticker(values.TareType("BOX")))
}

func TestNilID(t *testing.T) {
	require.Equal(t, uuid.Nil, values.NilID)
}