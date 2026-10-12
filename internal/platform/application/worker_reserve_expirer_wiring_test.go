package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/platform/application/builder"
)

// newTestBuilder builds a real builder without touching the database: the
// worker only needs the builder context (created in builder.New), no pool.
func newTestBuilder(t *testing.T) *builder.Builder {
	t.Helper()

	b, err := builder.New(context.Background(), "test")
	require.NoError(t, err)
	t.Cleanup(b.Shutdown)

	return b
}

// nil reserve expirer: the wiring short-circuits and must not spin a worker.
func TestStartReserveExpirer_NilExpirer_Noop(t *testing.T) {
	app := &Application{builder: newTestBuilder(t)}
	app.startReserveExpirer()
}

// non-nil reserve expirer: the worker is started on the builder background and
// is bound to the builder context, which Shutdown cancels.
func TestStartReserveExpirer_StartsWorker(t *testing.T) {
	app := &Application{
		builder:        newTestBuilder(t),
		reserveExpirer: &fakeReserveExpirer{},
	}
	app.startReserveExpirer()
}