// Package auth abstracts system keyring access for revu.
//
// Tokens live in the OS Secret Service (libsecret / KWallet / gnome-keyring).
// Only the opaque keyring_ref is persisted in the SQLite store; the secret is
// fetched at call-time and discarded immediately after use.
package auth

import (
	"errors"
	"fmt"

	gokeyring "github.com/zalando/go-keyring"
)

// Service is the keyring service name used by revu for all entries.
const Service = "revu"

// ErrNotFound is returned when a secret for a given ref is missing from the
// keyring. Callers can distinguish it from transport / backend errors.
var ErrNotFound = errors.New("keyring: secret not found")

// ErrEmptyRef is returned when an empty ref is passed to Set/Get/Delete.
var ErrEmptyRef = errors.New("keyring: empty ref")

// Keyring is the minimal surface the rest of the app depends on.
// Kept tiny so we can swap in a fake for tests.
type Keyring interface {
	Set(ref, secret string) error
	Get(ref string) (string, error)
	Delete(ref string) error
}

// System is the real implementation backed by github.com/zalando/go-keyring.
// On Linux this uses libsecret via D-Bus (Secret Service API).
type System struct{}

// New returns a production keyring backed by the OS Secret Service.
func New() *System { return &System{} }

// Set stores secret under (Service, ref), overwriting any existing entry.
func (s *System) Set(ref, secret string) error {
	if ref == "" {
		return ErrEmptyRef
	}
	if err := gokeyring.Set(Service, ref, secret); err != nil {
		return fmt.Errorf("keyring: set %s: %w", ref, err)
	}
	return nil
}

// Get returns the secret stored under (Service, ref). Returns ErrNotFound if
// the entry does not exist.
func (s *System) Get(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("keyring: empty ref")
	}
	secret, err := gokeyring.Get(Service, ref)
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("keyring: get %s: %w", ref, err)
	}
	return secret, nil
}

// Delete removes the entry at (Service, ref). Deleting a missing entry is a
// no-op (returns nil) so callers can use it defensively.
func (s *System) Delete(ref string) error {
	if ref == "" {
		return ErrEmptyRef
	}
	if err := gokeyring.Delete(Service, ref); err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("keyring: delete %s: %w", ref, err)
	}
	return nil
}
