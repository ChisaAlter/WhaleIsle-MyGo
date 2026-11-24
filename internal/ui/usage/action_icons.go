package usage

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

var usageSettingsIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="4.2"/><circle cx="8" cy="8" r="1.8"/><path d="M8 1.8v1.5M8 12.7v1.5M1.8 8h1.5M12.7 8h1.5M3.6 3.6l1.1 1.1M11.3 11.3l1.1 1.1M12.4 3.6l-1.1 1.1M4.7 11.3l-1.1 1.1"/></svg>`))

// settingsIconButton builds a coss outline SM button with the settings gear
// glyph, because coss's icon enum does not define a gear icon.
func settingsIconButton(c *ui.Context, key any, label string) ui.Element {
	return coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonOutline,
		Size:    coss.ButtonSM,
		Label:   label,
		Key:     key,
		Content: func() {
			t := design.For(c)
			ui.Icon(c, usageSettingsIcon).Size(design.IconSM, design.IconSM).Shrink(0).TextColor(t.Foreground).Opacity(design.OpacityIcon).MarginX(-2)
			ui.Text(c, label).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).FontWeight(design.WeightMedium).TextColor(t.Foreground).SingleLine()
		},
	})
}
