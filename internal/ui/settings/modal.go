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
	vw, vh := c.Size()
	bar := c.TitleBar()
	topInset, bottomInset, sideInset := settingsModalInsets(vw, vh, bar)
	w, h := settingsModalSizeFor(vw, vh, bar)
	ui.DialogBase(c.Key("settings-modal"), &open, func(backdrop, panel ui.Element) {
		ui.ShortcutModalScope(panel, "settings")
		if topInset > design.Sp(4) {
			backdrop.Padding(topInset, sideInset, bottomInset, sideInset)
		}
		mask := design.TW.Black
		progress := chrome.Reveal(c, backdrop, design.DurationPanel)
		chrome.RevealMask(backdrop, mask.Alpha(0.32), progress)
		panel.Width(w).Height(h).MaxWidthPercent(100).MaxHeightPercent(100).
			Role(ui.RoleDialog).Label(i18n.T(i18n.KeySettingsTitle)).
			Radius(design.RadiusXL2).Clip().Opacity(progress).
			Border(design.BorderWidth, t.Border).TextColor(t.PopoverForeground)
		if t.Glass || chrome.BackdropActive() {
			panel.Draw(chrome.SettingsPanelGlass(t.Popover, t.Background, mask.Alpha(0.32*progress), design.RadiusXL2))
			rim := design.TW.White.Alpha(0.70)
			if t.IsDark() {
				rim = design.TW.White.Alpha(0.14)
			}
			panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
				inset := design.BorderWidth
				inner := ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}
				if inner.W > 0 && inner.H > 0 {
					p.Stroke(inner, rim, max(0, design.RadiusXL2-inset), design.BorderWidth)
				}
			})
			if t.IsDark() {
				panel.Shadow(0, 12, 32, 0, design.TW.Black.Alpha(0.36)).
					Shadow(0, 3, 10, 0, design.TW.Black.Alpha(0.18))
			} else {
				panel.Shadow(0, 12, 32, 0, design.TW.Black.Alpha(0.12)).
					Shadow(0, 3, 10, 0, design.TW.Black.Alpha(0.06))
			}
		} else {
			panel.Background(t.Popover)
			for i := len(design.ShadowLg) - 1; i >= 0; i-- {
				l := design.ShadowLg[i]
				panel.Shadow(l.X, l.Y, l.Blur, l.Spread, design.TW.Black.Alpha(0.05))
			}
		}
		panel.Children(func() { View(c, m) })
	})
	if !open && m.OnClose != nil {
		ClearPreferenceErrorNotices(c)
		m.OnClose()
	}
}

// settingsModalInsets computes the viewport insets for the settings dialog.
// When the window carries overlaid title-bar controls (such as Windows - □ ×)
// and centering the modal would place its upper corners inside the caption
// band, the top inset rises below the window controls so the modal's close
// button never collides with them.
func settingsModalInsets(w, h float32, bar ui.TitleBar) (top, bottom, side float32) {
	base := design.Sp(4)
	top, bottom, side = base, base, base
	safeTop := settingsTitleBarTopInset(bar)
	if safeTop <= base {
		return top, bottom, side
	}
	baseW := max(0, min(design.Sp(220), w-2*base))
	baseH := max(0, min(design.Sp(200), h-2*base))
	topGap := (h - baseH) / 2
	horizGap := (w - baseW) / 2
	nearRight := bar.Right > 0 && horizGap < bar.Right+base
	nearLeft := bar.Left > 0 && horizGap < bar.Left+base
	if topGap < safeTop && (nearRight || nearLeft || (bar.Right == 0 && bar.Left == 0)) {
		top = safeTop
	}
	return top, bottom, side
}

// settingsModalSizeFor is the panel's size accounting for overlaid title-bar
// controls.
func settingsModalSizeFor(w, h float32, bar ui.TitleBar) (float32, float32) {
	top, bottom, side := settingsModalInsets(w, h, bar)
	return max(0, min(design.Sp(220), w-2*side)), max(0, min(design.Sp(200), h-top-bottom))
}

// settingsModalSize is the panel's size: at most 880×800, keeping coss's
// p-4 viewport inset on every side.
func settingsModalSize(w, h float32) (float32, float32) {
	return settingsModalSizeFor(w, h, ui.TitleBar{})
}
