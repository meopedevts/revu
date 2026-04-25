package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/meopedevts/revu/internal/github"
)

// checkResult is the output of a single doctor probe. OK false shows as ✗
// with Detail appended; OK true shows as ✓ with Name only.
type checkResult struct {
	Name   string
	Detail string
	OK     bool
}

// lookupPath abstracts exec.LookPath for testing.
type lookupPath func(string) (string, error)

// cmdRunner runs a command to completion and returns the combined exec error.
// Stdout/stderr are discarded — doctor only cares about exit status for
// system-probe commands (pkg-config, ldconfig).
type cmdRunner func(ctx context.Context, name string, args ...string) error

func defaultRunner(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func checkGHInPath(look lookupPath) checkResult {
	if _, err := look("gh"); err != nil {
		return checkResult{Name: "gh na PATH", Detail: "instale github.com/cli/cli"}
	}
	return checkResult{Name: "gh na PATH", OK: true}
}

func checkGHAuth(ctx context.Context, c github.Client) checkResult {
	err := c.AuthStatus(ctx)
	if err == nil {
		return checkResult{Name: "gh auth status", OK: true}
	}
	if errors.Is(err, github.ErrAuthExpired) {
		return checkResult{Name: "gh auth status", Detail: "sessão expirada — rode `gh auth login`"}
	}
	return checkResult{Name: "gh auth status", Detail: err.Error()}
}

func checkDBus(env func(string) string) checkResult {
	if env("DBUS_SESSION_BUS_ADDRESS") == "" {
		return checkResult{Name: "D-Bus session", Detail: "DBUS_SESSION_BUS_ADDRESS não setada"}
	}
	return checkResult{Name: "D-Bus session", OK: true}
}

// checkAppIndicator prefers pkg-config (precise) and falls back to ldconfig
// for systems without pkgconf installed.
func checkAppIndicator(ctx context.Context, look lookupPath, run cmdRunner) checkResult {
	if _, err := look("pkg-config"); err == nil {
		if err := run(ctx, "pkg-config", "--exists", "ayatana-appindicator3-0.1"); err == nil {
			return checkResult{Name: "libayatana-appindicator", OK: true}
		}
	}
	if err := run(ctx, "sh", "-c", "ldconfig -p | grep -q ayatana"); err == nil {
		return checkResult{Name: "libayatana-appindicator", OK: true}
	}
	return checkResult{Name: "libayatana-appindicator", Detail: "instale libayatana-appindicator"}
}

// runAllChecks is the composition used by the doctor command. Split from the
// Cobra wiring to keep it unit-testable.
func runAllChecks(ctx context.Context, c github.Client, db string) []checkResult {
	return []checkResult{
		checkGHInPath(exec.LookPath),
		checkGHAuth(ctx, c),
		checkDBus(os.Getenv),
		checkAppIndicator(ctx, exec.LookPath, defaultRunner),
		checkDBPath(db),
		checkSchemaVersion(ctx, db),
		checkPRCounts(ctx, db),
		checkProfiles(ctx, db),
	}
}
