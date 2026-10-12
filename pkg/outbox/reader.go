package outbox

import (
	"context"
	"log/slog"
	"math"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type MessageHandler interface {
	HandleMessage(ctx context.Context, message *Message) error
}

type MessageHandlerFunc func(ctx context.Context, message *Message) error

func (fn MessageHandlerFunc) HandleMessage(ctx context.Context, message *Message) error {
	return fn(ctx, message)
}

type Reader struct {
	started int32
	stopped int32
	errCh   chan error

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	outbox  OutboxReader
	handler MessageHandler

	interval    time.Duration
	maxAttempts int
	batchSize   int
	backoffBase float64
	backoffStep time.Duration

	updateTimeout  time.Duration
	handlerTimeout time.Duration

	messageType string
	delayFunc   DelayFunc
}

type DelayFunc func(attempt int) time.Duration

type ReaderOption func(r *Reader)

func ReaderWithMessageType(messageType string) ReaderOption {
	return func(r *Reader) {
		r.messageType = messageType
	}
}

func ReaderWithInterval(interval time.Duration) ReaderOption {
	return func(r *Reader) {
		r.interval = interval
	}
}

func ReaderWithHandlerTimeout(timeout time.Duration) ReaderOption {
	return func(r *Reader) {
		r.handlerTimeout = timeout
	}
}

func ReaderWithDelayFunc(fn DelayFunc) ReaderOption {
	return func(r *Reader) {
		r.delayFunc = fn
	}
}

func ReaderWithUpdateTimeout(timeout time.Duration) ReaderOption {
	return func(r *Reader) {
		r.updateTimeout = timeout
	}
}

func ReaderWithMaxAttempts(attempts int) ReaderOption {
	return func(r *Reader) {
		r.maxAttempts = attempts
	}
}

func ReaderWithBatchSize(size int) ReaderOption {
	return func(r *Reader) {
		r.batchSize = size
	}
}

func ReaderWithBackoff(base float64, step time.Duration) ReaderOption {
	return func(r *Reader) {
		r.backoffBase = base
		r.backoffStep = step
	}
}

func NewReader(
	outbox OutboxReader,
	handler MessageHandler,
	opts ...ReaderOption,
) *Reader {
	ctx, cancel := context.WithCancel(context.Background())

	const (
		interval       = 10 * time.Second
		handlerTimeout = 30 * time.Second
		updateTimeout  = 10 * time.Second
		maxAttempts    = 3

		batchSize   = 64
		backoffBase = 2
		backoffStep = time.Second
	)

	r := &Reader{
		outbox:         outbox,
		handler:        handler,
		ctx:            ctx,
		cancel:         cancel,
		interval:       interval,
		handlerTimeout: handlerTimeout,
		maxAttempts:    maxAttempts,
		batchSize:      batchSize,

		backoffBase: backoffBase,
		backoffStep: backoffStep,

		updateTimeout: updateTimeout,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Initialize after applying options
	// cause using reader values as parameters

	r.errCh = make(chan error, r.batchSize*2)
	r.delayFunc = func(attempt int) time.Duration {
		return time.Duration(math.Pow(r.backoffBase, float64(attempt))) * r.backoffStep
	}

	return r
}

func (r *Reader) Start() {
	if !atomic.CompareAndSwapInt32(&r.started, 0, 1) {
		return
	}

	r.wg.Add(1)
	go func() {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("outbox reader panic",
					slog.Any("panic", v),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()

		ticker := time.NewTicker(r.interval)

		defer r.wg.Done()
		defer close(r.errCh)
		defer ticker.Stop()

		r.handleMessages()

		for {
			select {
			case <-ticker.C:
				r.handleMessages()
			case <-r.ctx.Done():
				return
			}
		}
	}()
}

func (r *Reader) Errors() <-chan error {
	return r.errCh
}

func (r *Reader) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&r.stopped, 0, 1) {
		return nil
	}

	r.cancel() // signal stop

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Reader) handleMessages() {
	defer func() {
		if v := recover(); v != nil {
			slog.Error("outbox panic recovered",
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()

	readCtx, cancel := context.WithTimeout(r.ctx, 30*time.Second)
	defer cancel()

	ctx, err := r.outbox.BeginTx(readCtx)
	if err != nil {
		r.sendError(&DatabaseError{Err: err})
		return
	}
	defer func() {
		if err = r.outbox.Rollback(ctx); err != nil {
			r.sendError(&DatabaseError{Err: err})
		}
	}()

	messages, err := r.outbox.ReadMessages(ctx, r.messageType, r.batchSize)
	if err != nil {
		r.sendError(&ReadError{Err: err})
		return
	}

	messagesToComplete := make([]*Message, 0, len(messages))
	messagesToDiscard := make([]*Message, 0, len(messages))

	for _, msg := range messages {
		if r.handleMessage(ctx, msg) {
			switch msg.status {
			case MessageStatusCompleted:
				messagesToComplete = append(messagesToComplete, msg)
			case MessageStatusDiscarded:
				messagesToDiscard = append(messagesToDiscard, msg)
			}
		}
	}

	// use reader.ctx without cancel because when reader is stopped
	// we want to complete/discard messages anyways
	ctx = context.WithoutCancel(ctx)

	if len(messagesToComplete) > 0 {
		if err := r.outbox.CompleteMessages(ctx, messagesToComplete); err != nil {
			r.sendError(&CompleteError{Messages: copyMessages(messagesToComplete), Err: err})
		}
	}

	if len(messagesToDiscard) > 0 {
		if err := r.outbox.DiscardMessages(ctx, messagesToDiscard); err != nil {
			r.sendError(&DiscardError{Messages: copyMessages(messagesToDiscard), Err: err})
		}
	}

	if err := r.outbox.Commit(ctx); err != nil {
		r.sendError(&DatabaseError{Err: err})
		return
	}
}

func copyMessages(messages []*Message) []Message {
	copied := make([]Message, 0, len(messages))
	for _, msg := range messages {
		copied = append(copied, *msg)
	}
	return copied
}

func (r *Reader) handleMessage(txCtx context.Context, msg *Message) bool {
	ctx, cancel := context.WithTimeout(txCtx, r.handlerTimeout)
	defer cancel()

	if msg.scheduledAt.After(time.Now().UTC()) {
		return false
	}

	if msg.attempt >= r.maxAttempts {
		msg.discard()

		return true
	}

	if err := r.handler.HandleMessage(ctx, msg); err != nil {
		msg.incAttempt()

		r.sendError(&HandleError{Message: *msg, Err: err})

		if err = r.scheduleNextAttempt(txCtx, msg, err); err != nil {
			r.sendError(&ScheduleError{Message: *msg, Err: err})
		}

		return false
	}

	msg.complete()

	return true
}

func (r *Reader) sendError(err error) {
	select {
	case r.errCh <- err:
	default:
	}
}

func (r *Reader) scheduleNextAttempt(ctx context.Context, msg *Message, err error) error {
	ctx, cancel := context.WithTimeout(ctx, r.updateTimeout)
	defer cancel()

	msg.schedule(
		r.delayFunc(msg.attempt-1),
		err.Error(),
	)

	return r.outbox.ScheduleMessage(ctx, msg)
}
