package outbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeQuery struct {
	sql  string
	args []any
}

type fakeQuerier struct {
	queried  []fakeQuery
	rows     *fakeRows
	queryErr error
}

func (f *fakeQuerier) BeginTx(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (f *fakeQuerier) Commit(ctx context.Context) error {
	return nil
}

func (f *fakeQuerier) Rollback(ctx context.Context) error {
	return nil
}

func (f *fakeQuerier) Exec(ctx context.Context, sql string, arguments ...any) error {
	f.queried = append(f.queried, fakeQuery{sql: sql, args: arguments})

	return nil
}

func (f *fakeQuerier) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	f.queried = append(f.queried, fakeQuery{sql: sql, args: args})

	if f.queryErr != nil {
		return nil, f.queryErr
	}

	return f.rows, nil
}

type fakeRows struct {
	cells [][]any
	idx   int
	err   error
}

func (f *fakeRows) Scan(dest ...any) error {
	if f.idx >= len(f.cells) {
		return errors.New("scan out of range")
	}

	row := f.cells[f.idx]
	f.idx++

	for i, d := range dest {
		switch v := d.(type) {
		case *uuid.UUID:
			*v = row[i].(uuid.UUID)
		case *string:
			*v = row[i].(string)
		case *Payload:
			*v = row[i].(Payload)
		case *Metadata:
			*v = row[i].(Metadata)
		case *int:
			*v = row[i].(int)
		case *time.Time:
			*v = row[i].(time.Time)
		case *Status:
			*v = row[i].(Status)
		default:
			return errors.New("unsupported scan dest")
		}
	}

	return nil
}

func (f *fakeRows) Next() bool {
	return f.idx < len(f.cells)
}

func (f *fakeRows) Close() {}

func (f *fakeRows) Err() error {
	return f.err
}

func TestPostgresOutboxWriteMessage(t *testing.T) {
	querier := &fakeQuerier{}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	id := uuid.New()
	msg := NewMessage(
		Payload(`{"a":1}`),
		MessageWithID(id),
		MessageWithType("shipment.events"),
		MessageWithScheduledAt(time.Now().UTC()),
	)

	if err := outbox.WriteMessage(context.Background(), msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if len(querier.queried) != 1 {
		t.Fatalf("expected 1 query, got %d", len(querier.queried))
	}

	q := querier.queried[0]

	if !strings.Contains(q.sql, "INSERT INTO outbox_messages") ||
		!strings.Contains(q.sql, "id, type, payload, metadata, attempt, scheduled_at, status, err") ||
		!strings.Contains(q.sql, "VALUES ($1, $2, $3, $4, $5, $6, $7, $8)") {
		t.Fatalf("unexpected SQL: %s", q.sql)
	}

	if len(q.args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(q.args))
	}

	if q.args[0] != id || q.args[1] != "shipment.events" {
		t.Fatalf("unexpected args: %v", q.args)
	}
}

func TestPostgresOutboxCompleteMessages(t *testing.T) {
	querier := &fakeQuerier{}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	id1, id2 := uuid.New(), uuid.New()
	messages := []*Message{
		NewMessage(Payload("p"), MessageWithID(id1)),
		NewMessage(Payload("p"), MessageWithID(id2)),
	}

	if err := outbox.CompleteMessages(context.Background(), messages); err != nil {
		t.Fatalf("CompleteMessages failed: %v", err)
	}

	q := querier.queried[0]

	if !strings.Contains(q.sql, "status = $1") ||
		!strings.Contains(q.sql, "attempt = attempt + 1") ||
		!strings.Contains(q.sql, "id IN ($2, $3)") {
		t.Fatalf("unexpected SQL: %s", q.sql)
	}

	if q.args[0] != MessageStatusCompleted || q.args[1] != id1 || q.args[2] != id2 {
		t.Fatalf("unexpected args: %v", q.args)
	}
}

func TestPostgresOutboxDiscardMessages(t *testing.T) {
	querier := &fakeQuerier{}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	id := uuid.New()
	if err := outbox.DiscardMessages(context.Background(), []*Message{
		NewMessage(Payload("p"), MessageWithID(id)),
	}); err != nil {
		t.Fatalf("DiscardMessages failed: %v", err)
	}

	q := querier.queried[0]

	if !strings.Contains(q.sql, "status = $1") ||
		!strings.Contains(q.sql, "id IN ($2)") {
		t.Fatalf("unexpected SQL: %s", q.sql)
	}

	if q.args[0] != MessageStatusDiscarded || q.args[1] != id {
		t.Fatalf("unexpected args: %v", q.args)
	}
}

func TestPostgresOutboxScheduleMessage(t *testing.T) {
	querier := &fakeQuerier{}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	id := uuid.New()
	msg := NewMessage(Payload("p"), MessageWithID(id))
	msg.attempt = 2
	msg.err = "boom"
	msg.schedule(time.Minute, "boom")

	if err := outbox.ScheduleMessage(context.Background(), msg); err != nil {
		t.Fatalf("ScheduleMessage failed: %v", err)
	}

	q := querier.queried[0]

	if !strings.Contains(q.sql, "attempt = attempt + 1") ||
		!strings.Contains(q.sql, "scheduled_at = $1") ||
		!strings.Contains(q.sql, "err = $3") ||
		!strings.Contains(q.sql, "id = $4") {
		t.Fatalf("unexpected SQL: %s", q.sql)
	}

	if q.args[1] != MessageStatusTODO || q.args[2] != "boom" || q.args[3] != id {
		t.Fatalf("unexpected args: %v", q.args)
	}
}

func TestPostgresOutboxReadMessages(t *testing.T) {
	id := uuid.New()
	scheduledAt := time.Now().UTC()
	metadata := Metadata{"src": "wms"}

	rows := &fakeRows{
		cells: [][]any{
			{
				id,
				"shipment.events",
				Payload(`{"a":1}`),
				metadata,
				0,
				scheduledAt,
				Status(MessageStatusTODO),
				"",
			},
		},
	}

	querier := &fakeQuerier{rows: rows}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	messages, err := outbox.ReadMessages(context.Background(), "shipment.events", 64)
	if err != nil {
		t.Fatalf("ReadMessages failed: %v", err)
	}

	q := querier.queried[0]

	if !strings.Contains(q.sql, "WHERE type = $1 AND status = $2") ||
		!strings.Contains(q.sql, "scheduled_at < NOW()") ||
		!strings.Contains(q.sql, "LIMIT $3") ||
		!strings.Contains(q.sql, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("unexpected SQL: %s", q.sql)
	}

	if q.args[0] != "shipment.events" || q.args[1] != MessageStatusTODO || q.args[2] != 64 {
		t.Fatalf("unexpected args: %v", q.args)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.ID != id || msg.Type != "shipment.events" ||
		string(msg.Payload) != `{"a":1}` || msg.attempt != 0 ||
		!msg.scheduledAt.Equal(scheduledAt) || msg.status != MessageStatusTODO {
		t.Fatalf("scanned message mismatch: %+v", msg)
	}

	v, ok := msg.Metadata.Get("src")
	if !ok || v != "wms" {
		t.Fatalf("metadata mismatch: %v, %v", v, ok)
	}
}

func TestPostgresOutboxReadMessagesQueryError(t *testing.T) {
	querier := &fakeQuerier{queryErr: errors.New("boom")}
	outbox := NewPostgresOutbox(querier, "outbox_messages")

	if _, err := outbox.ReadMessages(context.Background(), "shipment.events", 64); err == nil {
		t.Fatal("expected error on query failure")
	}
}

func TestPostgresOutboxEmptyMessages(t *testing.T) {
	outbox := NewPostgresOutbox(&fakeQuerier{}, "outbox_messages")

	if err := outbox.CompleteMessages(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty messages list")
	}

	if err := outbox.DiscardMessages(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty messages list")
	}
}
