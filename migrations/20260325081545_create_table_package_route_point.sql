-- +goose Up
CREATE TABLE lmt.package_route_point (
    package_id UUID NOT NULL,
    route_point_id UUID NOT NULL,

    action lmt.package_point_action NOT NULL DEFAULT 'UNSPECIFIED',
    action_status lmt.package_point_action_status NOT NULL DEFAULT 'TODO',
    action_status_updated_at TIMESTAMPTZ,

    is_sample BOOL NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lmt.package IS 'Таблица грузомест';
COMMENT ON COLUMN lmt.package_route_point.package_id IS 'Идентификатор грузоместа';
COMMENT ON COLUMN lmt.package_route_point.route_point_id IS 'Идентификатор точки маршрута';

COMMENT ON COLUMN lmt.package_route_point.action IS 'Действие';
COMMENT ON COLUMN lmt.package_route_point.action_status IS 'Статус действия';
COMMENT ON COLUMN lmt.package_route_point.action_status_updated_at IS 'Время обновления статуса действия';

-- +goose Down
DROP TABLE lmt.package_route_point;
