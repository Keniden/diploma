-- +goose Up
CREATE TABLE lmt.route (
    route_id UUID PRIMARY KEY,
    shipment_id UUID NOT NULL,
    routecar_id BIGINT NOT NULL DEFAULT 0,
    distance BIGINT NOT NULL DEFAULT 0,
    duration BIGINT NOT NULL DEFAULT 0,

    is_sample BOOL NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lmt.route IS 'Таблица маршрутов';
COMMENT ON COLUMN lmt.route.route_id IS 'Идентификатор маршрута';
COMMENT ON COLUMN lmt.route.shipment_id IS 'Идентификатор перевозки';
COMMENT ON COLUMN lmt.route.routecar_id IS 'Идентификатор транспортного маршрута';
COMMENT ON COLUMN lmt.route.distance IS 'Дистанция маршрута в метрах';
COMMENT ON COLUMN lmt.route.duration IS 'Длительность маршрута в секундах';

CREATE INDEX route_shipment_id_idx ON lmt.route (shipment_id);
CREATE INDEX route_routecar_id_idx ON lmt.route (routecar_id);

-- +goose Down
DROP INDEX lmt.route_shipment_id_idx;
DROP INDEX lmt.route_routecar_id_idx;
DROP TABLE lmt.route;
