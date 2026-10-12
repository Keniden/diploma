-- +goose Up
-- Renumber shipment_status from old values to new values.
-- Old: UNSPECIFIED=0, NEW=1, ASSIGNED=2, LOADING=3, IN_PROGRESS=4, COMPLETED=5, CANCELED=6
-- New: UNSPECIFIED=0, NEW=1, ASSIGNED=2, RESERVE_LOADING=3, LOADING=4, IN_PROGRESS=5,
--       RESERVE_UNLOADING=6, UNLOADING=7, COMPLETED=8, CANCELED=9

-- CANCELED: renumber to new value.
UPDATE lmt.shipment SET shipment_status = 9 WHERE shipment_status = 6;

-- All completed/active shipments: finish as COMPLETED.
UPDATE lmt.shipment SET shipment_status = 8 WHERE shipment_status IN (3, 4, 5);

-- +goose Down
-- Restore old values from new COMPLETED/CANCELED (irreversible for 3/4 → 8, restore as 5).
UPDATE lmt.shipment SET shipment_status = 5 WHERE shipment_status = 8;
UPDATE lmt.shipment SET shipment_status = 6 WHERE shipment_status = 9;
