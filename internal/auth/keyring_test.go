package auth

import (
	"errors"
	"testing"
)

func TestFake_SetGetDelete(t *testing.T) {
	k := NewFake()

	if _, err := k.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on missing, got %v", err)
	}

	if err := k.Set("ref-1", "s3cret"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := k.Get("ref-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "s3cret" {
		t.Fatalf("got %q, want s3cret", got)
	}

	if err := k.Set("ref-1", "updated"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, _ = k.Get("ref-1")
	if got != "updated" {
		t.Fatalf("overwrite failed: got %q", got)
	}

	if err := k.Delete("ref-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := k.Get("ref-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Deleting a missing entry is a no-op.
	if err := k.Delete("ref-1"); err != nil {
		t.Fatalf("delete missing should be nil, got %v", err)
	}
}

func TestFake_EmptyRef(t *testing.T) {
	k := NewFake()
	if err := k.Set("", "x"); !errors.Is(err, ErrEmptyRef) {
		t.Errorf("Set empty: want ErrEmptyRef, got %v", err)
	}
	if _, err := k.Get(""); !errors.Is(err, ErrEmptyRef) {
		t.Errorf("Get empty: want ErrEmptyRef, got %v", err)
	}
	if err := k.Delete(""); !errors.Is(err, ErrEmptyRef) {
		t.Errorf("Delete empty: want ErrEmptyRef, got %v", err)
	}
}

// Compile-time check: System and Fake both implement Keyring.
var _ Keyring = (*System)(nil)
var _ Keyring = (*Fake)(nil)
