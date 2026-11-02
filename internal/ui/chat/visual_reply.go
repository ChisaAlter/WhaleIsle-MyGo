package chat

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// VisualReplyView sits outside ToolCardView's disclosure. HTML is always plain
// native text until interaction mounts a lifetime-bound AppContainer child.
// The chat process never loads the HTML or executes its scripts.
func VisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, fg, muted, border, field ui.Color) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	id := p.File.AttachmentID
	state := m.VisualReply
	active := state.SessionID == m.Snapshot.SelectedID && state.AttachmentID == id
	ui.Column(c.Key("published-html-"+id)).FillWidth().Gap(6).Padding(10).Background(field).Border(1, border).Radius(6).Children(func() {
		ui.Text(c, p.Title).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyChatVisualPublished)).FontSize(11).TextColor(muted)
		ui.Row(c).Gap(6).Children(func() {
			if m.OnInteractVisualReply != nil && m.OnInlineVisualReply == nil {
				if stdButton(c, palette, i18n.T(i18n.KeyChatVisualInteract), buttonPrimary, active && (state.Loading || state.InteractiveLoading || state.InteractiveActive)).
					Tooltip(i18n.T(i18n.KeyChatVisualIsolatedTooltip)).Clicked() {
					m.OnInteractVisualReply(id)
				}
			}
			if active && (state.InteractiveLoading || state.InteractiveActive) && m.OnStopInteractiveVisualReply != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyChatVisualStop), buttonDanger, false).Clicked() {
					m.OnStopInteractiveVisualReply()
				}
			}
			if stdButton(c, palette, i18n.T(i18n.KeyChatVisualSafePreview), buttonGhost, m.OnPreviewVisualReply == nil || (active && (state.Loading || state.PreviewLoading))).Clicked() {
				m.OnPreviewVisualReply(id)
			}
			if stdButton(c, palette, i18n.T(i18n.KeyChatVisualSource), buttonGhost, m.OnReadVisualReply == nil || (active && state.Loading)).Clicked() {
				m.OnReadVisualReply(id)
			}
			if active && state.Ready {
				if stdButton(c, palette, i18n.T(i18n.KeyChatVisualSaveAs), buttonGhost, m.OnSaveVisualReply == nil).Clicked() {
					m.OnSaveVisualReply(id)
				}
			}
			if active && m.OnCloseVisualReply != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyChatImageClose), buttonGhost, false).Clicked() {
					m.OnCloseVisualReply()
				}
			}
		})
		if m.OnInlineVisualReply != nil {
			inlineVisualReplyView(c, p, m, m.OnInlineVisualReply(id), fg, muted, field)
		}
		if !active {
			return
		}
		if state.Loading {
			ui.Text(c, i18n.T(i18n.KeyChatVisualReading)).TextColor(muted)
		}
		if state.Error != "" {
			ui.Text(c, state.Error).TextColor(muted)
		}
		if state.InteractiveLoading {
			ui.Text(c, i18n.T(i18n.KeyChatVisualPreparing)).TextColor(muted)
		}
		if state.InteractiveActive {
			if state.Surface != nil {
				c.After(100 * time.Millisecond)
				expanded := state.InteractiveExpanded
				if stdButton(c, palette, i18n.T(i18n.KeyChatVisualExpand), buttonGhost, m.HistoryStore == nil).Clicked() {
					m.HistoryStore.ToggleVisualReplyExpanded()
				}
				if !expanded {
					visualReplySurfaceSeat(c, state.Surface, m.ThemeID, m.SystemDark, 360, m.Inspector)
				}
				if expanded {
					open := true
					ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
						chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 960, 12).Children(func() {
							ui.Row(c).FillWidth().Children(func() {
								ui.Text(c, p.Title).Grow(1).TextColor(fg)
								if stdButton(c, palette, i18n.T(i18n.KeyChatRowCollapse), buttonGhost, false).Clicked() {
									open = false
								}
							})
							visualReplySurfaceSeat(c, state.Surface, m.ThemeID, m.SystemDark, 600, nil)
						})
					})
					if !open && m.HistoryStore != nil {
						m.HistoryStore.ToggleVisualReplyExpanded()
					}
				}
			}
		}
		if state.InteractiveError != "" {
			ui.Text(c, i18n.T(i18n.KeyChatVisualNoPage)+": "+state.InteractiveError).TextColor(muted)
		}
		if state.PreviewLoading {
			ui.Text(c, i18n.T(i18n.KeyChatVisualChecking)).TextColor(muted)
		}
		if state.PreviewError != "" {
			ui.Text(c, i18n.T(i18n.KeyChatVisualNoPreview)+": "+state.PreviewError).TextColor(muted)
		}
		if m.VisualReplyBitmap != nil && state.PreviewID != 0 {
			ui.Image(c, m.VisualReplyBitmap).FillWidth().AspectRatio(float32(state.PreviewWidth) / float32(state.PreviewCapturedHeight)).Fit(ui.Contain)
			ui.Text(c, i18n.T(i18n.KeyChatVisualStatic)).FontSize(11).TextColor(muted)
			if state.PreviewCapturedHeight < state.PreviewContentHeight {
				ui.Text(c, i18n.T(i18n.KeyChatVisualCapped)).FontSize(11).TextColor(muted)
			}
		}
		if state.Ready {
			source := state.HTML
			ui.TextArea(c.Key("html-source"), &source).ReadOnly(true).FillWidth().Lines(8, 20).FontSize(12).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyChatVisualSourceNote)).FontSize(11).TextColor(muted)
		}
	})
}

func visualReplySurfaceSeat(c *ui.Context, surface session.VisualReplySurface, themeID string, dark bool, height float32, inspector *InspectorNavigation) {
	resolved := theme.Resolve(themeID)
	colors := resolved.Active(dark)
	appearance := "light"
	if resolved.Scheme == theme.SchemeDark || (resolved.Scheme == theme.SchemeSystem && dark) {
		appearance = "dark"
	}
	if pixels, ok := surface.(interface {
		RenderPixels(*ui.Context, float32, string, map[string]string) bool
	}); ok {
		if pixels.RenderPixels(c, height, appearance, map[string]string{"--bg": colors.BG, "--background": colors.BG, "--surface": colors.Field, "--text": colors.FG, "--text-secondary": colors.Muted, "--border": colors.Border, "--accent": colors.Accent}) {
			return
		}
	}
	ui.Box(c).FillWidth().Height(height).DrawOver(func(p *ui.Painter, r ui.Rect) {
		clip := p.NativeSurfaceClip()
		if inspector != nil && inspector.Picking {
			clip = ui.Rect{}
		}
		_ = surface.Present(int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H), appearance, map[string]string{"--bg": colors.BG, "--background": colors.BG, "--surface": colors.Field, "--text": colors.FG, "--text-secondary": colors.Muted, "--border": colors.Border, "--accent": colors.Accent})
		// Presenting renews the helper's paint lease. Repaint, not rebuild, to
		// keep it: a build every 100ms redrew the whole window while shown.
		p.After(100 * time.Millisecond)
	})
}

func inlineVisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, page session.InlineVisualReply, fg, muted, field ui.Color) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if page.Loading {
		ui.Text(c, i18n.T(i18n.KeyChatVisualLoadingPage)).TextColor(muted)
		return
	}
	if page.Error != "" {
		ui.Text(c, page.Error).TextColor(muted)
		if page.Retry != nil && stdButton(c, palette, i18n.T(i18n.KeyChatImageRetry), buttonGhost, false).Clicked() {
			page.Retry()
		}
		return
	}
	if page.Surface == nil {
		return
	}
	// The seat repaints itself to keep the helper presented; the owner asks for
	// a build when the page's content height changes.
	height := float32(max(80, min(2000, page.Height)))
	if !page.Expanded {
		if stdButton(c, palette, i18n.T(i18n.KeyChatVisualExpand), buttonGhost, false).Clicked() {
			page.ToggleExpanded()
		}
		visualReplySurfaceSeat(c, page.Surface, m.ThemeID, m.SystemDark, height, m.Inspector)
		return
	}
	// Preserve the inline layout while moving the same child to the dialog seat.
	ui.Box(c).FillWidth().Height(height)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 960, 12).Children(func() {
			ui.Row(c).FillWidth().Children(func() {
				ui.Text(c, p.Title).Grow(1).TextColor(fg)
				if stdButton(c, palette, i18n.T(i18n.KeyChatRowCollapse), buttonGhost, false).Clicked() {
					open = false
				}
			})
			visualReplySurfaceSeat(c, page.Surface, m.ThemeID, m.SystemDark, 600, nil)
		})
	})
	if !open {
		page.ToggleExpanded()
	}
}
