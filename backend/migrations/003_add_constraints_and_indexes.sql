-- 003_add_constraints_and_indexes.sql
-- Adds the unique constraint required for upsert logic and performance indexes.
-- Must run AFTER 002 — will fail if duplicate (address, site_name) rows still exist.
BEGIN;

-- 1. Unique constraint — the ON CONFLICT target for all upsert queries
ALTER TABLE auctions
    ADD CONSTRAINT uq_auctions_address_site
    UNIQUE (address, site_name);

-- 2. Partial index for the "soonest upcoming" sort used by the API and Astro frontend.
--    Only indexes rows that will actually be queried — keeps it small and fast.
CREATE INDEX IF NOT EXISTS idx_auctions_upcoming
    ON auctions (date ASC)
    WHERE status NOT IN ('Past', 'Removed');

-- 3. Index for the per-site cleanup phase.
--    Lets the UPDATE ... WHERE site_name = $1 AND updated_at < cutoff run without a seq scan.
CREATE INDEX IF NOT EXISTS idx_auctions_site_updated
    ON auctions (site_name, updated_at DESC);

COMMIT;
