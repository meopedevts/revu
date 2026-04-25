package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// checkProfiles queries the profiles table directly (read-only) and reports
// the list with the active one flagged. Skipped cleanly on a fresh install
// (DB absent) since `revu doctor` runs before first boot.
func checkProfiles(ctx context.Context, path string) checkResult {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return checkResult{Name: "profiles", Detail: "DB ausente — skip", OK: true}
	}
	db, err := openReadOnlyDB(ctx, path)
	if err != nil {
		return checkResult{Name: "profiles", Detail: err.Error()}
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT name, auth_method, is_active FROM profiles ORDER BY created_at ASC`)
	if err != nil {
		return checkResult{Name: "profiles", Detail: err.Error()}
	}
	defer rows.Close()

	var parts []string
	usesKeyring := false
	for rows.Next() {
		var name, method string
		var active int
		if err := rows.Scan(&name, &method, &active); err != nil {
			return checkResult{Name: "profiles", Detail: err.Error()}
		}
		if method == "keyring" {
			usesKeyring = true
		}
		marker := "○"
		if active != 0 {
			marker = "●"
		}
		parts = append(parts, fmt.Sprintf("%s %s (%s)", marker, name, method))
	}
	if err := rows.Err(); err != nil {
		return checkResult{Name: "profiles", Detail: err.Error()}
	}
	if len(parts) == 0 {
		return checkResult{Name: "profiles", Detail: "nenhum profile — rode `revu run` pelo menos uma vez"}
	}
	detail := strings.Join(parts, ", ")
	if usesKeyring {
		detail += " · keyring usado (Secret Service via D-Bus)"
	}
	return checkResult{Name: "profiles", Detail: detail, OK: true}
}
