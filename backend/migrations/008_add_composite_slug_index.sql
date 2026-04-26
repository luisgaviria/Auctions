-- 008_add_composite_slug_index.sql
-- Replaces the two single-column slug indexes with one composite index that
-- covers the exact WHERE clause used by GET /auctions/:county_slug/:city_slug.
-- A composite (county_slug, city_slug) index satisfies both slug filters in a
-- single index scan; the individual indexes from 007 are kept as they still
-- help county-only queries.
BEGIN;

CREATE INDEX IF NOT EXISTS idx_auctions_county_city_slug
    ON auctions (county_slug, city_slug)
    WHERE status NOT IN (
        'cancelled','sold','removed','canceled',
        'sold back to mortgagee','back to mortgagee',
        'past','3rd party purchase','postponed'
    );

COMMIT;
