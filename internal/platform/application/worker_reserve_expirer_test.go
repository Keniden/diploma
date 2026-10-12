package application

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeReserveExpirer is a ports.ReserveExpirer recording each expiry pass. Its
// methods are safe to call repeatedly (idempotent), mirroring the concrete
// ReserveService which skips already-expired reserves.
type fakeReserveExpirer struct {
	waitingCalls atomic.Int32
	activeCalls  atomic.Int32
	err          error
}

func (f *fakeReserveExpirer) ExpireDueWaitingApprovalReserves(context.Context) error {
	f.waitingCalls.Add(1)
	return f.err
}

func (f *fakeReserveExpirer) ExpireDueActiveReserves(context.Context) error {
	f.activeCalls.Add(1)
	return f.err
}

// captureHandler records slog Error records for the error-logging assertion.
type captureHandler struct {
	mu    sync.Mutex
	errs  []string
	attrs slog.Handler
}

func newCaptureHandler() *captureHandler {
	return &captureHandler{}
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelError {
		var b bytes.Buffer
		b.WriteString(r.Message)
		r.Attrs(func(a slog.Attr) bool {
			b.WriteString(" ")
			b.WriteString(a.Key)
			b.WriteString("=")
			b.WriteString(a.Value.String())
			return true
		})
		h.mu.Lock()
		h.errs = append(h.errs, b.String())
		h.mu.Unlock()
	}
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func newFastExpirer(logger *slog.Logger, expirer *fakeReserveExpirer) *reserveExpirer {
	w := newReserveExpirer(logger, expirer)
	w.interval = 5 * time.Millisecond
	return w
}

// Lifecycle: Run pumps both expiry passes on each tick and returns when the
// context is canceled.
func TestReserveExpirer_Run_Lifecycle(t *testing.T) {
	fake := &fakeReserveExpirer{}
	ctx, cancel := context.WithCancel(context.Background())

	w := newFastExpirer(slog.Default(), fake)
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return fake.waitingCalls.Load() >= 1 && fake.activeCalls.Load() >= 1
	}, time.Second, time.Millisecond, "worker must run both expiry passes")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run must exit after the context is canceled")
	}
}

// Errors are pumped into the log and do not stop the loop.
func TestReserveExpirer_Run_LogsErrorsAndContinues(t *testing.T) {
	fake := &fakeReserveExpirer{err: errors.New("db down")}
	handler := newCaptureHandler()
	ctx, cancel := context.WithCancel(context.Background())

	w := newFastExpirer(slog.New(handler), fake)
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		handler.mu.Lock()
		defer handler.mu.Unlock()
		return len(handler.errs) >= 2
	}, time.Second, time.Millisecond, "both expiry errors must be logged")

	// The loop keeps going: further ticks still call the expirer.
	before := fake.waitingCalls.Load()
	require.Eventually(t, func() bool {
		return fake.waitingCalls.Load() > before
	}, time.Second, time.Millisecond, "the loop must continue after errors")

	handler.mu.Lock()
	joined := strings.Join(handler.errs, "\n")
	handler.mu.Unlock()
	require.Contains(t, joined, "reserve expirer: expire waiting approval reserves")
	require.Contains(t, joined, "reserve expirer: expire active reserves")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run must exit after the context is canceled")
	}
}

// Idempotent expiry: the worker holds no per-reserve state, so repeated passes
// (as after a crash/restart) are harmless and the loop keeps ticking.
func TestReserveExpirer_IdempotentExpiry(t *testing.T) {
	fake := &fakeReserveExpirer{}
	ctx, cancel := context.WithCancel(context.Background())

	w := newFastExpirer(slog.Default(), fake)
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return fake.waitingCalls.Load() >= 3
	}, time.Second, time.Millisecond, "worker must re-run the expiry pass over already-expired reserves")

	cancel()
	<-done
}