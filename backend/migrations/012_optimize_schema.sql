-- +goose Up

-- 1. Create indexes for heavily filtered queries
CREATE INDEX IF NOT EXISTS idx_auctions_status_date ON auctions(LOWER(status), date);
CREATE INDEX IF NOT EXISTS idx_auctions_county_city ON auctions(LOWER(county_slug), LOWER(city_slug));
CREATE INDEX IF NOT EXISTS idx_auctions_address_slug ON auctions(address_slug);