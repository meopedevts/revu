package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func TestLoad_MissingFileSeedsDefaults(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := m.Current(); got != Defaults() {
		t.Fatalf("current != defaults:\nwant %+v\ngot  %+v", Defaults(), got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not seeded: %v", err)
	}
}

func TestLoad_FromExistingFile(t *testing.T) {
	path := tempPath(t)
	writeJSON(t, path, map[string]any{
		"polling_interval_seconds": 60,
		"notifications_enabled":    false,
		"history_retention_days":   15,
		"window":                   map[string]any{"width": 720, "height": 900},
		"theme":                    "dark",
	})
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c := m.Current()
	if c.PollingIntervalSeconds != 60 || c.NotificationsEnabled || c.HistoryRetentionDays != 15 {
		t.Fatalf("overrides not honored: %+v", c)
	}
	if c.Window.Width != 720 || c.Window.Height != 900 {
		t.Fatalf("window not honored: %+v", c.Window)
	}
	if c.Theme != "dark" {
		t.Fatalf("theme not honored: %q", c.Theme)
	}
	// Unset field falls back to default.
	if c.NotificationTimeoutSeconds != Defaults().NotificationTimeoutSeconds {
		t.Fatalf("default fallback missing: %+v", c)
	}
}

func TestValidate_CoercesInsaneValues(t *testing.T) {
	path := tempPath(t)
	writeJSON(t, path, map[string]any{
		"polling_interval_seconds":     5, // too small — bumped to default
		"notification_timeout_seconds": -1,
		"status_refresh_every_n_ticks": 0,
		"window":                       map[string]any{"width": 10, "height": -1},
		"theme":                        "system", // unsupported in MVP → coerced to default
	})
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c := m.Current()
	d := Defaults()
	if c.PollingIntervalSeconds != d.PollingIntervalSeconds {
		t.Fatalf("polling not coerced: %d", c.PollingIntervalSeconds)
	}
	if c.NotificationTimeoutSeconds != d.NotificationTimeoutSeconds {
		t.Fatalf("timeout not coerced: %d", c.NotificationTimeoutSeconds)
	}
	if c.StatusRefreshEveryNTicks != d.StatusRefreshEveryNTicks {
		t.Fatalf("refresh ticks not coerced: %d", c.StatusRefreshEveryNTicks)
	}
	if c.Window.Width != d.Window.Width || c.Window.Height != d.Window.Height {
		t.Fatalf("window not coerced: %+v", c.Window)
	}
	if c.Theme != d.Theme {
		t.Fatalf("theme not coerced to default: %q", c.Theme)
	}
}

func TestUpdate_RejectsUnsupportedTheme(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c := m.Current()
	c.Theme = "system"

	err = m.Update(c)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	var found bool
	for _, fe := range ve.Errors {
		if fe.Field == "theme" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected theme field error, got: %+v", ve.Errors)
	}
}

func TestSubscribe_FiresOnFileChange(t *testing.T) {
	path := tempPath(t)
	writeJSON(t, path, map[string]any{"polling_interval_seconds": 60})

	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	var mu sync.Mutex
	var received []Config
	ch := make(chan struct{}, 4)
	m.Subscribe(func(c Config) {
		mu.Lock()
		received = append(received, c)
		mu.Unlock()
		select {
		case ch <- struct{}{}:
		default:
		}
	})

	// Give the watcher a moment to attach.
	time.Sleep(100 * time.Millisecond)
	writeJSON(t, path, map[string]any{"polling_interval_seconds": 120})

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("subscriber never fired")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Fatal("no config delivered")
	}
	last := received[len(received)-1]
	if last.PollingIntervalSeconds != 120 {
		t.Fatalf("subscriber got stale polling value: %d", last.PollingIntervalSeconds)
	}
}

func TestUpdate_WritesAtomicallyAndReloads(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	c := m.Current()
	c.PollingIntervalSeconds = 120
	c.NotificationsEnabled = false
	c.HistoryRetentionDays = 45
	c.Window.Width = 700
	c.Window.Height = 800

	if err := m.Update(c); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Current() returns the new value without waiting for the watcher.
	got := m.Current()
	if got.PollingIntervalSeconds != 120 || got.NotificationsEnabled || got.HistoryRetentionDays != 45 {
		t.Fatalf("current stale after update: %+v", got)
	}
	if got.Window.Width != 700 || got.Window.Height != 800 {
		t.Fatalf("window stale after update: %+v", got.Window)
	}

	// File content matches.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after update: %v", err)
	}
	var onDisk map[string]any
	if err := json.Unmarshal(b, &onDisk); err != nil {
		t.Fatalf("unmarshal after update: %v", err)
	}
	if v, ok := onDisk["polling_interval_seconds"].(float64); !ok || int(v) != 120 {
		t.Fatalf("polling on disk wrong: %v", onDisk["polling_interval_seconds"])
	}

	// No leftover .tmp file.
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tmp file not removed: err=%v", err)
	}
}

func TestUpdate_RejectsInvalid(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c := m.Current()
	c.PollingIntervalSeconds = 10    // below 30
	c.NotificationTimeoutSeconds = 0 // below 1
	c.HistoryRetentionDays = 0       // below 1

	err = m.Update(c)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if len(ve.Errors) < 3 {
		t.Fatalf("expected at least 3 field errors, got %d: %+v", len(ve.Errors), ve.Errors)
	}
	fields := map[string]bool{}
	for _, fe := range ve.Errors {
		fields[fe.Field] = true
	}
	for _, required := range []string{"polling_interval_seconds", "notification_timeout_seconds", "history_retention_days"} {
		if !fields[required] {
			t.Fatalf("missing field %q in errors: %+v", required, ve.Errors)
		}
	}

	// Current() unchanged on rejection.
	if m.Current().PollingIntervalSeconds != Defaults().PollingIntervalSeconds {
		t.Fatalf("current mutated after rejected update: %+v", m.Current())
	}
}

func TestUpdate_NotifiesSubscribers(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	ch := make(chan Config, 2)
	m.Subscribe(func(c Config) {
		select {
		case ch <- c:
		default:
		}
	})

	// Give watcher time to attach.
	time.Sleep(100 * time.Millisecond)

	c := m.Current()
	c.PollingIntervalSeconds = 90
	if err := m.Update(c); err != nil {
		t.Fatalf("update: %v", err)
	}

	select {
	case got := <-ch:
		if got.PollingIntervalSeconds != 90 {
			t.Fatalf("subscriber got stale polling: %d", got.PollingIntervalSeconds)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("subscriber never fired after Update")
	}
}

// REV-43 — NotificationCooldownMinutes is the throttle window the poller
// reads on every tick. Defaults to 360 (6h); 0 disables; negative coerces
// back to default; out-of-bounds rejected by Update.
func TestNotificationCooldown_DefaultAndCoercion(t *testing.T) {
	if Defaults().NotificationCooldownMinutes != 360 {
		t.Fatalf("default cooldown drift: %d", Defaults().NotificationCooldownMinutes)
	}

	path := tempPath(t)
	writeJSON(t, path, map[string]any{
		"notification_cooldown_minutes": -5,
	})
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := m.Current().NotificationCooldownMinutes; got != Defaults().NotificationCooldownMinutes {
		t.Fatalf("negative not coerced: %d", got)
	}

	// Zero is a valid (throttle-disabled) value and must be preserved.
	zeroPath := tempPath(t)
	writeJSON(t, zeroPath, map[string]any{
		"notification_cooldown_minutes": 0,
	})
	m2, err := Load(zeroPath)
	if err != nil {
		t.Fatalf("load zero: %v", err)
	}
	if got := m2.Current().NotificationCooldownMinutes; got != 0 {
		t.Fatalf("zero must round-trip (throttle disabled), got %d", got)
	}
}

func TestUpdate_RejectsCooldownOutOfBounds(t *testing.T) {
	path := tempPath(t)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c := m.Current()
	c.NotificationCooldownMinutes = 10081 // above max (10080 = 1 week)

	err = m.Update(c)
	if err == nil {
		t.Fatal("expected validation error for cooldown above max")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	var found bool
	for _, fe := range ve.Errors {
		if fe.Field == "notification_cooldown_minutes" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected notification_cooldown_minutes error, got: %+v", ve.Errors)
	}
}

func writeJSON(t *testing.T, path string, body any) {
	t.Helper()
	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
