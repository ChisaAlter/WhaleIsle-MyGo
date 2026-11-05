// Package pad paints the native floating Pad dock and its controls. Only each
// sandboxed HTML/script body uses a binding-free, native-clipped WebView.
package pad

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/pad"
	store "whaleisle/internal/store/pad"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Model struct {
	Desktop       *desktop.Desktop
	ThemeID       string
	SystemDark    bool
	Width, Height float32
}
type viewState struct {
	Menu, Delete, Models, Focus string
	ModelRequest                uint64
}
type cardDrag struct {
	ID         string
	Generation uint64
}
type dragging struct {
	Held  bool
	Moved bool
}
type splitDrag struct {
	Held                           bool
	Area, Total, Pair, Left, Delta float32
}

func copy(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}
func View(c *ui.Context, m Model) {
	if m.Desktop == nil || m.Desktop.Store == nil {
		return
	}
	d, s := m.Desktop, m.Desktop.Store
	snap := s.Snapshot()
	if !snap.Attached || snap.Disabled {
		d.Hide()
		return
	}
	if m.Width <= 0 || m.Height <= 0 {
		m.Width, m.Height = c.Size()
	}
	if m.Width < 100 || m.Height < 120 {
		d.Hide()
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	mode := "light"
	if selected := theme.Resolve(m.ThemeID); selected.Scheme == theme.SchemeDark || selected.Scheme == theme.SchemeSystem && m.SystemDark {
		mode = "dark"
	}
	appearance := desktop.Appearance{Mode: mode, Lang: i18n.Locale(), CSS: fmt.Sprintf(":root{--pad-fg:%s;--pad-fg-2:%s;--pad-fg-3:%s;--pad-bg:%s;--pad-bg-2:%s;--pad-border:%s;--pad-accent:%s;--pad-danger:%s;--pad-warn:#b37b21;--pad-ok:#3d9b70;--pad-font:system-ui,sans-serif;--pad-mono:Consolas,monospace;--pad-radius:6px}", colors.FG, colors.Secondary, colors.Muted, colors.BG, colors.Field, colors.Border, colors.Accent, colors.Danger)}
	d.BeginFrame(appearance)
	defer d.EndFrame()
	stateSeat := ui.Box(c.Key("pad-state-seat")).Absolute().Width(0).Height(0).Invisible()
	state := ui.Local(stateSeat, "native-pad-state", func() viewState { return viewState{} })
	state.Menu = ""
	defer func() {
		if state.Menu != "" {
			d.SetMasked(true)
		}
	}()
	if state.Focus != "" {
		if p, ok := find(snap, state.Focus); ok {
			focus(c, m, snap, p, state, colors)
			return
		}
		state.Focus = ""
	}
	width := min(snap.Layout.Width, m.Width-16)
	height := snap.Layout.Height
	if height <= 0 {
		height = min(min(560, m.Height-80), max(180, estimateHeight(d, snap, snap.Layout.Tree)+56))
		if len(snap.Panels) == 0 {
			height = min(320, m.Height-80)
		}
	}
	height = max(120, min(height, m.Height-48))
	position := store.PointXY{X: m.Width - width - 12, Y: 64}
	if snap.Layout.Dock != nil {
		position = *snap.Layout.Dock
	}
	position = bounded(position, width, height, m.Width, m.Height)
	if !snap.Layout.Expanded {
		ball(c, m, snap, colors)
		return
	}
	if len(snap.Panels) > 0 {
		c.After(time.Second)
	}
	ui.Overlay(c, func() {
		dock := ui.Column(c.Key("pad-dock")).Absolute().Left(position.X).Top(position.Y).Width(width).Height(height).Background(chrome.Color(colors.BG)).TextColor(chrome.Color(colors.FG)).Border(1, chrome.Color(colors.Border)).Radius(12).Clip()
		dock.Children(func() {
			ui.Row(c).FillWidth().Height(36).Padding(4, 6).Gap(4).AlignItems(ui.Center).Children(func() {
				grip := ui.Box(c.Key("pad-move")).Grow(1).Height(28).Focusable().Label(copy("移动面板坞", "Move panel dock"))
				grip.Children(func() { ui.Text(c, copy("面板", "Panels")).FontSize(13).Bold().TextColor(chrome.Color(colors.FG)) })
				moveDock(c, grip, s, position, width, height, m)
				if ui.Button(c, copy("新建", "New")).FontSize(12).Clicked() {
					d.Ask(copy("用 dsh-pad 的 pad_create 创建一张自定义悬浮面板，按工具描述里的面板编写指南来做。需求：", "Create a custom floating panel with dsh-pad pad_create. Follow its authoring guide. Requirements: "), true)
				}
				if ui.Button(c, copy("刷新", "Refresh")).FontSize(12).Disabled(len(snap.Panels) == 0).Clicked() {
					s.RefreshAll()
				}
				collapse := ui.Button(c, copy("收起", "Fold")).FontSize(12).Tooltip(copy("点击收起，拖动可移动面板", "Click to fold; drag to move the dock"))
				moved := moveDock(c, collapse, s, position, width, height, m)
				if collapse.Clicked() && !moved {
					s.UpdateLayout(func(l *store.Layout) {
						l.Expanded = false
						l.Ball = &store.PointXY{X: position.X + width - 32, Y: position.Y + 4}
					})
				}
			})
			if snap.Error != "" {
				ui.Text(c, snap.Error).FillWidth().FontSize(12).TextColor(chrome.Color(colors.Danger)).Padding(4, 10).Selectable()
			} else if snap.Loaded && !snap.Connected {
				ui.Text(c, copy("连接断开，正在重连…", "Disconnected. Reconnecting…")).FontSize(12).Padding(4, 10).TextColor(chrome.Color(colors.Muted))
			}
			askNotice(c, d, snap, colors)
			list := ui.Scroll(c.Key("pad-cards")).Grow(1).MinHeight(0).FillWidth().Padding(6).Children(func() {
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					if !snap.Loaded {
						ui.Text(c, copy("正在读取面板…", "Loading panels…")).FontSize(12)
						return
					}
					if len(snap.Panels) == 0 {
						empty(c, d, colors)
						return
					}
					tree(c, m, snap, snap.Layout.Tree, "root", state, colors)
				})
			})
			if drag, ok := ui.Drop[cardDrag](list); ok && drag.Generation == snap.Generation {
				s.DropCard(drag.ID, "", "root")
			}
			grip := ui.Box(c.Key("pad-size")).Height(10).FillWidth().Cursor(ui.CursorResizeNWSE).Focusable().Label(copy("调整面板坞大小", "Resize panel dock"))
			dx, dy, held := grip.Dragged()
			st := ui.Local(grip, "pad-size-drag", func() dragging { return dragging{} })
			if held {
				st.Held = true
				s.DragLayout(func(l *store.Layout) {
					l.Width = max(288, min(980, width+dx))
					l.Height = max(180, min(m.Height-position.Y-8, height+dy))
				}, false)
			} else if st.Held {
				st.Held = false
				s.DragLayout(func(*store.Layout) {}, true)
			}
			if grip.DoubleClicked() || grip.Shortcut(0, ui.KeyEnter) {
				s.UpdateLayout(func(l *store.Layout) { l.Height = 0 })
			}
		})
	})
}
func find(s store.Snapshot, id string) (store.Panel, bool) {
	for _, p := range s.Panels {
		if p.ID == id {
			return p, true
		}
	}
	return store.Panel{}, false
}
func bounded(p store.PointXY, w, h, vw, vh float32) store.PointXY {
	p.X = max(8, min(max(float32(8), vw-w-8), p.X))
	p.Y = max(40, min(max(float32(40), vh-h-8), p.Y))
	return p
}
func ball(c *ui.Context, m Model, snap store.Snapshot, colors theme.Colors) {
	s := m.Desktop.Store
	p := store.PointXY{X: m.Width - 44, Y: 64}
	if snap.Layout.Ball != nil {
		p = *snap.Layout.Ball
	}
	p = bounded(p, 32, 32, m.Width, m.Height)
	attention := false
	for _, v := range snap.Panels {
		if v.NeedsApproval || v.AgentRun != nil && v.AgentRun.Status == "failed" || snap.PanelErrors[v.ID] != "" {
			attention = true
		}
		for _, reading := range v.Data.Sources {
			if !reading.OK {
				attention = true
			}
		}
	}
	ui.Overlay(c, func() {
		button := ui.ButtonBase(c.Key("pad-ball")).Absolute().Left(p.X).Top(p.Y).Width(32).Height(32).Radius(16).Background(chrome.Color(colors.Field)).Border(1, chrome.Color(colors.Border)).Label(copy("展开面板", "Open panels")).Tooltip(copy("展开面板 · 拖动可移动", "Open panels · Drag to move")).Children(func() {
			ink := chrome.Color(colors.Accent)
			if attention {
				ink = chrome.Color(colors.Danger)
			}
			ui.Icon(c, starSVG).Size(20, 20).TextColor(ink)
		})
		dx, dy, held := button.Dragged()
		st := ui.Local(button, "pad-ball-drag", func() dragging { return dragging{} })
		if held {
			st.Held = true
			if dx != 0 || dy != 0 {
				st.Moved = true
				s.DragLayout(func(l *store.Layout) { l.Ball = &store.PointXY{X: p.X + dx, Y: p.Y + dy}; l.Dock = nil }, false)
			}
		} else if st.Held {
			st.Held = false
			s.DragLayout(func(l *store.Layout) {
				if l.Ball == nil {
					return
				}
				q := bounded(*l.Ball, 32, 32, m.Width, m.Height)
				if q.X < 34 {
					q.X = 8
				}
				if m.Width-q.X-32 < 34 {
					q.X = m.Width - 40
				}
				if q.Y < 66 {
					q.Y = 40
				}
				if m.Height-q.Y-32 < 34 {
					q.Y = m.Height - 40
				}
				l.Ball = &q
			}, true)
		}
		if button.Clicked() {
			if !st.Moved {
				s.UpdateLayout(func(l *store.Layout) { l.Expanded = true })
			}
			st.Moved = false
		}
	})
}
func moveDock(c *ui.Context, e ui.Element, s *store.Store, p store.PointXY, w, h float32, m Model) bool {
	dx, dy, held := e.Dragged()
	st := ui.Local(e, "pad-dock-drag", func() dragging { return dragging{} })
	moved := st.Moved
	if held {
		st.Held = true
		if dx != 0 || dy != 0 {
			st.Moved = true
			moved = true
		}
		s.DragLayout(func(l *store.Layout) {
			q := bounded(store.PointXY{X: p.X + dx, Y: p.Y + dy}, w, h, m.Width, m.Height)
			l.Dock = &q
		}, false)
	} else if st.Held {
		st.Held = false
		st.Moved = false
		s.DragLayout(func(*store.Layout) {}, true)
	}
	return moved
}
func empty(c *ui.Context, d *desktop.Desktop, colors theme.Colors) {
	ui.Column(c).FillWidth().Padding(12).Gap(10).Children(func() {
		ui.Text(c, copy("还没有面板", "No panels yet")).Bold().FontSize(14).TextColor(chrome.Color(colors.FG))
		ui.Text(c, copy("告诉 AI 你想长期观察什么。", "Tell AI what you want to keep watching.")).FillWidth().FontSize(13).TextColor(chrome.Color(colors.Secondary))
		for _, example := range []string{copy("观察这台机器的 CPU、内存和磁盘，每 3 秒刷新", "Watch this machine's CPU, memory and disk every 3 seconds"), copy("查看 API 账户余额", "Show my API account balance"), copy("跟踪项目最新 Release 和 Star", "Track a project's latest release and stars")} {
			if ui.Button(c, example).FillWidth().FontSize(12).Clicked() {
				d.Ask(example, false)
			}
		}
	})
}

var starSVG = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="currentColor"><path d="M14 6.3C14.8 10.8 17.2 13.2 21.7 14C17.2 14.8 14.8 17.2 14 21.7C13.2 17.2 10.8 14.8 6.3 14C10.8 13.2 13.2 10.8 14 6.3Z"/></svg>`))

func tree(c *ui.Context, m Model, snap store.Snapshot, n *store.Tree, path string, state *viewState, colors theme.Colors) {
	if n == nil {
		return
	}
	if n.Kind == "leaf" {
		if p, ok := find(snap, n.ID); ok {
			cardView(c.Key("pad-card-"+p.ID), m, snap, p, state, colors)
		}
		return
	}
	if n.Dir == "row" {
		row := ui.Row(c).FillWidth().Gap(0).AlignItems(ui.Start)
		row.Children(func() {
			for i, child := range n.Children {
				child, i := child, i
				if i > 0 {
					splitter(c, row, m.Desktop.Store, n, path, i-1)
				}
				weight := float32(1)
				if i < len(n.Weights) {
					weight = n.Weights[i]
				}
				ui.Column(c.Key(fmt.Sprint(path, i))).Grow(weight).MinWidth(0).Children(func() { tree(c, m, snap, child, fmt.Sprintf("%s.%d", path, i), state, colors) })
			}
		})
	} else {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			for i, child := range n.Children {
				tree(c, m, snap, child, fmt.Sprintf("%s.%d", path, i), state, colors)
			}
		})
	}
}
func splitter(c *ui.Context, row ui.Element, s *store.Store, n *store.Tree, path string, index int) {
	e := ui.Box(c.Key("pad-split-" + path + fmt.Sprint(index))).Width(6).Height(max(24, row.Bounds().H)).Cursor(ui.CursorResizeEW).Focusable().Label(copy("调整分栏比例", "Resize panel columns"))
	dx, _, held := e.Dragged()
	st := ui.Local(e, "split-drag-"+path+fmt.Sprint(index), func() splitDrag { return splitDrag{} })
	if held {
		if !st.Held && index+1 < len(n.Weights) {
			st.Held = true
			st.Area = max(1, row.Bounds().W-float32(len(n.Children)-1)*6)
			st.Total = 0
			for _, weight := range n.Weights {
				st.Total += weight
			}
			st.Pair = n.Weights[index] + n.Weights[index+1]
			st.Left = n.Weights[index] / st.Total * st.Area
			st.Delta = 0
		}
		st.Delta += dx
		s.DragLayout(func(l *store.Layout) {
			node := nodePath(l.Tree, path)
			if !st.Held || node == nil || index+1 >= len(node.Weights) {
				return
			}
			pairPixels := st.Pair / st.Total * st.Area
			minimum := min(float32(80), pairPixels/2)
			leftPixels := max(minimum, min(max(minimum, pairPixels-80), st.Left+st.Delta))
			left := leftPixels / st.Area * st.Total
			node.Weights[index], node.Weights[index+1] = left, st.Pair-left
		}, false)
	} else if st.Held {
		st.Held = false
		s.DragLayout(func(*store.Layout) {}, true)
	}
}
func nodePath(n *store.Tree, path string) *store.Tree {
	for _, v := range strings.Split(path, ".")[1:] {
		var i int
		_, _ = fmt.Sscan(v, &i)
		if n == nil || i < 0 || i >= len(n.Children) {
			return nil
		}
		n = n.Children[i]
	}
	return n
}

func cardView(c *ui.Context, m Model, snap store.Snapshot, p store.Panel, state *viewState, colors theme.Colors) {
	s, d := m.Desktop.Store, m.Desktop
	box := ui.Column(c).FillWidth().MinWidth(0).Gap(5).Padding(8).Background(chrome.Color(colors.Field)).Border(1, chrome.Color(colors.Border)).Radius(8)
	box.Children(func() {
		ui.Row(c).FillWidth().Gap(2).AlignItems(ui.Center).Children(func() {
			grip := ui.Button(c, "⠿").Width(22).Height(24).Label(copy("拖动卡片分栏或交换", "Drag panel to split or swap")).Drag(cardDrag{ID: p.ID, Generation: snap.Generation})
			_ = grip
			more := ui.Button(c, "⋯").Width(24).Height(24).Label(copy("更多操作", "More actions"))
			more.Menu(func(menu *ui.Menu) {
				if menu.Item(copy("让 AI 修改…", "Edit with AI…")).Chosen() {
					d.Ask(fmt.Sprintf(copy("修改悬浮面板「%s」（id: %s）：", "Edit floating panel \"%s\" (id: %s): "), p.Definition.Title, p.ID), false)
				}
				if menu.Item(copy("立即刷新", "Refresh now")).Disabled(p.NeedsApproval && p.Definition.Agent == nil).Chosen() {
					s.Do(store.Action{Generation: snap.Generation, Op: "refresh", PanelID: p.ID})
				}
				if menu.Item(copy("撤销上次修改", "Undo last edit")).Disabled(!p.CanUndo).Chosen() {
					s.Do(store.Action{Generation: snap.Generation, Op: "undo", PanelID: p.ID})
				}
				menu.Separator()
				if menu.Item(copy("面板模型", "Panel model") + " · " + modelLabel(snap.Catalog, p.Definition.Model)).Chosen() {
					state.Models, state.Delete, state.ModelRequest = p.ID, "", 0
					s.LoadModels()
				}
				menu.Separator()
				if menu.Item(copy("删除面板…", "Delete panel…")).Danger(true).Chosen() {
					state.Delete, state.Models = p.ID, ""
				}
			})
			if more.MenuOpen() {
				state.Menu, state.Delete, state.Models = p.ID, "", ""
			}
			title := p.Definition.Title
			if p.Definition.Icon != "" {
				title = p.Definition.Icon + " " + title
			}
			if ui.Button(c, title).Grow(1).MinWidth(0).FontSize(13).Label(copy("折叠或展开 ", "Fold or unfold ") + p.Definition.Title).Clicked() {
				s.Do(store.Action{Generation: snap.Generation, Op: "collapse", PanelID: p.ID, Collapsed: !p.Collapsed})
			}
			meta, alert := panelStatus(p, c.Now())
			ink := chrome.Color(colors.Muted)
			if alert {
				ink = chrome.Color(colors.Danger)
			}
			ui.Text(c, meta).MaxWidthPercent(38).NoWrap().Ellipsis("…").FontSize(11).TextColor(ink)
			if ui.Button(c, "↗").Width(24).Height(24).Label(copy("放大查看", "Expand panel")).Clicked() {
				state.Focus = p.ID
			}
		})
		if state.Delete == p.ID {
			ui.Column(c.Key("pad-delete")).FillWidth().Gap(5).Role(ui.RoleAlertDialog).Label(copy("删除面板", "Delete panel")).Children(func() {
				ui.Text(c, copy("删除后无法恢复。", "Deletion cannot be undone.")).FontSize(12)
				ui.Row(c).Gap(6).Children(func() {
					if ui.Button(c, copy("取消", "Cancel")).AutoFocus().Clicked() {
						state.Delete = ""
					}
					if ui.Button(c, copy("确认删除", "Delete")).Clicked() {
						s.Do(store.Action{Generation: snap.Generation, Op: "delete", PanelID: p.ID})
						state.Delete = ""
					}
				})
			})
		}
		if state.Models == p.ID {
			modelPicker(c, s, snap, p, state)
		}
		if p.Collapsed {
			return
		}
		if p.NeedsApproval {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				ui.Text(c, copy("本机数据源需要许可", "Allow these data sources")).FontSize(12).Bold()
				for _, source := range p.Definition.Sources {
					text := source.Command
					if source.Kind == "http" {
						method := source.Method
						if method == "" {
							method = "GET"
						}
						text = method + " " + source.URL
					}
					ui.Text(c, text).FillWidth().FontSize(12).Selectable()
				}
				if ui.Button(c, copy("允许运行", "Allow execution")).Clicked() {
					s.Do(store.Action{Generation: snap.Generation, Op: "approve", PanelID: p.ID, Fingerprint: p.Fingerprint})
				}
			})
		}
		if p.AgentRun != nil {
			label := ""
			if p.AgentRun.Status == "running" {
				label = copy("AI 正在刷新…", "AI is refreshing…")
			} else if p.AgentRun.Status == "failed" {
				label = copy("AI 刷新失败：", "AI refresh failed: ") + p.AgentRun.Error
			} else if p.Definition.Agent != nil {
				label = fmt.Sprintf(copy("AI 每 %d 分钟刷新 · 下次 %s", "AI every %d min · next %s"), p.Definition.Agent.EveryMinutes, clock(p.AgentRun.NextRunAt))
			}
			if label != "" {
				ui.Text(c, label).FillWidth().FontSize(12).TextColor(chrome.Color(colors.Secondary))
			}
		}
		if p.AwaitingData() {
			ui.Text(c, copy("正在读取数据…", "Waiting for first data…")).FontSize(12).TextColor(chrome.Color(colors.Muted))
		} else {
			height := panelHeight(d, snap, p)
			d.Body(c, p, height, false)
			resize := ui.Box(c.Key("pad-card-height")).Height(6).FillWidth().Cursor(ui.CursorResizeNS).Focusable().Label(copy("调整面板高度，回车恢复自动", "Resize panel height; Enter resets"))
			_, dy, held := resize.Dragged()
			st := ui.Local(resize, "pad-card-height-drag", func() dragging { return dragging{} })
			if held {
				st.Held = true
				s.DragLayout(func(l *store.Layout) { l.Heights[p.ID] = max(120, min(900, height+dy)) }, false)
			} else if st.Held {
				st.Held = false
				s.DragLayout(func(*store.Layout) {}, true)
			}
			if resize.DoubleClicked() || resize.Shortcut(0, ui.KeyEnter) {
				s.UpdateLayout(func(l *store.Layout) { delete(l.Heights, p.ID) })
			}
		}
		for id, r := range p.Data.Sources {
			if !r.OK {
				ui.Text(c, id+": "+r.Error).FillWidth().FontSize(12).TextColor(chrome.Color(colors.Danger))
			}
		}
		if text := snap.PanelErrors[p.ID]; text != "" {
			ui.Text(c, text).FillWidth().FontSize(12).Selectable().TextColor(chrome.Color(colors.Danger))
		}
		if text := snap.ActionErrors[p.ID]; text != "" {
			ui.Text(c, text).FillWidth().FontSize(12).Selectable().TextColor(chrome.Color(colors.Danger))
		}
	})
	if (state.Menu == p.ID || state.Delete == p.ID || state.Models == p.ID) && box.OverlayShortcut(0, ui.KeyEscape) {
		state.Menu = ""
		state.Delete = ""
		state.Models = ""
	}
	if _, over := ui.DragOver[cardDrag](box); over {
		box.Border(1, chrome.Color(colors.Accent))
	}
	if drag, x, y, ok := ui.DropAt[cardDrag](box); ok && drag.Generation == snap.Generation && drag.ID != p.ID {
		r := box.Bounds()
		zone := "center"
		rx, ry := (x-r.X)/max(1, r.W), (y-r.Y)/max(1, r.H)
		if rx < .3 {
			zone = "left"
		} else if rx > .7 {
			zone = "right"
		} else if ry < .3 {
			zone = "top"
		} else if ry > .7 {
			zone = "bottom"
		}
		s.DropCard(drag.ID, p.ID, zone)
	}
}
func panelHeight(d *desktop.Desktop, snap store.Snapshot, p store.Panel) float32 {
	if h := snap.Layout.Heights[p.ID]; h > 0 {
		return h
	}
	var fixed float32
	if json.Unmarshal(p.Definition.Height, &fixed) == nil && fixed > 0 {
		return fixed
	}
	return d.AutoHeight(p.ID)
}
func estimateHeight(d *desktop.Desktop, snap store.Snapshot, n *store.Tree) float32 {
	if n == nil {
		return 160
	}
	if n.Kind == "leaf" {
		p, ok := find(snap, n.ID)
		if !ok {
			return 0
		}
		if p.Collapsed {
			return 44
		}
		return panelHeight(d, snap, p) + 76
	}
	value := float32(0)
	for _, child := range n.Children {
		v := estimateHeight(d, snap, child)
		if n.Dir == "row" {
			value = max(value, v)
		} else {
			value += v + 8
		}
	}
	return value
}
func clock(raw string) string {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.Local().Format("15:04")
	}
	return "—"
}

func panelStatus(p store.Panel, now time.Time) (string, bool) {
	failed := p.NeedsApproval || p.AgentRun != nil && p.AgentRun.Status == "failed"
	latest := ""
	for _, r := range p.Data.Sources {
		if !r.OK {
			failed = true
		}
		if r.At > latest {
			latest = r.At
		}
	}
	if p.Data.Agent != nil && p.Data.Agent.At > latest {
		latest = p.Data.Agent.At
	}
	if p.AgentRun != nil && p.AgentRun.Status == "running" {
		return copy("AI 正在刷新…", "AI is refreshing…"), failed
	}
	if at, err := time.Parse(time.RFC3339Nano, latest); err == nil {
		seconds := max(0, int(now.Sub(at).Seconds()))
		if seconds < 60 {
			return fmt.Sprintf(copy("%d 秒前更新", "Updated %ds ago"), seconds), failed
		}
		if seconds < 3600 {
			return fmt.Sprintf(copy("%d 分钟前更新", "Updated %dm ago"), seconds/60), failed
		}
		return copy("更新于 ", "Updated at ") + at.Local().Format("01-02 15:04"), failed
	}
	return copy("等待数据", "Waiting for data"), failed
}
func modelPicker(c *ui.Context, s *store.Store, snap store.Snapshot, p store.Panel, state *viewState) {
	result := snap.ModelActions[p.ID]
	if state.ModelRequest != 0 && result.Request == state.ModelRequest && result.Succeeded {
		state.Models, state.ModelRequest = "", 0
		return
	}
	ui.Column(c.Key("pad-model-picker")).FillWidth().Gap(4).Role(ui.RoleDialog).Label(copy("面板模型", "Panel model")).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Text(c, copy("面板模型", "Panel model")).Grow(1).Bold().FontSize(12)
			if ui.Button(c, copy("关闭", "Close")).Clicked() {
				state.Models = ""
			}
		})
		if snap.ModelsLoading {
			ui.Text(c, copy("正在读取模型…", "Loading models…")).FontSize(12)
		}
		if snap.ModelsError != "" {
			ui.Text(c, snap.ModelsError).FillWidth().FontSize(12).TextColor(c.Theme().Danger).Selectable()
		}
		if result.Error != "" {
			ui.Text(c, result.Error).FillWidth().FontSize(12).TextColor(c.Theme().Danger).Selectable()
		}
		defaultLabel := copy("跟随全局默认", "Follow default")
		if snap.Catalog != nil && snap.Catalog.Default != nil {
			defaultLabel += " · " + modelLabel(snap.Catalog, snap.Catalog.Default)
		}
		if p.Definition.Model == nil {
			defaultLabel = "✓ " + defaultLabel
		}
		if ui.Button(c, defaultLabel).FillWidth().Disabled(result.Busy).Clicked() {
			state.ModelRequest = s.ChooseModel(snap.Generation, p.ID, nil)
		}
		if snap.Catalog == nil {
			return
		}
		ui.Scroll(c).MaxHeight(240).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				for _, provider := range snap.Catalog.Providers {
					ui.Text(c, provider.Name).Bold().FontSize(12)
					for _, model := range provider.Models {
						selection := &store.ModelSelection{Provider: provider.ID, Model: model.ID, ReasoningEffort: model.DefaultReasoningEffort}
						label := model.Name
						if p.Definition.Model != nil && p.Definition.Model.Provider == provider.ID && p.Definition.Model.Model == model.ID {
							label = "✓ " + label
						}
						if ui.Button(c, label).FillWidth().FontSize(12).Tooltip(model.Description).Disabled(result.Busy).Clicked() {
							state.ModelRequest = s.ChooseModel(snap.Generation, p.ID, selection)
						}
					}
				}
			})
		})
	})
}
func modelLabel(catalog *store.Catalog, selection *store.ModelSelection) string {
	if selection == nil {
		return copy("跟随全局默认", "Follow default")
	}
	if catalog != nil {
		for _, provider := range catalog.Providers {
			if provider.ID == selection.Provider {
				for _, model := range provider.Models {
					if model.ID == selection.Model {
						return model.Name
					}
				}
			}
		}
	}
	return selection.Model
}
func focus(c *ui.Context, m Model, snap store.Snapshot, p store.Panel, state *viewState, colors theme.Colors) {
	open := true
	ui.DialogBase(c.Key("pad-focus"), &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .42))
		panel.Width(min(640, m.Width*.92)).MaxHeight(max(0, m.Height-48)).Background(chrome.Color(colors.BG)).TextColor(chrome.Color(colors.FG)).Border(1, chrome.Color(colors.Border)).Radius(16).Padding(8, 14, 12).Gap(0).Label(p.Definition.Title).Children(func() {
			ui.Row(c).FillWidth().Height(40).Gap(8).AlignItems(ui.Center).Children(func() {
				if p.Definition.Icon != "" {
					ui.Text(c, p.Definition.Icon)
				}
				ui.Text(c, p.Definition.Title).Grow(1).MinWidth(0).NoWrap().Ellipsis("…").Bold()
				if ui.Button(c, "×").Size(28, 28).Label(copy("关闭", "Close")).AutoFocus().Clicked() {
					open = false
				}
			})
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(0, m.Height-108)).Children(func() {
				askNotice(c, m.Desktop, snap, colors)
				if p.AwaitingData() {
					ui.Text(c, copy("正在读取数据…", "Waiting for first data…"))
				} else {
					m.Desktop.Body(c, p, panelHeight(m.Desktop, snap, p), true)
				}
			})
		})
	})
	if !open {
		state.Focus = ""
	}
}

func askNotice(c *ui.Context, d *desktop.Desktop, snap store.Snapshot, colors theme.Colors) {
	if snap.Notice == "" {
		return
	}
	ui.Text(c, snap.Notice).FillWidth().FontSize(12).TextColor(chrome.Color(colors.Danger)).Padding(4, 10).Selectable()
	if pending := d.PendingAsk(); pending != "" {
		ui.Row(c).Wrap().Padding(4, 10).Gap(6).Children(func() {
			if ui.Button(c, copy("插入现有草稿", "Insert into draft")).Clicked() {
				d.Ask(pending, true)
			}
			if ui.Button(c, copy("复制问题", "Copy question")).Clicked() {
				mygo.Clipboard.WriteText(pending)
			}
			if ui.Button(c, copy("取消", "Dismiss")).Clicked() {
				d.DismissAsk()
			}
		})
	}
}
