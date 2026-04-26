package store

import (
	"context"
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

// TestWithLogger_NilIsNoOp cobre o guard `if l != nil` no Option.
// New(..., WithLogger(nil)) não deve substituir o [slog.Default]
// nem panicar — opção comum quando callers não querem injetar logger.
func TestWithLogger_NilIsNoOp(t *testing.T) {
	st := New(":memory:", WithLogger(nil)).(*sqliteStore)
	if st.log == nil {
		t.Fatal("log must remain default, got nil")
	}
	// Sanity: o store carrega e fecha sem panicar mesmo com Option nil.
	if err := st.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(context.Background()) })
}
