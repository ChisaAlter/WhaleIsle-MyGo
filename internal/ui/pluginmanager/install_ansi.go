package pluginmanager

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/ansi"
)

// Installation output keeps its existing theme mapping while sharing the inert
// cursor/SGR replay with conversation terminal cards.
func installANSI(raw string, theme *ui.Theme) [][]ui.Span {
	return ansi.Parse(raw, ansi.Palette{
		Primary: theme.Text, Tertiary: theme.TextMuted, Background: theme.Surface, DimBase: theme.TextMuted,
		Error: theme.Danger, ErrorSecondary: theme.Danger,
		Success: theme.Success, SuccessSecondary: theme.Success,
		Warning: theme.Warning, WarningSecondary: theme.Warning, Business: theme.Accent,
	})
}
