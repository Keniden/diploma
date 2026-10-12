package primitives_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/primitives"
)

func TestNewPage_Valid(t *testing.T) {
	page, err := primitives.NewPage[string](10, "token-1")
	require.NoError(t, err)
	require.Equal(t, int64(10), page.Size)
	require.Equal(t, "token-1", page.Token)
}

func TestNewPage_InvalidSize(t *testing.T) {
	tests := []int64{0, -1, -100}
	for _, size := range tests {
		_, err := primitives.NewPage[string](size, "token-1")
		require.ErrorIs(t, err, primitives.ErrInvalidPageSize, "size %d", size)
	}
}

func TestPaginationResponse_ZeroValueUsable(t *testing.T) {
	var resp primitives.PaginationResponse[string]
	require.Nil(t, resp.Items)
	require.False(t, resp.HasMore)
	require.Nil(t, resp.NextPage)
}