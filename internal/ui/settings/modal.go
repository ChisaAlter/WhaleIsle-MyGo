package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// ModalView keeps the workspace tree mounted. DialogBase owns input isolation,
// Tab focus containment and restoring the invoking control when dismissed.
// All dismissal paths use the existing OnClose, including its draft policy.
func ModalView(c *ui.Context, m Model) {
	if m.Store == nil && m.Desktop == nil && m.SectionContent == nil {
		return
	}
	open := true
	resolved := theme.ResolveWithSystem(m.ThemeID, m.SystemDark)
	colors := resolved.Active(m.SystemDark)
	w, h := c.Size()
	w, h = settingsModalSize(w, h)
	ui.DialogBase(c.Key("settings-modal"), &open, func(backdrop, panel ui.Element) {
		ui.ShortcutModalScope(panel, "settings")
		dark := resolved.Scheme == theme.SchemeDark || resolved.Scheme == theme.SchemeSystem && m.SystemDark
		mask := ui.RGBA(0, 0, 0, .24)
		if dark {
			mask = ui.RGBA(0, 0, 0, .5)
		}
		progress := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
		chrome.RevealMask(backdrop, mask, progress)
		panel.Width(w).Height(h).MaxWidthPercent(100).MaxHeightPercent(100).
			Radius(28).Clip().Opacity(progress).Draw(chrome.SettingsPanelGlass(settingsPanelBase(colors, dark), chrome.Color(colors.BG), mask.Alpha(progress), 28)).
			Children(func() { View(c, m) })
		chrome.MenuElevation(panel)
		// --dsw-elevation-stroke: the harness draws a 0.5px hairline in
		// --dsw-elevation-stroke-color, which is --dsw-alias-border-l4 over the
		// light half and --dsw-alias-border-l3 over the dark one
		// (gradient-shadow-text.css). The shell carries --dsw-alias-border-l2 as
		// theme.Colors.Border, the same hairline family the rest of the chrome
		// draws, so the panel uses that. It paints after the glass, which runs
		// after the element's own border and would otherwise cover it.
		panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
			border := chrome.Color(colors.Border)
			if border.A == 0 || r.W <= 0 || r.H <= 0 {
				return
			}
			p.Stroke(r, border, 28, 0.5)
		})
	})
	if !open && m.OnClose != nil {
		ClearPreferenceErrorNotices(c)
		m.OnClose()
	}
}

// settingsPanelBase is the colour the panel's backdrop layer fills at 80%
// under blur(40px) saturate(1.5). The 0.3.13 panel reads the raised layer
// (--dsw-alias-bg-layer-1): white in the light half and rgb(35, 35, 36) in the
// dark half. 0.3.10 read the selector token, which is lighter in both halves
// and made the dark panel a visibly lighter grey. Derived families keep the
// selector token until their raised layer is measured.
func settingsPanelBase(colors theme.Colors, dark bool) ui.Color {
	switch colors.SelectorFill {
	case "rgb(245, 246, 247)":
		return ui.RGB(255, 255, 255)
	case "rgb(53, 54, 56)":
		return chrome.Color(colors.Field)
	}
	return chrome.Color(colors.SelectorFill)
}

func settingsModalSize(w, h float32) (float32, float32) {
	return max(0, min(float32(800), w-48)), max(0, min(float32(800), h-48))
}
