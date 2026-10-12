-- +goose Up
CREATE TABLE lmt.route_point (
    route_point_id UUID PRIMARY KEY,
    route_id UUID NOT NULL,
    shipment_id UUID NOT NULL,
    routecar_point_id BIGINT NOT NULL DEFAULT 0,
    seq_num INT NOT NULL DEFAULT 0,
    distance BIGINT NOT NULL DEFAULT 0,

    is_sample BOOL NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lmt.route_point IS 'Таблица точек маршрута';
COMMENT ON COLUMN lmt.route_point.route_point_id IS 'Идентификатор точки маршрута';
COMMENT ON COLUMN lmt.route_point.route_id IS 'Идентификатор маршрута';
COMMENT ON COLUMN lmt.route_point.shipment_id IS 'Идентификатор перевозки';
COMMENT ON COLUMN lmt.route_point.routecar_point_id IS 'Идентификатор точки маршрута сквозной';
COMMENT ON COLUMN lmt.route_point.seq_num IS 'Порядковый номер точки маршрута';
COMMENT ON COLUMN lmt.route_point.distance IS 'Расстояние до предыдущей точки маршрута';

CREATE INDEX route_point_route_id_shipment_id_idx ON lmt.route_point (route_id, shipment_id);

-- +goose Down
DROP INDEX lmt.route_point_route_id_shipment_id_idx;
DROP TABLE lmt.route_point;
