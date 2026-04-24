package store

import "time"

// PRRecord is the persisted shape of a tracked pull request (SPEC §5.3).
// JSON tags mirror state.json exactly — fields are snake_case on disk.
type PRRecord struct {
	ID             string     `json:"id"`
	Number         int        `json:"number"`
	Repo           string     `json:"repo"`
	Title          string     `json:"title"`
	Author         string     `json:"author"`
	URL            string     `json:"url"`
	State          string     `json:"state"`
	IsDraft        bool       `json:"is_draft"`
	Additions      int        `json:"additions"`
	Deletions      int        `json:"deletions"`
	ReviewPending  bool       `json:"review_pending"`
	FirstSeenAt    time.Time  `json:"first_seen_at"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
}

// snapshot is the on-disk envelope.
type snapshot struct {
	PRs        map[string]PRRecord `json:"prs"`
	LastPollAt *time.Time          `json:"last_poll_at,omitempty"`
}
