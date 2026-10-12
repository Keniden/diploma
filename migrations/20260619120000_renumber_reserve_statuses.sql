-- +goose Up
-- Renumber reserve status Released from 4 to 5.
-- Old: Queued=0, WaitingApproval=1, Active=2, Expired=3, Released=4
-- New: Queued=0, WaitingApproval=1, Active=2, ExpiredWaitingApproval=3, ExpiredActive=4, Released=5

UPDATE lmt.reserve SET status = 5 WHERE status = 4;

-- +goose Down

UPDATE lmt.reserve SET status = 4 WHERE status = 5;
