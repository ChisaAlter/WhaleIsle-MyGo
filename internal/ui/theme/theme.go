// Package theme provides shell design tokens inspired by the Electron
// harness --dsw-* CSS variables (see reference/electron mobile tokens and
// src/shared/themes.js FAMILY_SEEDS).
//
// Config stores a theme id. A profile with no stored theme starts on
// FirstRunThemeID, the dark palette the product ships; Resolve maps that id to
// a Theme with light and dark palettes. BackgroundColor returns a value
// suitable for mygo.WindowOptions.BackgroundColor, including the
// light-dark(#light, #dark) form when the theme follows the system scheme.
// Optional window wallpaper uses config goWallpaperPath; see wallpaper.go
// (mygo v0.3.5 has no WindowOptions.BackgroundImage — the shell paints
// wallpaper in the root view when the path is set).
package theme

import "strings"

// Well-known theme ids accepted by Resolve.
const (
	// DefaultID is the deepseek family id, which Resolve also uses for unknown
	// ids. internal/config Defaults stores FirstRunThemeID instead.
	DefaultID = "deepseek"
	// IDSystem follows the OS light/dark preference via light-dark() strings.
	IDSystem = "system"
	// IDDeepseekDark forces the deepseek dark palette.
	IDDeepseekDark = "deepseek.dark"
	// IDDeepseekLight forces the deepseek light palette.
	IDDeepseekLight = "deepseek.light"
	// FirstRunThemeID is the theme of a fresh profile (internal/config
	// DefaultTheme): the deepseek family, which follows the OS light/dark
	// preference like the harness install default. A stored theme id always
	// wins.
	FirstRunThemeID = DefaultID
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
	BG              string // --dsw-alias-bg-base
	FG              string // --dsw-alias-label-primary
	Muted           string // --dsw-alias-label-tertiary
	Caption         string // --dsw-alias-label-caption / composer placeholder
	Accent          string // --dsw-alias-state-business-primary (deepseek-500 light, rgb(103, 158, 254) dark)
	Border          string // --dsw-alias-border-l2
	Field           string // raised surface: --dsw-alias-bg-layer-1 (dark #232324; light reads module-platform #f5f6f7 where the token is #ffffff)
	Sidebar         string // --dsw-specific-sidebar-fill
	Titlebar        string // titlebar / chrome strip
	Hover           string // --dsw-alias-interactive-bg-hover
	Pressed         string // --dsw-alias-interactive-bg-active
	ButtonFG        string // --dsw-alias-label-primary-foreground / on-accent text
	Bubble          string // --dsw-specific-bubble
	InputMajor      string // --dsw-specific-input-major (glass-opacity 80%)
	ButtonInfoFill  string // --dsw-alias-button-info-fill
	ButtonInfoHover string // --dsw-alias-button-info-hover (also active)
	SelectorFill    string // --dsw-specific-selector
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
	BG:      "#ffffff",
	FG:      "#0f1115",
	Muted:   "#81858c",
	Caption: "#adb2b8",
	Accent:  "#4176e6",
	Border:  "rgba(0, 0, 0, 0.10)",
	// Raised surface. --dsw-alias-bg-layer-1 is rgb(255, 255, 255) here
	// (mixWallpaperSurfaces' light "raised" fallback, neutral-bluish-00); this
	// value instead reads --dsw-alias-bg-module-platform (rgb(245, 246, 247)),
	// which keeps light-mode wells visible against the white base. Matching the
	// sheet would need #ffffff and is left to a wave that can retune the surfaces
	// sharing the token (see the Wave 9 token report).
	Field:           "#f5f6f7",
	Sidebar:         "#f9fafb",
	Titlebar:        "#f9fafb",
	Hover:           "rgba(38, 49, 72, 0.06)",
	Pressed:         "rgba(38, 49, 72, 0.1)",
	ButtonFG:        "#ffffff",
	Bubble:          "rgb(237, 243, 254)",
	InputMajor:      "rgba(255, 255, 255, 0.8)",
	ButtonInfoFill:  "rgb(65, 118, 230)",
	ButtonInfoHover: "rgb(122, 170, 255)",
	SelectorFill:    "rgb(245, 246, 247)",
}

// deepseekDark mirrors official dsh web dark tokens.
var deepseekDark = Colors{
	BG:      "#151517",
	FG:      "#f9fafb",
	Muted:   "#adb2b8",
	Caption: "#81858c",
	// --dsw-alias-state-business-primary: the harness stylesheet resolves it to
	// --dsw-static-deepseek-400 (rgb(122, 170, 255)) in the dark half. The
	// app-level dsh-webui-tokens.css copy says rgb(103, 158, 254); the cascaded
	// value is the harness one, which the shell follows.
	Accent: "#7aaaff",
	Border: "rgba(255, 255, 255, 0.12)",
	// Raised surface, --dsw-alias-bg-layer-1: rgb(35, 35, 36), which is also
	// mixWallpaperSurfaces' dark "raised" fallback (neutral-bluish-875). The light
	// half reads --dsw-alias-bg-module-platform instead (see deepseekLight.Field).
	Field:    "#232324",
	Sidebar:  "#1b1b1c",
	Titlebar: "#232324",
	Hover:    "rgba(255, 255, 255, 0.08)",
	Pressed:  "rgba(255, 255, 255, 0.14)",
	ButtonFG: "#0f1115",
	Bubble:   "rgb(44, 44, 46)",
	// --dsw-specific-input-major: color-mix(rgb(44, 44, 46) 80%, transparent).
	InputMajor: "rgba(44, 44, 46, 0.8)",
	// --dsw-alias-button-info-fill follows state-business-primary here
	// (--dsw-static-deepseek-400); its hover is deepseek-500.
	ButtonInfoFill:  "rgb(122, 170, 255)",
	ButtonInfoHover: "rgb(65, 118, 230)",
	SelectorFill:    "rgb(53, 54, 56)",
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
	base = selectedPalettes(base)
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
