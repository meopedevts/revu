// Package uilimits holds numeric thresholds shared between the Go core and
// the React frontend. These values are mirrored to TS by cmd/gentsconst, so
// frontend code never hardcodes a magic number that could drift.
package uilimits

// DetailsDiffLimit caps the (additions+deletions) a PR can have before the
// inline diff view is skipped. Backend returns "" above the limit; the
// frontend uses the same value to render an explanatory placeholder.
const DetailsDiffLimit = 500

// PollSafetyIntervalMS is the fallback period (milliseconds) the frontend
// uses to re-sync the PR list when Wails events appear to be missed.
const PollSafetyIntervalMS = 120_000
