package frame

import (
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
)

func inspectorTypeCandidates(rows []session.InspectorRecord, query string) []string {
	query = strings.ToLower(inspectorFilterQuery(query))
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
	query = inspectorFilterQuery(query)
	p.Filter, p.draft = query, query
	p.filterOpen = false
	p.filterActive = -1
	p.closeInspectorDetails()
	p.collapsed = map[string]inspectorDisclosure{}
	p.list = ui.ListState{}
	previousOffset := p.tableY.Y
	p.tableY = ui.ScrollState{}
	p.scroll = inspectorScrollState{mode: "scroll", requestedFrom: &previousOffset}
	p.pulse.revealKey = ""
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
	ui.Row(c).Children(func() {
		anchor := inspectorButton(c, "Inspector type filter").Tooltip(paneLabel("筛选类型", "Filter types")).Size(17, 17).Padding(2).Radius(3).Selected(inspectorFilterQuery(p.Filter) != "").Expanded(p.filterOpen)
		ink := c.Theme().Text
		if anchor.Hovered() || inspectorFilterQuery(p.Filter) != "" {
			anchor.Background(c.Theme().SurfaceHover)
			ink = c.Theme().Accent
		}
		anchor.Children(func() { ui.Icon(c, inspectorFilterIcon).Size(13, 13).TextColor(ink) })
		if anchor.Clicked() {
			if p.filterOpen {
				p.cancelFilter()
			} else {
				p.filterOpen = true
				p.draft = p.Filter
				p.filterQuery = p.draft
				p.filterActive = -1
				p.filterHovered = -1
				p.filterList = ui.ListState{}
			}
		}
		ui.PopoverBase(c, anchor, &p.filterOpen, func(panel ui.Element) {
			width, height := c.Size()
			panel = chrome.PopoverReveal(c, panel.Width(290).MaxWidth(width-24).MaxHeight(height-24).Margin(4, 0, 0, 0).Padding(10).Radius(6).Background(c.Theme().Surface).Border(.5, c.Theme().Border).FontSize(11).FontWeight(400))
			ui.Column(c).FillWidth().Children(func() {
				input := ui.TextInputBase(c, &p.draft).FillWidth().Label("Inspector type query").Placeholder(paneLabel("输入类型片段，如 delta", "Type a fragment, e.g. delta")).AutoFocus().Padding(5, 6).Border(.5, c.Theme().Border).Radius(4).Background(ui.Transparent).FontSize(11)
				if p.filterQuery != p.draft {
					p.filterQuery = p.draft
					p.filterActive = -1
					p.filterHovered = -1
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
				ui.Text(c, paneLabel("默认包含匹配（*关键词*）；Enter 确认，Esc 取消。", "Contains matching (*keyword*). Enter confirms; Esc cancels.")).FontSize(11).FontWeight(400).Margin(6, 0).TextColor(c.Theme().TextMuted)
				if len(items) == 0 {
					ui.Text(c, paneLabel("没有匹配候选，仍可确认当前关键词。", "No suggestions. You can still apply this keyword.")).FontSize(11)
				} else {
					p.filterList.Key = func(i int) any { return items[i] }
					hoveredItem := -1
					ui.List(c, &p.filterList, len(items), func(i int) {
						row := inspectorButton(c, items[i]).FillWidth().Padding(4, 6).Selected(i == p.filterActive).Tooltip(items[i])
						if row.Hovered() {
							hoveredItem = i
							if p.filterHovered != i {
								p.filterHovered = i
								p.filterActive = i
							}
						}
						ink := c.Theme().Text
						if i == p.filterActive || (p.filterActive < 0 && row.Hovered()) {
							row.Background(c.Theme().SurfaceHover)
							ink = c.Theme().Accent
						}
						row.Children(func() { inspectorText(c, items[i]).Grow(1).TextColor(ink) })
						if row.Clicked() {
							p.applyInspectorFilter(items[i])
							anchor.Focus()
						}
					}).Height(min(float32(180), float32(len(items)*22))).FillWidth()
					p.filterHovered = hoveredItem
				}
				action := func(label string) ui.Element {
					button := inspectorButton(c, label).Padding(3, 6).Border(.5, c.Theme().Border).Radius(4)
					button.Children(func() { inspectorText(c, label) })
					return button
				}
				ui.Row(c).Gap(8).Margin(8, 0, 0, 0).Justify(ui.End).Children(func() {
					if action(paneLabel("清空", "Clear")).Clicked() {
						p.draft = ""
						p.filterActive = -1
						input.Focus()
					}
					if action(paneLabel("取消", "Cancel")).Clicked() {
						p.cancelFilter()
						anchor.Focus()
					}
					if action(paneLabel("确认筛选", "Apply filter")).Clicked() {
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
	rail := ui.Box(c).FillWidth().Height(5).Shrink(0).Cursor(ui.CursorResizeNS).Focusable().Role(ui.RoleSplitter).Label("Inspector details divider").Range(15, 80, float64(p.inspectorDetailsPercent())).Step(5)
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
