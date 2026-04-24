package notifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meopedevts/revu/internal/store"
)

func TestFormatBody(t *testing.T) {
	pr := store.PRRecord{
		Number:    123,
		Title:     "feat: add foo",
		Author:    "alice",
		Additions: 142,
		Deletions: 37,
	}
	got := FormatBody(pr)
	want := "PR #123 — feat: add foo\npor @alice · +142 -37"
	if got != want {
		t.Fatalf("FormatBody mismatch:\nwant %q\ngot  %q", want, got)
	}
}

func TestExtractIcon_WritesAndReuses(t *testing.T) {
	// Redirect TMPDIR so the test does not pollute the real /tmp/revu dir.
	t.Setenv("TMPDIR", t.TempDir())

	data := []byte("\x89PNG\r\n\x1a\nfake png body")
	path, err := extractIcon(data, "test-icon.png")
	if err != nil {
		t.Fatalf("first extract: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("want absolute path, got %s", path)
	}
	if !strings.HasSuffix(path, "test-icon.png") {
		t.Fatalf("unexpected path: %s", path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != string(data) {
		t.Fatal("extracted content does not match input")
	}

	// Second call with same-size data must reuse without touching disk
	// semantics beyond stat.
	path2, err := extractIcon(data, "test-icon.png")
	if err != nil {
		t.Fatalf("second extract: %v", err)
	}
	if path2 != path {
		t.Fatalf("reuse path mismatch: %s vs %s", path, path2)
	}

	// Different-size content triggers rewrite.
	data2 := append([]byte{}, data...)
	data2 = append(data2, "more bytes"...)
	path3, err := extractIcon(data2, "test-icon.png")
	if err != nil {
		t.Fatalf("rewrite extract: %v", err)
	}
	if path3 != path {
		t.Fatalf("path should remain stable on rewrite: %s vs %s", path3, path)
	}
	got2, err := os.ReadFile(path3)
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != string(data2) {
		t.Fatal("rewrite did not update file content")
	}
}
