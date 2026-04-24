package store

import (
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// Shared test helpers used by all *_test.go in this package.

func newClock(start time.Time) (func() time.Time, *time.Time) {
	t := start
	return func() time.Time { return t }, &t
}

func mkSummary(id, repo string, number int, title, author string, draft bool) github.PRSummary {
	return github.PRSummary{
		ID:        id,
		Number:    number,
		Repo:      repo,
		Title:     title,
		URL:       "https://github.com/" + repo + "/pull/" + itoa(number),
		Author:    author,
		IsDraft:   draft,
		UpdatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// silences the "unused" warning for itoa when building store_test.go in
// isolation — used by mkSummary.
var _ = itoa

func TestPackageCompiles(t *testing.T) {
	// Intentional placeholder; real coverage lives in *_test.go siblings.
}
