package frame

import (
	"math"
	"runtime"
	"strings"

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

// workspaceCaptionHeight is the native title band. Conversation view tabs
// occupy their own row below it, outside the window-drag region.
const workspaceCaptionHeight = float32(48)

// The band's content row matches the reference plate's padding, 12px top and
// 4px bottom (harness-chrome-inject.js:89; ConversationRoot.module.css:49-66
// padding-top 12 with a .titleRow of min-height 32 at :169-173). So the title,
// and the trailing cluster sit at y = 12..44, centred on 28,
// and the last 4 DIP of the band is empty above its hairline.
const (
	workspaceCaptionInsetTop    = float32(12)
	workspaceCaptionInsetBottom = float32(4)
	workspaceCaptionContent     = workspaceCaptionHeight - workspaceCaptionInsetTop - workspaceCaptionInsetBottom
)

// AppFrame owns one auto-sized title row across conversation and details.
// The right dock begins in row two; native caption controls remain an overlay.
func workspaceSharedCaption(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, hero bool) float32 {
	row := ui.Row(c.Key("workspace-shared-caption")).FillWidth().MinHeight(workspaceCaptionHeight).
		Padding(workspaceCaptionInsetTop, 0, workspaceCaptionInsetBottom, 0).
		Shrink(0).DragWindow()
	observeConversationTabInput(row, "caption", m.ChatPaneTab)
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
		ui.Column(c.Key("workspace-caption-conversation")).Width(m.Sizes.Center + max(float32(0), m.RightOverlayReserve)).Shrink(0).MinWidth(0).Children(func() {
			if m.WorkspacePage {
				ui.Box(c).Height(workspaceCaptionContent)
				return
			}
			if mainSubagentConversationHeader(c, m, m.captionRightInset) {
				return
			}
			if (hero && snap.SelectedID == "") || conversationHeaderHidden(snap, hasPendingStopInteraction(m.Interactions, snap.SelectedID)) {
				conversationBlankHeader(c, m, snap)
				return
			}
			conversationHeaderTitleRow(c, m, snap, colors)
		})
		// Installed AppFrame keeps clusterVisible true for native desktop
		// platforms, including narrow windows; label density concedes instead.
		if m.WorkspaceTrailing != nil || m.GlobalPluginSlot != nil {
			densityWidth := m.Sizes.Center
			if details > 0 {
				densityWidth = min(densityWidth, details)
			}
			// The installed resolver uses compact below 720, cozy below 900.
			// Desktop's visible cluster is always over the caption span; an open
			// details track uses the smaller of its width and the conversation.
			density := "full"
			if densityWidth < 720 {
				density = "compact"
			} else if densityWidth < 900 {
				density = "cozy"
			}
			// titlebar-density.ts: the cluster sits 12px below the band's top edge
			// and ends at the native caption reserve (dshd-wco-controls = 112px).
			trailing = ui.Row(c.Key("workspace-titlebar-trailing")).Absolute().Top(12).Right(controls).Height(32).Gap(8)
			observeConversationTabInput(trailing, "trailing", m.ChatPaneTab)
			trailing.Children(func() {
				renderGlobalShellSlot(c, m, "shell.titlebar.trailing", func() {
					if m.WorkspaceTrailing != nil {
						m.WorkspaceTrailing(c, density)
					}
				})
			})
		}
		// The reference's corner order is terminal, right panel, minimize,
		// maximize, close: the panel toggles come from the trailing cluster and the
		// native overlay keeps the close, so the shell adds the two missing.
		captionWindowButtons(c, chrome.Color(colors.Muted))
	})
	// Build-time Bounds is the previous frame. Painting follows layout/commit,
	// so this observes actual new geometry, including a trailing width of zero.
	// Request another frame only when that geometry changes, never on a timer.
	row.DrawOver(func(_ *ui.Painter, bounds ui.Rect) {
		width := float32(0)
		if m.WorkspaceTrailing != nil || m.GlobalPluginSlot != nil {
			width = trailing.Bounds().W
		}
		if measure.trailing != width || measure.height != bounds.H {
			measure.trailing, measure.height = width, bounds.H
			c.Invalidate()
		}
	})
	// The band's bottom hairline anchors the header the way the client's
	// conversation-header ::after does.
	layout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	if layout.Active() {
		row.Background(chrome.Color(layout.ChatBackground))
	}
	stroke, ink := float32(.5), chrome.Color(colors.Border)
	if m.SystemDark {
		ink = ui.RGBA(255, 255, 255, 0.02)
	}
	if hero {
		stroke = 0
	}
	if layout.Skin == theme.SkinClaude {
		stroke, ink = layout.HeaderBorder, chrome.Color(layout.BorderColor)
	}
	if stroke > 0 {
		row.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - stroke, W: r.W, H: stroke}, ink, 0)
		})
	}
	return max(workspaceCaptionHeight, row.Bounds().H)
}

// conversationViewTabs keeps pointer, arrow-key and accessibility navigation
// together. Compact title actions must never remove the view switcher.
func conversationViewOptions(m Model) (ids, labels []string) {
	ids = []string{"chat"}
	labels = []string{i18n.T(i18n.KeyChatPaneTabChat)}
	if !m.DisableDeveloperTools {
		ids = append(ids, "trajectory")
		labels = append(labels, i18n.T(i18n.KeyChatPaneTabTrajectory))
	}
	seen := map[string]bool{"chat": true, "trajectory": true}
	for _, view := range m.ConversationViews {
		if strings.TrimSpace(view.Handle) == "" || view.ID == "chat" || view.ID == "trajectory" {
			continue
		}
		id := "plugin:" + view.Handle
		if seen[id] {
			continue
		}
		seen[id] = true
		label := strings.TrimSpace(view.Label)
		if label == "" {
			label = view.ID
		}
		if label == "" {
			continue
		}
		ids, labels = append(ids, id), append(labels, label)
	}
	return ids, labels
}

func selectConversationView(m Model, id string) {
	if m.ChatPaneTab == nil {
		return
	}
	before := *m.ChatPaneTab
	*m.ChatPaneTab = id
	if strings.HasPrefix(id, "plugin:") {
		if m.OnSelectConversationView != nil {
			m.OnSelectConversationView(strings.TrimPrefix(id, "plugin:"))
		}
	} else if m.OnSelectBuiltinConversationView != nil {
		m.OnSelectBuiltinConversationView(id)
	} else if m.OnClearConversationView != nil {
		m.OnClearConversationView()
	}
	traceConversationTabSelection(before, id, m.ChatPaneTab)
}

func conversationViewTabs(c *ui.Context, m Model, colors theme.Colors, compact bool) {
	if m.ChatPaneTab == nil {
		return
	}
	ids, labels := conversationViewOptions(m)
	if len(ids) <= 1 {
		return
	}
	selected := 0
	for i, id := range ids {
		if *m.ChatPaneTab == id {
			selected = i
			break
		}
	}
	// ConversationRoot adds 8px inside the header's leading clearance. Keep
	// the phone menu clearance used by the title band rather than hiding tabs
	// underneath the floating drawer button.
	leftInset := float32(28)
	if m.Sizes != nil && m.Sizes.drawerMode() {
		leftInset = 56
	}
	ui.ScrollHorizontal(c.Key("conversation-view-scroll")).FillWidth().MinWidth(0).Height(28).Shrink(0).Children(func() {
		tabs := ui.TabsBase(c.Key("conversation-view-tabs"), &selected, len(ids))
		tabs.List.Label("Conversation views").MinWidth(0).Height(28).Shrink(0).AlignItems(ui.Start).Gap(36).Padding(0, 28, 0, leftInset).
			BorderWidth(0, 0, .5, 0).BorderColor(chrome.Color(colors.Border)).Children(func() {
			for index, id := range ids {
				ink := chrome.Color(colors.Muted)
				if selected == index {
					ink = chrome.Color(colors.Accent)
				}
				tab := tabs.Tab(index).Label(labels[index]).Height(27).Padding(0)
				if selected == index {
					tab.ScrollIntoView()
				}
				observeConversationTabInput(tab, id, m.ChatPaneTab)
				tab.Children(func() {
					ui.Column(c).FillHeight().Children(func() {
						ui.Text(c, labels[index]).FontSize(theme.InterfaceSize(13)).FixedLineHeight(16).FontWeight(500).TextColor(ink)
						ui.Box(c).Grow(1)
						line := ui.Box(c).FillWidth().Height(2).Margin(0, 0, 1, 0).Radius(2)
						if selected == index {
							line.Background(chrome.Color(colors.Accent))
						}
					})
				})
			}
		})
		// mainView commits paint-local bindings before deferred OnChange notices
		// run. Resolve the composite's input now, while this paint still owns its
		// tab pointer, so the selected view reaches that same commit.
		if tabs.List.Changed() {
			selectConversationView(m, ids[selected])
		}
	})
}
