-- +goose Up
ALTER TABLE lmt.reserve
ADD COLUMN expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE lmt.reserve
DROP COLUMN expires_at;
