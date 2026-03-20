-- 004_updated_at_trigger.sql
-- Creates a reusable trigger function and attaches it to the auctions table.
-- The trigger bumps updated_at on every UPDATE, which the cleanup phase relies on
-- to identify auctions not seen in the last scrape run.
BEGIN;

-- 1. Reusable function — can be attached to any table that has an updated_at column
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- 2. Attach to auctions
DROP TRIGGER IF EXISTS trg_auctions_updated_at ON auctions;
CREATE TRIGGER trg_auctions_updated_at
    BEFORE UPDATE ON auctions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;
