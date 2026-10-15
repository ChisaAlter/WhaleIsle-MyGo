package projects

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
)

// filteredProjects preserves the Host listing's order and identities. Search is
// a literal, case-insensitive title substring, with no trimming or normalization.
// Only the exact archived lifecycle belongs to history; transient states remain
// in the unarchived list. Neither the listing nor session selection is changed.
func filteredProjects(projects []projectflow.Project, query string, history bool) []projectflow.Project {
	query = strings.ToLower(query)
	var result []projectflow.Project
	for _, p := range projects {
		if (p.Lifecycle == "archived") == history && (query == "" || strings.Contains(strings.ToLower(p.Title), query)) {
			result = append(result, p)
		}
	}
	return result
}

func projectFilterControls(c *ui.Context, m *Model) {
	label := tr("搜索 Project 标题", "Search Project titles")
	if ui.TextInput(c, &m.SearchQuery).FillWidth().Label(label).Placeholder(label).Changed() && m.OnSearchQuery != nil {
		m.OnSearchQuery(m.SearchQuery)
	}
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("未归档", "Unarchived")).Disabled(!m.History).Clicked() && m.OnHistory != nil {
			m.History = false
			m.OnHistory(false)
		}
		if ui.Button(c, tr("已归档", "Archived")).Disabled(m.History).Clicked() && m.OnHistory != nil {
			m.History = true
			m.OnHistory(true)
		}
		if m.SearchQuery != "" && ui.Button(c, tr("清除搜索", "Clear search")).Clicked() && m.OnSearchQuery != nil {
			m.SearchQuery = ""
			m.OnSearchQuery("")
		}
	})
}

func projectEmptyLabel(m Model) string {
	if len(m.Listing.Projects) == 0 {
		return tr("尚无 Project", "No Projects yet")
	}
	if m.SearchQuery != "" {
		return tr("没有匹配的 Project", "No matching Projects")
	}
	if m.History {
		return tr("暂无已归档 Project", "No archived Projects")
	}
	return tr("暂无未归档 Project", "No unarchived Projects")
}
