-- +goose Up
-- review_state captures the state of the review I personally submitted on the
-- PR: PENDING (not reviewed yet) | APPROVED | CHANGES_REQUESTED | COMMENTED.
-- REV-16 decouples this from review_pending (which tracked "still in the gh
-- search result") so the UI can show two badges and keep a merged-but-not-yet-
-- reviewed PR out of the history tab.
ALTER TABLE prs ADD COLUMN review_state TEXT NOT NULL DEFAULT 'PENDING';

-- Backfill: rows with review_pending=1 clearly weren't reviewed yet; rows with
-- review_pending=0 had been dropped from the search previously (either review
-- was submitted OR the PR was closed). We can't distinguish those two cases
-- from history alone, so we mark them COMMENTED — the next enrich tick will
-- correct it with the actual review state fetched from gh.
UPDATE prs SET review_state = CASE
    WHEN review_pending = 1 THEN 'PENDING'
    ELSE 'COMMENTED'
END;

CREATE INDEX idx_prs_review_state ON prs(review_state);

-- +goose Down
DROP INDEX IF EXISTS idx_prs_review_state;
ALTER TABLE prs DROP COLUMN review_state;
