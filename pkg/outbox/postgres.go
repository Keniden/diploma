package outbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TODO(LASTMILE-040): the real repo builds queries with Masterminds/squirrel,
// which is not in this skeleton's go.mod. Plain SQL with $N placeholders is
// used instead; column names (id, type, payload, metadata, attempt,
// scheduled_at, status, err) were inferred from the real repo's builder calls
// and the migrations. Switch back to squirrel once it is allowed in go.mod.

const outboxColumns = "id, type, payload, metadata, attempt, scheduled_at, status, err"

type PostgresOutbox struct {
	Querier
	table string
}

func NewPostgresOutbox(q Querier, table string) *PostgresOutbox {
	return &PostgresOutbox{
		Querier: q,
		table:   table,
	}
}

func (o *PostgresOutbox) WriteMessage(ctx context.Context, message *Message) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		o.table, outboxColumns, placeholders(8),
	)

	args := []any{
		message.ID,
		message.Type,
		message.Payload,
		message.Metadata,
		message.attempt,
		message.scheduledAt,
		message.status,
		message.err,
	}

	if err := o.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("executor: %w", err)
	}

	return nil
}

func (o *PostgresOutbox) DiscardMessages(ctx context.Context, messages []*Message) error {
	ids, err := messageIDs(messages)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(
		"UPDATE %s SET status = $1 WHERE id IN (%s)",
		o.table, placeholdersFrom(2, len(ids)),
	)

	args := make([]any, 0, len(ids)+1)
	args = append(args, MessageStatusDiscarded)
	for _, id := range ids {
		args = append(args, id)
	}

	if err := o.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("executor: %w", err)
	}

	return nil
}

func (o *PostgresOutbox) CompleteMessages(ctx context.Context, messages []*Message) error {
	ids, err := messageIDs(messages)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(
		"UPDATE %s SET status = $1, attempt = attempt + 1 WHERE id IN (%s)",
		o.table, placeholdersFrom(2, len(ids)),
	)

	args := make([]any, 0, len(ids)+1)
	args = append(args, MessageStatusCompleted)
	for _, id := range ids {
		args = append(args, id)
	}

	if err := o.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("executor: %w", err)
	}

	return nil
}

func (o *PostgresOutbox) ScheduleMessage(ctx context.Context, msg *Message) error {
	query := fmt.Sprintf(
		"UPDATE %s SET attempt = attempt + 1, scheduled_at = $1, status = $2, err = $3 WHERE id = $4",
		o.table,
	)

	args := []any{
		msg.scheduledAt,
		MessageStatusTODO,
		msg.err,
		msg.ID,
	}

	if err := o.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("executor: %w", err)
	}

	return nil
}

func (o *PostgresOutbox) ReadMessages(
	ctx context.Context,
	messageType string,
	batchSize int,
) ([]*Message, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE type = $1 AND status = $2 AND scheduled_at < NOW() ORDER BY scheduled_at LIMIT $3 FOR UPDATE SKIP LOCKED",
		outboxColumns, o.table,
	)

	rows, err := o.Query(ctx, query, messageType, MessageStatusTODO, batchSize)
	if err != nil {
		return nil, fmt.Errorf("querier: %w", err)
	}
	defer rows.Close()

	var events []*Message

	for rows.Next() {
		event := new(Message)

		if err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Payload,
			&event.Metadata,
			&event.attempt,
			&event.scheduledAt,
			&event.status,
			&event.err,
		); err != nil {
			return nil, fmt.Errorf("scanner: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}

	return events, nil
}

func messageIDs(messages []*Message) ([]uuid.UUID, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages: empty list")
	}

	ids := make([]uuid.UUID, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			return nil, fmt.Errorf("messages: nil message")
		}

		ids = append(ids, msg.ID)
	}

	return ids, nil
}

func placeholders(n int) string {
	return placeholdersFrom(1, n)
}

func placeholdersFrom(start, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", start+i)
	}

	return strings.Join(parts, ", ")
}
