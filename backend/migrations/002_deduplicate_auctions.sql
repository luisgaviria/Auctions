-- 002_deduplicate_auctions.sql
-- Removes duplicate (address, site_name) rows before the unique constraint is added.
-- For each group of duplicates, keeps the row with the highest id.
-- Repoints any favorites referencing a duplicate row to the surviving canonical row.
BEGIN;

-- Step A: map every duplicate id to the canonical (max id) row for that group
CREATE TEMP TABLE dedup_map AS
SELECT
    dup.id  AS old_id,
    keep.id AS canonical_id
FROM auctions AS dup
JOIN (
    SELECT address, site_name, MAX(id) AS id
    FROM   auctions
    GROUP  BY address, site_name
    HAVING COUNT(*) > 1
) AS keep
  ON  keep.address   = dup.address
  AND keep.site_name = dup.site_name
  AND keep.id       != dup.id;   -- only the non-surviving rows

-- Step B: repoint favorites from duplicate rows to the canonical row,
--         but only when the user doesn't already have a favorite for the canonical
UPDATE favorites f
SET    auction_id = dm.canonical_id
FROM   dedup_map dm
WHERE  f.auction_id = dm.old_id
  AND  NOT EXISTS (
      SELECT 1 FROM favorites f2
      WHERE  f2.user_id    = f.user_id
        AND  f2.auction_id = dm.canonical_id
  );

-- Step C: drop any remaining favorites pointing to duplicate rows
--         (these are users who already favorited the canonical, so no data is lost)
DELETE FROM favorites f
USING  dedup_map dm
WHERE  f.auction_id = dm.old_id;

-- Step D: delete the duplicate auction rows
DELETE FROM auctions
WHERE id IN (SELECT old_id FROM dedup_map);

DROP TABLE dedup_map;

COMMIT;
