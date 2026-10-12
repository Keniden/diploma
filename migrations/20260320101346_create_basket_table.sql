-- +goose Up
CREATE TABLE lmt.basket (
    driver_id UUID NOT NULL,
    item_id UUID NOT NULL,
    routecar_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (driver_id, item_id)
);

COMMENT ON TABLE lmt.basket IS 'Таблица корзины';
COMMENT ON COLUMN lmt.basket.driver_id IS 'Идентификатор водителя';
COMMENT ON COLUMN lmt.basket.item_id IS 'Идентификатор элемента';
COMMENT ON COLUMN lmt.basket.routecar_id IS 'Идентификатор маршрута';

-- +goose Down
DROP TABLE lmt.basket;
