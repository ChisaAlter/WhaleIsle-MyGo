package frame

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

const inspectorRowHeight float32 = 30

// Chromium's monospace default for this Windows locale is NSimSun; this was
// verified through CSS.getPlatformFontsForNode in the installed inspector.
func inspectorFont() string {
	if runtime.GOOS == "windows" {
		return "NSimSun, monospace"
	}
	return "monospace"
}

var inspectorPickIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3"><circle cx="8" cy="8" r="4.5"/><path d="M8 0v5M8 11v5M0 8h5M11 8h5"/></svg>`))
var inspectorFilterIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3"><path d="M2 3h12L9.5 8v5l-3-1.5V8L2 3Z"/></svg>`))
var inspectorDownIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.2"><path d="m3 4.5 3 3 3-3"/></svg>`))

func inspectorButton(c *ui.Context, label string) ui.Element {
	return ui.ButtonBase(c).Label(label).BorderWidth(0).Radius(0).Padding(0).MinHeight(0).
		Background(ui.Transparent).FontSize(11).FontWeight(400).Cursor(ui.CursorPointer).Justify(ui.Start).AlignItems(ui.Center)
}

func inspectorText(c *ui.Context, text string) ui.Element {
	return ui.Text(c, text).FontSize(11).LineHeight(0).SingleLine().Ellipsis("…").MinWidth(0)
}

func inspectorToolbar(c *ui.Context, p *SessionInspectorPaneState, source session.InspectorSnapshot, rows []session.InspectorRecord, hasMore, busy bool, loadOlder func(), colors theme.Colors) {
	ink, border := chrome.Color(colors.FG), chrome.Color(colors.Border)
	toolbar := ui.Row(c.Key("inspector-toolbar")).Label("Inspector toolbar").FillWidth().Shrink(0).Gap(8).Padding(6, 8).
		AlignItems(ui.Center).FontSize(11).TextColor(ink).BorderWidth(0, 0, .5, 0).BorderColor(border)
	toolbar.Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			if p.Mode == "" {
				p.Mode = "session-log"
			}
			mode := &p.Mode
			parts := ui.SelectBase(c.Key("inspector-mode-select"), mode)
			parts.Trigger.Label(paneLabel("展示形式", "Presentation")).Padding(2, 4).BorderWidth(0).Radius(0).
				Background(chrome.Color(colors.Field)).FontSize(11).MinHeight(0).Children(func() {
				label := paneLabel("原始数据", "Raw Log")
				if *mode == "chat-node" {
					label = paneLabel("对话分组", "Chat Group")
				}
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					inspectorText(c, label)
					ui.Icon(c, inspectorDownIcon).Size(12, 12).TextColor(ink)
				})
			})
			parts.Popup(func(panel ui.Element) {
				panel.MinWidth(120).Padding(3).Background(chrome.Color(colors.Field)).Border(.5, border).Radius(4)
				for _, item := range []struct{ key, label string }{{"session-log", paneLabel("原始数据", "Raw Log")}, {"chat-node", paneLabel("对话分组", "Chat Group")}} {
					option := parts.Item(item.key).Label(item.label).Padding(4, 6)
					if option.Highlighted() {
						option.Background(chrome.Color(colors.Hover))
					}
					option.Children(func() { inspectorText(c, item.label) })
				}
			})
			if parts.Trigger.Changed() {
				p.changeInspectorMode(*mode, source.SessionID)
			}
			pick := inspectorButton(c.Key("inspector-pick"), paneLabel("在对话面板中选取元素", "Pick an element in Chat View")).Tooltip(paneLabel("在对话面板中选取元素", "Pick an element in Chat View")).Size(22, 22).Padding(3).Radius(4).Selected(p.navigation.Picking)
			pickInk := ink
			if pick.Hovered() || p.navigation.Picking {
				pick.Background(chrome.Color(colors.Hover))
				pickInk = chrome.Color(colors.Accent)
			}
			pick.Children(func() { ui.Icon(c, inspectorPickIcon).Size(16, 16).TextColor(pickInk) })
			if pick.Clicked() {
				p.toggleInspectorPicking()
			}
			count := strconv.Itoa(len(rows))
			if query := inspectorFilterQuery(p.Filter); query != "" {
				matches := 0
				for _, row := range rows {
					if strings.Contains(strings.ToLower(row.Type), strings.ToLower(query)) {
						matches++
					}
				}
				count = fmt.Sprintf("%d / %d", matches, len(rows))
			}
			inspectorText(c, "· "+count)
		})
		if hasMore {
			label := paneLabel("加载更早", "Load earlier")
			if busy {
				label = paneLabel("加载中…", "Loading…")
			}
			button := inspectorButton(c.Key("inspector-older"), label).Disabled(busy)
			button.Children(func() { inspectorText(c, label) })
			if button.Clicked() {
				p.list.FollowEnd = false
				if loadOlder != nil {
					loadOlder()
				}
			}
		}
		label := paneLabel("跟踪最新", "Follow Latest")
		button := inspectorButton(c.Key("inspector-latest"), label)
		button.Children(func() { inspectorText(c, label) })
		if button.Clicked() {
			p.list.FollowEnd = true
			p.list.ScrollToEnd()
		}
	})
}

func (p *SessionInspectorPaneState) changeInspectorMode(mode, sessionID string) {
	p.Mode = mode
	p.filterOpen, p.detailsDragging = false, false
	p.filterActive = -1
	p.navigation = chat.InspectorNavigation{SessionID: sessionID}
	p.pendingPick, p.pendingPickBinding = nil, ""
	p.breadcrumb, p.tree = nil, inspectorJSONTreeState{}
	p.Filter, p.draft, p.selected = "", "", ""
	p.collapsed = map[string]bool{}
	p.list = ui.ListState{FollowEnd: true}
	p.tableX, p.tableY = ui.ScrollState{}, ui.ScrollState{}
	p.detailsPercent = 0
}

func inspectorDepths(rows []session.InspectorRecord) (map[string]int, map[string]bool) {
	depth, branches := map[string]int{}, map[string]bool{}
	for _, row := range rows {
		if row.Parent != "" {
			depth[row.Key] = depth[row.Parent] + 1
			branches[row.Parent] = true
		}
	}
	return depth, branches
}

func inspectorStickyIndexes(rows []session.InspectorRecord, branches map[string]bool, offset float32) []int {
	byKey := make(map[string]int, len(rows))
	for i, row := range rows {
		byKey[row.Key] = i
	}
	ancestors := func(index int) []int {
		if index < 0 || index >= len(rows) {
			return nil
		}
		var out []int
		row := rows[index]
		if branches[row.Key] {
			out = append(out, index)
		}
		seen := map[string]bool{row.Key: true}
		for row.Parent != "" && !seen[row.Parent] {
			at, ok := byKey[row.Parent]
			if !ok {
				break
			}
			seen[row.Parent] = true
			out = append(out, at)
			row = rows[at]
		}
		for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
			out[l], out[r] = out[r], out[l]
		}
		return out
	}
	var pinned []int
	first := max(0, int(math.Floor(float64(offset/inspectorRowHeight))))
	for len(pinned) < len(rows) {
		next := ancestors(min(len(rows)-1, first+len(pinned)))
		if len(next) <= len(pinned) {
			break
		}
		match := true
		for level, index := range pinned {
			if next[level] != index {
				match = false
				break
			}
		}
		if !match {
			break
		}
		pinned = append(pinned, next[len(pinned)])
	}
	return pinned
}

// The horizontal viewport owns both the header and the virtual vertical list.
// Narrow docks retain the reference's 660px table rather than wrapping cells.
func inspectorTable(c *ui.Context, p *SessionInspectorPaneState, rows []session.InspectorRecord, colors theme.Colors) {
	depths, _ := inspectorDepths(rows)
	filtered := inspectorTypeRows(rows, p.Filter)
	_, branches := inspectorDepths(filtered)
	shown := inspectorDisclosedRows(filtered, p.collapsed)
	viewport := ui.ScrollHorizontal(c.Key("inspector-table-x")).Label("Inspector table horizontal scroll").TrackScroll(&p.tableX).Grow(1).MinHeight(0).FillWidth()
	width := max(float32(660), viewport.Bounds().W)
	viewport.Children(func() {
		ui.Column(c.Key("inspector-table")).Role(ui.RoleTable).Label(paneLabel("会话数据诊断", "Session Log")).Width(width).FillHeight().Font(inspectorFont()).FontSize(11).TextColor(chrome.Color(colors.FG)).Children(func() {
			inspectorTableHeader(c, p, rows, width, colors)
			host := ui.Box(c.Key("inspector-table-rows")).Grow(1).MinHeight(0).FillWidth().Clip()
			host.Children(func() {
				pinned := inspectorStickyIndexes(shown, branches, p.tableY.Y)
				isPinned := func(i int) bool {
					for _, index := range pinned {
						if i == index {
							return true
						}
					}
					return false
				}
				p.list.Key = func(i int) any { return shown[i].Key }
				p.list.Label = func(i int) string {
					if isPinned(i) {
						return ""
					}
					return inspectorRecordIdentity(shown[i]) + " · " + shown[i].Type
				}
				ui.List(c.Key("inspector-log-rows"), &p.list, len(shown), func(i int) {
					if isPinned(i) {
						// Keep the measured flow space; only the pinned instance owns
						// controls, focus, and accessible record content.
						ui.Box(c).Height(inspectorRowHeight).Invisible()
						return
					}
					row := shown[i]
					inspectorTableRow(c.Key("inspector-row:"+row.Key), p, row, depths[row.Key], branches[row.Key], false, width, colors)
				}).TrackScroll(&p.tableY).Fill().Children(func() {
					if len(shown) == 0 {
						ui.Text(c, paneLabel("暂无记录", "No records")).Padding(12).FontSize(11).TextColor(chrome.Color(colors.Muted))
					}
					// A child of the Y scroll owner receives wheel and scroll keys
					// from the fixed band through the ordinary input ancestor chain.
					if len(pinned) > 0 {
						ui.Column(c.Key("inspector-sticky-ancestors")).Absolute().Left(0).Right(0).Top(0).Children(func() {
							for _, index := range pinned {
								row := shown[index]
								inspectorTableRow(c.Key("inspector-pinned:"+row.Key), p, row, depths[row.Key], branches[row.Key], true, width, colors)
							}
						})
					}
				})
			})
		})
	})
}

func inspectorCell(c *ui.Context, width float32) ui.Element {
	return ui.Row(c).Role(ui.RoleCell).Width(width).Shrink(0).Height(inspectorRowHeight).Padding(4, 8).AlignItems(ui.Center).Clip()
}

func inspectorTableHeader(c *ui.Context, p *SessionInspectorPaneState, rows []session.InspectorRecord, width float32, colors theme.Colors) {
	ui.Row(c.Key("inspector-table-header")).Role(ui.RoleRow).Label("Inspector table header").FillWidth().Height(inspectorRowHeight).Shrink(0).FontWeight(700).Background(chrome.Color(colors.Field)).BorderWidth(0, 0, .5, 0).BorderColor(chrome.Color(colors.BorderL1)).Children(func() {
		inspectorCell(c, 64).Role(ui.RoleColumnHeader).Children(func() { inspectorText(c, paneLabel("序号", "Index")) })
		inspectorCell(c, 190).Role(ui.RoleColumnHeader).Gap(6).Children(func() { inspectorText(c, paneLabel("类型", "Type")); inspectorTypeFilter(c, p, rows) })
		inspectorCell(c, 65).Role(ui.RoleColumnHeader).Children(func() { inspectorText(c, "T/S") })
		if p.Mode != "chat-node" {
			inspectorCell(c, 100).Role(ui.RoleColumnHeader).Children(func() { inspectorText(c, paneLabel("时间 (UTC)", "Time (UTC)")) })
		}
		dataWidth := width - 319
		if p.Mode != "chat-node" {
			dataWidth -= 100
		}
		inspectorCell(c, dataWidth).Role(ui.RoleColumnHeader).Children(func() { inspectorText(c, paneLabel("数据", "Data")) })
	})
}

func inspectorTableRow(c *ui.Context, p *SessionInspectorPaneState, row session.InspectorRecord, depth int, branch, sticky bool, width float32, colors theme.Colors) {
	ink := chrome.Color(colors.FG)
	context := inspectorFilterQuery(p.Filter) != "" && !strings.Contains(strings.ToLower(row.Type), strings.ToLower(inspectorFilterQuery(p.Filter)))
	if context {
		ink = chrome.Color(colors.Muted)
	}
	container := ui.Row(c).Role(ui.RoleRow).Label("Inspector record "+row.Key).Checked(p.selected == row.Key).FillWidth().Height(inspectorRowHeight).Shrink(0).BorderWidth(0, 0, .5, 0).BorderColor(chrome.Color(colors.BorderL1))
	if container.Hovered() || p.selected == row.Key {
		container.Background(chrome.Color(colors.Hover))
	}
	if sticky {
		container.Background(chrome.Color(colors.Field))
		if p.selected == row.Key {
			ink = chrome.Color(colors.Accent)
		}
	}
	if context {
		container.Tooltip(paneLabel("匹配记录的父级上下文", "Parent context of a matching record"))
	}
	selectRow := func() { p.selectInspectorRow(row) }
	container.Focusable()
	if container.Clicked() || container.Shortcut(0, ui.KeyEnter) || container.Shortcut(0, ui.KeySpace) {
		selectRow()
	}
	container.Children(func() {
		identity := inspectorRecordIdentity(row)
		inspectorCell(c, 64).Children(func() { inspectorText(c, identity).TextColor(chrome.Color(colors.Muted)).Tooltip(identity).Focusable() })
		inspectorCell(c, 190).Padding(4, 8, 4, 8+float32(depth)*20).Children(func() {
			if branch {
				marker := "▾"
				label := paneLabel("收起子行", "Collapse children")
				if p.collapsed[row.Key] {
					marker = "▸"
					label = paneLabel("展开子行", "Expand children")
				}
				button := inspectorButton(c.Key("disclosure"), label+" "+identity).Width(16).Shrink(0).Expanded(!p.collapsed[row.Key])
				button.Children(func() { inspectorText(c, marker).TextColor(ink) })
				if button.Clicked() {
					if p.collapsed == nil {
						p.collapsed = map[string]bool{}
					}
					p.collapsed[row.Key] = !p.collapsed[row.Key]
					p.list.FollowEnd = false
				}
			}
			button := inspectorButton(c.Key("record"), row.Type).MinWidth(0).Shrink(1)
			button.Children(func() { inspectorText(c, row.Type).TextColor(ink) })
			if button.Clicked() {
				selectRow()
			}
		})
		inspectorCell(c, 65).Children(func() { inspectorText(c, inspectorRecordLocation(row)).TextColor(chrome.Color(colors.FG)) })
		if p.Mode != "chat-node" {
			inspectorCell(c, 100).Children(func() {
				text := ""
				if row.Time != 0 || inspectorHasTime(row.Raw) {
					text = time.UnixMilli(int64(row.Time)).UTC().Format("15:04:05.000")
				}
				inspectorText(c, text).TextColor(chrome.Color(colors.FG))
			})
		}
		dataWidth := width - 319
		if p.Mode != "chat-node" {
			dataWidth -= 100
		}
		inspectorCell(c, dataWidth).Children(func() {
			preview := inspectorPreview(row.Raw)
			button := inspectorButton(c.Key("preview"), preview).FillWidth().MinWidth(0)
			button.Children(func() { inspectorText(c, preview).TextColor(chrome.Color(colors.FG)) })
			if button.Clicked() {
				selectRow()
			}
		})
	})
}

func inspectorRecordIdentity(row session.InspectorRecord) string {
	if row.Identity != "" {
		return row.Identity
	}
	if index := strings.LastIndex(row.Key, "/chunk:"); index >= 0 {
		return row.Key[index+7:]
	}
	if strings.HasPrefix(row.Key, "event:") {
		return strings.TrimPrefix(row.Key, "event:")
	}
	if strings.HasPrefix(row.Key, "attempt:") {
		return strings.TrimPrefix(row.Key, "attempt:")
	}
	if row.Seq != 0 {
		return strconv.FormatFloat(row.Seq, 'f', -1, 64)
	}
	return strings.TrimPrefix(row.Key, "chat:")
}

func inspectorRecordLocation(row session.InspectorRecord) string {
	if row.Location != "" {
		return row.Location
	}
	for _, field := range inspectorFields(row.Raw) {
		if field.name == "data" {
			var turn, step string
			for _, value := range inspectorFields(field.raw) {
				if value.name == "turn" {
					turn = inspectorScalar(value.raw)
				}
				if value.name == "step" {
					step = inspectorScalar(value.raw)
				}
			}
			if turn != "" {
				if step != "" {
					return turn + "/" + step
				}
				return turn
			}
		}
	}
	return ""
}
