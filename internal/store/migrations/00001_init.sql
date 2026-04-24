-- +goose Up
CREATE TABLE prs (
    id TEXT PRIMARY KEY,
    number INTEGER NOT NULL,
    repo TEXT NOT NULL,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    url TEXT NOT NULL,
    state TEXT NOT NULL,
    is_draft INTEGER NOT NULL,
    additions INTEGER NOT NULL DEFAULT 0,
    deletions INTEGER NOT NULL DEFAULT 0,
    review_pending INTEGER NOT NULL,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    last_notified_at TEXT
);

CREATE TABLE meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX idx_prs_pending ON prs(review_pending) WHERE review_pending = 1;
CREATE INDEX idx_prs_state ON prs(state);

-- +goose Down
DROP INDEX IF EXISTS idx_prs_state;
DROP INDEX IF EXISTS idx_prs_pending;
DROP TABLE IF EXISTS meta;
DROP TABLE IF EXISTS prs;
