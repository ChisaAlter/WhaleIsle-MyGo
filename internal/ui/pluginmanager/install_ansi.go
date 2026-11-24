package pluginmanager

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/widgets/ansi"
)

// installANSI replays installation output's cursor/SGR codes, as the
// conversation terminal cards do, in the window's design tokens.
func installANSI(raw string, t *design.Tokens) [][]ui.Span {
	return ansi.Parse(raw, ansi.Palette{
		Primary: t.Foreground, Tertiary: t.MutedForeground, Background: t.Muted, DimBase: t.MutedForeground,
		Error: t.DestructiveForeground, ErrorSecondary: t.DestructiveForeground,
		Success: t.SuccessForeground, SuccessSecondary: t.SuccessForeground,
		Warning: t.WarningForeground, WarningSecondary: t.WarningForeground, Business: t.Primary,
	})
}
