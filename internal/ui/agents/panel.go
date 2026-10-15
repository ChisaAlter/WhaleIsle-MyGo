// Package agents is the desktop「智能体团队」pane (Electron ui-agents-panel).
package agents

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model drives the agents / subagent team list for the active session.
type Model struct {
	Store       *session.Store
	Manager     *session.AgentPresetManager
	Snapshot    session.ViewState
	ThemeID     string
	SystemDark  bool
	PromptDraft *string // draft for subagents/prompt to the selected child
	OnSelect    func(sessionID string)
	OnStop      func(sessionID string) // session/cancel for a running child
	OnInterrupt func(parentID, childID string)
	OnPrompt    func(parentID, childID, text string)
	OnOpenURL   func(string)
	OnClose     func()
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

	ui.Column(c).Fill().MinHeight(0).Background(field).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12, 10, 12).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAgentsTitle)).FontSize(14).FontWeight(600).TextColor(fg).Grow(1)
			runningN := 0
			for _, ch := range children {
				if ch.Running {
					runningN++
				}
			}
			if runningN > 0 && m.OnInterrupt != nil && rootID != "" {
				if ui.Button(c, i18n.T(i18n.KeyAgentsInterruptAll)).FontSize(11).Clicked() {
					for _, ch := range children {
						if ch.Running {
							m.OnInterrupt(rootID, ch.SessionID)
						}
					}
				}
			}
			if m.OnClose != nil {
				if ui.Button(c, i18n.T(i18n.KeyAgentsClose)).FontSize(12).BorderWidth(0).Background(ui.Transparent).
					TextColor(muted).Clicked() {
					m.OnClose()
				}
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(border)
		if m.Manager != nil {
			presetManagerView(c, m.Manager, fg, muted)
		}
		ui.Column(c).FillWidth().Padding(10, 12, 10, 12).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAgentsRootLabel)).FontSize(11).TextColor(muted)
			rootBtn := ui.Button(c, rootTitle).FontSize(13).FontWeight(600).TextColor(fg).
				BorderWidth(0).Background(ui.Transparent).Padding(0, 0, 0, 0)
			if rootBtn.Clicked() && m.OnSelect != nil && rootID != "" {
				m.OnSelect(rootID)
			}
			if rootID == "" {
				ui.Text(c, i18n.T(i18n.KeyAgentsPickSession)).FontSize(12).TextColor(muted)
				return
			}
			ui.Text(c, i18n.T(i18n.KeyAgentsChildren)).FontSize(11).TextColor(muted)
			if len(children) == 0 {
				ui.Text(c, i18n.T(i18n.KeyAgentsEmpty)).FontSize(12).TextColor(muted)
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
				ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
					btn := ui.Button(c, "").Grow(1).Background(bg).BorderWidth(0).
						Padding(8, 8, 8, 8).Radius(8)
					btn.Children(func() {
						ui.Column(c).FillWidth().Gap(2).Children(func() {
							ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
								ink := muted
								if row.Running {
									ink = accent
								}
								ui.Text(c, "●").FontSize(10).TextColor(ink)
								ui.Text(c, label).FontSize(13).TextColor(fg).Grow(1)
							})
							if sub != "" {
								ui.Text(c, sub).FontSize(11).TextColor(muted)
							}
						})
					})
					if btn.Clicked() && m.OnSelect != nil {
						m.OnSelect(row.SessionID)
					}
					if row.Running && m.OnStop != nil {
						if ui.Button(c, i18n.T(i18n.KeyAgentsStop)).FontSize(11).Clicked() {
							m.OnStop(row.SessionID)
						}
					}
					if row.Running && m.OnInterrupt != nil {
						if ui.Button(c, i18n.T(i18n.KeyAgentsInterrupt)).FontSize(11).Clicked() {
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
					ui.Text(c, i18n.T(i18n.KeyAgentsPromptPlaceholder)).FontSize(11).TextColor(muted)
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						ui.TextInput(c, m.PromptDraft).Grow(1).Placeholder(i18n.T(i18n.KeyAgentsPromptPlaceholder))
						if ui.Button(c, i18n.T(i18n.KeyAgentsPromptSend)).FontSize(11).Clicked() {
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
