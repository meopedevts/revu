package profiles

import (
	"errors"
	"strings"
	"testing"
)

// FuzzIsUniqueNameViolation blinda a detecção de violação de UNIQUE em
// profiles.name contra refator que afrouxe o match (ex: regex). Invariante:
// resultado bate com [strings.Contains] da substring canônica. Seed cobre
// canonical, lowercase, constraints irmãs, ordem trocada e null bytes.
func FuzzIsUniqueNameViolation(f *testing.F) {
	if isUniqueNameViolation(nil) {
		f.Fatal("nil deve ser false")
	}
	seeds := []string{
		"",
		"UNIQUE constraint failed: profiles.name",
		"UNIQUE constraint failed: profiles.id",
		"UNIQUE constraint failed: profiles.email",
		"unique constraint failed: profiles.name",
		"some prefix: UNIQUE constraint failed: profiles.name",
		"constraint UNIQUE failed: profiles.name",
		"\x00UNIQUE constraint failed: profiles.name\x00",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got := isUniqueNameViolation(errors.New(raw))
		want := strings.Contains(raw, "UNIQUE constraint failed: profiles.name")
		if got != want {
			t.Fatalf("inconsistente: raw=%q got=%v want=%v", raw, got, want)
		}
	})
}
