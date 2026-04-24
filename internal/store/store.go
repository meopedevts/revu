package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// Store is the persistence surface used by the poller and the UI bridge.
type Store interface {
	Load() error
	Save() error
	GetAll() []PRRecord
	GetPending() []PRRecord
	GetHistory() []PRRecord
	UpdateFromPoll(prs []github.PRSummary) (novos []PRRecord)
	RefreshPRStatus(id string, details github.PRDetails) error
	SetRetentionDays(days int)
}

// ErrNotFound is returned by RefreshPRStatus when the id is unknown.
var ErrNotFound = errors.New("pr not tracked")

// Option customizes the fileStore.
type Option func(*fileStore)

// WithClock injects a time source; useful for deterministic tests.
func WithClock(now func() time.Time) Option {
	return func(s *fileStore) { s.now = now }
}

// WithRetention sets the number of days a non-OPEN PR is kept after it was
// last seen. Zero disables retention (history grows unbounded).
func WithRetention(days int) Option {
	return func(s *fileStore) { s.retentionDays = days }
}

// New constructs a Store backed by a JSON file at path. Path is only created
// on first Save — Load tolerates a missing file (fresh install).
func New(path string, opts ...Option) Store {
	s := &fileStore{
		path:          path,
		prs:           map[string]PRRecord{},
		now:           time.Now,
		retentionDays: 30,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type fileStore struct {
	path          string
	now           func() time.Time
	retentionDays int

	mu         sync.RWMutex
	prs        map[string]PRRecord
	lastPollAt *time.Time
}

func (s *fileStore) Load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if snap.PRs == nil {
		snap.PRs = map[string]PRRecord{}
	}
	s.prs = snap.PRs
	s.lastPollAt = snap.LastPollAt
	return nil
}

// Save writes the current state atomically: it writes to a temp file in the
// target directory, fsyncs it, and renames over the destination. Retention
// runs first so the on-disk file never contains entries that should have
// been evicted.
func (s *fileStore) Save() error {
	s.mu.Lock()
	s.applyRetentionLocked()
	snap := snapshot{
		PRs:        s.prs,
		LastPollAt: s.lastPollAt,
	}
	s.mu.Unlock()

	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	// On any error past this point we want the temp file gone.
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}

func (s *fileStore) GetAll() []PRRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedRecords(s.prs, func(a, b PRRecord) bool {
		return a.LastSeenAt.After(b.LastSeenAt)
	})
}

func (s *fileStore) GetPending() []PRRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedFiltered(s.prs, func(r PRRecord) bool { return r.ReviewPending })
}

func (s *fileStore) GetHistory() []PRRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedFiltered(s.prs, func(r PRRecord) bool { return !r.ReviewPending })
}

// UpdateFromPoll merges a fresh `gh search prs` result into the store. A PR
// counts as "novo" when (a) it was never seen, or (b) review_pending flipped
// false→true since the last poll. PRs that disappear from the result are
// marked review_pending=false (request removed, review submitted, or PR
// closed — store cannot distinguish without a status refresh).
func (s *fileStore) UpdateFromPoll(prs []github.PRSummary) []PRRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	polledIDs := make(map[string]struct{}, len(prs))
	var novos []PRRecord

	for _, pr := range prs {
		polledIDs[pr.ID] = struct{}{}
		existing, found := s.prs[pr.ID]
		if !found {
			rec := PRRecord{
				ID:            pr.ID,
				Number:        pr.Number,
				Repo:          pr.Repo,
				Title:         pr.Title,
				Author:        pr.Author,
				URL:           pr.URL,
				State:         "OPEN",
				IsDraft:       pr.IsDraft,
				ReviewPending: true,
				FirstSeenAt:   now,
				LastSeenAt:    now,
			}
			s.prs[pr.ID] = rec
			novos = append(novos, rec)
			continue
		}
		wasPending := existing.ReviewPending
		existing.Title = pr.Title
		existing.IsDraft = pr.IsDraft
		existing.Author = pr.Author
		existing.URL = pr.URL
		existing.Repo = pr.Repo
		existing.LastSeenAt = now
		existing.ReviewPending = true
		if existing.State == "" {
			existing.State = "OPEN"
		}
		s.prs[pr.ID] = existing
		if !wasPending {
			novos = append(novos, existing)
		}
	}

	for id, rec := range s.prs {
		if _, stillPolled := polledIDs[id]; stillPolled {
			continue
		}
		if rec.ReviewPending {
			rec.ReviewPending = false
			s.prs[id] = rec
		}
	}

	s.lastPollAt = &now
	return novos
}

// RefreshPRStatus applies enrichment (additions/deletions, state, mergedAt,
// isDraft) from `gh pr view` to a tracked PR.
func (s *fileStore) RefreshPRStatus(id string, details github.PRDetails) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.prs[id]
	if !ok {
		return ErrNotFound
	}
	rec.Additions = details.Additions
	rec.Deletions = details.Deletions
	rec.IsDraft = details.IsDraft
	if details.State != "" {
		rec.State = details.State
	}
	// A merged PR overrides State (gh returns state=CLOSED with mergedAt set).
	if details.MergedAt != nil {
		rec.State = "MERGED"
	}
	s.prs[id] = rec
	return nil
}

// SetRetentionDays updates the retention window at runtime. Takes effect on
// the next Save. Non-positive values disable retention (history unbounded).
func (s *fileStore) SetRetentionDays(days int) {
	s.mu.Lock()
	s.retentionDays = days
	s.mu.Unlock()
}

// applyRetentionLocked drops non-OPEN records whose LastSeenAt is older than
// retentionDays. Caller must hold s.mu.
func (s *fileStore) applyRetentionLocked() {
	if s.retentionDays <= 0 {
		return
	}
	cutoff := s.now().UTC().AddDate(0, 0, -s.retentionDays)
	for id, rec := range s.prs {
		if rec.State == "OPEN" || rec.State == "" {
			continue
		}
		if rec.LastSeenAt.Before(cutoff) {
			delete(s.prs, id)
		}
	}
}

func sortedRecords(m map[string]PRRecord, less func(a, b PRRecord) bool) []PRRecord {
	out := make([]PRRecord, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

func sortedFiltered(m map[string]PRRecord, keep func(PRRecord) bool) []PRRecord {
	out := make([]PRRecord, 0)
	for _, r := range m {
		if keep(r) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeenAt.After(out[j].LastSeenAt) })
	return out
}
