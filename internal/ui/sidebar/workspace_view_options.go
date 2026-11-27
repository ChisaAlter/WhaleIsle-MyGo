package sidebar

import (
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
)

type workspaceViewChoice struct {
	id, label, group string
	selected         bool
	run              func()
	disabled         bool
}

// The shipped browser has one dense, grouped menu. Picking one choice closes
// it; all three sections read the same live store projection as the list/search.
func workspaceViewOptionsItems(m Model) ([]coss.MenuItem, func(string)) {
	group, order := m.Snapshot.WorkspaceLayout.GroupBy, m.Snapshot.WorkspaceLayout.OrderBy
	if group == "" {
		group = "workspace"
	}
	if order == "" {
		order = "updated"
	}
	filter := sharedArchiveFilter(m)
	onGrouping := m.OnWorkspaceGrouping
	onOrdering := m.OnWorkspaceOrdering
	onArchivedFilter := m.OnArchivedFilter
	browseFilter := m.BrowseArchivedFilter
	searchFilter := m.SearchArchivedFilter
	canFilter := canSetArchiveFilter(m)
	groupPick := func(value string) func() {
		return func() {
			if onGrouping != nil {
				onGrouping(value)
			}
		}
	}
	orderPick := func(value string) func() {
		return func() {
			if onOrdering != nil {
				onOrdering(value)
			}
		}
	}
	filterPick := func(value string) func() {
		return func() {
			setArchiveFilter(Model{OnArchivedFilter: onArchivedFilter, BrowseArchivedFilter: browseFilter, SearchArchivedFilter: searchFilter}, value)
		}
	}
	choices := []workspaceViewChoice{
		{"workspace", workspaceText("按工作区", "By workspace"), i18n.T(i18n.KeyWSGroupByLabel), group == "workspace", groupPick("workspace"), onGrouping == nil},
		{"workspace-tree", workspaceText("按工作区树", "Workspace tree"), "", group == "workspace-tree", groupPick("workspace-tree"), onGrouping == nil},
		{"flat", workspaceText("单列表", "Single list"), "", group == "flat", groupPick("flat"), onGrouping == nil},
		{"manual", workspaceText("手动排序", "Manual order"), i18n.T(i18n.KeyWSOrderByLabel), order == "manual", orderPick("manual"), onOrdering == nil},
		{"updated", workspaceText("最近更新", "Recently updated"), "", order == "updated", orderPick("updated"), onOrdering == nil},
		{"default", i18n.T(i18n.KeyWSViewOptionsHideArchived), i18n.T(i18n.KeyWSFilterByLabel), filter == "default", filterPick("default"), !canFilter},
		{"show", i18n.T(i18n.KeyWSViewOptionsShowArchived), "", filter == "show", filterPick("show"), !canFilter},
		{"only", workspaceText("仅显示已归档", "Only archived"), "", filter == "only", filterPick("only"), !canFilter},
	}
	items := make([]coss.MenuItem, 0, len(choices)+5)
	runners := make(map[string]func(), len(choices))
	for index, choice := range choices {
		if choice.group != "" {
			if index > 0 {
				items = append(items, coss.MenuItem{Kind: coss.MenuSeparator})
			}
			items = append(items, coss.MenuItem{Kind: coss.MenuLabel, Label: choice.group})
		}
		item := coss.MenuItem{
			Kind:     coss.MenuAction,
			ID:       choice.id,
			Label:    choice.label,
			Disabled: choice.disabled,
		}
		if choice.selected {
			item.Icon = coss.IconCheck
		} else {
			item.Inset = true
		}
		items = append(items, item)
		runners[choice.id] = choice.run
	}
	return items, func(id string) {
		if run := runners[id]; run != nil {
			run()
		}
	}
}
