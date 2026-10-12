-- +goose Up

CREATE INDEX IF NOT EXISTS reserve_status_expires_at_idx
ON lmt.reserve (status, expires_at)
WHERE expires_at IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS lmt.reserve_status_expires_at_idx;
