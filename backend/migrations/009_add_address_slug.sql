-- 009_add_address_slug.sql
-- Adds address_slug for shareable property report URLs.
-- e.g. "123 MAIN STREET" + "WORCESTER" → "123-main-street-worcester"
-- The column is TEXT (not UNIQUE) because two different auctioneers can
-- list the same physical address; the report route returns the first
-- active match, which is the correct behaviour.
BEGIN;

ALTER TABLE auctions ADD COLUMN IF NOT EXISTS address_slug TEXT;

CREATE INDEX IF NOT EXISTS idx_auctions_address_slug ON auctions (address_slug);

COMMIT;
