package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
)

// composerSelect uses coss.Select while preserving the 220 DIP cap for the
// closed trigger.
func composerSelect(c *ui.Context, selected *string, options []string) ui.Element {
	opts := make([]coss.SelectOption, len(options))
	for i, option := range options {
		opts[i] = coss.SelectOption{Value: option, Label: option}
	}
	tr := coss.Select(c, selected, opts, coss.SelectOptions{
		Size:  coss.InputSM,
		Label: *selected,
	}).MaxWidth(220).Tooltip(*selected)
	return tr
}
