-- +goose Up
ALTER TABLE lmt.package_route_point
ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE lmt.package_route_point
DROP COLUMN deleted_at;
