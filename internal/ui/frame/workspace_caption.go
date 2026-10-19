package frame

import (
	"math"
	"runtime"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type workspaceCaptionMeasure struct {
	trailing float32
	height   float32
}

func conversationIsManaged(snapshot session.ViewState) bool {
	for _, item := range snapshot.Items {
		if item.SessionID == snapshot.SelectedID {
			return item.Presentation != nil && item.Presentation.Composer == "managed"
		}
	}
	return false
}

// AppFrame owns one auto-sized title row across conversation and details.
// The right dock begins in row two; native caption controls remain an overlay.
func workspaceSharedCaption(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, hero bool) float32 {
	row := ui.Row(c.Key("workspace-shared-caption")).FillWidth().MinHeight(52).Shrink(0).DragWindow()
	conversationFileDrop(c, m, row, snap.SelectedID)
	measure := ui.Local(row, "trailing-measure", func() workspaceCaptionMeasure { return workspaceCaptionMeasure{} })
	controls := max(float32(112), c.TitleBar().Right+8)
	if runtime.GOOS == "darwin" {
		controls = 8
	}
	details := float32(0)
	if m.FilePaneWidth != nil && m.RightPaneOpen {
		details = *m.FilePaneWidth
	}
	trailingInset := float32(math.Ceil(float64(measure.trailing + controls)))
	m.captionRightInset = max(float32(28), max(float32(0), trailingInset-details)+24)
	var trailing ui.Element
	row.Children(func() {
		ui.Column(c.Key("workspace-caption-conversation")).Width(m.Sizes.Center).Shrink(0).MinWidth(0).Children(func() {
			if m.WorkspacePage {
				ui.Box(c).Height(52)
				return
			}
			if mainSubagentConversationHeader(c, m, m.captionRightInset) {
				return
			}
			if hero {
				ui.Row(c).FillWidth().Height(52).Padding(0, m.captionRightInset, 0, 20).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0)
					jobsHeader(c, m)
				})
				return
			}
			conversationHeaderTitleRow(c, m, snap, colors)
			if m.HideChatViewTabs || m.DisableDeveloperTools || conversationIsManaged(snap) {
				ui.Box(c).Height(13) // source bottom padding12 + transparent border1
				return
			}
			conversationViewTabs(c, m, colors)
		})
		if m.WorkspaceTrailing != nil {
			densityWidth := m.Sizes.Center
			if details > 0 {
				densityWidth = min(densityWidth, details)
			}
			density := "full"
			if densityWidth < 720 {
				density = "compact"
			} else if densityWidth < 900 {
				density = "cozy"
			}
			trailing = ui.Row(c.Key("workspace-titlebar-trailing")).Absolute().Top(8).Right(controls).Height(32).Gap(8)
			trailing.Children(func() { m.WorkspaceTrailing(c, density) })
		}
	})
	// Build-time Bounds is the previous frame. Painting follows layout/commit,
	// so this observes actual new geometry, including a trailing width of zero.
	// Request another frame only when that geometry changes, never on a timer.
	row.DrawOver(func(_ *ui.Painter, bounds ui.Rect) {
		width := float32(0)
		if m.WorkspaceTrailing != nil {
			width = trailing.Bounds().W
		}
		if measure.trailing != width || measure.height != bounds.H {
			measure.trailing, measure.height = width, bounds.H
			c.Invalidate()
		}
	})
	return max(float32(52), row.Bounds().H)
}

func conversationViewTabs(c *ui.Context, m Model, colors theme.Colors) {
	ui.Row(c).FillWidth().Padding(4, 28, 0, 28).AlignItems(ui.End).Gap(36).Children(func() {
		tab := derefString(m.ChatPaneTab)
		if tab == "" {
			tab = "chat"
		}
		paint := func(id, label string) {
			ink := chrome.Color(colors.Muted)
			if tab == id {
				ink = chrome.Color(colors.Accent)
			}
			ui.Column(c).Children(func() {
				button := ui.Button(c, label).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(16)).FontWeight(500).
					BorderWidth(0).Background(ui.Transparent).TextColor(ink).Padding(0, 0, 9, 0)
				if button.Clicked() && m.ChatPaneTab != nil {
					*m.ChatPaneTab = id
				}
				line := ui.Box(c).FillWidth().Height(2)
				if tab == id {
					line.Background(chrome.Color(colors.Accent)).Radius(2)
				}
			})
		}
		paint("chat", i18n.T(i18n.KeyChatPaneTabChat))
		paint("trajectory", i18n.T(i18n.KeyChatPaneTabTrajectory))
		ui.Box(c).Grow(1).MinWidth(0)
	})
	ui.Box(c).FillWidth().Height(1).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - .5, W: r.W, H: .5}, chrome.Color(colors.Border), 0)
	})
}
