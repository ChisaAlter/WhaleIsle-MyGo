package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
		chrome.NoteBackdropOverlayActive()
		ui.ShortcutModalScope(panel, "settings")
		if topInset > design.Sp(4) {
			backdrop.Padding(topInset, sideInset, bottomInset, sideInset)
		}
		glassActive := settingsGlassActive(t)
		scrim := modalScrim(t.IsDark(), glassActive)
		borderWidth := design.BorderWidth
		borderColor := t.Border
		if glassActive {
			borderWidth = design.BorderWidth / 2
			borderColor = chrome.GlassOuterBorder(t.IsDark())
		}
		dlgRadius := theme.ContextRadii(c).Dialog
		progress := chrome.Reveal(c, backdrop, design.DurationPanel)
		chrome.ApplyDialogBackdropMask(c, backdrop, scrim, progress)
		panel.Width(w).Height(h).MaxWidthPercent(100).MaxHeightPercent(100).
			Role(ui.RoleDialog).Label(i18n.T(i18n.KeySettingsTitle)).
			Radius(dlgRadius).Clip().
			Border(borderWidth, borderColor).TextColor(t.PopoverForeground)
		chrome.RevealDialogPanel(c, panel, progress)
		if glassActive {
			// Single GPU OpEffect modal blur on the outer frame (R=24/26,
			// high vibrancy saturation); left rail and right options pane
			// paint their disjoint layered translucent fills over it.
			panel.Material(chrome.ModalGlassMaterial())
			panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
				chrome.DrawGlassRim(p, r, dlgRadius)
			})
			chrome.ModalElevation(panel, t.IsDark())
		} else {
			panel.Background(t.Popover)
			for i := len(design.ShadowLg) - 1; i >= 0; i-- {
				l := design.ShadowLg[i]
				panel.Shadow(l.X, l.Y, l.Blur, l.Spread, design.TW.Black.Alpha(0.05))
			}
		}
		panel.Children(func() {
			modalW := settingsModalWidthHintState(c)
			prevW := *modalW
			*modalW = w
			defer func() { *modalW = prevW }()
			View(c, m)
		})
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
