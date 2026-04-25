package notifier

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus/v5"

	"github.com/meopedevts/revu/assets"
	"github.com/meopedevts/revu/internal/store"
)

// Notifier sends desktop notifications for tracked PRs. MVP only exposes
// Notify — click actions, replace semantics, and closed-signal handling are
// out of scope (see SPEC §12). SetEnabled / SetExpireTimeout are the hooks
// used by the config watcher to reconfigure mid-flight.
type Notifier interface {
	Notify(pr store.PRRecord) error
	SetEnabled(enabled bool)
	SetExpireTimeout(d time.Duration)
	Close() error
}

// dbusNotifier publishes to org.freedesktop.Notifications via D-Bus, with
// the transient=true hint so daemons like swaync/mako/dunst skip the history
// pane. The tray remains the source of truth for past PRs.
type dbusNotifier struct {
	conn     *dbus.Conn
	iconPath string

	mu            sync.RWMutex
	expireTimeout time.Duration
	enabled       bool

	closeOnce sync.Once
}

// Option customizes the notifier.
type Option func(*dbusNotifier)

// WithExpireTimeout sets the notification visibility. Zero means "server
// decides"; negative values are clamped to zero.
func WithExpireTimeout(d time.Duration) Option {
	return func(n *dbusNotifier) {
		if d < 0 {
			d = 0
		}
		n.expireTimeout = d
	}
}

// New opens a session bus connection and extracts the embedded idle icon to
// a tempdir so D-Bus can reference it by absolute path.
func New(opts ...Option) (Notifier, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbus session bus: %w", err)
	}
	iconPath, err := extractIcon(assets.NotifierIcon, "revu-notifier.png")
	if err != nil {
		return nil, fmt.Errorf("extract icon: %w", err)
	}
	n := &dbusNotifier{
		conn:          conn,
		iconPath:      iconPath,
		expireTimeout: 5 * time.Second,
		enabled:       true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

// SetEnabled toggles notification delivery at runtime. When disabled,
// Notify becomes a silent no-op. Safe for concurrent calls.
func (n *dbusNotifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	n.enabled = enabled
	n.mu.Unlock()
}

// SetExpireTimeout adjusts the notification visibility window at runtime.
// Non-positive values mean "server decides".
func (n *dbusNotifier) SetExpireTimeout(d time.Duration) {
	if d < 0 {
		d = 0
	}
	n.mu.Lock()
	n.expireTimeout = d
	n.mu.Unlock()
}

func (n *dbusNotifier) Notify(pr store.PRRecord) error {
	if n == nil || n.conn == nil {
		return errors.New("notifier not initialized")
	}
	n.mu.RLock()
	enabled := n.enabled
	timeout := n.expireTimeout
	n.mu.RUnlock()
	if !enabled {
		return nil
	}
	note := notify.Notification{
		AppName:       "revu",
		AppIcon:       n.iconPath,
		Summary:       fmt.Sprintf("Review solicitado · %s", pr.Repo),
		Body:          FormatBody(pr),
		ExpireTimeout: timeout,
		Hints: map[string]dbus.Variant{
			"transient": dbus.MakeVariant(true),
		},
	}
	if _, err := notify.SendNotification(n.conn, note); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}

func (n *dbusNotifier) Close() error {
	n.closeOnce.Do(func() {
		// SessionBus is shared across the process; drop our reference without
		// closing the underlying connection.
		n.conn = nil
	})
	return nil
}

// FormatBody builds the notification body per SPEC §5.4. Exported so tests
// can assert the layout independently of D-Bus.
func FormatBody(pr store.PRRecord) string {
	return fmt.Sprintf("PR #%d — %s\npor @%s · +%d -%d",
		pr.Number, pr.Title, pr.Author, pr.Additions, pr.Deletions)
}

// extractIcon writes the embedded icon bytes to a stable tempfile and
// returns its absolute path. If the file already exists with matching size
// (e.g. previous run), it is reused.
func extractIcon(data []byte, name string) (string, error) {
	dir := filepath.Join(os.TempDir(), "revu")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("mkdir tmp: %w", err)
	}
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(data)) {
		return path, nil
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write icon: %w", err)
	}
	return path, nil
}
