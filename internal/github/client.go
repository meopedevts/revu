package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Client is the surface used by the poller and by `revu doctor`.
type Client interface {
	AuthStatus(ctx context.Context) error
	ListReviewRequested(ctx context.Context) ([]PRSummary, error)
	GetPRDetails(ctx context.Context, url string) (*PRDetails, error)
}

// reviewStateFor picks the review state for login among latestReviews and
// normalizes it to the four-value domain persisted by the store. Any review
// in states GitHub doesn't expose here (PENDING, DISMISSED) collapses into
// "PENDING" so the UI can show "still yours to review".
func reviewStateFor(login string, reviews []rawLatestReview) string {
	if login == "" {
		return "PENDING"
	}
	for _, r := range reviews {
		if r.Author.Login != login {
			continue
		}
		switch r.State {
		case "APPROVED", "CHANGES_REQUESTED", "COMMENTED":
			return r.State
		default:
			return "PENDING"
		}
	}
	return "PENDING"
}

// TokenProvider returns the PAT for the currently active profile, or "" when
// the profile defers to the ambient `gh auth login` session. The client
// resolves it per call so switching profiles takes effect immediately, and
// so tokens never live on a long-lived struct field.
type TokenProvider interface {
	TokenForActive(ctx context.Context) (string, error)
}

// noopTokenProvider satisfies TokenProvider by always returning "". Used when
// the caller has no profile service wired (e.g., unit tests of unrelated
// flows, or `gh auth status` bootstrap).
type noopTokenProvider struct{}

func (noopTokenProvider) TokenForActive(context.Context) (string, error) { return "", nil }

type ghClient struct {
	exec   Executor
	tokens TokenProvider

	// whoMu guards the cache of `gh api user --jq .login` results keyed by
	// token. A per-token cache is required because REV-15 lets the user
	// switch active profiles — each profile's token resolves to a different
	// login and the enrichment needs "which review is mine" at poll time.
	whoMu sync.Mutex
	who   map[string]string
}

// NewClient wires a Client around the given Executor. The optional tokens
// argument enables per-call GH_TOKEN injection (keyring profiles). Pass nil
// to rely entirely on the ambient gh CLI session.
func NewClient(e Executor, tokens ...TokenProvider) Client {
	var tp TokenProvider = noopTokenProvider{}
	if len(tokens) > 0 && tokens[0] != nil {
		tp = tokens[0]
	}
	return &ghClient{exec: e, tokens: tp, who: make(map[string]string)}
}

func (c *ghClient) AuthStatus(ctx context.Context) error {
	// AuthStatus is only meaningful for the gh-cli session; never inject a
	// token here — that would mask a broken ambient login.
	_, stderr, err := c.exec.Run(ctx, "gh", "auth", "status")
	if err != nil {
		return classify(stderr, err)
	}
	return nil
}

func (c *ghClient) ListReviewRequested(ctx context.Context) ([]PRSummary, error) {
	args := []string{
		"search", "prs",
		"--review-requested=@me",
		"--state=open",
		"--json", "number,title,url,repository,author,isDraft,updatedAt",
		"--limit", "100",
	}
	stdout, stderr, err := c.runGH(ctx, args...)
	if err != nil {
		return nil, classify(stderr, err)
	}
	var raw []rawSearchPR
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil, fmt.Errorf("decode search prs: %w", err)
	}
	out := make([]PRSummary, 0, len(raw))
	for _, r := range raw {
		out = append(out, PRSummary{
			ID:        fmt.Sprintf("%s#%d", r.Repository.NameWithOwner, r.Number),
			Number:    r.Number,
			Repo:      r.Repository.NameWithOwner,
			Title:     r.Title,
			URL:       r.URL,
			Author:    r.Author.Login,
			IsDraft:   r.IsDraft,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (c *ghClient) GetPRDetails(ctx context.Context, url string) (*PRDetails, error) {
	stdout, stderr, err := c.runGH(ctx,
		"pr", "view", url,
		"--json", "additions,deletions,state,mergedAt,isDraft,latestReviews",
	)
	if err != nil {
		return nil, classify(stderr, err)
	}
	var raw rawPRView
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil, fmt.Errorf("decode pr view: %w", err)
	}
	login, _ := c.whoAmI(ctx) // best-effort; failure just collapses to PENDING
	return &PRDetails{
		Additions:   raw.Additions,
		Deletions:   raw.Deletions,
		State:       raw.State,
		MergedAt:    raw.MergedAt,
		IsDraft:     raw.IsDraft,
		ReviewState: reviewStateFor(login, raw.LatestReviews),
	}, nil
}

// whoAmI returns the login associated with the currently active profile's
// token, caching the result per-token. A failure is returned as ("", err)
// and callers fall back to treating the review as PENDING — missing this
// field never breaks polling.
func (c *ghClient) whoAmI(ctx context.Context) (string, error) {
	token, err := c.tokens.TokenForActive(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve token for whoami: %w", err)
	}
	key := token
	if key == "" {
		key = "__ambient__"
	}
	c.whoMu.Lock()
	if cached, ok := c.who[key]; ok {
		c.whoMu.Unlock()
		return cached, nil
	}
	c.whoMu.Unlock()

	stdout, stderr, err := c.runGH(ctx, "api", "user", "--jq", ".login")
	if err != nil {
		return "", classify(stderr, err)
	}
	login := strings.TrimSpace(string(stdout))

	c.whoMu.Lock()
	c.who[key] = login
	c.whoMu.Unlock()
	return login, nil
}

// runGH resolves the active token, builds the env override, and invokes gh.
// When the token is empty (gh-cli profile) the ambient env is used untouched.
// The token variable goes out of scope immediately after the child process
// exits — no struct fields, no logs.
func (c *ghClient) runGH(ctx context.Context, args ...string) ([]byte, []byte, error) {
	token, err := c.tokens.TokenForActive(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve token: %w", err)
	}
	if token == "" {
		return c.exec.Run(ctx, "gh", args...)
	}
	env := append(os.Environ(), "GH_TOKEN="+token)
	if envRunner, ok := c.exec.(EnvExecutor); ok {
		return envRunner.RunEnv(ctx, env, "gh", args...)
	}
	// Executor does not support env injection. Caller passed a non-env-aware
	// impl despite wiring a TokenProvider — surface a clear error instead of
	// silently falling back to ambient.
	return nil, nil, fmt.Errorf("executor does not support GH_TOKEN injection")
}

// classify maps the stderr of a failed `gh` invocation to a sentinel error.
// Matching is substring-based because gh does not expose structured error
// codes — fragile to version changes, acceptable for the MVP.
func classify(stderr []byte, runErr error) error {
	s := string(stderr)
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "not logged"),
		strings.Contains(lower, "authentication required"),
		strings.Contains(lower, "authentication failed"):
		return ErrAuthExpired
	case strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "api rate limit exceeded"):
		return ErrRateLimited
	case strings.Contains(lower, "could not resolve"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "timeout"):
		return ErrTransient
	}
	if runErr == nil {
		runErr = fmt.Errorf("gh failed")
	}
	return fmt.Errorf("gh: %s: %w", firstLine(s), runErr)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if s == "" {
		return "no stderr"
	}
	return s
}
