// Package config loads and hot-reloads the runtime configuration from
// ~/.config/revu/config.json. Subscribers receive every validated change so
// components (poller, notifier, store) can reconfigure mid-flight without
// restarting the binary.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the user-editable shape persisted to JSON. Field names mirror
// SPEC §7 exactly — casing preserved for mapstructure binding.
type Config struct {
	PollingIntervalSeconds     int          `mapstructure:"polling_interval_seconds" json:"polling_interval_seconds"`
	NotificationsEnabled       bool         `mapstructure:"notifications_enabled" json:"notifications_enabled"`
	NotificationTimeoutSeconds int          `mapstructure:"notification_timeout_seconds" json:"notification_timeout_seconds"`
	StatusRefreshEveryNTicks   int          `mapstructure:"status_refresh_every_n_ticks" json:"status_refresh_every_n_ticks"`
	HistoryRetentionDays       int          `mapstructure:"history_retention_days" json:"history_retention_days"`
	StartHidden                bool         `mapstructure:"start_hidden" json:"start_hidden"`
	Window                     WindowConfig `mapstructure:"window" json:"window"`
}

// WindowConfig carries the initial Wails window geometry.
type WindowConfig struct {
	Width  int `mapstructure:"width" json:"width"`
	Height int `mapstructure:"height" json:"height"`
}

// Defaults returns the SPEC §7 baseline. Used when no config file exists
// (fresh install) and as a fallback when an invalid change lands on disk.
func Defaults() Config {
	return Config{
		PollingIntervalSeconds:     300,
		NotificationsEnabled:       true,
		NotificationTimeoutSeconds: 5,
		StatusRefreshEveryNTicks:   12,
		HistoryRetentionDays:       30,
		StartHidden:                true,
		Window:                     WindowConfig{Width: 480, Height: 640},
	}
}

// Manager owns a viper instance and broadcasts validated changes to
// subscribers. Safe for concurrent readers and one-shot Subscribe.
type Manager struct {
	log *slog.Logger
	v   *viper.Viper

	mu          sync.RWMutex
	current     Config
	subscribers []func(Config)
}

// Option customizes the Manager.
type Option func(*Manager)

// WithLogger injects a logger.
func WithLogger(l *slog.Logger) Option {
	return func(m *Manager) {
		if l != nil {
			m.log = l
		}
	}
}

// Load reads path and starts watching it. Missing files are tolerated — the
// defaults are written to disk so the user has a template to edit. Returns
// an error only when the directory cannot be created or the file exists but
// is unreadable/invalid.
func Load(path string, opts ...Option) (*Manager, error) {
	m := &Manager{
		log: slog.Default(),
		v:   viper.New(),
	}
	for _, opt := range opts {
		opt(m)
	}
	m.applyDefaults()
	m.v.SetConfigFile(path)
	m.v.SetConfigType("json")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir config dir: %w", err)
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// Seed the file with defaults so the user has something to edit.
		if err := m.v.WriteConfigAs(path); err != nil {
			return nil, fmt.Errorf("write default config: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat config: %w", err)
	}

	if err := m.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	c, err := m.decodeAndValidate()
	if err != nil {
		return nil, fmt.Errorf("config invalid on load: %w", err)
	}
	m.mu.Lock()
	m.current = c
	m.mu.Unlock()

	m.v.OnConfigChange(func(_ fsnotify.Event) { m.onChange() })
	m.v.WatchConfig()

	return m, nil
}

// Current returns a copy of the active configuration.
func (m *Manager) Current() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Subscribe registers a callback that receives every validated change. The
// callback fires on the viper watcher goroutine — keep it short, fan out if
// needed. Returns an unsubscribe function.
func (m *Manager) Subscribe(fn func(Config)) func() {
	m.mu.Lock()
	idx := len(m.subscribers)
	m.subscribers = append(m.subscribers, fn)
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if idx < len(m.subscribers) {
			m.subscribers[idx] = nil
		}
	}
}

func (m *Manager) onChange() {
	c, err := m.decodeAndValidate()
	if err != nil {
		m.log.Warn("config reload rejected; keeping previous values", "err", err)
		return
	}
	m.mu.Lock()
	prev := m.current
	m.current = c
	subs := append([]func(Config){}, m.subscribers...)
	m.mu.Unlock()
	if prev == c {
		return
	}
	for _, fn := range subs {
		if fn == nil {
			continue
		}
		fn(c)
	}
}

func (m *Manager) decodeAndValidate() (Config, error) {
	var c Config
	if err := m.v.Unmarshal(&c); err != nil {
		return Config{}, fmt.Errorf("unmarshal: %w", err)
	}
	if err := validate(&c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// applyDefaults registers every field default in viper so partial configs
// (and empty files) fill in safely.
func (m *Manager) applyDefaults() {
	d := Defaults()
	m.v.SetDefault("polling_interval_seconds", d.PollingIntervalSeconds)
	m.v.SetDefault("notifications_enabled", d.NotificationsEnabled)
	m.v.SetDefault("notification_timeout_seconds", d.NotificationTimeoutSeconds)
	m.v.SetDefault("status_refresh_every_n_ticks", d.StatusRefreshEveryNTicks)
	m.v.SetDefault("history_retention_days", d.HistoryRetentionDays)
	m.v.SetDefault("start_hidden", d.StartHidden)
	m.v.SetDefault("window.width", d.Window.Width)
	m.v.SetDefault("window.height", d.Window.Height)
}

// validate coerces obviously-broken values back into safe defaults. Total
// garbage (e.g. negative polling) is replaced silently with the spec
// baseline — better than crashing the daemon over a stray keystroke.
func validate(c *Config) error {
	d := Defaults()
	if c.PollingIntervalSeconds < 30 {
		c.PollingIntervalSeconds = d.PollingIntervalSeconds
	}
	if c.NotificationTimeoutSeconds < 0 {
		c.NotificationTimeoutSeconds = d.NotificationTimeoutSeconds
	}
	if c.StatusRefreshEveryNTicks < 1 {
		c.StatusRefreshEveryNTicks = d.StatusRefreshEveryNTicks
	}
	if c.HistoryRetentionDays < 0 {
		c.HistoryRetentionDays = d.HistoryRetentionDays
	}
	if c.Window.Width < 240 {
		c.Window.Width = d.Window.Width
	}
	if c.Window.Height < 240 {
		c.Window.Height = d.Window.Height
	}
	return nil
}
