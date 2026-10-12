-- +goose Up
CREATE TYPE lmt.package_point_action AS ENUM ('UNSPECIFIED', 'LOAD', 'UNLOAD');
CREATE TYPE lmt.package_point_action_status AS ENUM ('UNSPECIFIED', 'TODO', 'DONE', 'CANCELED');

-- +goose Down
DROP TYPE lmt.package_point_action;
DROP TYPE lmt.package_point_action_status;
