-- +goose Up
CREATE TABLE lmt.package (
    package_id UUID NOT NULL,
    shipment_id UUID NOT NULL,

    is_return BOOLEAN NOT NULL DEFAULT FALSE,
    external_id BIGINT NOT NULL DEFAULT 0,
    tare_id TEXT NOT NULL DEFAULT '',

    is_sample BOOL NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE lmt.package IS 'Таблица грузомест';
COMMENT ON COLUMN lmt.package.package_id IS 'Идентификатор грузоместа';
COMMENT ON COLUMN lmt.package.shipment_id IS 'Идентификатор перевозки';
COMMENT ON COLUMN lmt.package.is_return IS 'Является ли возвратом';
COMMENT ON COLUMN lmt.package.external_id IS 'Внешний идентификатор грузоместа';
COMMENT ON COLUMN lmt.package.tare_id IS 'Идентификатор тары';

-- +goose Down
DROP TABLE lmt.package;
