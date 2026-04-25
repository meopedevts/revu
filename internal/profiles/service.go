package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/meopedevts/revu/internal/auth"
)

// CommandRunner is the subset of the github exec.Executor that the service
// needs to validate tokens. Declared locally so profiles does not import
// github (that would create a cycle — github will depend on profiles).
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// RunnerWithEnv lets the service inject GH_TOKEN when the command runner
// supports it. Optional: when the runner only implements CommandRunner we
// fall back to setting GH_TOKEN on the process env during validation (not
// used in production — the github exec runner implements this interface).
type RunnerWithEnv interface {
	RunEnv(ctx context.Context, env []string, name string, args ...string) (stdout, stderr []byte, err error)
}

// Service coordinates the repository, the keyring, and GitHub for all
// profile-level operations. Never logs or returns tokens.
type Service struct {
	repo    *Repository
	keys    auth.Keyring
	runner  CommandRunner
	now     func() time.Time
	log     *slog.Logger
	refPref string // keyring ref prefix, defaults to "profile-"

	subsMu      sync.Mutex
	subscribers map[int]func(Profile)
	nextSubID   int
}

// Option customizes a Service built via NewService.
type Option func(*Service)

// WithClock injects a deterministic time source for tests.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// WithLogger injects a logger. Defaults to slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(s *Service) {
		if l != nil {
			s.log = l
		}
	}
}

// NewService wires the collaborators. All arguments are required except opts.
func NewService(repo *Repository, keys auth.Keyring, runner CommandRunner, opts ...Option) *Service {
	s := &Service{
		repo:        repo,
		keys:        keys,
		runner:      runner,
		now:         time.Now,
		log:         slog.Default(),
		refPref:     "profile-",
		subscribers: map[int]func(Profile){},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SubscribeActive registers fn to be called whenever the active profile
// changes (via Create{MakeActive:true}, SetActive, or Update that modifies
// the active row). Returns an unsubscribe function. fn runs synchronously on
// the caller's goroutine — keep it cheap.
func (s *Service) SubscribeActive(fn func(Profile)) func() {
	s.subsMu.Lock()
	id := s.nextSubID
	s.nextSubID++
	s.subscribers[id] = fn
	s.subsMu.Unlock()
	return func() {
		s.subsMu.Lock()
		delete(s.subscribers, id)
		s.subsMu.Unlock()
	}
}

func (s *Service) fanoutActive(p Profile) {
	s.subsMu.Lock()
	subs := make([]func(Profile), 0, len(s.subscribers))
	for _, fn := range s.subscribers {
		subs = append(subs, fn)
	}
	s.subsMu.Unlock()
	for _, fn := range subs {
		func() {
			defer func() { _ = recover() }()
			fn(p)
		}()
	}
}

// List returns every profile.
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	return s.repo.List(ctx)
}

// Get returns a single profile by id.
func (s *Service) Get(ctx context.Context, id string) (Profile, error) {
	return s.repo.Get(ctx, id)
}

// GetActive returns the active profile. Migration 00002 seeds a gh-cli row
// with is_active=1, so a properly migrated DB always has one.
func (s *Service) GetActive(ctx context.Context) (Profile, error) {
	return s.repo.GetActive(ctx)
}

// Count returns how many profiles exist. Useful for the doctor command.
func (s *Service) Count(ctx context.Context) (int, error) {
	return s.repo.Count(ctx)
}

// ActiveProfileID returns the id of the active profile. Thin wrapper around
// GetActive — exists to satisfy the poller's ActiveProfileProvider without
// pulling the whole Profile over the call.
func (s *Service) ActiveProfileID(ctx context.Context) (string, error) {
	p, err := s.repo.GetActive(ctx)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

// CreateParams carries the data required to create a profile. Token is
// required when method=keyring; ignored otherwise.
type CreateParams struct {
	Name       string
	Method     AuthMethod
	Token      string
	MakeActive bool
}

// Create validates inputs, stores the token in the keyring (if any), and
// writes the profile row. On keyring mode, the token is validated against
// GitHub before being persisted. On failure, any partial state is rolled
// back so the user sees an atomic outcome.
func (s *Service) Create(ctx context.Context, p CreateParams) (Profile, error) {
	if !p.Method.Valid() {
		return Profile{}, ErrInvalidMethod
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return Profile{}, fmt.Errorf("profile name required")
	}

	id, err := NewID()
	if err != nil {
		return Profile{}, err
	}

	prof := Profile{
		ID:         id,
		Name:       name,
		AuthMethod: p.Method,
		CreatedAt:  s.now().UTC(),
	}

	var username string
	switch p.Method {
	case AuthKeyring:
		if p.Token == "" {
			return Profile{}, ErrTokenRequired
		}
		username, err = s.validateTokenString(ctx, p.Token)
		if err != nil {
			return Profile{}, err
		}
		ref := s.refPref + id
		if err := s.keys.Set(ref, p.Token); err != nil {
			return Profile{}, fmt.Errorf("store token: %w", err)
		}
		prof.KeyringRef = ref
		prof.GitHubUsername = username
		now := s.now().UTC()
		prof.LastValidatedAt = &now
	case AuthGHCLI:
		prof.KeyringRef = "gh-cli"
	}

	if err := s.repo.Create(ctx, prof); err != nil {
		if prof.AuthMethod == AuthKeyring && prof.KeyringRef != "" {
			_ = s.keys.Delete(prof.KeyringRef)
		}
		return Profile{}, err
	}

	if p.MakeActive {
		if err := s.repo.SetActive(ctx, prof.ID); err != nil {
			return Profile{}, fmt.Errorf("activate new profile: %w", err)
		}
		prof.IsActive = true
		s.fanoutActive(prof)
	}
	s.log.Info("profile created",
		slog.String("id", prof.ID),
		slog.String("name", prof.Name),
		slog.String("method", string(prof.AuthMethod)))
	return prof, nil
}

// Update applies the non-nil fields in u. If u.Token is non-nil and the
// profile uses the keyring, the token is validated and replaces the current
// secret atomically (validation fails → nothing changes).
//
//nolint:gocyclo // pipeline coeso de validação + persistência + atualização atômica do segredo; quebrar em helpers obscurece o fluxo.
func (s *Service) Update(ctx context.Context, id string, u Update) (Profile, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}

	next := cur
	if u.Name != nil {
		n := strings.TrimSpace(*u.Name)
		if n == "" {
			return Profile{}, fmt.Errorf("profile name required")
		}
		next.Name = n
	}
	if u.Method != nil {
		if !u.Method.Valid() {
			return Profile{}, ErrInvalidMethod
		}
		next.AuthMethod = *u.Method
	}

	tokenChanged := u.Token != nil && *u.Token != ""

	switch next.AuthMethod {
	case AuthKeyring:
		if tokenChanged {
			username, err := s.validateTokenString(ctx, *u.Token)
			if err != nil {
				return Profile{}, err
			}
			ref := next.KeyringRef
			if ref == "" || ref == "gh-cli" {
				ref = s.refPref + next.ID
			}
			if err := s.keys.Set(ref, *u.Token); err != nil {
				return Profile{}, fmt.Errorf("store token: %w", err)
			}
			next.KeyringRef = ref
			next.GitHubUsername = username
			now := s.now().UTC()
			next.LastValidatedAt = &now
		} else if cur.AuthMethod != AuthKeyring {
			return Profile{}, ErrTokenRequired
		}
	case AuthGHCLI:
		// Switching to gh-cli: drop the token from the keyring if one existed.
		if cur.AuthMethod == AuthKeyring && cur.KeyringRef != "" && cur.KeyringRef != "gh-cli" {
			if err := s.keys.Delete(cur.KeyringRef); err != nil {
				s.log.Warn("delete old keyring entry", slog.String("err", err.Error()))
			}
		}
		next.KeyringRef = "gh-cli"
		next.GitHubUsername = ""
	}

	if err := s.repo.UpdateFields(ctx, next); err != nil {
		return Profile{}, err
	}
	if next.IsActive {
		s.fanoutActive(next)
	}
	s.log.Info("profile updated", slog.String("id", next.ID), slog.String("name", next.Name))
	return next, nil
}

// Delete removes the profile. Rejects deletion of the last remaining profile
// or of the currently active profile (caller must switch first).
func (s *Service) Delete(ctx context.Context, id string) error {
	prof, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if prof.IsActive {
		return ErrCannotDeleteActive
	}
	total, err := s.repo.Count(ctx)
	if err != nil {
		return err
	}
	if total <= 1 {
		return ErrCannotDeleteLast
	}
	if prof.AuthMethod == AuthKeyring && prof.KeyringRef != "" && prof.KeyringRef != "gh-cli" {
		if err := s.keys.Delete(prof.KeyringRef); err != nil {
			s.log.Warn("delete keyring entry", slog.String("id", id), slog.String("err", err.Error()))
		}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("profile deleted", slog.String("id", id), slog.String("name", prof.Name))
	return nil
}

// SetActive flips the active profile atomically and notifies subscribers.
func (s *Service) SetActive(ctx context.Context, id string) error {
	if err := s.repo.SetActive(ctx, id); err != nil {
		return err
	}
	p, err := s.repo.Get(ctx, id)
	if err == nil {
		s.fanoutActive(p)
	}
	s.log.Info("active profile changed", slog.String("id", id))
	return nil
}

// ValidateToken asks GitHub who the token belongs to. Returns the GitHub
// login on success. The token is never logged; errors are surfaced without
// the secret value.
func (s *Service) ValidateToken(ctx context.Context, token string) (string, error) {
	return s.validateTokenString(ctx, token)
}

// TokenForActive fetches the secret for the currently active profile.
// - gh-cli profile → empty string, nil (caller uses ambient gh auth).
// - keyring profile → the PAT, freshly read. Caller must not retain it.
func (s *Service) TokenForActive(ctx context.Context) (string, error) {
	p, err := s.GetActive(ctx)
	if err != nil {
		return "", err
	}
	return s.TokenFor(ctx, p)
}

// TokenFor returns the PAT for a specific profile, or "" for gh-cli profiles.
func (s *Service) TokenFor(_ context.Context, p Profile) (string, error) {
	if p.AuthMethod != AuthKeyring {
		return "", nil
	}
	if p.KeyringRef == "" {
		return "", fmt.Errorf("profile %s missing keyring ref", p.ID)
	}
	token, err := s.keys.Get(p.KeyringRef)
	if err != nil {
		return "", fmt.Errorf("read keyring: %w", err)
	}
	return token, nil
}

// validateTokenString runs `gh api user` with GH_TOKEN=<token> set only in
// the child process env. Returns the login; on forbidden/401 returns
// ErrTokenInvalid so the UI can show a friendly message. If the PAT lacks
// the user scope (fine-grained), falls back to `gh api /octocat` and returns
// an empty username — caller writes "" and shows "(username indisponível)".
func (s *Service) validateTokenString(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrTokenRequired
	}
	env := append(os.Environ(), "GH_TOKEN="+token)

	stdout, stderr, err := s.runWithEnv(ctx, env, "gh", "api", "user")
	if err == nil {
		var body struct {
			Login string `json:"login"`
		}
		if jerr := json.Unmarshal(stdout, &body); jerr != nil {
			return "", fmt.Errorf("decode gh api user: %w", jerr)
		}
		if body.Login == "" {
			return "", fmt.Errorf("gh api user returned empty login")
		}
		return body.Login, nil
	}

	low := strings.ToLower(string(stderr))
	if strings.Contains(low, "401") || strings.Contains(low, "unauthorized") ||
		strings.Contains(low, "bad credentials") {
		return "", ErrTokenInvalid
	}
	// fine-grained PAT without user scope → 403. Fall back to /octocat which
	// only requires a valid token.
	if strings.Contains(low, "403") || strings.Contains(low, "forbidden") {
		if _, _, ferr := s.runWithEnv(ctx, env, "gh", "api", "/octocat"); ferr == nil {
			return "", nil
		}
		return "", ErrTokenInvalid
	}
	return "", fmt.Errorf("gh api user failed: %w", err)
}

// runWithEnv prefers RunEnv when the runner implements it; otherwise falls
// back to running the command without env overrides (only reached by tests
// that supply a fake runner which ignores env anyway).
func (s *Service) runWithEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, []byte, error) {
	if r, ok := s.runner.(RunnerWithEnv); ok {
		return r.RunEnv(ctx, env, name, args...)
	}
	return s.runner.Run(ctx, name, args...)
}

// RepositoryHandle exposes the repo for callers that need a CRUD touchpoint
// outside the service surface (primarily the smoke test harness).
func (s *Service) RepositoryHandle() *Repository { return s.repo }
