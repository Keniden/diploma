-- +goose Up
CREATE TABLE lmt.reserve (
    reserve_id UUID PRIMARY KEY,
    route_point_id UUID NOT NULL,
    shipment_id UUID NOT NULL,
    task_id TEXT NOT NULL,
    scanned_at TIMESTAMP WITH TIME ZONE,
    gate_id UUID NOT NULL,
    gate_name VARCHAR(255) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lmt.reserve IS 'Таблица резервов ворот';
COMMENT ON COLUMN lmt.reserve.reserve_id IS 'Идентификатор резерва';
COMMENT ON COLUMN lmt.reserve.route_point_id IS 'Идентификатор точки маршрута';
COMMENT ON COLUMN lmt.reserve.shipment_id IS 'Идентификатор перевозки';
COMMENT ON COLUMN lmt.reserve.task_id IS 'Идентификатор задачи';
COMMENT ON COLUMN lmt.reserve.scanned_at IS 'Дата и время сканирования';
COMMENT ON COLUMN lmt.reserve.gate_id IS 'Идентификатор ворот';
COMMENT ON COLUMN lmt.reserve.gate_name IS 'Название ворот';

CREATE INDEX reserve_route_point_id_shipment_id_idx ON lmt.reserve (route_point_id, shipment_id);

-- +goose Down
DROP INDEX lmt.reserve_route_point_id_shipment_id_idx;
DROP TABLE lmt.reserve;
