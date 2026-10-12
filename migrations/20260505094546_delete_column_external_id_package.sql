-- +goose Up
ALTER TABLE lmt.package
DROP COLUMN external_id;

-- +goose Down
ALTER TABLE lmt.package
ADD COLUMN external_id BIGINT NOT NULL DEFAULT 0;
