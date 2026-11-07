package agents

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
	if len(v.Rows) == 0 && !v.Busy {
		ui.Text(c, i18n.T(i18n.KeyAgentsEmpty)).FontSize(theme.InterfaceSize(13)).TextColor(muted)
		ui.Text(c, AgentsPanelLocale("empty.title")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.Text(c, AgentsPanelLocale("empty.body")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
		button := ui.Button(c.Key("agent/"+row.Address.ParentSessionID+"/"+row.Address.ChildSessionID+"/"+row.Address.Mode), "").Label(row.Label).Tooltip(FormatSubagentSwitcherAria(row.Label)).FillWidth().MinWidth(0).Padding(8).Radius(8).BorderWidth(0).Background(ui.Transparent).Disabled(v.Busy || !row.CanOpen())
		if button.Hovered() {
			button.Background(c.Theme().SurfaceHover)
		}
		button.Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				if row.Address.Mode != "external" {
					ink := muted
					if row.Running {
						ink = c.Theme().Accent
					}
					ui.Box(c).Size(6, 6).Radius(3).Background(ink).Shrink(0)
				}
				ui.Text(c, row.Label).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").FontSize(theme.InterfaceSize(13))
				meta := agentText("当前未运行", "Inactive")
				if !row.ActivityKnown {
					meta = agentText("模式未知", "Activity unknown")
				}
				if row.Running {
					meta = agentText("正在运行", "Running")
				}
				switch row.Address.Mode {
				case "external":
					meta = agentText("外部任务", "External")
				case "one-shot":
					meta += " · " + agentText("一次性", "One-shot")
				case "continuable":
					meta += " · " + agentText("可继续", "Continuable")
				}
				ui.Text(c, meta).FontSize(theme.InterfaceSize(12)).TextColor(muted).Shrink(0)
			})
		})
		if row.CanOpen() {
			buttons = append(buttons, button)
			addresses = append(addresses, row.Address)
		}
		if button.Clicked() {
			open(row.Address)
		}
	}
	catalogKeyboard(buttons, func(index int) { open(addresses[index]) }, func() {
		if v.Depth > 0 {
			m.Store.BackAgentNavigation(v.Token)
		}
	})
}
