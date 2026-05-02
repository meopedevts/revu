// Package assets embeds binary resources (tray icons, app icon, etc.) into
// the revu binary via go:embed. Callers extract PNGs to a tempdir at boot
// and reference them by absolute path (required by D-Bus AppIcon and some
// tray backends). SVG sources em assets/brand/tray/ são a fonte de verdade
// — regenerar via `task icons`.
package assets

import _ "embed"

// Tray icons — 4 estados (idle / pending / attention / error) × 5 tamanhos
// (22 / 24 / 44 / 48 / 256). REV-51.
//
// Os vars sem sufixo de tamanho (TrayIdle, TrayPending, ...) apontam pro
// PNG 256×256 — fyne.io/systray expõe um único SetIcon e o backend SNI
// faz o downscale. Variantes menores ficam embedadas pra consumers que
// precisem do tamanho exato (ex.: hicolor theme spec, futuras integrações
// .desktop/notification daemons que rejeitem upscale).

//go:embed icons/tray-idle-256.png
var TrayIdle []byte

//go:embed icons/tray-idle-22.png
var TrayIdle22 []byte

//go:embed icons/tray-idle-24.png
var TrayIdle24 []byte

//go:embed icons/tray-idle-44.png
var TrayIdle44 []byte

//go:embed icons/tray-idle-48.png
var TrayIdle48 []byte

//go:embed icons/tray-pending-256.png
var TrayPending []byte

//go:embed icons/tray-pending-22.png
var TrayPending22 []byte

//go:embed icons/tray-pending-24.png
var TrayPending24 []byte

//go:embed icons/tray-pending-44.png
var TrayPending44 []byte

//go:embed icons/tray-pending-48.png
var TrayPending48 []byte

//go:embed icons/tray-attention-256.png
var TrayAttention []byte

//go:embed icons/tray-attention-22.png
var TrayAttention22 []byte

//go:embed icons/tray-attention-24.png
var TrayAttention24 []byte

//go:embed icons/tray-attention-44.png
var TrayAttention44 []byte

//go:embed icons/tray-attention-48.png
var TrayAttention48 []byte

//go:embed icons/tray-error-256.png
var TrayError []byte

//go:embed icons/tray-error-22.png
var TrayError22 []byte

//go:embed icons/tray-error-24.png
var TrayError24 []byte

//go:embed icons/tray-error-44.png
var TrayError44 []byte

//go:embed icons/tray-error-48.png
var TrayError48 []byte

// NotifierIcon é o ícone default ao lado das notificações D-Bus. SPEC §5.4
// manda AppIcon ser path absoluto — caller extrai esses bytes pra tempfile
// no boot. Usa o variant pending (acento brand) pra reconhecimento visual.
//
//go:embed icons/tray-pending-256.png
var NotifierIcon []byte

// WindowIcon é o bitmap 256×256 usado na barra de título Wails / alt-tab /
// entrada de taskbar.
//
//go:embed icons/tray-pending-256.png
var WindowIcon []byte
