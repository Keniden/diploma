-- +goose Up
ALTER TABLE lmt.package
ADD PRIMARY KEY (package_id, shipment_id);

-- +goose Down
ALTER TABLE lmt.package
DROP CONSTRAINT package_pkey;
