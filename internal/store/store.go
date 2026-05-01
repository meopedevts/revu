package store

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// Store is the persistence surface used by the poller and the UI bridge.
// I/O methods take a ctx so callers (poller, Wails bridge, CLI) can propagate
// cancellation and deadlines down to the SQLite driver. Methods that don't
// touch the DB (SetRetentionDays, SetActiveProfileID, DB) stay ctx-free.
type Store interface {
	Load(ctx context.Context) error
	Save(ctx context.Context) error
	Close(ctx context.Context) error
	GetAll(ctx context.Context) []PRRecord
	GetPending(ctx context.Context) []PRRecord
	GetHistory(ctx context.Context) []PRRecord
	// GetByID returns the record with the given id, or (zero, false) if the
	// store does not track it. Used by the app bridge to resolve a PR id
	// coming off the frontend into the URL needed for gh-backed calls.
	GetByID(ctx context.Context, id string) (PRRecord, bool)
	UpdateFromPoll(ctx context.Context, prs []github.PRSummary) (novos, vanished []PRRecord)
	RefreshPRStatus(ctx context.Context, id string, details github.PRDetails) error
	// MarkNotified persiste o instante em que o desktop notify foi enviado
	// pra o PR identificado por id. Usado pelo poller pra throttle de
	// re-requests via janela de cooldown.
	MarkNotified(ctx context.Context, id string, when time.Time) error
	SetRetentionDays(days int)
	SetActiveProfileID(id string)
	ClearHistory(ctx context.Context) (int, error)
	// DB returns the underlying *sql.DB so sibling packages (e.g. profiles)
	// can share the same connection / WAL session instead of opening a
	// second handle. Returns nil before Load or after Close.
	DB() *sql.DB
}

// ErrNotFound is returned by RefreshPRStatus when the id is unknown.
var ErrNotFound = errors.New("pr not tracked")

// Option customizes a Store built via New.
type Option func(*sqliteStore)

// WithClock injects a time source; useful for deterministic tests.
func WithClock(now func() time.Time) Option {
	return func(s *sqliteStore) {
		if now != nil {
			s.now = now
		}
	}
}

// WithRetention sets the number of days a non-OPEN PR is kept after it was
// last seen. Zero disables retention (history grows unbounded).
func WithRetention(days int) Option {
	return func(s *sqliteStore) { s.retentionDays = days }
}

// WithLogger injects a slog logger. Used by the one-shot JSON import to
// surface migration results.
func WithLogger(l *slog.Logger) Option {
	return func(s *sqliteStore) {
		if l != nil {
			s.log = l
		}
	}
}

// WithJSONMigration points the store at a legacy state.json path. On Load,
// if the DB has not yet absorbed it, the file is parsed and imported in a
// single transaction, then renamed as a backup. Absent file is a no-op.
func WithJSONMigration(path string) Option {
	return func(s *sqliteStore) { s.jsonStatePath = path }
}

// withDBOpener substitui a função usada por Load pra abrir o handle do
// DB. Existe **apenas** pra testes que exercitam o error path do
// openDB — produção sempre usa o opener default. Nome unexported sinaliza
// uso restrito ao pacote.
func withDBOpener(open func(string) (*sql.DB, error)) Option {
	return func(s *sqliteStore) { s.openDB = open }
}

// New constructs a Store backed by SQLite at dbPath. The DB is opened and
// migrated by Load — New never touches the filesystem.
func New(dbPath string, opts ...Option) Store {
	s := &sqliteStore{
		path:          dbPath,
		now:           time.Now,
		retentionDays: 30,
		log:           slog.Default(),
		openDB:        openDB,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
