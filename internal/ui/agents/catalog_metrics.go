package agents

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func agentMetricLabel(row session.AgentNavigationRow, now float64) (string, string) {
	state := agentText("模式未知", "Activity unknown")
	if row.Address.Mode == "external" {
		return "", ""
	}
	if row.ActivityKnown {
		state = agentText("当前未运行", "Inactive")
		if row.Running {
			state = agentText("正在运行", "Running")
		} else if row.Metrics.Completed {
			state = agentText("已完成", "Completed")
		}
	}
	var compact, exact []string
	compact = append(compact, state)
	exact = append(exact, state)
	if row.Metrics.Tokens != nil {
		n := *row.Metrics.Tokens
		label := fmt.Sprintf("%.0f tokens", n)
		small := label
		if n >= 1e6 {
			small = fmt.Sprintf("%.1fM tokens", n/1e6)
		} else if n >= 1000 {
			small = fmt.Sprintf("%.1fK tokens", n/1000)
		}
		compact = append(compact, small)
		exact = append(exact, label)
	}
	// An active span cannot be extrapolated without a known activity state.
	if row.ActivityKnown || row.Metrics.Timing != nil && row.Metrics.Timing.Since == nil {
		if ms, ok := row.Metrics.Duration(row.Running, now); ok {
			seconds := int64(math.Floor(ms / 1000))
			compact = append(compact, fmt.Sprintf("%dh %dm %ds", seconds/3600, seconds/60%60, seconds%60))
			exact = append(exact, fmt.Sprintf("%d s active", seconds))
		}
	}
	return strings.Join(compact, " · "), strings.Join(exact, " · ")
}

// Scoped native directory keyboard navigation: no page-wide keys, and external
// metadata never acquires focus or becomes an addressed navigation action.
func catalogKeyboard(buttons []ui.Element, open func(int), back func()) {
	for i, b := range buttons {
		next := -1
		if b.Shortcut(0, ui.KeyHome) {
			next = 0
		} else if b.Shortcut(0, ui.KeyEnd) {
			next = len(buttons) - 1
		} else if b.Shortcut(0, ui.KeyDown) {
			next = i + 1
			if next >= len(buttons) {
				next = len(buttons) - 1
			}
		} else if b.Shortcut(0, ui.KeyUp) {
			next = i - 1
			if next < 0 {
				next = 0
			}
		}
		if next >= 0 {
			buttons[next].Focus()
			return
		}
		if b.Shortcut(0, ui.KeyRight) {
			open(i)
			return
		}
		if b.Shortcut(0, ui.KeyLeft) {
			back()
			return
		}
	}
}

func catalogRowsView(c *ui.Context, m Model, v session.AgentNavigationView, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	nowMs := float64(c.Now().UnixMilli())
	if len(v.Rows) == 0 && !v.Busy {
		ui.Column(c).FillWidth().MinHeight(160).Padding(16).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
			ui.Text(c, AgentsPanelLocale("empty.title")).FontSize(theme.InterfaceSize(14)).FontWeight(500).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
			ui.Text(c, AgentsPanelLocale("empty.body")).MaxWidth(220).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		})
	}
	var buttons []ui.Element
	var addresses []session.AgentSessionAddress
	open := func(address session.AgentSessionAddress) {
		if m.OnOpenConversation != nil {
			m.OnOpenConversation(v.Token, address)
		} else {
			m.Store.OpenAgentNavigation(v.Token, address)
		}
	}
	for _, row := range v.Rows {
		row := row
		disabled := v.Busy || !row.CanOpen()
		button := ui.Button(c.Key("agent/"+row.Address.ParentSessionID+"/"+row.Address.ChildSessionID+"/"+row.Address.Mode), "").Label(row.Label).Tooltip(FormatSubagentSwitcherAria(row.Label)).FillWidth().MinWidth(0).MinHeight(36).Padding(8, 10).Radius(6).BorderWidth(0).Background(chrome.Color(colors.Field)).Disabled(disabled)
		if !disabled {
			button.Cursor(ui.CursorPointer)
			if button.Hovered() {
				button.Background(chrome.Color(colors.Hover))
			}
		}
		button.Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				if row.Address.Mode != "external" {
					state := widgets.StateDone
					if !row.ActivityKnown {
						state = widgets.StateIdle
					} else if row.Running {
						state = widgets.StateRunning
					}
					widgets.StateDot(c, state)
				}
				ui.Text(c, row.Label).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
				if row.Address.Mode != "external" {
					if row.Metrics.Tokens != nil {
						ui.Text(c, FormatSubagentTokens(*row.Metrics.Tokens)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted).Shrink(0)
					}
					if row.ActivityKnown || row.Metrics.Timing != nil && row.Metrics.Timing.Since == nil {
						if ms, ok := row.Metrics.Duration(row.Running, nowMs); ok {
							durText := FormatSubagentDuration(ms)
							exactTip := FormatSubagentExactDurationTitle(FormatSubagentExactDuration(ms))
							ui.Text(c, durText).Tooltip(exactTip).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted).Shrink(0)
						}
					}
				}
				meta := AgentsPanelLocale("activity.inactive")
				if !row.ActivityKnown {
					meta = agentText("模式未知", "Activity unknown")
				}
				if row.Running {
					meta = AgentsPanelLocale("activity.running")
				} else if row.ActivityKnown && row.Metrics.Completed {
					meta = AgentsPanelLocale("activity.completed")
				}
				switch row.Address.Mode {
				case "external":
					meta = AgentsPanelLocale("mode.external")
				case "one-shot":
					meta += " · " + AgentsPanelLocale("mode.oneShot")
				case "continuable":
					meta += " · " + AgentsPanelLocale("mode.continuable")
				case "unknown":
					meta += " · " + AgentsPanelLocale("mode.unknown")
				}
				ui.Text(c, meta).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted).Shrink(0)
			})
		})
		if row.CanOpen() {
			buttons = append(buttons, button)
			addresses = append(addresses, row.Address)
		}
		if button.Clicked() {
			open(row.Address)
		}
		if m.OnOpenChat != nil && row.CanOpen() {
			sidebarAria := FormatSubagentOpenSidebarAria(row.Label)
			sbBtn := ui.Button(c.Key("agent-sidebar/"+row.Address.ParentSessionID+"/"+row.Address.ChildSessionID), SubagentLocale("open.sidebar")).
				Label(sidebarAria).Tooltip(sidebarAria).Height(24).Padding(0, 8).Radius(6).BorderWidth(0).
				Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy)
			if !v.Busy {
				sbBtn.Cursor(ui.CursorPointer)
			}
			if sbBtn.Clicked() {
				m.OnOpenChat(v.Token, row.Address)
			}
		}
	}
	catalogKeyboard(buttons, func(index int) { open(addresses[index]) }, func() {
		if v.Depth > 0 {
			m.Store.BackAgentNavigation(v.Token)
		}
	})
}
