-- +goose Up
-- SQLite does not support DEFAULT (subquery) on ALTER TABLE, so we add the
-- column with a safe placeholder and backfill in the same migration. Existing
-- rows get mapped to whichever profile is currently active (the gh-cli seed
-- on a freshly migrated DB, or the user's current active profile on upgrade).
ALTER TABLE prs ADD COLUMN profile_id TEXT NOT NULL DEFAULT '';

UPDATE prs SET profile_id = COALESCE(
    (SELECT id FROM profiles WHERE is_active = 1 LIMIT 1),
    ''
);

CREATE INDEX idx_prs_profile_pending ON prs(profile_id, review_pending);

-- +goose Down
DROP INDEX IF EXISTS idx_prs_profile_pending;
-- SQLite 3.35+ supports DROP COLUMN; revu targets modernc/sqlite which
-- bundles a recent SQLite so this works in tests and production.
ALTER TABLE prs DROP COLUMN profile_id;
