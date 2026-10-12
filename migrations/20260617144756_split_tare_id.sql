-- +goose Up
-- 1. Add new columns
ALTER TABLE lmt.package ADD COLUMN tare_type TEXT NOT NULL DEFAULT '';
ALTER TABLE lmt.package ADD COLUMN tare_num BIGINT NOT NULL DEFAULT 0;

-- 2. Backfill: split tare_id at the letter/digit boundary
--    tare_type = everything before first digit
--    tare_num  = everything from first digit onwards (cast to bigint)
UPDATE lmt.package SET
    tare_type = COALESCE(SUBSTRING(tare_id FROM '^[^0-9]*'), ''),
    tare_num  = COALESCE(NULLIF(REGEXP_REPLACE(tare_id, '^[^0-9]*', ''), '')::BIGINT, 0);

-- 3. Drop old text column, rename new numeric column
ALTER TABLE lmt.package DROP COLUMN tare_id;
ALTER TABLE lmt.package RENAME COLUMN tare_num TO tare_id;

-- +goose Down
-- Reverse: recreate text column, drop new columns
ALTER TABLE lmt.package RENAME COLUMN tare_id TO tare_num;
ALTER TABLE lmt.package ADD COLUMN tare_id TEXT NOT NULL DEFAULT '';

-- Reconstruct original tare_id from tare_type + tare_num
UPDATE lmt.package SET
    tare_id = tare_type || tare_num::TEXT
WHERE tare_num > 0;

ALTER TABLE lmt.package DROP COLUMN tare_type;
ALTER TABLE lmt.package DROP COLUMN tare_num;
