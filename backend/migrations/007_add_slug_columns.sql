-- 007_add_slug_columns.sql
-- Adds city_slug and county_slug for programmatic SEO landing pages.
-- e.g. city_slug = "worcester", county_slug = "worcester-county"
BEGIN;

ALTER TABLE auctions
    ADD COLUMN IF NOT EXISTS city_slug   TEXT,
    ADD COLUMN IF NOT EXISTS county_slug TEXT;

CREATE INDEX IF NOT EXISTS idx_auctions_city_slug   ON auctions (city_slug);
CREATE INDEX IF NOT EXISTS idx_auctions_county_slug ON auctions (county_slug);

COMMIT;
