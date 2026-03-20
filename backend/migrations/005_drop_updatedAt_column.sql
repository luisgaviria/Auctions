-- 005_drop_updatedAt_column.sql
-- Removes the redundant camelCase updatedAt column added by Supabase.
-- The snake_case updated_at column (added in 001) is the canonical one,
-- kept in sync by the trigger installed in 004.
BEGIN;

ALTER TABLE auctions DROP COLUMN IF EXISTS "updatedAt";

COMMIT;
