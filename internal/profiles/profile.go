// Package profiles manages GitHub authentication profiles persisted in the
// SQLite store. Each profile chooses between using the ambient `gh auth login`
// session or a Personal Access Token stored in the system keyring.
//
// Only the opaque keyring_ref lives in the database. Tokens never touch the
// store, logs, or long-lived structs — they are fetched from the keyring only
// when a `gh` invocation is about to run and dropped immediately after.
package profiles

import (
	"errors"
	"time"
)

// AuthMethod enumerates how a profile authenticates to GitHub.
type AuthMethod string

const (
	// AuthGHCLI defers to the ambient `gh auth` session on the host.
	AuthGHCLI AuthMethod = "gh-cli"
	// AuthKeyring uses a PAT fetched from the OS keyring via keyring_ref.
	AuthKeyring AuthMethod = "keyring"
)

// Valid reports whether m is a recognized AuthMethod.
func (m AuthMethod) Valid() bool {
	return m == AuthGHCLI || m == AuthKeyring
}

// Profile is the domain shape surfaced to the poller and UI. It never carries
// the secret — callers fetch it from the Keyring using KeyringRef when they
// actually need to invoke `gh`.
type Profile struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	AuthMethod      AuthMethod `json:"auth_method"`
	KeyringRef      string     `json:"keyring_ref,omitempty"`
	GitHubUsername  string     `json:"github_username,omitempty"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	LastValidatedAt *time.Time `json:"last_validated_at,omitempty"`
}

// Update is the mutable payload accepted by Service.Update. Zero-value fields
// are left untouched; a non-nil Token triggers keyring replacement.
type Update struct {
	Name   *string     `json:"name,omitempty"`
	Method *AuthMethod `json:"auth_method,omitempty"`
	// Token is opt-in replacement of the stored PAT. Empty = do not touch.
	// The service validates it before writing.
	Token *string `json:"token,omitempty"`
}

// Sentinel errors returned by the service. Callers branch on these via
// errors.Is — never compare messages.
var (
	ErrNotFound           = errors.New("profile not found")
	ErrNameTaken          = errors.New("profile name already in use")
	ErrInvalidMethod      = errors.New("invalid auth method")
	ErrTokenRequired      = errors.New("token required for keyring method")
	ErrTokenInvalid       = errors.New("token rejected by github")
	ErrCannotDeleteLast   = errors.New("cannot delete the last profile")
	ErrCannotDeleteActive = errors.New("cannot delete the active profile; switch first")
	ErrNoActiveProfile    = errors.New("no active profile")
)
