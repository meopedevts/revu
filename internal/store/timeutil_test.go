package store

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

// TestParseTimePtr_NullReturnsNilNoErr cobre o fast-path NULL: NullString
// inválida vira (nil, nil) sem tentar parsear. Documenta o contrato pra
// callers que usam columns nullable.
func TestParseTimePtr_NullReturnsNilNoErr(t *testing.T) {
	got, err := parseTimePtr(sql.NullString{Valid: false})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != nil {
		t.Fatalf("Valid=false must return nil ptr, got %v", *got)
	}
}

// TestParseTimePtr_MalformedStringErrors cobre `if err != nil` em
// parseTimePtr quando o conteúdo não é RFC3339. Garante que o erro do
// parseTime se propaga em vez de virar zero-time silencioso.
func TestParseTimePtr_MalformedStringErrors(t *testing.T) {
	_, err := parseTimePtr(sql.NullString{String: "not-a-timestamp", Valid: true})
	if err == nil {
		t.Fatal("expected error for malformed timestamp")
	}
	if !strings.Contains(err.Error(), "parse timestamp") {
		t.Errorf("error should reference parse timestamp, got %v", err)
	}
}

// TestParseTimePtr_ValidStringReturnsTime garante que o happy path
// retorna o ponteiro com o timestamp correto em UTC.
func TestParseTimePtr_ValidStringReturnsTime(t *testing.T) {
	want := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	got, err := parseTimePtr(sql.NullString{String: formatTime(want), Valid: true})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil ptr")
	}
	if !got.Equal(want) {
		t.Fatalf("want %v, got %v", want, *got)
	}
}
