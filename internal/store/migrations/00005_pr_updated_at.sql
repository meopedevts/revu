-- +goose Up
-- updated_at espelha o `updatedAt` do `gh search prs` pro PR. Usado por
-- UpdateFromPoll pra detectar mudança em PR já conhecido (force-push,
-- novo commit, etc.) e gatilhar re-enrich de additions/deletions/state
-- — sem isso, o diff stats fica congelado nos valores capturados na
-- primeira passagem enquanto o PR continuar na lista de pendentes.
ALTER TABLE prs ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE prs DROP COLUMN updated_at;
