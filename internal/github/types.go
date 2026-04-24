package github

import (
	"errors"
	"time"
)

// PRSummary is the flattened shape returned by ListReviewRequested. The ID
// follows the store convention "owner/repo#number" (see SPEC §5.3).
type PRSummary struct {
	ID        string    `json:"id"`
	Number    int       `json:"number"`
	Repo      string    `json:"repo"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	IsDraft   bool      `json:"isDraft"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PRDetails is the enrichment returned by GetPRDetails (diff + current state
// + the review state for the active user). ReviewState follows the GitHub
// REST vocabulary narrowed to what the store persists: PENDING, APPROVED,
// CHANGES_REQUESTED, COMMENTED.
type PRDetails struct {
	Additions   int        `json:"additions"`
	Deletions   int        `json:"deletions"`
	State       string     `json:"state"`
	MergedAt    *time.Time `json:"mergedAt"`
	IsDraft     bool       `json:"isDraft"`
	ReviewState string     `json:"reviewState"`
}

// rawLatestReview mirrors an entry of `gh pr view --json latestReviews`.
// Only the minimum fields needed to attribute the review and read its state
// are declared; anything else in the payload is ignored by encoding/json.
type rawLatestReview struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State string `json:"state"`
}

// rawPRView is the shape of the JSON payload returned by GetPRDetails.
type rawPRView struct {
	Additions     int               `json:"additions"`
	Deletions     int               `json:"deletions"`
	State         string            `json:"state"`
	MergedAt      *time.Time        `json:"mergedAt"`
	IsDraft       bool              `json:"isDraft"`
	LatestReviews []rawLatestReview `json:"latestReviews"`
}

// Sentinel errors let callers branch on failure mode (poller backoff, tray
// error state). They are returned via errors.Is after classify().
var (
	ErrAuthExpired = errors.New("gh auth expired")
	ErrRateLimited = errors.New("github rate limited")
	ErrTransient   = errors.New("gh transient failure")
)

// rawSearchPR mirrors the nested JSON from `gh search prs` so we can decode
// the wire shape before flattening into PRSummary.
type rawSearchPR struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	IsDraft   bool      `json:"isDraft"`
	UpdatedAt time.Time `json:"updatedAt"`
}
