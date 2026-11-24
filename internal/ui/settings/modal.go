package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// ModalView keeps the workspace tree mounted. DialogBase owns input isolation,
// Tab focus containment and restoring the invoking control when dismissed.
// All dismissal paths use the existing OnClose, including its draft policy.
//
// The panel is coss's dialog popup at settings size: popover surface, 1 DIP
// border, the 2xl radius, shadow-lg/5 over a black/32 backdrop, fading and
// scaling in along the dialog's own motion. With the optional glass
// appearance on, the popover token is translucent and the panel samples the
// window behind it.
func ModalView(c *ui.Context, m Model) {
	if m.Store == nil && m.Desktop == nil && m.SectionContent == nil {
		return
	}
	open := true
	t := tk(c)
	w, h := c.Size()
	w, h = settingsModalSize(w, h)
	ui.DialogBase(c.Key("settings-modal"), &open, func(backdrop, panel ui.Element) {
		ui.ShortcutModalScope(panel, "settings")
		mask := design.TW.Black
		progress := chrome.Reveal(c, backdrop, design.DurationPanel)
		chrome.RevealMask(backdrop, mask.Alpha(0.32), progress)
		panel.Width(w).Height(h).MaxWidthPercent(100).MaxHeightPercent(100).
			Role(ui.RoleDialog).Label(i18n.T(i18n.KeySettingsTitle)).
			Radius(design.RadiusXL2).Clip().Opacity(progress).
			Border(design.BorderWidth, t.Border).TextColor(t.PopoverForeground)
		if t.Glass {
			panel.Draw(chrome.SettingsPanelGlass(t.Popover, t.Background, mask.Alpha(0.32*progress), design.RadiusXL2))
		} else {
			panel.Background(t.Popover)
		}
		for i := len(design.ShadowLg) - 1; i >= 0; i-- {
			l := design.ShadowLg[i]
			panel.Shadow(l.X, l.Y, l.Blur, l.Spread, design.TW.Black.Alpha(0.05))
		}
		panel.Children(func() { View(c, m) })
	})
	if !open && m.OnClose != nil {
		ClearPreferenceErrorNotices(c)
		m.OnClose()
	}
}

// settingsModalSize is the panel's size: at most 880×800, keeping coss's
// p-4 viewport inset on every side.
func settingsModalSize(w, h float32) (float32, float32) {
	inset := 2 * design.Sp(4)
	return max(0, min(design.Sp(220), w-inset)), max(0, min(design.Sp(200), h-inset))
}
