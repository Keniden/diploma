-- +goose Up
CREATE TYPE lmt.package_status AS ENUM ('UNSPECIFIED', 'READY_TO_LOAD', 'LOADED', 'UNLOADING', 'UNLOADED');

ALTER TABLE lmt.package
ADD COLUMN package_status lmt.package_status;

-- +goose Down
ALTER TABLE lmt.package
DROP COLUMN package_status;

DROP TYPE lmt.package_status;