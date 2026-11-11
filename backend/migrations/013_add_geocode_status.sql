-- +goose Up
ALTER TABLE auctions
    ADD COLUMN IF NOT EXISTS geocode_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS geocode_error TEXT,
    ADD COLUMN IF NOT EXISTS geocode_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS geocode_retry_after TIMESTAMPTZ;

-- +goose Down
ALTER TABLE auctions
    DROP COLUMN IF EXISTS geocode_retry_after,
    DROP COLUMN IF EXISTS geocode_attempts,
    DROP COLUMN IF EXISTS geocode_error,
    DROP COLUMN IF EXISTS geocode_status;
