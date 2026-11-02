package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

var workspaceTreeMenuArt = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"><path d="M3 2v9h3M3 6h3"/><rect x="6" y="3.5" width="7" height="4" rx=".7"/><rect x="6" y="9" width="7" height="4" rx=".7"/></svg>`))
var workspaceFlatMenuArt = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-linecap="round"><path d="M5 4h8M5 8h8M5 12h8M2 4h.1M2 8h.1M2 12h.1"/></svg>`))
var workspaceOrderMenuArt = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"><path d="m5 5 3-3 3 3M8 2v12m-3-3 3 3 3-3"/></svg>`))

type workspaceViewChoice struct {
	id, label, group string
	art              *ui.SVG
	selected         bool
	run              func()
	disabled         bool
}

// The shipped browser has one dense, grouped menu. Picking one choice closes
// it; all three sections read the same live store projection as the list/search.
func workspaceViewOptionsMenu(c *ui.Context, m Model, open *bool) {
	group, order := m.Snapshot.WorkspaceLayout.GroupBy, m.Snapshot.WorkspaceLayout.OrderBy
	if group == "" {
		group = "workspace"
	}
	if order == "" {
		order = "updated"
	}
	filter := sharedArchiveFilter(m)
	groupPick := func(value string) func() {
		return func() {
			if m.OnWorkspaceGrouping != nil {
				m.OnWorkspaceGrouping(value)
			}
		}
	}
	orderPick := func(value string) func() {
		return func() {
			if m.OnWorkspaceOrdering != nil {
				m.OnWorkspaceOrdering(value)
			}
		}
	}
	filterPick := func(value string) func() { return func() { setArchiveFilter(m, value) } }
	choices := []workspaceViewChoice{
		{"workspace", workspaceText("按工作区", "By workspace"), workspaceText("分组方式", "Group by"), workspaceFolderClosedArtwork, group == "workspace", groupPick("workspace"), m.OnWorkspaceGrouping == nil},
		{"workspace-tree", workspaceText("按工作区树", "Workspace tree"), "", workspaceTreeMenuArt, group == "workspace-tree", groupPick("workspace-tree"), m.OnWorkspaceGrouping == nil},
		{"flat", workspaceText("单列表", "Single list"), "", workspaceFlatMenuArt, group == "flat", groupPick("flat"), m.OnWorkspaceGrouping == nil},
		{"manual", workspaceText("手动排序", "Manual order"), workspaceText("排序方式", "Order by"), workspaceOrderMenuArt, order == "manual", orderPick("manual"), m.OnWorkspaceOrdering == nil},
		{"updated", workspaceText("最近更新", "Recently updated"), "", sidebarRailIcons["clock"], order == "updated", orderPick("updated"), m.OnWorkspaceOrdering == nil},
		{"default", workspaceText("隐藏已归档", "Hide archived"), workspaceText("筛选列表与搜索", "Filter list and search"), archiveArtwork, filter == "default", filterPick("default"), !canSetArchiveFilter(m)},
		{"show", workspaceText("全部对话（显示已归档）", "All conversations (show archived)"), "", workspaceFlatMenuArt, filter == "show", filterPick("show"), !canSetArchiveFilter(m)},
		{"only", workspaceText("仅显示已归档", "Only archived"), "", archiveArtwork, filter == "only", filterPick("only"), !canSetArchiveFilter(m)},
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	for index, choice := range choices {
		if choice.group != "" {
			if index > 0 {
				ui.Divider(c).Margin(4, 0)
			}
			ui.Text(c.Key("view-label-"+choice.id), choice.group).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(4, 8).SingleLine()
		}
		item := ui.ButtonBase(c.Key("view-choice-"+choice.id)).Label(choice.label).Role(ui.RoleMenuItem).
			FillWidth().Height(32).Padding(4, 8).Gap(8).Radius(6).BorderWidth(0).
			Background(ui.Transparent).AlignItems(ui.Center).Disabled(choice.disabled)
		if item.Hovered() || item.Focused() {
			item.Background(chrome.Color(colors.Hover))
		}
		item.Children(func() {
			ui.Icon(c, choice.art).Size(16, 16).TextColor(muted)
			ui.Text(c, choice.label).FontSize(theme.InterfaceSize(13)).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
			ui.Box(c).Width(12).Height(16).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				if !choice.selected {
					return
				}
				path := &ui.Path{}
				path.MoveTo(r.X+1, r.Y+8)
				path.LineTo(r.X+4, r.Y+11)
				path.LineTo(r.X+11, r.Y+4)
				p.StrokePath(path, 1.3, fg)
			})
		})
		if item.Clicked() {
			choice.run()
			if open != nil {
				*open = false
			}
		}
	}
}
