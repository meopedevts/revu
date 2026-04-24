// Package config loads and hot-reloads the runtime configuration from
// ~/.config/revu/config.json. Subscribers receive every validated change so
// components (poller, notifier, store) can reconfigure mid-flight without
// restarting the binary.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	Theme                      string       `mapstructure:"theme" json:"theme"`
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
		Theme:                      "light",
	}
}

// Manager owns a viper instance and broadcasts validated changes to
// subscribers. Safe for concurrent readers and one-shot Subscribe.
type Manager struct {
	log  *slog.Logger
	v    *viper.Viper
	path string

	mu          sync.RWMutex
	current     Config
	subscribers []func(Config)
}

// FieldError describes a single validation failure for a config field.
// Serialized as-is across the Wails bridge so the frontend can attach each
// message to its form input.
type FieldError struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}

// ValidationError aggregates FieldError values from a strict validation
// pass. Returned by Manager.Update so callers can distinguish invalid input
// from I/O failures.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

// Error implements error. Concatenates field messages for logs.
func (v *ValidationError) Error() string {
	if v == nil || len(v.Errors) == 0 {
		return "config validation failed"
	}
	parts := make([]string, 0, len(v.Errors))
	for _, e := range v.Errors {
		parts = append(parts, e.Field+": "+e.Msg)
	}
	return "config validation failed: " + strings.Join(parts, "; ")
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
	m.path = path
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

// Update validates c with strict bounds and writes it atomically to disk.
// The viper watcher picks up the rename and fires subscribers via onChange;
// m.current is also updated inline so the next Current() call does not race
// the fsnotify goroutine. Returns *ValidationError on invalid input, or a
// wrapped I/O error when persistence fails.
func (m *Manager) Update(c Config) error {
	if err := validateStrict(&c); err != nil {
		return err
	}
	if m.path == "" {
		return errors.New("update config: manager has no persistent path")
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("update config: marshal: %w", err)
	}

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("update config: mkdir: %w", err)
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("update config: write tmp: %w", err)
	}
	if err := os.Rename(tmp, m.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("update config: rename: %w", err)
	}

	m.mu.Lock()
	prev := m.current
	m.current = c
	subs := append([]func(Config){}, m.subscribers...)
	m.mu.Unlock()

	if prev != c {
		m.broadcast(subs, c)
	}
	return nil
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
	m.broadcast(subs, c)
}

func (m *Manager) broadcast(subs []func(Config), c Config) {
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
	m.v.SetDefault("theme", d.Theme)
}

// validateStrict enforces UI-facing bounds and returns a *ValidationError
// listing every offending field. Used by Update so the frontend can render
// inline messages instead of silently coercing values. Ranges intentionally
// match the zod schema on the frontend — keep them in sync.
func validateStrict(c *Config) error {
	var fe []FieldError
	if c.PollingIntervalSeconds < 30 || c.PollingIntervalSeconds > 3600 {
		fe = append(fe, FieldError{Field: "polling_interval_seconds", Msg: "deve estar entre 30 e 3600 segundos"})
	}
	if c.NotificationTimeoutSeconds < 1 || c.NotificationTimeoutSeconds > 30 {
		fe = append(fe, FieldError{Field: "notification_timeout_seconds", Msg: "deve estar entre 1 e 30 segundos"})
	}
	if c.StatusRefreshEveryNTicks < 1 || c.StatusRefreshEveryNTicks > 1000 {
		fe = append(fe, FieldError{Field: "status_refresh_every_n_ticks", Msg: "deve estar entre 1 e 1000 ticks"})
	}
	if c.HistoryRetentionDays < 1 || c.HistoryRetentionDays > 365 {
		fe = append(fe, FieldError{Field: "history_retention_days", Msg: "deve estar entre 1 e 365 dias"})
	}
	if c.Window.Width < 240 || c.Window.Width > 3840 {
		fe = append(fe, FieldError{Field: "window.width", Msg: "deve estar entre 240 e 3840 pixels"})
	}
	if c.Window.Height < 240 || c.Window.Height > 2160 {
		fe = append(fe, FieldError{Field: "window.height", Msg: "deve estar entre 240 e 2160 pixels"})
	}
	if c.Theme != "light" && c.Theme != "dark" {
		fe = append(fe, FieldError{Field: "theme", Msg: "deve ser light ou dark"})
	}
	if len(fe) > 0 {
		return &ValidationError{Errors: fe}
	}
	return nil
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
	if c.Theme != "light" && c.Theme != "dark" {
		c.Theme = d.Theme
	}
	return nil
}
