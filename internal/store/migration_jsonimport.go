package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// jsonImportSnapshot mirrors the on-disk envelope of the legacy JSON store.
// Declared here (not in types.go) to keep the import path the only consumer
// of the old shape.
type jsonImportSnapshot struct {
	PRs        map[string]PRRecord `json:"prs"`
	LastPollAt *time.Time          `json:"last_poll_at,omitempty"`
}

// migrateJSONIfPresent performs the one-shot import from `state.json` (if
// any) into the SQLite database. Idempotent: once `meta.json_migrated_at`
// is set, subsequent calls are no-ops aside from renaming a leftover
// state.json file (if the previous run crashed between commit and rename).
//
// Rules:
//  1. If meta.json_migrated_at is set: if state.json still exists, rename it
//     to state.json.migrated-<unix> with a warning, then return.
//  2. If meta.json_migrated_at is absent and state.json does not exist:
//     return nil (fresh install, nothing to migrate).
//  3. Otherwise: parse JSON, insert PRs in a single transaction along with
//     meta.last_poll_at and meta.json_migrated_at, commit, then rename
//     state.json. On any transaction failure, state.json stays intact.
func migrateJSONIfPresent(ctx context.Context, db *sql.DB, jsonPath string, now time.Time, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}

	migratedAt, migratedPresent, err := getMeta(ctx, db, metaJSONMigratedAt)
	if err != nil {
		return err
	}
	if migratedPresent {
		if fileExists(jsonPath) {
			if err := renameAsMigrated(jsonPath, now); err != nil {
				log.WarnContext(ctx, "state.json leftover rename failed", "path", jsonPath, "err", err)
				return nil
			}
			log.WarnContext(ctx, "state.json was already migrated; renamed leftover",
				"path", jsonPath, "migrated_at", migratedAt)
		}
		return nil
	}

	if !fileExists(jsonPath) {
		return nil
	}

	b, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read state.json: %w", err)
	}
	var snap jsonImportSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return fmt.Errorf("decode state.json: %w", err)
	}

	count, err := runJSONImportTx(ctx, db, snap, now)
	if err != nil {
		return err
	}

	if err := renameAsMigrated(jsonPath, now); err != nil {
		log.WarnContext(ctx, "state.json migrated in DB but rename failed — next boot will rename leftover",
			"path", jsonPath, "err", err)
	}

	log.InfoContext(ctx, "migrated PRs from state.json to revu.db",
		"count", count, "state_json", jsonPath)
	return nil
}

// runJSONImportTx inserts every PR from snap into the DB and records the
// migration meta row. Errors trigger a rollback; state.json is untouched.
func runJSONImportTx(ctx context.Context, db *sql.DB, snap jsonImportSnapshot, now time.Time) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin migration tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO prs (
		id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, rec := range snap.PRs {
		state := rec.State
		if state == "" {
			state = "OPEN"
		}
		firstSeen := rec.FirstSeenAt
		if firstSeen.IsZero() {
			firstSeen = now
		}
		lastSeen := rec.LastSeenAt
		if lastSeen.IsZero() {
			lastSeen = now
		}
		// Legacy JSON never carried review_state. Mirror the migration 00004
		// backfill rule: pending rows become PENDING, non-pending rows become
		// COMMENTED so the REV-16 history predicate accepts them.
		reviewState := rec.ReviewState
		if reviewState == "" {
			if rec.ReviewPending {
				reviewState = "PENDING"
			} else {
				reviewState = "COMMENTED"
			}
		}
		if _, err := stmt.ExecContext(ctx,
			rec.ID, rec.Number, rec.Repo, rec.Title, rec.Author, rec.URL,
			state, boolToInt(rec.IsDraft), rec.Additions, rec.Deletions,
			boolToInt(rec.ReviewPending), reviewState, formatTime(firstSeen), formatTime(lastSeen),
			formatTimePtr(rec.LastNotifiedAt),
		); err != nil {
			return 0, fmt.Errorf("insert pr %s: %w", rec.ID, err)
		}
		count++
	}

	if snap.LastPollAt != nil {
		if _, err := tx.ExecContext(ctx, qSetMeta, metaLastPollAt, formatTime(*snap.LastPollAt)); err != nil {
			return 0, fmt.Errorf("set last_poll_at: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, qSetMeta, metaJSONMigratedAt, formatTime(now)); err != nil {
		return 0, fmt.Errorf("set json_migrated_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit migration: %w", err)
	}
	committed = true
	return count, nil
}

// getMeta is a package-level helper so the one-shot import code can peek
// at the meta table without constructing a full sqliteStore.
func getMeta(ctx context.Context, db *sql.DB, key string) (string, bool, error) {
	var v string
	err := db.QueryRowContext(ctx, qGetMeta, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get meta %s: %w", key, err)
	}
	return v, true, nil
}

// fileExists is a small helper; any error reading stat is treated as absent
// so migration is never blocked by permissions we can't act on.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// renameAsMigrated renames the legacy state.json file out of the way so the
// next boot sees a clean directory. Suffix is the Unix timestamp so repeated
// leftover-rename attempts don't collide.
func renameAsMigrated(jsonPath string, now time.Time) error {
	target := fmt.Sprintf("%s.migrated-%d", jsonPath, now.UTC().Unix())
	if err := os.Rename(jsonPath, target); err != nil {
		return fmt.Errorf("rename %s → %s: %w", jsonPath, target, err)
	}
	return nil
}
