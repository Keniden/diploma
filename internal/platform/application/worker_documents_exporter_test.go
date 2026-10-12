package application

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type fakeDocumentsReader struct {
	started chan struct{}
	stopped chan struct{}
	errs    chan error
}

func newFakeDocumentsReader() *fakeDocumentsReader {
	return &fakeDocumentsReader{
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
		errs:    make(chan error, 4),
	}
}

func (f *fakeDocumentsReader) Start() {
	f.started <- struct{}{}
}

func (f *fakeDocumentsReader) Stop(context.Context) error {
	close(f.stopped)
	return nil
}

func (f *fakeDocumentsReader) Errors() <-chan error { return f.errs }

func TestDocumentsExporterStartStopDrainsErrors(t *testing.T) {
	reader := newFakeDocumentsReader()
	exporter := newDocumentsExporter(slog.Default(), []outboxReader{reader})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		exporter.Run(ctx)
		close(done)
	}()

	exporter.Start()
	select {
	case <-reader.started:
	case <-time.After(time.Second):
		t.Fatal("reader must be started")
	}

	reader.errs <- context.DeadlineExceeded

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run must exit after the context is canceled")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()

	if err := exporter.Stop(stopCtx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	select {
	case <-reader.stopped:
	case <-time.After(time.Second):
		t.Fatal("reader must be stopped")
	}
}