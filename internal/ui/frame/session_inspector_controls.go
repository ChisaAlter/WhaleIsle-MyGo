package frame

import (
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/store/session"
)

func inspectorTypeCandidates(rows []session.InspectorRecord, query string) []string {
	query = strings.ToLower(strings.Trim(strings.TrimSpace(query), "* "))
	seen := map[string]bool{}
	var out []string
	for _, row := range rows {
		if row.Type != "" && !seen[row.Type] && strings.Contains(strings.ToLower(row.Type), query) {
			seen[row.Type] = true
			out = append(out, row.Type)
		}
	}
	sort.Strings(out)
	return out
}
func (p *SessionInspectorPaneState) applyInspectorFilter(query string) {
	p.Filter, p.draft = query, query
	p.filterOpen = false
	p.filterActive = -1
	p.selected = ""
	p.breadcrumb = nil
	p.tree = inspectorJSONTreeState{}
	p.collapsed = map[string]bool{}
	p.list = ui.ListState{}
	p.list.ScrollTo(0, ui.Start)
}
func (p *SessionInspectorPaneState) cancelFilter() {
	p.draft = p.Filter
	p.filterOpen = false
	p.filterActive = -1
}
func (p *SessionInspectorPaneState) moveFilterCandidate(delta, count int) {
	if count == 0 {
		p.filterActive = -1
		return
	}
	if p.filterActive < 0 || p.filterActive >= count {
		if delta < 0 {
			p.filterActive = count - 1
		} else {
			p.filterActive = 0
		}
	} else {
		p.filterActive = (p.filterActive + delta + count) % count
	}
	p.filterList.ScrollTo(p.filterActive, ui.Center)
}
func (p *SessionInspectorPaneState) confirmFilter(items []string) {
	query := p.draft
	if p.filterActive >= 0 && p.filterActive < len(items) {
		query = items[p.filterActive]
	}
	p.applyInspectorFilter(query)
}
func inspectorTypeFilter(c *ui.Context, p *SessionInspectorPaneState, rows []session.InspectorRecord) {
	ui.Row(c).Gap(6).Padding(6).Children(func() {
		anchor := ui.Button(c, paneLabel("事件类型筛选", "Filter event type")).Label("Inspector type filter").Selected(p.Filter != "")
		if anchor.Clicked() {
			if p.filterOpen {
				p.cancelFilter()
			} else {
				p.filterOpen = true
				p.draft = p.Filter
				p.filterQuery = p.draft
				p.filterActive = -1
				p.filterList = ui.ListState{}
			}
		}
		if p.Filter != "" {
			ui.Text(c, "*"+p.Filter+"*").Grow(1)
		}
		ui.Popover(c, anchor, &p.filterOpen, func() {
			ui.Column(c).Width(300).Gap(6).Children(func() {
				input := ui.TextInput(c, &p.draft).FillWidth().Label("Inspector type query").Placeholder(paneLabel("事件类型片段", "Type fragment")).AutoFocus()
				if p.filterQuery != p.draft {
					p.filterQuery = p.draft
					p.filterActive = -1
					p.filterList = ui.ListState{}
				}
				items := inspectorTypeCandidates(rows, p.draft)
				if !input.Composing() {
					if input.Shortcut(0, ui.KeyDown) {
						p.moveFilterCandidate(1, len(items))
					}
					if input.Shortcut(0, ui.KeyUp) {
						p.moveFilterCandidate(-1, len(items))
					}
					if input.Submitted() || input.Shortcut(0, ui.KeyEnter) {
						p.confirmFilter(items)
						anchor.Focus()
					}
				}
				ui.Text(c, paneLabel("仅应用或选择候选后筛选；Esc 取消", "Apply or choose a candidate to filter; Esc cancels")).FontSize(11)
				if len(items) == 0 {
					ui.Text(c, paneLabel("没有匹配类型", "No matching types"))
				} else {
					p.filterList.Key = func(i int) any { return items[i] }
					ui.List(c, &p.filterList, len(items), func(i int) {
						if ui.Button(c, items[i]).FillWidth().Selected(i == p.filterActive).Tooltip(items[i]).Clicked() {
							p.applyInspectorFilter(items[i])
							anchor.Focus()
						}
					}).Height(min(float32(160), float32(len(items)*32))).FillWidth()
				}
				ui.Row(c).Gap(6).Children(func() {
					if ui.Button(c, paneLabel("清除", "Clear")).Clicked() {
						p.draft = ""
						p.filterActive = -1
						input.Focus()
					}
					if ui.Button(c, paneLabel("取消", "Cancel")).Clicked() {
						p.cancelFilter()
						anchor.Focus()
					}
					if ui.Button(c, paneLabel("应用", "Apply")).Clicked() {
						p.applyInspectorFilter(p.draft)
						anchor.Focus()
					}
				})
			})
		})
		if !p.filterOpen {
			p.draft = p.Filter
			p.filterActive = -1
		}
	})
}
func (p *SessionInspectorPaneState) inspectorDetailsPercent() float32 {
	if p.detailsPercent == 0 {
		return 40
	}
	return min(float32(80), max(float32(15), p.detailsPercent))
}
func (p *SessionInspectorPaneState) resizeInspectorDetails(dy, available float32) {
	if available > 0 {
		p.detailsPercent = min(float32(80), max(float32(15), p.inspectorDetailsPercent()-dy/available*100))
	}
}

// Freeze the pointer origin and available height for the whole drag, so an
// excursion beyond a bound does not move the origin when the pointer returns.
func (p *SessionInspectorPaneState) dragInspectorDetails(dy, available float32, held bool) {
	if !held {
		p.detailsDragging = false
		return
	}
	if !p.detailsDragging {
		p.detailsDragging = true
		p.detailsDragBase = p.inspectorDetailsPercent()
		p.detailsDragDelta = 0
		p.detailsDragAvailable = available
	}
	p.detailsDragDelta += dy
	p.detailsPercent = p.detailsDragBase
	p.resizeInspectorDetails(p.detailsDragDelta, p.detailsDragAvailable)
}
func (p *SessionInspectorPaneState) detailsKey(key ui.Key) {
	value := p.inspectorDetailsPercent()
	switch key {
	case ui.KeyUp:
		value += 5
	case ui.KeyDown:
		value -= 5
	case ui.KeyHome:
		value = 15
	case ui.KeyEnd:
		value = 80
	}
	p.detailsPercent = min(float32(80), max(float32(15), value))
}
func inspectorDetailsDivider(c *ui.Context, p *SessionInspectorPaneState, available float32) {
	rail := ui.Box(c).FillWidth().Height(6).Shrink(0).Cursor(ui.CursorResizeNS).Focusable().Role(ui.RoleSplitter).Label("Inspector details divider").Range(15, 80, float64(p.inspectorDetailsPercent())).Step(5)
	for _, key := range []ui.Key{ui.KeyUp, ui.KeyDown, ui.KeyHome, ui.KeyEnd} {
		if rail.Shortcut(0, key) {
			p.detailsKey(key)
		}
	}
	_, dy, held := rail.Dragged()
	if held {
		rail.Focus()
	}
	p.dragInspectorDetails(dy, available, held)
	rail.Draw(func(painter *ui.Painter, r ui.Rect) {
		color := c.Theme().Border
		if held || rail.Hovered() {
			color = c.Theme().Accent
		}
		painter.Fill(r, color, 0)
		if rail.FocusVisible() {
			painter.FocusRing(r, [4]float32{})
		}
	})
}
