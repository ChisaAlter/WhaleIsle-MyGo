package frame

import (
	"math"
	"runtime"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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

type headerSwapState struct {
	owner       string
	initialized bool
	wasHero     bool
	swapping    bool
	startedAt   time.Time
}

// conversationHeaderSwapOpacity fades the conversation title row in over
// widgets.SwapDuration (100ms) on the commit leaving Hero.
func conversationHeaderSwapOpacity(c *ui.Context, host ui.Element, owner string, hero, attempted bool) float32 {
	st := ui.Local(host, "hero-header-swap", func() headerSwapState { return headerSwapState{} })
	now := c.Now()
	if !st.initialized || st.owner != owner {
		*st = headerSwapState{owner: owner, initialized: true, wasHero: hero}
		return 1
	}
	reduced := c.Preferences().ReduceMotion
	if hero {
		st.wasHero = true
		st.swapping = false
		st.startedAt = time.Time{}
		return 1
	}
	if st.wasHero && attempted && !reduced && !now.IsZero() {
		st.wasHero = false
		st.swapping = true
		st.startedAt = now
	}
	st.wasHero = false
	if reduced || !st.swapping || now.IsZero() {
		st.swapping = false
		return 1
	}
	elapsed := now.Sub(st.startedAt)
	if elapsed >= widgets.SwapDuration {
		st.swapping = false
		return 1
	}
	c.AnimationFrame()
	if elapsed <= 0 {
		return 0
	}
	return widgets.DesignEase(float32(elapsed) / float32(widgets.SwapDuration))
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
		convCol := ui.Column(c.Key("workspace-caption-conversation")).Width(m.Sizes.Center + max(float32(0), m.RightOverlayReserve)).Shrink(0).MinWidth(0)
		if swap := conversationHeaderSwapOpacity(c, convCol, conversationEnterOwner(snap), hero, snap.PromptAttempted); swap < 1 {
			convCol = convCol.Opacity(swap)
		}
		convCol.Children(func() {
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
		if m.WorkspaceTrailing != nil {
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
			trailing = ui.Row(c.Key("workspace-titlebar-trailing")).Absolute().Top(12).Right(controls).Height(32).Gap(design.Sp(2))
			observeConversationTabInput(trailing, "trailing", m.ChatPaneTab)
			trailing.Children(func() {
				m.WorkspaceTrailing(c, density)
			})
		}
		// The reference's corner order is terminal, right panel, minimize,
		// maximize, close: the panel toggles come from the trailing cluster and the
		// native overlay keeps the close, so the shell adds the two missing.
		t := tk(c)
		captionWindowButtons(c, t.MutedForeground)
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
	layout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	tCaption := tk(c)
	if layout.Active() {
		row.Background(theme.Color(layout.ChatBackground))
	} else if tCaption.Glass || chrome.BackdropActive() {
		row.Background(ui.Transparent)
	}
	stroke, ink := design.BorderWidth/2, tCaption.Border
	if m.SystemDark {
		ink = design.TW.White.Alpha(0.02)
	}
	// The header's one hairline sits under the view tabs, not between the title
	// row and them (ConversationRoot.module.css .header::after vs .tabs), and a
	// blank header carries none at all (.headerBlank::after).
	if hero || conversationTabsShown(m, snap) || conversationHeaderHidden(snap, hasPendingStopInteraction(m.Interactions, snap.SelectedID)) {
		stroke = 0
	}
	if layout.Skin == theme.SkinClaude {
		stroke, ink = layout.HeaderBorder, theme.Color(layout.BorderColor)
	}
	if stroke > 0 {
		row.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - stroke, W: r.W, H: stroke}, ink, 0)
		})
	}
	return max(workspaceCaptionHeight, row.Bounds().H)
}

// conversationTabsShown reports whether this frame also paints the 对话/轨迹
// strip under the band — the same gate frame.go's chat column applies — so the
// caption knows its bottom edge is not the header's hairline.
func conversationTabsShown(m Model, snap session.ViewState) bool {
	if snap.SelectedID == "" || m.HideChatViewTabs || conversationIsManaged(snap) ||
		MainConversationChildActive(m) ||
		conversationHeaderHidden(snap, hasSessionInteractions(m.Interactions, snap.SelectedID)) {
		return false
	}
	ids, _ := conversationViewOptions(m)
	return m.ChatPaneTab != nil && len(ids) > 1
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
	return ids, labels
}

func selectConversationView(m Model, id string) {
	if m.ChatPaneTab == nil {
		return
	}
	before := *m.ChatPaneTab
	*m.ChatPaneTab = id
	traceConversationTabSelection(before, id, m.ChatPaneTab)
}

func conversationViewTabs(c *ui.Context, m Model, colors theme.Colors, compact bool) {
	_ = colors
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
	leftInset := design.Sp(7)
	if m.Sizes != nil && m.Sizes.drawerMode() {
		leftInset = design.Sp(14)
	}
	t := tk(c)
	accent := t.Primary
	if colors.Accent != "" {
		accent = theme.Color(colors.Accent)
	}
	// The one header hairline is .header::after: a 0.5 DIP border line at the
	// strip's bottom edge, across the conversation column's full width.
	ui.ScrollHorizontal(c.Key("conversation-view-scroll")).FillWidth().MinWidth(0).Height(28).Shrink(0).
		BorderWidth(0, 0, design.BorderWidth/2, 0).BorderColor(t.Border).Children(func() {
		tabs := ui.TabsBase(c.Key("conversation-view-tabs"), &selected, len(ids))
		tabs.List.Label("Conversation views").MinWidth(0).Height(28).Shrink(0).AlignItems(ui.Start).Gap(design.Sp(9)).Padding(0, design.Sp(7), 0, leftInset).Children(func() {
			for index, id := range ids {
				ink := t.MutedForeground
				if selected == index {
					ink = accent
				}
				tab := tabs.Tab(index).Label(labels[index]).Height(27).Padding(0)
				if selected == index {
					tab.ScrollIntoView()
				}
				observeConversationTabInput(tab, id, m.ChatPaneTab)
				tab.Children(func() {
					ui.Column(c).FillHeight().Children(func() {
						ui.Text(c, labels[index]).FontSize(theme.InterfaceSize(13)).FixedLineHeight(16).FontWeight(design.WeightMedium).TextColor(ink)
						ui.Box(c).Grow(1)
						line := ui.Box(c).FillWidth().Height(2).Margin(0, 0, design.BorderWidth, 0).Radius(design.RadiusXS)
						if selected == index {
							line.Background(accent)
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
