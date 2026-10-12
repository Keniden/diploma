-- +goose Up

-- 1. shipment_status: lmt.shipment_status ENUM → smallint
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status DROP DEFAULT;
ALTER TABLE lmt.shipment
    ALTER COLUMN shipment_status TYPE smallint
        USING CASE shipment_status::text
            WHEN 'UNSPECIFIED' THEN 0
            WHEN 'NEW' THEN 1
            WHEN 'ASSIGNED' THEN 2
            WHEN 'LOADING' THEN 3
            WHEN 'IN_PROGRESS' THEN 4
            WHEN 'COMPLETED' THEN 5
            WHEN 'CANCELED' THEN 6
            ELSE 0
        END;
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status SET DEFAULT 1;

-- 2. package_status: lmt.package_status ENUM → smallint
UPDATE lmt.package SET package_status = 'UNSPECIFIED' WHERE package_status IS NULL;
ALTER TABLE lmt.package ALTER COLUMN package_status DROP DEFAULT;
ALTER TABLE lmt.package
    ALTER COLUMN package_status TYPE smallint
        USING CASE package_status::text
            WHEN 'UNSPECIFIED' THEN 0
            WHEN 'READY_TO_LOAD' THEN 1
            WHEN 'LOADED' THEN 2
            WHEN 'UNLOADING' THEN 3
            WHEN 'UNLOADED' THEN 4
            ELSE 0
        END;
ALTER TABLE lmt.package ALTER COLUMN package_status SET NOT NULL;
ALTER TABLE lmt.package ALTER COLUMN package_status SET DEFAULT 0;

-- 3. action: lmt.package_point_action ENUM → smallint
ALTER TABLE lmt.package_route_point ALTER COLUMN action DROP DEFAULT;
ALTER TABLE lmt.package_route_point
    ALTER COLUMN action TYPE smallint
        USING CASE action::text
            WHEN 'UNSPECIFIED' THEN 0
            WHEN 'LOAD' THEN 1
            WHEN 'UNLOAD' THEN 2
            ELSE 0
        END;
ALTER TABLE lmt.package_route_point ALTER COLUMN action SET DEFAULT 0;

-- 4. action_status: lmt.package_point_action_status ENUM → smallint
ALTER TABLE lmt.package_route_point ALTER COLUMN action_status DROP DEFAULT;
ALTER TABLE lmt.package_route_point
    ALTER COLUMN action_status TYPE smallint
        USING CASE action_status::text
            WHEN 'UNSPECIFIED' THEN 0
            WHEN 'TODO' THEN 1
            WHEN 'DONE' THEN 2
            WHEN 'CANCELED' THEN 3
            ELSE 0
        END;
ALTER TABLE lmt.package_route_point ALTER COLUMN action_status SET DEFAULT 1;

-- 5. Drop unused ENUM types
DROP TYPE IF EXISTS lmt.shipment_status;
DROP TYPE IF EXISTS lmt.package_status;
DROP TYPE IF EXISTS lmt.package_point_action;
DROP TYPE IF EXISTS lmt.package_point_action_status;

-- +goose Down

-- 1. Recreate ENUM types
CREATE TYPE lmt.shipment_status AS ENUM (
    'UNSPECIFIED', 'NEW', 'ASSIGNED', 'LOADING', 'IN_PROGRESS', 'COMPLETED', 'CANCELED'
);
CREATE TYPE lmt.package_status AS ENUM (
    'UNSPECIFIED', 'READY_TO_LOAD', 'LOADED', 'UNLOADING', 'UNLOADED'
);
CREATE TYPE lmt.package_point_action AS ENUM (
    'UNSPECIFIED', 'LOAD', 'UNLOAD'
);
CREATE TYPE lmt.package_point_action_status AS ENUM (
    'UNSPECIFIED', 'TODO', 'DONE', 'CANCELED'
);

-- 2. Convert shipment_status smallint → ENUM
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status DROP DEFAULT;
ALTER TABLE lmt.shipment
    ALTER COLUMN shipment_status TYPE lmt.shipment_status
        USING (CASE shipment_status
            WHEN 0 THEN 'UNSPECIFIED'::lmt.shipment_status
            WHEN 1 THEN 'NEW'::lmt.shipment_status
            WHEN 2 THEN 'ASSIGNED'::lmt.shipment_status
            WHEN 3 THEN 'LOADING'::lmt.shipment_status
            WHEN 4 THEN 'IN_PROGRESS'::lmt.shipment_status
            WHEN 5 THEN 'COMPLETED'::lmt.shipment_status
            WHEN 6 THEN 'CANCELED'::lmt.shipment_status
            ELSE 'NEW'::lmt.shipment_status
        END);
ALTER TABLE lmt.shipment ALTER COLUMN shipment_status SET DEFAULT 'NEW'::lmt.shipment_status;

-- 3. Convert package_status smallint → ENUM
ALTER TABLE lmt.package ALTER COLUMN package_status DROP DEFAULT;
ALTER TABLE lmt.package ALTER COLUMN package_status DROP NOT NULL;
ALTER TABLE lmt.package
    ALTER COLUMN package_status TYPE lmt.package_status
        USING (CASE package_status
            WHEN 0 THEN 'UNSPECIFIED'::lmt.package_status
            WHEN 1 THEN 'READY_TO_LOAD'::lmt.package_status
            WHEN 2 THEN 'LOADED'::lmt.package_status
            WHEN 3 THEN 'UNLOADING'::lmt.package_status
            WHEN 4 THEN 'UNLOADED'::lmt.package_status
            ELSE 'UNSPECIFIED'::lmt.package_status
        END);
ALTER TABLE lmt.package ALTER COLUMN package_status SET DEFAULT 'UNSPECIFIED'::lmt.package_status;

-- 4. Convert action smallint → ENUM
ALTER TABLE lmt.package_route_point ALTER COLUMN action DROP DEFAULT;
ALTER TABLE lmt.package_route_point
    ALTER COLUMN action TYPE lmt.package_point_action
        USING (CASE action
            WHEN 0 THEN 'UNSPECIFIED'::lmt.package_point_action
            WHEN 1 THEN 'LOAD'::lmt.package_point_action
            WHEN 2 THEN 'UNLOAD'::lmt.package_point_action
            ELSE 'UNSPECIFIED'::lmt.package_point_action
        END);
ALTER TABLE lmt.package_route_point ALTER COLUMN action SET DEFAULT 'UNSPECIFIED'::lmt.package_point_action;

-- 5. Convert action_status smallint → ENUM
ALTER TABLE lmt.package_route_point ALTER COLUMN action_status DROP DEFAULT;
ALTER TABLE lmt.package_route_point
    ALTER COLUMN action_status TYPE lmt.package_point_action_status
        USING (CASE action_status
            WHEN 0 THEN 'UNSPECIFIED'::lmt.package_point_action_status
            WHEN 1 THEN 'TODO'::lmt.package_point_action_status
            WHEN 2 THEN 'DONE'::lmt.package_point_action_status
            WHEN 3 THEN 'CANCELED'::lmt.package_point_action_status
            ELSE 'TODO'::lmt.package_point_action_status
        END);
ALTER TABLE lmt.package_route_point ALTER COLUMN action_status SET DEFAULT 'TODO'::lmt.package_point_action_status;
