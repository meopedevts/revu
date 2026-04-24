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
// out of scope (see SPEC §12).
type Notifier interface {
	Notify(pr store.PRRecord) error
	Close() error
}

// dbusNotifier publishes to org.freedesktop.Notifications via D-Bus, with
// the transient=true hint so daemons like swaync/mako/dunst skip the history
// pane. The tray remains the source of truth for past PRs.
type dbusNotifier struct {
	conn          *dbus.Conn
	iconPath      string
	expireTimeout time.Duration

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
	iconPath, err := extractIcon(assets.IconIdle, "tray-idle.png")
	if err != nil {
		return nil, fmt.Errorf("extract icon: %w", err)
	}
	n := &dbusNotifier{
		conn:          conn,
		iconPath:      iconPath,
		expireTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n, nil
}

func (n *dbusNotifier) Notify(pr store.PRRecord) error {
	if n == nil || n.conn == nil {
		return errors.New("notifier not initialized")
	}
	note := notify.Notification{
		AppName:       "revu",
		AppIcon:       n.iconPath,
		Summary:       fmt.Sprintf("Review solicitado · %s", pr.Repo),
		Body:          FormatBody(pr),
		ExpireTimeout: n.expireTimeout,
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir tmp: %w", err)
	}
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(data)) {
		return path, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write icon: %w", err)
	}
	return path, nil
}
