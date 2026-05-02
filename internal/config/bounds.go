package config

// Bound is the inclusive [Min, Max] range for an integer config field.
type Bound struct {
	Min int
	Max int
}

// Bounds aggregates the validation bounds enforced by validateStrict and
// exposed to the frontend (via cmd/gentsconst) so the Zod schema can reuse
// the same numbers without manual duplication.
type Bounds struct {
	PollingIntervalSeconds      Bound
	NotificationTimeoutSeconds  Bound
	NotificationCooldownMinutes Bound
	StatusRefreshEveryNTicks    Bound
	HistoryRetentionDays        Bound
	WindowWidth                 Bound
	WindowHeight                Bound
	ValidThemes                 []string
}

// Limits returns the canonical bounds. validateStrict consumes these, and
// cmd/gentsconst serialises them into the generated TS constants.
func Limits() Bounds {
	return Bounds{
		PollingIntervalSeconds:      Bound{Min: 30, Max: 3600},
		NotificationTimeoutSeconds:  Bound{Min: 1, Max: 30},
		NotificationCooldownMinutes: Bound{Min: 0, Max: 10080},
		StatusRefreshEveryNTicks:    Bound{Min: 1, Max: 1000},
		HistoryRetentionDays:        Bound{Min: 1, Max: 365},
		WindowWidth:                 Bound{Min: 240, Max: 3840},
		WindowHeight:                Bound{Min: 240, Max: 2160},
		ValidThemes:                 []string{"light", "dark", "auto"},
	}
}
