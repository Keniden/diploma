-- +goose Up
CREATE TABLE lmt.shipment (
    shipment_id UUID PRIMARY KEY,
    shipment_short_id BIGINT NOT NULL,
    shipment_status lmt.shipment_status NOT NULL DEFAULT 'NEW',
    driver_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    supplier_id BIGINT NOT NULL DEFAULT 0,
    car_license_plate TEXT,
    closed_at TIMESTAMPTZ,

    is_sample BOOL NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE lmt.shipment IS 'Таблица для хранения информации о перевозках';
COMMENT ON COLUMN lmt.shipment.shipment_id IS 'Идентификатор перевозки';
COMMENT ON COLUMN lmt.shipment.shipment_short_id IS 'Человекочитаемый идентификатор перевозки';
COMMENT ON COLUMN lmt.shipment.shipment_status IS 'Статус перевозки';
COMMENT ON COLUMN lmt.shipment.driver_id IS 'Идентификатор водителя';
COMMENT ON COLUMN lmt.shipment.supplier_id IS 'Идентификатор поставщика';
COMMENT ON COLUMN lmt.shipment.car_license_plate IS 'Номер ТС';
COMMENT ON COLUMN lmt.shipment.closed_at IS 'Время закрытия перевозки';

CREATE INDEX shipment_driver_id_idx ON lmt.shipment (driver_id);

-- +goose Down
DROP INDEX lmt.shipment_driver_id_idx;
DROP TABLE lmt.shipment;
