package agents

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func agentMetricLabel(row session.AgentNavigationRow, now float64) (string, string) {
	state := agentText("状态未知", "Activity unknown")
	if row.Address.Mode == "external" {
		return "", ""
	}
	if row.ActivityKnown {
		state = agentText("未活动", "Inactive")
		if row.Running {
			state = agentText("运行中", "Running")
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
	focusFirst := false
	count := agentText("目录状态未知", "Catalog state unknown")
	if v.RootCountsKnown {
		count = fmt.Sprintf(agentText("根目录 %d 个 · 已知运行 %d 个", "Root: %d children · %d known running"), v.RootChildren, v.RootRunning)
		if v.RootActivityUnknown > 0 {
			count += fmt.Sprintf(agentText(" · %d 个状态未知", " · %d activity unknown"), v.RootActivityUnknown)
		}
	}
	entry := ui.Button(c, count).FillWidth().FontSize(theme.InterfaceSize(11)).Tooltip(agentText("进入当前层目录；方向键导航，右方向键进入，左方向键返回", "Focus current directory; arrows navigate, Right enters, Left returns"))
	focusFirst = entry.Clicked() || entry.Shortcut(0, ui.KeyDown)
	if entry.Shortcut(0, ui.KeyLeft) && v.Depth > 0 {
		m.Store.BackAgentNavigation(v.Token)
	}
	ui.Text(c, agentText("当前会话的子代理", "Children of this session")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	if len(v.Rows) == 0 && !v.Busy {
		ui.Text(c, i18n.T(i18n.KeyAgentsEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	var buttons []ui.Element
	var paths [][]session.AgentSessionAddress
	now := float64(time.Now().UnixMilli())
	tree := m.Store.AgentCatalogTree(v.Token)
	for index, node := range tree {
		row := node.Row
		if row.ActivityKnown && row.Running && row.Metrics.Timing != nil && row.Metrics.Timing.Since != nil {
			c.After(time.Second)
		}
		ui.Column(c.Key(fmt.Sprintf("catalog:%s:%d:%d", v.SessionID, v.Token, index))).FillWidth().Gap(2).Padding(0, 0, 0, float32(node.Depth*14)).Children(func() {
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				toggle := "▸"
				if node.Expanded {
					toggle = "▾"
				}
				if row.CanOpen() && ui.Button(c, toggle).Label(agentText("展开/收起 ", "Expand/collapse ")+row.Label).Disabled(v.Busy).Clicked() {
					m.Store.ToggleAgentCatalogBranch(v.Token, node.Path)
				}
				title := row.Label + " · " + row.Address.Mode
				if row.CanOpen() && row.Running {
					title = "● " + title
				}
				_, tooltip := agentMetricLabel(row, now)
				button := ui.Button(c, title).Grow(1).FontSize(theme.InterfaceSize(12)).Tooltip(tooltip).Disabled(v.Busy || !row.CanOpen())
				if row.CanOpen() && !v.Busy {
					if len(buttons) == 0 {
						button.AutoFocus()
					}
					buttons = append(buttons, button)
					paths = append(paths, node.Path)
				}
				if button.Clicked() {
					m.Store.OpenAgentCatalogPath(v.Token, node.Path)
				}
				if m.OnOpenConversation != nil && ui.Button(c, agentText("在主对话打开", "Open in conversation")).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy || !row.CanOpen()).Clicked() {
					m.OnOpenConversation(v.Token, row.Address)
				}
				if m.OnOpenChat != nil && ui.Button(c, agentText("在聊天面板打开", "Open chat aside")).FontSize(theme.InterfaceSize(11)).Disabled(v.Busy || !row.CanOpen()).Clicked() {
					m.OnOpenChat(v.Token, row.Address)
				}
				if node.Depth == 0 && row.CanOpen() && row.Running && ui.Button(c, i18n.T(i18n.KeyAgentsInterrupt)).FontSize(theme.InterfaceSize(11)).Disabled(v.ActionBusy).Clicked() {
					m.Store.InterruptAgentNavigation(v.Token, []session.AgentSessionAddress{row.Address})
				}
			})
			if node.Busy {
				ui.Text(c, agentText("正在读取子目录…", "Loading children…")).FontSize(10).TextColor(muted)
			}
			if node.Error != "" {
				ui.Text(c, node.Error).FontSize(10).TextColor(muted)
			}
			if label, tooltip := agentMetricLabel(row, now); label != "" {
				ui.Text(c, label).FontSize(theme.InterfaceSize(10)).TextColor(muted).Tooltip(tooltip)
			}
		})
	}
	if len(buttons) == 0 {
		entry.AutoFocus()
	}
	if focusFirst && len(buttons) > 0 {
		buttons[0].Focus()
	}
	catalogKeyboard(buttons, func(i int) { m.Store.OpenAgentCatalogPath(v.Token, paths[i]) }, func() {
		if v.Depth > 0 {
			m.Store.BackAgentNavigation(v.Token)
		}
	})
}
