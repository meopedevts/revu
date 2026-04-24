package store

import (
	"database/sql"
	"fmt"
	"time"
)

// formatTime renders t as RFC3339Nano in UTC. Storing timestamps as TEXT
// (instead of relying on the SQLite driver's time scanner) keeps round-trips
// deterministic across drivers and makes values human-readable on the CLI.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime inverts formatTime. Errors surface as wrapped with context so
// callers can tell which column failed.
func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

// parseTimePtr reads a nullable timestamp column. NULL → (nil, nil);
// non-empty parses via parseTime.
func parseTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// formatTimePtr returns a sql.NullString from an optional time pointer.
func formatTimePtr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}
