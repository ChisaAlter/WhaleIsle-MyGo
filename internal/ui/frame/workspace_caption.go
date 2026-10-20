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

// workspaceCaptionHeight is the single title band above every column: the
// desktop caption's 48px (titlebar-density.ts sits in the band's trailing
// cluster). The band never grows a second row.
const workspaceCaptionHeight = float32(48)

// AppFrame owns one auto-sized title row across conversation and details.
// The right dock begins in row two; native caption controls remain an overlay.
func workspaceSharedCaption(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, hero bool) float32 {
	row := ui.Row(c.Key("workspace-shared-caption")).FillWidth().MinHeight(workspaceCaptionHeight).Shrink(0).DragWindow()
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
				ui.Box(c).Height(workspaceCaptionHeight)
				return
			}
			if mainSubagentConversationHeader(c, m, m.captionRightInset) {
				return
			}
			if hero {
				ui.Row(c).FillWidth().Height(workspaceCaptionHeight).Padding(0, m.captionRightInset, 0, 20).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0)
					jobsHeader(c, m)
				})
				return
			}
			// Title, status actions and the 对话/轨迹 switcher share this one 48px
			// row; rc.2 has no second tab row under the band.
			conversationHeaderTitleRow(c, m, snap, colors)
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
			// titlebar-density.ts: the cluster sits 12px below the band's top edge
			// and ends at the native caption reserve (dshd-wco-controls = 112px).
			trailing = ui.Row(c.Key("workspace-titlebar-trailing")).Absolute().Top(12).Right(controls).Height(32).Gap(8)
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
	// The band's bottom hairline anchors the header the way the client's
	// conversation-header ::after does.
	row.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - .5, W: r.W, H: .5}, chrome.Color(colors.Border), 0)
	})
	return max(workspaceCaptionHeight, row.Bounds().H)
}

// conversationViewTabs renders the 对话/轨迹 switcher inside the title band.
// The preference (ui-conversation viewTabs) still owns whether it exists; it
// never adds a row of its own, so the band stays 48px in every state.
func conversationViewTabs(c *ui.Context, m Model, colors theme.Colors, compact bool) {
	if compact {
		// The original collapses the trailing cluster instead of squeezing it; the
		// band keeps the title and the panel/terminal utilities at narrow widths.
		return
	}
	tab := derefString(m.ChatPaneTab)
	if tab == "" {
		tab = "chat"
	}
	paint := func(id, label string) {
		ink := chrome.Color(colors.Muted)
		if tab == id {
			ink = chrome.Color(colors.Accent)
		}
		ui.Column(c).Shrink(0).Height(workspaceCaptionHeight).Justify(ui.End).Children(func() {
			button := ui.Button(c, label).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).
				BorderWidth(0).Background(ui.Transparent).TextColor(ink).Padding(0, 2, 9, 2)
			if button.Clicked() && m.ChatPaneTab != nil {
				*m.ChatPaneTab = id
			}
			line := ui.Box(c).FillWidth().Height(2).Radius(1)
			if tab == id {
				line.Background(chrome.Color(colors.Accent))
			}
		})
	}
	ui.Row(c).Key("conversation-view-tabs").Shrink(0).Height(workspaceCaptionHeight).AlignItems(ui.End).Gap(18).
		Margin(0, 10, 0, 10).Children(func() {
		paint("chat", i18n.T(i18n.KeyChatPaneTabChat))
		paint("trajectory", i18n.T(i18n.KeyChatPaneTabTrajectory))
	})
}
