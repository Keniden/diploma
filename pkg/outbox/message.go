package outbox

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID       uuid.UUID
	Type     string
	Payload  Payload
	Metadata Metadata

	attempt     int
	scheduledAt time.Time
	status      Status
	err         string
}

type MessageOption func(m *Message)

func MessageWithID(id uuid.UUID) MessageOption {
	return func(m *Message) {
		m.ID = id
	}
}

func MessageWithType(messageType string) MessageOption {
	return func(m *Message) {
		m.Type = messageType
	}
}

func MessageWithScheduledAt(scheduledAt time.Time) MessageOption {
	return func(m *Message) {
		m.scheduledAt = scheduledAt
	}
}

func MessageWithMetadata(md Metadata) MessageOption {
	return func(m *Message) {
		m.Metadata = md
	}
}

func NewMessage(
	payload Payload,
	opts ...MessageOption,
) *Message {
	m := &Message{
		ID:       uuid.New(),
		Type:     "",
		Payload:  payload,
		Metadata: make(Metadata),

		attempt:     0,
		scheduledAt: time.Now().UTC(),
		status:      MessageStatusTODO,
		err:         "",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *Message) discard() {
	m.status = MessageStatusDiscarded
}

func (m *Message) complete() {
	m.status = MessageStatusCompleted
}

func (m *Message) schedule(delay time.Duration, err string) {
	m.status = MessageStatusTODO
	m.scheduledAt = time.Now().UTC().Add(delay)
	m.err = err
}

func (m *Message) incAttempt() {
	m.attempt++
}

type Payload []byte

type Metadata map[string]any

func NewMetadata() Metadata {
	return make(Metadata)
}

func (md Metadata) Get(key string) (any, bool) {
	if md == nil {
		return nil, false
	}

	v, ok := md[key]
	return v, ok
}

func (md Metadata) Set(key string, value any) {
	if md == nil {
		return
	}

	md[key] = value
}

type Status string

const (
	MessageStatusTODO      = "TODO"
	MessageStatusCompleted = "COMPLETED"
	MessageStatusDiscarded = "DISCARDED"
)
