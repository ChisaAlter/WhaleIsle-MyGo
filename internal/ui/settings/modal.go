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
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		maskAlpha := float32(.24)
		if resolved.Scheme == theme.SchemeDark {
			maskAlpha = .5
		}
		backdrop.Background(ui.RGBA(0, 0, 0, maskAlpha))
		panel.Width(w).Height(h).MaxWidthPercent(100).MaxHeightPercent(100).Background(chrome.Color(colors.BG)).Radius(28).Clip().Children(func() { View(c, m) })
	})
	if !open && m.OnClose != nil {
		m.OnClose()
	}
}
func settingsModalSize(w, h float32) (float32, float32) {
	return max(0, min(float32(800), w-48)), max(0, min(float32(800), h-48))
}
