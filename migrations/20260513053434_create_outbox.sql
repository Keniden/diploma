-- +goose Up
CREATE TABLE lmt.outbox (
    id UUID NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    attempt INTEGER NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'TODO',
    err TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE lmt.outbox;
