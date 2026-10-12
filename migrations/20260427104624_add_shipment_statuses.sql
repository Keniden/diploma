-- +goose Up
ALTER TYPE lmt.shipment_status ADD VALUE 'ASSIGNED';
ALTER TYPE lmt.shipment_status ADD VALUE 'LOADING';

-- +goose Down
UPDATE lmt.shipment
SET shipment_status = 'NEW'::lmt.shipment_status
WHERE shipment_status IN ('ASSIGNED'::lmt.shipment_status, 'LOADING'::lmt.shipment_status);

ALTER TYPE lmt.shipment_status RENAME TO shipment_status_old;

CREATE TYPE lmt.shipment_status AS ENUM ('UNSPECIFIED', 'NEW', 'IN_PROGRESS', 'COMPLETED', 'CANCELED');

ALTER TABLE lmt.shipment ALTER COLUMN shipment_status DROP DEFAULT;
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status TYPE lmt.shipment_status USING (shipment_status::text::lmt.shipment_status);
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status SET DEFAULT 'NEW'::lmt.shipment_status;

DROP TYPE lmt.shipment_status_old;
