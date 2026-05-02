package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/store"
	"github.com/meopedevts/revu/internal/tray"
)

// newTestStore retorna um store SQLite em arquivo temp já carregado.
// :memory: não funciona aqui porque o caller precisa que múltiplas conexões
// vejam o mesmo dataset (Acknowledge/AcknowledgedAt rodam em chamadas
// distintas). Arquivo temp resolve sem WAL.
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "revu.db")
	s := store.New(dbPath, store.WithRetention(0))
	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("store load: %v", err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return s
}

func TestHasUnacked(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		pending []store.PRRecord
		ackedAt time.Time
		want    bool
	}{
		{
			name:    "empty pending",
			pending: nil,
			ackedAt: now,
			want:    false,
		},
		{
			name:    "all PRs older than ack",
			pending: []store.PRRecord{{LastSeenAt: now.Add(-2 * time.Hour)}},
			ackedAt: now.Add(-1 * time.Hour),
			want:    false,
		},
		{
			name:    "one PR newer than ack",
			pending: []store.PRRecord{{LastSeenAt: now.Add(-2 * time.Hour)}, {LastSeenAt: now}},
			ackedAt: now.Add(-1 * time.Hour),
			want:    true,
		},
		{
			name:    "PR exactly at ack instant — não conta",
			pending: []store.PRRecord{{LastSeenAt: now}},
			ackedAt: now,
			want:    false,
		},
		{
			name:    "ack zero (nunca acked) + pending → attention",
			pending: []store.PRRecord{{LastSeenAt: now}},
			ackedAt: time.Time{},
			want:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasUnacked(tc.pending, tc.ackedAt); got != tc.want {
				t.Fatalf("hasUnacked = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSyncTrayState_OnPollCompleted(t *testing.T) {
	s := newTestStore(t)
	tr := tray.New(nil, nil, nil, nil)
	ctx := context.Background()

	// Sem pending, sem err → idle.
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPollCompleted})
	if got := tr.State(); got != tray.StateIdle {
		t.Fatalf("empty store: got %s, want idle", got)
	}

	// Insere um PR pending. Sem ack ainda — qualquer pending é attention.
	s.UpdateFromPoll(ctx, []github.PRSummary{
		{ID: "octo/h#1", Number: 1, Repo: "octo/h", Title: "t", Author: "a", URL: "u"},
	})
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPollCompleted})
	if got := tr.State(); got != tray.StateAttention {
		t.Fatalf("pending sem ack: got %s, want attention", got)
	}

	// Ack agora — pending vira só pending (sem badge attention).
	if err := s.Acknowledge(ctx, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPollCompleted})
	if got := tr.State(); got != tray.StatePending {
		t.Fatalf("pending pós-ack: got %s, want pending", got)
	}

	// Erro de tick → vira error (overrides pending).
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPollCompleted, Err: "rate limit"})
	if got := tr.State(); got != tray.StateError {
		t.Fatalf("Err set: got %s, want error", got)
	}

	// Próximo tick OK → erro limpa, volta pra pending.
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPollCompleted})
	if got := tr.State(); got != tray.StatePending {
		t.Fatalf("Err cleared: got %s, want pending", got)
	}
}

// TestSyncTrayState_IgnoresNonPollCompleted: EventPRNew / EventPRStatusChanged
// não devem mexer no estado — toda derivação acontece em EventPollCompleted
// pra ter um snapshot consistente do store.
func TestSyncTrayState_IgnoresNonPollCompleted(t *testing.T) {
	s := newTestStore(t)
	tr := tray.New(nil, nil, nil, nil)
	tr.SetPending(true)
	before := tr.State()

	syncTrayState(tr, s, poller.Event{Kind: poller.EventPRNew})
	syncTrayState(tr, s, poller.Event{Kind: poller.EventPRStatusChanged})

	if got := tr.State(); got != before {
		t.Fatalf("non-completed events mudaram State: %s → %s", before, got)
	}
}

func TestSeedTrayState(t *testing.T) {
	s := newTestStore(t)
	tr := tray.New(nil, nil, nil, nil)
	ctx := context.Background()

	// Boot com store vazio — idle.
	seedTrayState(ctx, tr, s)
	if got := tr.State(); got != tray.StateIdle {
		t.Fatalf("empty boot: got %s, want idle", got)
	}

	// Boot com pending sem ack — attention.
	s.UpdateFromPoll(ctx, []github.PRSummary{
		{ID: "x/y#1", Number: 1, Repo: "x/y", Title: "t", Author: "a", URL: "u"},
	})
	seedTrayState(ctx, tr, s)
	if got := tr.State(); got != tray.StateAttention {
		t.Fatalf("pending sem ack: got %s, want attention", got)
	}

	// Boot com pending após ack — só pending.
	if err := s.Acknowledge(ctx, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	seedTrayState(ctx, tr, s)
	if got := tr.State(); got != tray.StatePending {
		t.Fatalf("pending pós-ack: got %s, want pending", got)
	}
}
