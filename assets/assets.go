// Package assets embeds binary resources (tray icons, etc.) into the revu
// binary via go:embed. Callers extract them to a tempdir at boot and
// reference by absolute path (required by D-Bus AppIcon and some tray
// backends).
package assets

import _ "embed"

//go:embed icons/tray-idle.png
var IconIdle []byte
