package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeOutboxReader struct {
	messages []*Message
	readErr  error
	beginErr error

	completeErr error
	discardErr  error
	scheduleErr error
	commitErr   error
	rollbackErr error

	completed []*Message
	discarded []*Message
	scheduled []*Message

	commitCalls   int
	rollbackCalls int
	beginTxCalls  int
}

func (f *fakeOutboxReader) BeginTx(ctx context.Context) (context.Context, error) {
	f.beginTxCalls++

	if f.beginErr != nil {
		return ctx, f.beginErr
	}

	return ctx, nil
}

func (f *fakeOutboxReader) Commit(ctx context.Context) error {
	f.commitCalls++

	return f.commitErr
}

func (f *fakeOutboxReader) Rollback(ctx context.Context) error {
	f.rollbackCalls++

	return f.rollbackErr
}

func (f *fakeOutboxReader) ReadMessages(ctx context.Context, messageType string, batchSize int) ([]*Message, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}

	return f.messages, nil
}

func (f *fakeOutboxReader) ScheduleMessage(ctx context.Context, message *Message) error {
	f.scheduled = append(f.scheduled, message)

	return f.scheduleErr
}

func (f *fakeOutboxReader) CompleteMessages(ctx context.Context, messages []*Message) error {
	f.completed = append(f.completed, messages...)

	return f.completeErr
}

func (f *fakeOutboxReader) DiscardMessages(ctx context.Context, messages []*Message) error {
	f.discarded = append(f.discarded, messages...)

	return f.discardErr
}

func completedMessage(t *testing.T, opts ...MessageOption) *Message {
	t.Helper()

	opts = append([]MessageOption{MessageWithScheduledAt(time.Now().UTC().Add(-time.Hour))}, opts...)

	return NewMessage(Payload("p"), opts...)
}

func TestReaderCompletesMessage(t *testing.T) {
	outbox := &fakeOutboxReader{
		messages: []*Message{completedMessage(t, MessageWithID(uuid.New()))},
	}

	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		return nil
	}))

	reader.handleMessages()

	if len(outbox.completed) != 1 {
		t.Fatalf("expected 1 completed message, got %d", len(outbox.completed))
	}

	if outbox.completed[0].status != MessageStatusCompleted {
		t.Fatalf("message must be COMPLETED, got %q", outbox.completed[0].status)
	}

	if len(outbox.discarded) != 0 {
		t.Fatalf("expected no discarded messages, got %d", len(outbox.discarded))
	}

	if outbox.commitCalls != 1 {
		t.Fatalf("expected 1 commit, got %d", outbox.commitCalls)
	}
}

func TestReaderSchedulesRetryOnHandlerError(t *testing.T) {
	outbox := &fakeOutboxReader{
		messages: []*Message{completedMessage(t, MessageWithID(uuid.New()))},
	}

	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		return errors.New("handler boom")
	}))

	reader.handleMessages()

	if len(outbox.scheduled) != 1 {
		t.Fatalf("expected 1 scheduled message, got %d", len(outbox.scheduled))
	}

	msg := outbox.scheduled[0]
	if msg.status != MessageStatusTODO {
		t.Fatalf("status must be TODO after schedule, got %q", msg.status)
	}

	if msg.attempt != 1 {
		t.Fatalf("attempt must be 1, got %d", msg.attempt)
	}

	if msg.err != "handler boom" {
		t.Fatalf("err must be stored, got %q", msg.err)
	}

	if !msg.scheduledAt.After(time.Now().UTC()) {
		t.Fatalf("scheduledAt must be in the future, got %s", msg.scheduledAt)
	}

	if len(outbox.completed) != 0 {
		t.Fatalf("expected no completed messages, got %d", len(outbox.completed))
	}

	// Error must be reported to Errors() channel.
	select {
	case err := <-reader.Errors():
		var handleErr *HandleError
		if !errors.As(err, &handleErr) {
			t.Fatalf("expected HandleError, got %T: %v", err, err)
		}
	default:
		t.Fatal("expected HandleError on Errors()")
	}
}

func TestReaderDiscardsAfterMaxAttempts(t *testing.T) {
	outbox := &fakeOutboxReader{
		messages: []*Message{
			completedMessage(t, MessageWithID(uuid.New()), func(m *Message) {
				m.attempt = 5
			}),
		},
	}

	handlerCalled := false
	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		handlerCalled = true

		return nil
	}))

	reader.handleMessages()

	if handlerCalled {
		t.Fatal("handler must not be called after max attempts")
	}

	if len(outbox.discarded) != 1 {
		t.Fatalf("expected 1 discarded message, got %d", len(outbox.discarded))
	}

	if outbox.discarded[0].status != MessageStatusDiscarded {
		t.Fatalf("message must be DISCARDED, got %q", outbox.discarded[0].status)
	}
}

func TestReaderSkipsFutureMessage(t *testing.T) {
	outbox := &fakeOutboxReader{
		messages: []*Message{
			completedMessage(t, MessageWithID(uuid.New()), MessageWithScheduledAt(time.Now().UTC().Add(time.Hour))),
		},
	}

	handlerCalled := false
	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		handlerCalled = true

		return nil
	}))

	reader.handleMessages()

	if handlerCalled {
		t.Fatal("handler must not be called for future message")
	}

	if len(outbox.completed) != 0 || len(outbox.discarded) != 0 || len(outbox.scheduled) != 0 {
		t.Fatalf("future message must not be mutated: completed=%d discarded=%d scheduled=%d",
			len(outbox.completed), len(outbox.discarded), len(outbox.scheduled))
	}
}

func TestReaderReportsReadError(t *testing.T) {
	outbox := &fakeOutboxReader{readErr: errors.New("select failed")}

	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		return nil
	}))

	reader.handleMessages()

	select {
	case err := <-reader.Errors():
		var readErr *ReadError
		if !errors.As(err, &readErr) {
			t.Fatalf("expected ReadError, got %T: %v", err, err)
		}
	default:
		t.Fatal("expected ReadError on Errors()")
	}

	if outbox.commitCalls != 0 {
		t.Fatalf("commit must not be called, got %d", outbox.commitCalls)
	}
}

func TestReaderReportsBeginError(t *testing.T) {
	outbox := &fakeOutboxReader{beginErr: errors.New("begin failed")}

	reader := NewReader(outbox, MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		return nil
	}))

	reader.handleMessages()

	select {
	case err := <-reader.Errors():
		var dbErr *DatabaseError
		if !errors.As(err, &dbErr) {
			t.Fatalf("expected DatabaseError, got %T: %v", err, err)
		}
	default:
		t.Fatal("expected DatabaseError on Errors()")
	}
}

func TestReaderStartStop(t *testing.T) {
	handlerCalled := make(chan struct{}, 1)
	outbox := &fakeOutboxReader{
		messages: []*Message{completedMessage(t, MessageWithID(uuid.New()))},
	}

	reader := NewReader(
		outbox,
		MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
			select {
			case handlerCalled <- struct{}{}:
			default:
			}

			return nil
		}),
		ReaderWithInterval(time.Hour),
	)

	reader.Start()

	select {
	case <-handlerCalled:
	case <-time.After(time.Second):
		t.Fatal("handler must be called after Start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := reader.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Stop must be idempotent.
	if err := reader.Stop(ctx); err != nil {
		t.Fatalf("second Stop must not fail: %v", err)
	}

	select {
	case _, ok := <-reader.Errors():
		if ok {
			t.Fatal("Errors() channel must be closed after Stop")
		}
	default:
		t.Fatal("Errors() channel must be closed after Stop")
	}
}
