package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// TestSQLite_Load_OpenDBError exercita o error path em Load quando o
// opener do DB falha. Usa o seam withDBOpener pra injetar uma função
// que retorna sentinel — produção sempre usa openDB. Mata mutantes que
// invertam `if err != nil { return err }` na linha 44.
func TestSQLite_Load_OpenDBError(t *testing.T) {
	sentinel := errors.New("opener boom")
	s := New("/tmp/revu-failure-test.db", withDBOpener(func(_ string) (*sql.DB, error) {
		return nil, sentinel
	})).(*sqliteStore)

	err := s.Load(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err: want sentinel, got %v", err)
	}
	if s.db != nil {
		t.Errorf("db: want nil after opener failure, got non-nil")
	}
}

// TestSQLite_Load_MigrateJSONError força migrateJSONIfPresent a falhar
// com JSON malformado em jsonStatePath. Confirma que Load wrappa o
// erro e que o handle é fechado (db permanece nil). Mata mutantes na
// linha 49 (chamada migrateJSONIfPresent) e linha 50 (db.Close no
// rollback).
func TestSQLite_Load_MigrateJSONError(t *testing.T) {
	dir := t.TempDir()
	badJSON := filepath.Join(dir, "state.json")
	if err := os.WriteFile(badJSON, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("seed bad json: %v", err)
	}
	dbPath := filepath.Join(dir, "store.db")

	s := New(dbPath, WithJSONMigration(badJSON)).(*sqliteStore)

	err := s.Load(context.Background())
	if err == nil {
		t.Fatal("err: expected failure from malformed json, got nil")
	}
	if !strings.Contains(err.Error(), "import legacy state.json") {
		t.Errorf("err: want wrap 'import legacy state.json:', got %v", err)
	}
	if s.db != nil {
		t.Errorf("db: want nil after migration failure (Load must Close), got non-nil")
	}
}

// TestUpsertPolled_QueryRowCtxCancel exercita o error path
// `case err != nil` (linha 273) em upsertPolled quando o
// QueryRowContext devolve erro. Cancela o ctx antes da execução —
// driver respeita ctx e Scan() devolve [context.Canceled]. Pré-popula
// um PR pra forçar o branch que executa a query (não-novo).
func TestUpsertPolled_QueryRowCtxCancel(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: ctx", "alice", false)
	if novos, _ := s.UpdateFromPoll(context.Background(), []github.PRSummary{pr}); len(novos) != 1 {
		t.Fatalf("seed: want 1 novo, got %d", len(novos))
	}

	db := s.handle()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = upsertPolled(ctx, tx, []github.PRSummary{pr}, time.Now(), "")
	if err == nil {
		t.Fatal("err: expected failure from canceled ctx, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err: want context.Canceled, got %v", err)
	}
}

// TestSQLite_GetByID_CtxCanceled valida que o ctx cancelado pelo caller
// chega no driver SQLite na rota de leitura. Usa loadRecord direto pra
// observar o erro (a fachada GetByID silencia e devolve (zero, false)).
// Mata mutantes que troquem `db.QueryRowContext(ctx, ...)` por
// `Background()` em loadRecord — cobertura REV-26 do critério "leitura".
func TestSQLite_GetByID_CtxCanceled(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: ctx", "alice", false)
	if novos, _ := s.UpdateFromPoll(context.Background(), []github.PRSummary{pr}); len(novos) != 1 {
		t.Fatalf("seed: want 1 novo, got %d", len(novos))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := s.loadRecord(ctx, pr.ID)
	if err == nil {
		t.Fatal("err: expected failure from canceled ctx, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err: want context.Canceled, got %v", err)
	}

	// Façade pública silencia o erro mas devolve zero/false — confirma o
	// contrato externo do GetByID em ctx cancelado.
	if rec, ok := s.GetByID(ctx, pr.ID); ok || rec.ID != "" {
		t.Fatalf("GetByID with canceled ctx: want (zero,false), got (%+v,%v)", rec, ok)
	}
}

// TestSQLite_ClearHistory_CtxCanceled valida que ClearHistory respeita
// ctx cancelado e não apaga nada. Pré-popula um registro non-OPEN, então
// cancela ctx antes de chamar. Mata mutantes que troquem
// `db.ExecContext(ctx, ...)` por `Background()` ou `db.Exec(...)` —
// cobertura REV-26 do critério "escrita".
func TestSQLite_ClearHistory_CtxCanceled(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: ctx", "alice", false)
	bg := context.Background()
	s.UpdateFromPoll(bg, []github.PRSummary{pr})
	if err := s.RefreshPRStatus(bg, pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatalf("seed RefreshPRStatus: %v", err)
	}
	// Tira do polling pra entrar no histórico.
	s.UpdateFromPoll(bg, nil)
	if got := len(s.GetHistory(bg)); got != 1 {
		t.Fatalf("seed: want 1 history record, got %d", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n, err := s.ClearHistory(ctx)
	if err == nil {
		t.Fatal("err: expected failure from canceled ctx, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err: want context.Canceled, got %v", err)
	}
	if n != 0 {
		t.Fatalf("rows affected: want 0 on canceled ctx, got %d", n)
	}
	// History deve continuar intocado.
	if got := len(s.GetHistory(bg)); got != 1 {
		t.Fatalf("history must survive canceled ClearHistory; got %d", got)
	}
}
