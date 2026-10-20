package chrome

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// Color parses a CSS color token from the theme package (#hex or rgb/rgba).
func Color(s string) ui.Color { return theme.Color(s) }

// TitleBarColorsFrom maps a theme palette onto title-bar fills.
func TitleBarColorsFrom(c theme.Colors) TitleBarColors {
	return TitleBarColors{
		Background: theme.Color(c.Titlebar),
		Foreground: theme.Color(c.FG),
		Muted:      theme.Color(c.Muted),
		Border:     theme.Color(c.Border),
	}
}
