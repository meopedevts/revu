-- +goose Up
-- REV-54 enriquece o card com branch (headRefName) e avatar do autor.
-- Branch sai do gh search/full-view; avatarUrl idem. Nullable enquanto o
-- backfill via poll/enrich não passa por todos os registros existentes —
-- frontend trata "" como ausente e cai no fallback visual.
ALTER TABLE prs ADD COLUMN branch TEXT NOT NULL DEFAULT '';
ALTER TABLE prs ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE prs DROP COLUMN avatar_url;
ALTER TABLE prs DROP COLUMN branch;
