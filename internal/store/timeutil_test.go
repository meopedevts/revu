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

// FuzzParseTimePtr varre input arbitrário pra parseTimePtr garantindo:
// nunca panic; (err != nil) ⇒ ptr nil; (err == nil) ⇒ timezone UTC e
// round-trip via formatTime estável. Seed cobre RFC3339Nano canônico,
// timezones, edge years, ns precision e malformed.
func FuzzParseTimePtr(f *testing.F) {
	seeds := []string{
		"",
		"2026-04-25T10:30:45Z",
		"2026-04-25T10:30:45.999999999Z",
		"2026-04-25T10:30:45+03:00",
		"2026-04-25T10:30:45-05:30",
		"0001-01-01T00:00:00Z",
		"9999-12-31T23:59:59.999999999Z",
		"not-a-timestamp",
		"2026-13-45T99:99:99Z",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got, err := parseTimePtr(sql.NullString{String: raw, Valid: true})
		if err != nil {
			if got != nil {
				t.Fatalf("err != nil mas got != nil: err=%v ptr=%v", err, *got)
			}
			return
		}
		if got == nil {
			t.Fatalf("err == nil mas got == nil para raw=%q", raw)
		}
		if got.Location() != time.UTC {
			t.Fatalf("location não-UTC: %v (raw=%q)", got.Location(), raw)
		}
		again, err2 := parseTime(formatTime(*got))
		if err2 != nil {
			t.Fatalf("round-trip falhou: %v (raw=%q)", err2, raw)
		}
		if !again.Equal(*got) {
			t.Fatalf("round-trip diverge: %v != %v (raw=%q)", again, *got, raw)
		}
	})
}
