package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// ConfirmOptions is the content of one in-app confirmation. The settings domain
// hosts it so every destructive surface can ask in the application's own
// chrome instead of opening an operating-system message box.
type ConfirmOptions struct {
	Title        string
	Body         string
	Detail       string
	ConfirmLabel string
	CancelLabel  string
	OtherLabel   string
	// TechnicalDetails is the optional evidence block (port of the shell update
	// dialog's technicalDetails). It stays collapsed so it never widens the
	// question, but the exact version, route and checksum stay one click away.
	TechnicalDetails string
	// TechnicalLabel captions that block; empty hides it.
	TechnicalLabel string
	// TechnicalOpen holds the disclosure state across frames; the caller owns it
	// so a closed block stays closed and an opened one stays open.
	TechnicalOpen *bool
	// Danger paints the confirm action as the destructive action.
	Danger bool
	// Width is 380 for an ordinary confirmation; update readers may request 640.
	Width float32
	// Colors is kept for callers; the dialog follows the window's coss tokens.
	Colors    theme.Colors
	OnConfirm func()
	OnCancel  func()
	OnOther   func()
}

// ConfirmDialog paints one confirmation as coss's AlertDialog and reports the
// answer: the popover panel over a black/32 backdrop, the bold title, the
// muted description, an optional technical disclosure and the muted footer
// strip with the quiet cancel (outline) and the answer (default, or
// destructive when the question is). It paints nothing when called with an
// empty title, so a caller can render it unconditionally while its own state
// decides whether a question is open.
func ConfirmDialog(c *ui.Context, opts ConfirmOptions) {
	if opts.Title == "" {
		return
	}
	open := true
	settled := false
	finish := func(callback func()) {
		if settled {
			return
		}
		settled = true
		open = false
		if callback != nil {
			callback()
		}
	}
	cancelLabel := opts.CancelLabel
	if cancelLabel == "" {
		cancelLabel = i18n.Pick("取消", "Cancel")
	}
	width := opts.Width
	if width <= 0 {
		width = design.Sp(112)
	}
	bodyDialogFrame(c, &open, width, func() {
		t := tk(c)
		ui.Column(c).FillWidth().Shrink(0).Gap(design.Sp(2)).Padding(design.Sp(6), design.Sp(6), design.Sp(3), design.Sp(6)).Children(func() {
			kText(c, opts.Title, design.TextStyle{Size: design.TextXL.Size, LineHeight: design.TextXL.Size}, design.WeightBold, t.PopoverForeground).FillWidth().Role(ui.RoleHeading)
		})
		// The panel sizes to its content, so the scroll keeps a viewport
		// budget instead of Grow: the panel's inset max height minus the
		// 56 DIP title block and the 69 DIP footer strip.
		_, vh := c.Size()
		bodyMax := max(float32(0), vh-2*design.Sp(4)-design.Sp(14)-design.Sp(17)-design.BorderWidth)
		ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).Shrink(1).MaxHeight(bodyMax).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(0, design.Sp(6), design.Sp(6), design.Sp(6)).Children(func() {
				if opts.Body != "" {
					kText(c, opts.Body, design.TextSM, design.WeightNormal, t.MutedForeground).FillWidth()
				}
				if opts.Detail != "" {
					kHint(c, opts.Detail)
				}
				if opts.TechnicalDetails != "" && opts.TechnicalLabel != "" && opts.TechnicalOpen != nil {
					coss.Collapsible(c, opts.TechnicalOpen, func(open bool) {
						glyph := coss.IconChevronRight
						if open {
							glyph = coss.IconChevronDown
						}
						ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, glyph.SVG()).Size(design.IconSM, design.IconSM).TextColor(t.MutedForeground).PassThrough()
							kText(c, opts.TechnicalLabel, design.TextXS, design.WeightMedium, t.MutedForeground)
						})
					}, func() {
						kHint(c, opts.TechnicalDetails).Selectable()
					})
				}
			})
		})
		footerBg, footerBorder := dialogFooterStyle(t)
		ui.Row(c).FillWidth().Shrink(0).Wrap().Gap(design.Sp(2)).Justify(ui.End).Padding(design.Sp(4), design.Sp(6)).
			Background(footerBg).BorderWidth(footerBorder, 0, 0, 0).BorderColor(t.Border).Children(func() {
			if coss.Button(c, cancelLabel, coss.ButtonOptions{Variant: coss.ButtonOutline}).AutoFocus().Clicked() {
				finish(opts.OnCancel)
			}
			if opts.OtherLabel != "" && coss.Button(c, opts.OtherLabel, coss.ButtonOptions{Variant: coss.ButtonGhost}).Clicked() {
				finish(opts.OnOther)
			}
			variant := coss.ButtonDefault
			if opts.Danger {
				variant = coss.ButtonDestructive
			}
			if coss.Button(c, opts.ConfirmLabel, coss.ButtonOptions{Variant: variant}).Clicked() {
				finish(opts.OnConfirm)
			}
		})
	}, ui.DialogOptions{BackdropDismissDisabled: true})
	if !open {
		finish(opts.OnCancel)
	}
}

// bodyDialogFrame is the coss dialog frame settings dialogs share: the
// black/32 backdrop and the popover panel (2xl radius, border, shadow-lg/5),
// at most width wide inside coss's 16 DIP viewport inset. body builds the
// panel's children.
func bodyDialogFrame(c *ui.Context, open *bool, width float32, body func(), options ...ui.DialogOptions) {
	var dismissal ui.DialogOptions
	if len(options) > 0 {
		dismissal = options[0]
	}
	ui.DialogBaseWithOptions(c, open, dismissal, func(backdrop, panel ui.Element) {
		t := tk(c)
		w, h := c.Size()
		inset := design.Sp(4)
		dlgRadius := theme.ContextRadii(c).Dialog
		backdrop.Column().Justify(ui.Center).AlignItems(ui.Center).Padding(inset)
		glassActive := settingsGlassActive(t)
		scrim := modalScrim(t.IsDark(), glassActive)
		borderWidth := design.BorderWidth
		borderColor := t.Border
		if glassActive {
			borderWidth = design.BorderWidth / 2
			borderColor = settingsGlassOuterBorder(t.IsDark())
		}
		progress := chrome.Reveal(c, backdrop, design.DurationPanel)
		chrome.ApplyDialogBackdropMask(c, backdrop, scrim, progress)
		panel.Width(max(0, min(width, w-2*inset))).MaxHeight(max(0, h-2*inset)).Radius(dlgRadius).Clip().
			Border(borderWidth, borderColor).TextColor(t.PopoverForeground).Role(ui.RoleDialog)
		chrome.RevealDialogPanel(c, panel, progress)
		if glassActive {
			panel.Material(chrome.MenuGlassMaterial())
			fill := settingsDialogGlassFill(t.IsDark())
			panel.Draw(func(p *ui.Painter, r ui.Rect) {
				if r.W > 0 && r.H > 0 {
					p.Fill(r, fill, dlgRadius)
				}
			})
			panel.DrawOver(drawSettingsGlassRim(t.IsDark(), dlgRadius))
			chrome.ModalElevation(panel, t.IsDark())
		} else {
			panel.Background(t.Popover)
			for i := len(design.ShadowLg) - 1; i >= 0; i-- {
				l := design.ShadowLg[i]
				panel.Shadow(l.X, l.Y, l.Blur, l.Spread, design.TW.Black.Alpha(0.05))
			}
		}
		panel.Children(body)
	})
}

func modalScrimAlpha(dark bool) float32 {
	if dark {
		return 0.48
	}
	return 0.24
}

func modalScrim(dark, glass bool) ui.Color {
	if glass {
		if dark {
			return design.TW.Black.Alpha(0.32)
		}
		return design.TW.Gray900.Alpha(0.12)
	}
	return design.TW.Black.Alpha(modalScrimAlpha(dark))
}

func dialogFooterStyle(t *design.Tokens) (ui.Color, float32) {
	if t.Glass || chrome.BackdropActive() {
		if t.IsDark() {
			return design.TW.Black.Alpha(0.14), design.BorderWidth / 2
		}
		return design.TW.Black.Alpha(0.03), design.BorderWidth / 2
	}
	return t.Muted.Alpha(0.72), design.BorderWidth
}
