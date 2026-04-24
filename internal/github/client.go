package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Client is the surface used by the poller and by `revu doctor`.
type Client interface {
	AuthStatus(ctx context.Context) error
	ListReviewRequested(ctx context.Context) ([]PRSummary, error)
	GetPRDetails(ctx context.Context, url string) (*PRDetails, error)
}

type ghClient struct {
	exec Executor
}

// NewClient wires a Client around the given Executor. Tests inject a fake
// Executor; production callers pass DefaultExecutor().
func NewClient(e Executor) Client { return &ghClient{exec: e} }

func (c *ghClient) AuthStatus(ctx context.Context) error {
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
	stdout, stderr, err := c.exec.Run(ctx, "gh", args...)
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
	stdout, stderr, err := c.exec.Run(ctx, "gh",
		"pr", "view", url,
		"--json", "additions,deletions,state,mergedAt,isDraft",
	)
	if err != nil {
		return nil, classify(stderr, err)
	}
	var d PRDetails
	if err := json.Unmarshal(stdout, &d); err != nil {
		return nil, fmt.Errorf("decode pr view: %w", err)
	}
	return &d, nil
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
