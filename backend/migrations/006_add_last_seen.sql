-- 006_add_last_seen.sql
-- Adds a last_seen column for the Tag & Sweep sync strategy.
-- Every upsert stamps last_seen = NOW().  A post-scrape sweep deletes any row
-- whose last_seen is older than 6 hours, giving scrapers a grace period so a
-- single transient failure never wipes live data.
BEGIN;

ALTER TABLE auctions ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ;

-- Backfill: treat the last update time as the first "seen" timestamp so existing
-- rows are not swept on the very first run after deployment.
UPDATE auctions SET last_seen = updated_at WHERE last_seen IS NULL;

COMMIT;
