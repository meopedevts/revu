package config

import (
	"encoding/json"
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
