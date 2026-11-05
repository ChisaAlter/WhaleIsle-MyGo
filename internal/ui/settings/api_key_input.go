package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

var draftEyeIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M1.5 8s2.3-4.3 6.5-4.3 6.5 4.3 6.5 4.3-2.3 4.3-6.5 4.3S1.5 8 1.5 8Z"/><circle cx="8" cy="8" r="2"/></svg>`))
var modelTrashIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"><path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5"/></svg>`))
var modelExpandIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"><path d="m4 6 4 4 4-4"/></svg>`))

// Reveal belongs to this exact draft value. Saved credentials remain write-only.
func apiKeyInput(c *ui.Context, key string, value *string, placeholder string, disabled bool) bool {
	row := ui.Row(c.Key("api-key/" + key)).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center)
	shown := ui.Local(row, "revealed-value", func() string { return "" })
	changed := false
	row.Children(func() {
		input := ui.TextInput(c, value).Label("API Key").Grow(1).MinWidth(0).Height(36).Radius(8).Border(.5, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).Placeholder(placeholder).Disabled(disabled)
		if *value == "" || *shown != *value {
			input.Password()
		}
		if input.Changed() {
			*shown = ""
			changed = true
		}
		if *value != "" {
			label := providerCopy("显示密钥", "Show key")
			if *shown == *value {
				label = providerCopy("隐藏密钥", "Hide key")
			}
			toggle := ui.Button(c, "").Label(label).Tooltip(label).Size(32, 32).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(disabled)
			toggle.Children(func() { ui.Icon(c, draftEyeIcon).Size(16, 16).TextColor(c.Theme().TextMuted).PassThrough() })
			if toggle.Clicked() {
				if *shown == *value {
					*shown = ""
				} else {
					*shown = *value
				}
			}
		}
	})
	return changed
}
