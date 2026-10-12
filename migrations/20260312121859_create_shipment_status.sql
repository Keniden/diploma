-- +goose Up
CREATE TYPE lmt.shipment_status AS ENUM ('UNSPECIFIED', 'NEW', 'IN_PROGRESS', 'COMPLETED', 'CANCELED');

-- +goose Down
DROP TYPE lmt.shipment_status;
