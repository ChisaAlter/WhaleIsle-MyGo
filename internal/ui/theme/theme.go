// Package theme provides shell design tokens inspired by the Electron
// harness --dsw-* CSS variables (see reference/electron mobile tokens and
// src/shared/themes.js FAMILY_SEEDS).
//
// Config stores a theme id (default "deepseek"). Resolve maps that id to a
// Theme with light and dark palettes. BackgroundColor returns a value
// suitable for mygo.WindowOptions.BackgroundColor, including the
// light-dark(#light, #dark) form when the theme follows the system scheme.
// Optional window wallpaper uses config goWallpaperPath; see wallpaper.go
// (mygo v0.3.5 has no WindowOptions.BackgroundImage — the shell paints
// wallpaper in the root view when the path is set).
package theme

import "strings"

// Well-known theme ids accepted by Resolve.
const (
	// DefaultID is the config.json default (internal/config Defaults).
	DefaultID = "deepseek"
	// IDSystem follows the OS light/dark preference via light-dark() strings.
	IDSystem = "system"
	// IDDeepseekDark forces the deepseek dark palette.
	IDDeepseekDark = "deepseek.dark"
	// IDDeepseekLight forces the deepseek light palette.
	IDDeepseekLight = "deepseek.light"
)

// Scheme values for Theme.Scheme.
const (
	SchemeLight  = "light"
	SchemeDark   = "dark"
	SchemeSystem = "system"
)

// Colors holds semantic UI colors as CSS color strings (hex or rgba).
// Field names mirror the Electron chrome tokens (bg/fg/accent/…) plus
// shell chrome surfaces (sidebar, titlebar).
type Colors struct {
	BG       string // --dsw-alias-bg-base
	FG       string // --dsw-alias-label-primary
	Muted    string // --dsw-alias-label-tertiary
	Accent   string // --dsw-alias-state-business-primary / deepseek-500
	Border   string // --dsw-alias-border-l2
	Field    string // --dsw-alias-bg-layer-1 / module-platform
	Sidebar  string // --dsw-specific-sidebar-fill
	Titlebar string // titlebar / chrome strip
	Hover    string // --dsw-alias-interactive-bg-hover
	ButtonFG string // contrast text on accent / primary button
}

// Theme is a resolved family with both palettes and a scheme preference.
type Theme struct {
	ID     string // family id after resolve (never empty)
	Name   string
	Scheme string // light | dark | system
	Light  Colors
	Dark   Colors
}

// deepseekLight mirrors official dsh web light tokens (dsh-webui-tokens.css).
var deepseekLight = Colors{
	BG:       "#ffffff",
	FG:       "#0f1115",
	Muted:    "#81858c",
	Accent:   "#4176e6",
	Border:   "rgba(0, 0, 0, 0.10)",
	Field:    "#f5f6f7",
	Sidebar:  "#f9fafb",
	Titlebar: "#f9fafb",
	Hover:    "rgba(38, 49, 72, 0.06)",
	ButtonFG: "#ffffff",
}

// deepseekDark mirrors official dsh web dark tokens.
var deepseekDark = Colors{
	BG:       "#151517",
	FG:       "#f9fafb",
	Muted:    "#adb2b8",
	Accent:   "#679efe",
	Border:   "rgba(255, 255, 255, 0.12)",
	Field:    "#232324",
	Sidebar:  "#1b1b1c",
	Titlebar: "#232324",
	Hover:    "rgba(255, 255, 255, 0.08)",
	ButtonFG: "#0f1115",
}

// Default returns the deepseek family with system scheme (light-dark window bg).
func Default() Theme {
	return Resolve(DefaultID)
}

// Resolve maps a config theme id to a Theme. Unknown or empty ids fall back
// to deepseek. Accepted ids:
//
//	deepseek          – family deepseek, scheme system (default)
//	system            – same as deepseek / system
//	deepseek.dark     – family deepseek, scheme dark
//	deepseek.light    – family deepseek, scheme light
//	dark / light      – deepseek family pinned to that scheme
func Resolve(id string) Theme {
	id = strings.TrimSpace(strings.ToLower(id))
	base := Theme{
		ID:     DefaultID,
		Name:   "DeepSeek",
		Scheme: SchemeSystem,
		Light:  deepseekLight,
		Dark:   deepseekDark,
	}
	switch id {
	case "", DefaultID, IDSystem:
		return base
	case IDDeepseekDark, SchemeDark:
		base.Scheme = SchemeDark
		return base
	case IDDeepseekLight, SchemeLight:
		base.Scheme = SchemeLight
		return base
	default:
		// Unknown family: still deepseek tokens so the shell never has a
		// blank palette; keep system scheme so the window tracks the OS.
		return base
	}
}

// ResolveWithSystem pins scheme to light or dark when the theme follows the
// system preference. Fixed light/dark themes ignore systemDark.
func ResolveWithSystem(id string, systemDark bool) Theme {
	t := Resolve(id)
	if t.Scheme != SchemeSystem {
		return t
	}
	if systemDark {
		t.Scheme = SchemeDark
	} else {
		t.Scheme = SchemeLight
	}
	return t
}

// BackgroundColor returns a CSS color for mygo.WindowOptions.BackgroundColor.
// System (and default deepseek) themes use light-dark so the empty window
// matches the desktop appearance before the first paint.
func (t Theme) BackgroundColor() string {
	switch t.Scheme {
	case SchemeDark:
		return t.Dark.BG
	case SchemeLight:
		return t.Light.BG
	default:
		return LightDark(t.Light.BG, t.Dark.BG)
	}
}

// Active returns the palette for the theme's scheme. For system scheme it
// uses systemDark to pick light or dark colors.
func (t Theme) Active(systemDark bool) Colors {
	switch t.Scheme {
	case SchemeDark:
		return t.Dark
	case SchemeLight:
		return t.Light
	default:
		if systemDark {
			return t.Dark
		}
		return t.Light
	}
}

// LightDark formats a CSS light-dark() pair accepted by MyGo color fields.
func LightDark(light, dark string) string {
	return "light-dark(" + light + ", " + dark + ")"
}

// KnownIDs lists the theme ids Resolve treats as first-class (not unknown).
func KnownIDs() []string {
	return []string{DefaultID, IDSystem, IDDeepseekDark, IDDeepseekLight, SchemeDark, SchemeLight}
}
