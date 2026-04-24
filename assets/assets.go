// Package assets embeds binary resources (tray icons, app icon, etc.) into
// the revu binary via go:embed. Callers extract PNGs to a tempdir at boot
// and reference them by absolute path (required by D-Bus AppIcon and some
// tray backends). SVG sources live alongside the PNGs as the source of
// truth — regenerate via `task icons`.
package assets

import _ "embed"

// Tray icons (32px — upscaled by Waybar / SNI consumers).

//go:embed icons/tray-idle-32.png
var TrayIdle []byte

//go:embed icons/tray-pending-32.png
var TrayPending []byte

//go:embed icons/tray-error-32.png
var TrayError []byte

// NotifierIcon is the default icon rendered next to D-Bus notifications.
// SPEC §5.4 mandates AppIcon be an absolute file path — caller extracts
// these bytes to a tempfile at boot.
//
//go:embed icons/tray-pending-64.png
var NotifierIcon []byte

// WindowIcon is the 256×256 bitmap used for the Wails title bar /
// alt-tab / taskbar entry.
//
//go:embed icons/tray-pending-256.png
var WindowIcon []byte

// IconIdle is kept as a transitional alias for code that imported the
// original placeholder. Will be removed once all call sites move to the
// explicit state names above.
var IconIdle = TrayIdle
