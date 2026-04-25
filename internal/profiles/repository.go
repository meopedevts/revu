package profiles

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// DBTX is the subset of [*sql.DB] / [*sql.Tx] the repository needs. Accepting
// either lets callers opt into transactions for multi-step operations.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Repository performs CRUD on the `profiles` table. No business rules here —
// the service layer owns validation, activation, and keyring side-effects.
type Repository struct {
	db *sql.DB
}

// NewRepository wraps a [*sql.DB]. The store owns the handle lifecycle.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns all profiles ordered by created_at ASC. Never nil.
func (r *Repository) List(ctx context.Context) ([]Profile, error) {
	rows, err := r.db.QueryContext(ctx, qSelectAll)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	defer rows.Close()
	out := make([]Profile, 0)
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profiles: %w", err)
	}
	return out, nil
}

// Get returns the profile by id, or ErrNotFound.
func (r *Repository) Get(ctx context.Context, id string) (Profile, error) {
	return r.getFrom(ctx, r.db, id)
}

// GetActive returns the single active profile or ErrNoActiveProfile.
// Exactly one row should match thanks to the unique partial index seeded by
// migration 00002.
func (r *Repository) GetActive(ctx context.Context) (Profile, error) {
	row := r.db.QueryRowContext(ctx, qSelectActive)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNoActiveProfile
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get active: %w", err)
	}
	return p, nil
}

// Create inserts a new profile. If active=true, the caller is responsible for
// running the activation transaction (service layer does it).
func (r *Repository) Create(ctx context.Context, p Profile) error {
	_, err := r.db.ExecContext(ctx, qInsert,
		p.ID, p.Name, string(p.AuthMethod), nullString(p.KeyringRef),
		nullString(p.GitHubUsername), boolToInt(p.IsActive),
		p.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullTimePtr(p.LastValidatedAt),
	)
	if err != nil {
		if isUniqueNameViolation(err) {
			return ErrNameTaken
		}
		return fmt.Errorf("insert profile: %w", err)
	}
	return nil
}

// UpdateFields rewrites name, auth_method, keyring_ref, github_username, and
// last_validated_at for the profile. Pass the fully-resolved values; service
// layer handles "leave alone" semantics.
func (r *Repository) UpdateFields(ctx context.Context, p Profile) error {
	res, err := r.db.ExecContext(ctx, qUpdateFields,
		p.Name, string(p.AuthMethod), nullString(p.KeyringRef),
		nullString(p.GitHubUsername), nullTimePtr(p.LastValidatedAt), p.ID,
	)
	if err != nil {
		if isUniqueNameViolation(err) {
			return ErrNameTaken
		}
		return fmt.Errorf("update profile: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update profile rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the profile by id. Service layer ensures invariants (not
// last, not active).
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, qDelete, id)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete profile rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive flips is_active atomically: clears the current active row and
// sets target's flag. Target must exist. Runs inside a transaction so the
// unique partial index never sees two rows with is_active=1 mid-flight.
func (r *Repository) SetActive(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Ensure target exists.
	if _, err := r.getFrom(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, qClearActive); err != nil {
		return fmt.Errorf("clear active: %w", err)
	}
	if _, err := tx.ExecContext(ctx, qSetActive, id); err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set-active: %w", err)
	}
	committed = true
	return nil
}

// Count returns the total number of profiles.
func (r *Repository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, qCount).Scan(&n); err != nil {
		return 0, fmt.Errorf("count profiles: %w", err)
	}
	return n, nil
}

// getFrom reads a single row from either [*sql.DB] or [*sql.Tx].
func (r *Repository) getFrom(ctx context.Context, q DBTX, id string) (Profile, error) {
	row := q.QueryRowContext(ctx, qSelectByID, id)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}

// NewID returns a 32-hex random identifier matching the seed in migration
// 00002.
func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("rand id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProfile(s scanner) (Profile, error) {
	var (
		p                    Profile
		method               string
		keyringRef, username sql.NullString
		createdAt            string
		lastValidated        sql.NullString
		isActive             int
	)
	err := s.Scan(
		&p.ID, &p.Name, &method, &keyringRef, &username,
		&isActive, &createdAt, &lastValidated,
	)
	if err != nil {
		return Profile{}, err
	}
	p.AuthMethod = AuthMethod(method)
	if keyringRef.Valid {
		p.KeyringRef = keyringRef.String
	}
	if username.Valid {
		p.GitHubUsername = username.String
	}
	p.IsActive = isActive != 0
	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Profile{}, fmt.Errorf("parse created_at: %w", err)
	}
	p.CreatedAt = t
	if lastValidated.Valid {
		t, err := time.Parse(time.RFC3339Nano, lastValidated.String)
		if err != nil {
			return Profile{}, fmt.Errorf("parse last_validated_at: %w", err)
		}
		p.LastValidatedAt = &t
	}
	return p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// isUniqueNameViolation is a coarse check on modernc sqlite's error text.
// Good enough for our single unique constraint (profiles.name).
func isUniqueNameViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed: profiles.name")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
