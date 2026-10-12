-- +goose Up

ALTER TABLE lmt.reserve
ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 0;

ALTER TABLE lmt.reserve
ALTER COLUMN gate_name SET DEFAULT '';

CREATE INDEX IF NOT EXISTS reserve_shipment_id_status_idx ON lmt.reserve (shipment_id, status);

-- +goose Down

DROP INDEX IF EXISTS lmt.reserve_shipment_id_status_idx;

ALTER TABLE lmt.reserve
DROP COLUMN IF EXISTS status;

