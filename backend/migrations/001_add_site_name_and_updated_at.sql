-- 001_add_site_name_and_updated_at.sql
-- Adds site_name and updated_at columns to the auctions table.
-- Backfills site_name from the existing logo path (e.g. '/baystate.webp' -> 'baystate').
-- Widens link to TEXT to handle long URLs from sites like Patriot and Jake.
BEGIN;

-- 1. Add new columns (nullable first so backfill can run before NOT NULL is enforced)
ALTER TABLE auctions
    ADD COLUMN IF NOT EXISTS site_name  VARCHAR(100),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 2. Widen link — VARCHAR(255) is too short for full hrefs built by several scrapers
ALTER TABLE auctions
    ALTER COLUMN link TYPE TEXT;

-- 3. Backfill site_name from logo path: '/baystate.webp' -> 'baystate'
UPDATE auctions
SET site_name = REPLACE(REPLACE(logo, '.webp', ''), '/', '')
WHERE site_name IS NULL
  AND logo IS NOT NULL
  AND logo != '';

-- 4. Any rows with no logo get a safe fallback so NOT NULL can be applied
UPDATE auctions
SET site_name = 'unknown'
WHERE site_name IS NULL OR site_name = '';

-- 5. Enforce NOT NULL now that every row has a value
ALTER TABLE auctions
    ALTER COLUMN site_name SET NOT NULL;

COMMIT;
