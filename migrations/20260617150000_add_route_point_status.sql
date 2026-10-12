-- +goose Up

ALTER TABLE lmt.route_point
ADD COLUMN IF NOT EXISTS route_point_status smallint NOT NULL DEFAULT 0;

COMMENT ON COLUMN lmt.route_point.route_point_status IS 'Статус точки маршрута';

-- +goose Down

ALTER TABLE lmt.route_point
DROP COLUMN IF EXISTS route_point_status;
