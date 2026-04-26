-- 010_add_deep_link_columns.sql
-- Adds three columns to support deep linking from the property report page.
--
--   legal_description  – raw text captured by the scraper (e.g. "B13812/P506")
--                        used as source for Book/Page extraction.
--   registry_deep_link – full masslandrecords.com URL pre-seeded with Book/Page
--                        so the buyer lands directly on the deed, not the portal.
--   assessor_pid       – Vision Government Solutions (VGSI) numeric parcel ID;
--                        populated by the post-scrape second-pass function.
BEGIN;

ALTER TABLE auctions ADD COLUMN IF NOT EXISTS legal_description  TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_deep_link TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS assessor_pid       TEXT;

COMMIT;
