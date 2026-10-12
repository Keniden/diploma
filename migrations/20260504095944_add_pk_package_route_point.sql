-- +goose Up
ALTER TABLE lmt.package_route_point
ADD COLUMN shipment_id UUID NOT NULL;

ALTER TABLE lmt.package_route_point
ADD PRIMARY KEY (package_id, route_point_id, shipment_id);

-- +goose Down
ALTER TABLE lmt.package_route_point
DROP CONSTRAINT package_route_point_pkey;

ALTER TABLE lmt.package_route_point
DROP COLUMN shipment_id;
