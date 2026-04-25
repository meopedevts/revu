package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// sqliteStore is the SQLite-backed implementation of Store. Writes serialize
// through database/sql's pool + SQLite's WAL mode (single writer, concurrent
// readers). The RWMutex guards retentionDays only — every DB mutation is
// already serialized by the driver.
type sqliteStore struct {
	path          string
	jsonStatePath string
	now           func() time.Time
	retentionDays int
	log           *slog.Logger

	// openDB é injetável via withDBOpener pra exercitar error paths em
	// testes. Em produção sempre aponta pro openDB package-level.
	openDB func(string) (*sql.DB, error)

	dbMu sync.Mutex // guards db handle lifecycle (Load/Close)
	db   *sql.DB

	mu              sync.RWMutex // guards retentionDays + activeProfileID
	activeProfileID string
}

// Load opens the underlying SQLite database, applies pending migrations, and
// performs the one-shot JSON import if configured. Idempotent: calling Load
// twice is a no-op for the second call.
func (s *sqliteStore) Load() error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	if s.db != nil {
		return nil
	}
	db, err := s.openDB(s.path)
	if err != nil {
		return err
	}
	if s.jsonStatePath != "" {
		ctx := context.Background()
		if err := migrateJSONIfPresent(ctx, db, s.jsonStatePath, s.now().UTC(), s.log); err != nil {
			_ = db.Close()
			return fmt.Errorf("import legacy state.json: %w", err)
		}
	}
	s.db = db
	return nil
}

// Save runs a WAL checkpoint so the main DB file absorbs pending WAL frames.
// Does not close the handle.
func (s *sqliteStore) Save() error {
	db := s.handle()
	if db == nil {
		return nil
	}
	if _, err := db.ExecContext(context.Background(), `PRAGMA wal_checkpoint(PASSIVE)`); err != nil {
		return fmt.Errorf("wal checkpoint: %w", err)
	}
	return nil
}

// Close flushes the WAL with a TRUNCATE checkpoint and closes the DB. Safe
// to call multiple times — subsequent calls return nil.
func (s *sqliteStore) Close() error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	if s.db == nil {
		return nil
	}
	_, _ = s.db.ExecContext(context.Background(), `PRAGMA wal_checkpoint(TRUNCATE)`)
	err := s.db.Close()
	s.db = nil
	if err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}
	return nil
}

// handle returns the current DB handle under lock. Nil before Load or after
// Close.
func (s *sqliteStore) handle() *sql.DB {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	return s.db
}

// DB satisfies Store.DB — see Store interface doc.
func (s *sqliteStore) DB() *sql.DB { return s.handle() }

func (s *sqliteStore) GetAll() []PRRecord {
	db := s.handle()
	if db == nil {
		return nil
	}
	return mustScan(db, qSelectPRsAll)
}

func (s *sqliteStore) GetPending() []PRRecord {
	db := s.handle()
	if db == nil {
		return nil
	}
	return mustScan(db, qSelectPRsPending)
}

func (s *sqliteStore) GetHistory() []PRRecord {
	db := s.handle()
	if db == nil {
		return nil
	}
	return mustScan(db, qSelectPRsHistory)
}

func (s *sqliteStore) GetByID(id string) (PRRecord, bool) {
	rec, ok, err := s.loadRecord(context.Background(), id)
	if err != nil || !ok {
		return PRRecord{}, false
	}
	return rec, true
}

// SetRetentionDays updates the retention window. Takes effect on the next
// UpdateFromPoll — non-positive disables retention.
func (s *sqliteStore) SetRetentionDays(days int) {
	s.mu.Lock()
	s.retentionDays = days
	s.mu.Unlock()
}

// SetActiveProfileID records the profile id that new prs should be tagged
// with. The poller sets it at startup and whenever the user switches
// profiles. Empty string = legacy / untagged inserts (migration default).
func (s *sqliteStore) SetActiveProfileID(id string) {
	s.mu.Lock()
	s.activeProfileID = id
	s.mu.Unlock()
}

// ClearHistory removes every non-OPEN record from the store, regardless of
// age. Returns the number of rows deleted. Backs the "Limpar histórico
// agora" action in the settings UI.
func (s *sqliteStore) ClearHistory() (int, error) {
	db := s.handle()
	if db == nil {
		return 0, errors.New("store is not loaded")
	}
	ctx := context.Background()
	res, err := db.ExecContext(ctx, qClearHistory)
	if err != nil {
		return 0, fmt.Errorf("clear history: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("clear history rows affected: %w", err)
	}
	return int(n), nil
}

// UpdateFromPoll merges a fresh `gh search prs` result into the store in a
// single transaction. Retention runs at the end of the same tx so evicted
// records never linger.
//
// Returns two slices:
//   - novos: PRs that are new on this poll or that just came back into the
//     search after having dropped out (re-request). The poller enriches and
//     notifies these.
//   - vanished: PRs that were in review_pending=1 before this poll but are
//     absent from the current search result. Under REV-16 these still need
//     re-enrichment so state + review_state converge with GitHub (the PR may
//     have been merged/closed, or our review may have just been submitted).
//     Callers must not notify on these — they're not new work.
func (s *sqliteStore) UpdateFromPoll(prs []github.PRSummary) ([]PRRecord, []PRRecord) {
	db := s.handle()
	if db == nil {
		return nil, nil
	}
	ctx := context.Background()
	now := s.now().UTC()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	s.mu.RLock()
	profID := s.activeProfileID
	retention := s.retentionDays
	s.mu.RUnlock()

	polledIDs := indexPolledIDs(prs)

	vanishedIDs, err := selectVanishedIDs(ctx, tx, polledIDs)
	if err != nil {
		return nil, nil
	}

	novos, err := upsertPolled(ctx, tx, prs, now, profID)
	if err != nil {
		return nil, nil
	}

	if err := markNotPolled(ctx, tx, polledIDs); err != nil {
		return nil, nil
	}

	vanished := loadVanishedRecords(ctx, tx, vanishedIDs)

	if err := enforceRetention(ctx, tx, retention, now); err != nil {
		return nil, nil
	}

	if _, err := tx.ExecContext(ctx, qSetMeta, metaLastPollAt, formatTime(now)); err != nil {
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, nil
	}
	committed = true
	return novos, vanished
}

// indexPolledIDs materializa um set dos PR IDs do poll corrente —
// usado por selectVanishedIDs e markNotPolled. Pure function.
func indexPolledIDs(prs []github.PRSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(prs))
	for _, pr := range prs {
		out[pr.ID] = struct{}{}
	}
	return out
}

// upsertPolled aplica cada PR do poll corrente sobre a transação `tx`:
// PRs novos viram INSERT, existentes ganham UPDATE de mutable fields,
// e existentes que estavam fora de pendentes (re-request) são
// devolvidos como `novos` pra o caller notificar. `profID` etiqueta
// inserts; é capturado uma única vez antes do loop pra evitar lock
// repetido.
func upsertPolled(
	ctx context.Context,
	tx *sql.Tx,
	prs []github.PRSummary,
	now time.Time,
	profID string,
) ([]PRRecord, error) {
	var novos []PRRecord
	for _, pr := range prs {
		var prevPending int
		err := tx.QueryRowContext(ctx, `SELECT review_pending FROM prs WHERE id = ?`, pr.ID).
			Scan(&prevPending)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			rec, err := insertNewPolled(ctx, tx, pr, now, profID)
			if err != nil {
				return nil, err
			}
			novos = append(novos, rec)
		case err != nil:
			return nil, err
		default:
			rec, isReRequest, err := updateExistingPolled(ctx, tx, pr, now, prevPending)
			if err != nil {
				return nil, err
			}
			if isReRequest {
				novos = append(novos, rec)
			}
		}
	}
	return novos, nil
}

// insertNewPolled cria um PRRecord novo a partir do summary e persiste
// via qInsertPR. Retorna o registro construído (mesmo formato dos
// novos).
func insertNewPolled(
	ctx context.Context,
	tx *sql.Tx,
	pr github.PRSummary,
	now time.Time,
	profID string,
) (PRRecord, error) {
	rec := PRRecord{
		ID:            pr.ID,
		Number:        pr.Number,
		Repo:          pr.Repo,
		Title:         pr.Title,
		Author:        pr.Author,
		URL:           pr.URL,
		State:         "OPEN",
		IsDraft:       pr.IsDraft,
		ReviewPending: true,
		ReviewState:   "PENDING",
		FirstSeenAt:   now,
		LastSeenAt:    now,
	}
	if _, err := tx.ExecContext(ctx, qInsertPR,
		rec.ID, rec.Number, rec.Repo, rec.Title, rec.Author, rec.URL,
		rec.State, boolToInt(rec.IsDraft), rec.Additions, rec.Deletions,
		boolToInt(rec.ReviewPending), rec.ReviewState,
		formatTime(rec.FirstSeenAt),
		formatTime(rec.LastSeenAt), formatTimePtr(rec.LastNotifiedAt),
		profID,
	); err != nil {
		return PRRecord{}, err
	}
	return rec, nil
}

// updateExistingPolled aplica os campos mutáveis de `pr` sobre o registro
// existente e detecta re-request (quando review_pending era 0 e o PR
// reapareceu no poll). Quando é re-request, retorna o registro pós-update
// e isReRequest=true pra o caller acumular nos novos.
func updateExistingPolled(
	ctx context.Context,
	tx *sql.Tx,
	pr github.PRSummary,
	now time.Time,
	prevPending int,
) (PRRecord, bool, error) {
	if _, err := tx.ExecContext(ctx, qUpdatePRMutable,
		pr.Title, pr.Author, pr.URL, pr.Repo, boolToInt(pr.IsDraft),
		formatTime(now), pr.ID,
	); err != nil {
		return PRRecord{}, false, err
	}
	if prevPending != 0 {
		return PRRecord{}, false, nil
	}
	rec, ok := loadRecordTx(ctx, tx, pr.ID)
	if !ok {
		return PRRecord{}, false, nil
	}
	return rec, true, nil
}

// markNotPolled garante que toda linha review_pending=1 que NÃO está no
// poll corrente vire review_pending=0. Quando o poll vem vazio, usa a
// query estática qMarkNotPendingAll; senão monta um IN(?, ?, ...) com
// placeholders fixos pra evitar SQL injection.
func markNotPolled(ctx context.Context, tx *sql.Tx, polledIDs map[string]struct{}) error {
	if len(polledIDs) == 0 {
		if _, err := tx.ExecContext(ctx, qMarkNotPendingAll); err != nil {
			return err
		}
		return nil
	}
	ids := make([]any, 0, len(polledIDs))
	placeholders := make([]string, 0, len(polledIDs))
	for id := range polledIDs {
		ids = append(ids, id)
		placeholders = append(placeholders, "?")
	}
	// #nosec G202 — placeholders são uma sequência fixa de "?" gerada acima; valores entram via parâmetros.
	q := `UPDATE prs SET review_pending = 0
		WHERE review_pending = 1 AND id NOT IN (` + strings.Join(placeholders, ",") + `)`
	if _, err := tx.ExecContext(ctx, q, ids...); err != nil {
		return err
	}
	return nil
}

// loadVanishedRecords resolve o slice de PRRecord pra cada id que sumiu
// do poll. Erros de load individual são silenciados (semântica
// pre-existente — vanished que não bate no DB significa concorrência
// resolvida).
func loadVanishedRecords(ctx context.Context, tx *sql.Tx, ids []string) []PRRecord {
	var out []PRRecord
	for _, id := range ids {
		if rec, ok := loadRecordTx(ctx, tx, id); ok {
			out = append(out, rec)
		}
	}
	return out
}

// enforceRetention apaga registros mais velhos que `days` quando a
// retenção está habilitada (>0). No-op silencioso quando desabilitada.
func enforceRetention(ctx context.Context, tx *sql.Tx, days int, now time.Time) error {
	if days <= 0 {
		return nil
	}
	cutoff := now.AddDate(0, 0, -days)
	if _, err := tx.ExecContext(ctx, qDeleteRetention, formatTime(cutoff)); err != nil {
		return err
	}
	return nil
}

// selectVanishedIDs returns the ids of PRs currently flagged review_pending=1
// that are absent from the incoming poll result. Must be called before the
// "mark not pending" UPDATE so the set is not diluted.
func selectVanishedIDs(ctx context.Context, tx *sql.Tx, polled map[string]struct{}) ([]string, error) {
	var q string
	var args []any
	if len(polled) == 0 {
		q = `SELECT id FROM prs WHERE review_pending = 1`
	} else {
		placeholders := make([]string, 0, len(polled))
		args = make([]any, 0, len(polled))
		for id := range polled {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		q = `SELECT id FROM prs WHERE review_pending = 1 AND id NOT IN (` + strings.Join(placeholders, ",") + `)`
	}
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("select vanished: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan vanished: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// RefreshPRStatus applies enrichment details to a tracked PR. Preserves the
// MERGED-override semantics: a non-nil MergedAt always wins over state.
// Empty details.ReviewState leaves the stored value untouched, so callers that
// only care about diff/status fields (legacy sites, tests) don't accidentally
// reset the review flag.
func (s *sqliteStore) RefreshPRStatus(id string, details github.PRDetails) error {
	db := s.handle()
	if db == nil {
		return ErrNotFound
	}
	ctx := context.Background()
	rec, ok, err := s.loadRecord(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	rec.Additions = details.Additions
	rec.Deletions = details.Deletions
	rec.IsDraft = details.IsDraft
	if details.State != "" {
		rec.State = details.State
	}
	if details.MergedAt != nil {
		rec.State = "MERGED"
	}
	if details.ReviewState != "" {
		rec.ReviewState = details.ReviewState
	}
	if _, err := db.ExecContext(ctx, qUpdatePRStatus,
		rec.Additions, rec.Deletions, boolToInt(rec.IsDraft), rec.State, rec.ReviewState, rec.ID,
	); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *sqliteStore) loadRecord(ctx context.Context, id string) (PRRecord, bool, error) {
	db := s.handle()
	rec, err := scanRow(db.QueryRowContext(ctx, qSelectPRByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return PRRecord{}, false, nil
	}
	if err != nil {
		return PRRecord{}, false, fmt.Errorf("load pr: %w", err)
	}
	return rec, true, nil
}

func loadRecordTx(ctx context.Context, tx *sql.Tx, id string) (PRRecord, bool) {
	rec, err := scanRow(tx.QueryRowContext(ctx, qSelectPRByID, id))
	if err != nil {
		return PRRecord{}, false
	}
	return rec, true
}

func scanRow(row interface {
	Scan(dest ...any) error
}) (PRRecord, error) {
	var (
		rec             PRRecord
		isDraft, pend   int
		firstSeen       string
		lastSeen        string
		lastNotifiedRaw sql.NullString
	)
	err := row.Scan(
		&rec.ID, &rec.Number, &rec.Repo, &rec.Title, &rec.Author, &rec.URL,
		&rec.State, &isDraft, &rec.Additions, &rec.Deletions,
		&pend, &rec.ReviewState, &firstSeen, &lastSeen, &lastNotifiedRaw,
	)
	if err != nil {
		return PRRecord{}, err
	}
	rec.IsDraft = isDraft != 0
	rec.ReviewPending = pend != 0
	rec.FirstSeenAt, err = parseTime(firstSeen)
	if err != nil {
		return PRRecord{}, err
	}
	rec.LastSeenAt, err = parseTime(lastSeen)
	if err != nil {
		return PRRecord{}, err
	}
	rec.LastNotifiedAt, err = parseTimePtr(lastNotifiedRaw)
	if err != nil {
		return PRRecord{}, err
	}
	return rec, nil
}

func mustScan(db *sql.DB, query string) []PRRecord {
	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]PRRecord, 0)
	for rows.Next() {
		rec, err := scanRow(rows)
		if err != nil {
			return out
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return out
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// getMetaString returns a meta value or (empty,false,nil) if absent.
// Thin wrapper around the package-level helper so test bodies read naturally.
func (s *sqliteStore) getMetaString(ctx context.Context, key string) (string, bool, error) {
	db := s.handle()
	if db == nil {
		return "", false, nil
	}
	return getMeta(ctx, db, key)
}

// setMetaString upserts a meta row.
func (s *sqliteStore) setMetaString(ctx context.Context, key, value string) error {
	db := s.handle()
	if db == nil {
		return errors.New("store not loaded")
	}
	_, err := db.ExecContext(ctx, qSetMeta, key, value)
	if err != nil {
		return fmt.Errorf("set meta %s: %w", key, err)
	}
	return nil
}

// newSQLiteFromDB wraps an already-open DB for tests that want to bypass the
// path-based Load flow. Exported only within the package.
func newSQLiteFromDB(db *sql.DB, opts ...Option) *sqliteStore {
	s := &sqliteStore{
		db:            db,
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
