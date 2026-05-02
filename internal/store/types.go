package store

import "time"

// PRRecord is the persisted shape of a tracked pull request (SPEC §5.3).
// JSON tags are still honored because the record crosses the Wails bridge
// into the React frontend via EventsEmit — removing them would silently
// break TS bindings.
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
	ReviewState    string     `json:"review_state"`
	Branch         string     `json:"branch"`
	AvatarURL      string     `json:"avatar_url"`
	FirstSeenAt    time.Time  `json:"first_seen_at"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
}
