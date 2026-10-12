-- +goose Up
-- Insert IN_TRANSIT between LOADED and UNLOADING.
-- Old: UNSPECIFIED=0, READY_TO_LOAD=1, LOADED=2, UNLOADING=3, UNLOADED=4, CANCELED=5
-- New: UNSPECIFIED=0, READY_TO_LOAD=1, LOADED=2, IN_TRANSIT=3, UNLOADING=4, UNLOADED=5, CANCELED=6

UPDATE lmt.package SET package_status = 6 WHERE package_status = 5;
UPDATE lmt.package SET package_status = 5 WHERE package_status = 4;
UPDATE lmt.package SET package_status = 4 WHERE package_status = 3;

-- +goose Down
UPDATE lmt.package SET package_status = 3 WHERE package_status = 4;
UPDATE lmt.package SET package_status = 4 WHERE package_status = 5;
UPDATE lmt.package SET package_status = 5 WHERE package_status = 6;
