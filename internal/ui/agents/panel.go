// Package agents is the desktop「智能体团队」pane (Electron ui-agents-panel).
package agents

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/theme"
)

// Model drives the agents / subagent team list for the active session.
type Model struct {
	ChatPreferences    settings.ChatPreferences
	BusyEnter          string
	ComposerOverride   func(*ui.Context, session.AgentSessionAddress) bool
	ChildDraft         func(binding string, address session.AgentSessionAddress) *composer.DraftState
	MainConversation   bool
	OnOpenConversation func(uint64, session.AgentSessionAddress)
	Interactions       *interaction.Store
	OnOpenPlanResource func(planpreview.Document)
	ChatOnly           bool
	OnOpenChat         func(uint64, session.AgentSessionAddress)
	Store              *session.Store
	Manager            *session.AgentPresetManager
	// PresetManagerOpen shows the preset manager when the pane's ⋯ menu turns
	// it on; nil keeps it hidden, like the original surface does.
	PresetManagerOpen *bool
	Snapshot          session.ViewState
	ThemeID           string
	SystemDark        bool
	PromptDraft       *string // draft for subagents/prompt to the selected child
	OnSelect          func(sessionID string)
	OnStop            func(sessionID string) // session/cancel for a running child
	OnInterrupt       func(parentID, childID string)
	OnPrompt          func(parentID, childID, text string)
	OnOpenURL         func(string)
	OnOpenDocument    func(uint64, session.AgentSessionAddress, string)
	OnClose           func()
}

// View paints child sessions of the selected root (ParentID tree).
func View(c *ui.Context, m Model) {
	if m.Store != nil {
		navigationView(c, m)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	rootID := m.Snapshot.SelectedID
	// If selection is already a child, climb to its parent as the team root.
	for _, it := range m.Snapshot.Items {
		if it.SessionID == rootID && it.ParentID != "" {
			rootID = it.ParentID
			break
		}
	}

	children := childSessions(m.Snapshot.Items, rootID)
	rootTitle := i18n.T(i18n.KeyAgentsRootFallback)
	for _, it := range m.Snapshot.Items {
		if it.SessionID == rootID {
			if t := strings.TrimSpace(it.Title); t != "" {
				rootTitle = t
			}
			break
		}
	}

	paneBg := field
	if design.For(c).Glass || chrome.BackdropActive() {
		paneBg = ui.Transparent
	}
	ui.Column(c).Fill().MinHeight(0).Background(paneBg).Children(func() {
		ui.Row(c).FillWidth().MinHeight(36).Padding(8, 12, 8, 12).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAgentsTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			runningN := 0
			for _, ch := range children {
				if ch.Running {
					runningN++
				}
			}
			if runningN > 0 && m.OnInterrupt != nil && rootID != "" {
				if coss.Button(c, i18n.T(i18n.KeyAgentsInterruptAll), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonXS}).Clicked() {
					for _, ch := range children {
						if ch.Running {
							m.OnInterrupt(rootID, ch.SessionID)
						}
					}
				}
			}
			if m.OnClose != nil {
				if coss.Button(c, i18n.T(i18n.KeyAgentsClose), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					m.OnClose()
				}
			}
		})
		ui.Box(c).FillWidth().Height(0.5).Background(border)
		if m.Manager != nil && m.PresetManagerOpen != nil && *m.PresetManagerOpen {
			presetManagerView(c, m.Manager, fg, muted)
		}
		ui.Column(c).FillWidth().Padding(10, 12, 10, 12).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAgentsRootLabel)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			rootBtn := coss.Button(c, rootTitle, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonLG}).FillWidth().Justify(ui.Start)
			if rootBtn.Clicked() && m.OnSelect != nil && rootID != "" {
				m.OnSelect(rootID)
			}
			if rootID == "" {
				ui.Text(c, i18n.T(i18n.KeyAgentsPickSession)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				return
			}
			ui.Text(c, i18n.T(i18n.KeyAgentsChildren)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			if len(children) == 0 {
				ui.Text(c, i18n.T(i18n.KeyAgentsEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				return
			}
			for _, it := range children {
				row := it
				label := strings.TrimSpace(row.Title)
				if label == "" {
					label = row.SessionID
				}
				sub := ""
				if row.Cwd != "" {
					sub = filepath.Base(row.Cwd)
				}
				bg := ui.Transparent
				if row.SessionID == m.Snapshot.SelectedID {
					bg = chrome.Color(colors.Hover)
					if bg.A == 0 {
						bg = chrome.Color("rgba(0,0,0,0.05)")
					}
				}
				ui.Row(c).FillWidth().Gap(6).Radius(10).Border(0.5, border).Padding(10, 12, 10, 12).Background(bg).AlignItems(ui.Center).Children(func() {
					btn := coss.Button(c, "", coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Label:   label,
						Content: func() {
							ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
								ui.Row(c).FillWidth().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
									ink := muted
									pillBg := muted.Alpha(0.12)
									statusLabel := agentText("空闲", "Idle")
									if row.Running {
										ink = accent
										pillBg = accent.Alpha(0.14)
										statusLabel = agentText("运行中", "Running")
									}
									ui.Text(c, "●").FontSize(theme.InterfaceSize(10)).TextColor(ink)
									ui.Text(c, label).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).Grow(1)
									ui.Row(c).Radius(design.RadiusFull).Padding(design.Sp(0.5), design.Sp(2)).Background(pillBg).AlignItems(ui.Center).Children(func() {
										ui.Text(c, statusLabel).FontSize(theme.InterfaceSize(11)).TextColor(ink)
									})
								})
								if sub != "" {
									ui.Text(c, sub).FontSize(theme.InterfaceSize(11)).TextColor(muted)
								}
							})
						},
					}).Grow(1).Padding(0).Radius(design.RadiusMD)
					if btn.Clicked() && m.OnSelect != nil {
						m.OnSelect(row.SessionID)
					}
					if row.Running && m.OnStop != nil {
						if coss.Button(c, i18n.T(i18n.KeyAgentsStop), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() {
							m.OnStop(row.SessionID)
						}
					}
					if row.Running && m.OnInterrupt != nil {
						if coss.Button(c, i18n.T(i18n.KeyAgentsInterrupt), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonXS}).Clicked() {
							m.OnInterrupt(rootID, row.SessionID)
						}
					}
				})
			}
			if m.PromptDraft != nil && m.OnPrompt != nil {
				target := m.Snapshot.SelectedID
				if target == "" || target == rootID {
					// Prefer first child when root is selected.
					if len(children) > 0 {
						target = children[0].SessionID
					} else {
						target = ""
					}
				}
				isChild := false
				for _, ch := range children {
					if ch.SessionID == target {
						isChild = true
						break
					}
				}
				if isChild && target != "" {
					ui.Text(c, i18n.T(i18n.KeyAgentsPromptPlaceholder)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Grow(1).MinWidth(0).Children(func() {
							coss.Input(c, m.PromptDraft, coss.InputOptions{Size: coss.InputSM, Placeholder: i18n.T(i18n.KeyAgentsPromptPlaceholder)})
						})
						if coss.Button(c, i18n.T(i18n.KeyAgentsPromptSend), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
							text := strings.TrimSpace(*m.PromptDraft)
							if text != "" {
								m.OnPrompt(rootID, target, text)
								*m.PromptDraft = ""
							}
						}
					})
				}
			}
		})
	})
}

func childSessions(items []session.Item, parentID string) []session.Item {
	if parentID == "" {
		return nil
	}
	var out []session.Item
	for _, it := range items {
		if it.ParentID == parentID {
			out = append(out, it)
		}
	}
	return out
}
