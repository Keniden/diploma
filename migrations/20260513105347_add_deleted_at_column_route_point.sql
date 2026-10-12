-- +goose Up
ALTER TABLE lmt.route_point
ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE lmt.route_point
DROP COLUMN deleted_at;
