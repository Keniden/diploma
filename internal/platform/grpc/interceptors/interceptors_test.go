package interceptors

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"last-mile-task/internal/platform/requestid"
)

func TestRequestIDPropagation(t *testing.T) {
	interc := RequestID()

	ctx, cancel := context.WithCancel(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestid.Header, "req-123",
	)))
	defer cancel()

	var got string
	_, err := interc(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"},
		func(ctx context.Context, _ any) (any, error) {
			got = requestid.Extract(ctx)
			return "ok", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "req-123", got)
}

func TestRequestIDNoMetadata(t *testing.T) {
	interc := RequestID()

	_, err := interc(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"},
		func(ctx context.Context, _ any) (any, error) {
			assert.Empty(t, requestid.Extract(ctx))
			return "ok", nil
		},
	)
	require.NoError(t, err)
}

func TestRecovery(t *testing.T) {
	interc := Recovery()

	_, err := interc(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(_ context.Context, _ any) (any, error) {
			panic("boom")
		},
	)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "expected a grpc status error")
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "boom")
}

func TestRecoveryPassesThrough(t *testing.T) {
	interc := Recovery()

	_, err := interc(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Ok"},
		func(_ context.Context, _ any) (any, error) {
			return "ok", nil
		},
	)
	require.NoError(t, err)
}

func TestLogging(t *testing.T) {
	interc := Logging()

	sentinel := errors.New("handler error")

	_, err := interc(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Fail"},
		func(_ context.Context, _ any) (any, error) {
			return nil, sentinel
		},
	)
	require.ErrorIs(t, err, sentinel)
}