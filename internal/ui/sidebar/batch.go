package sidebar

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

// VisibleBatchTargets mirrors the rendered tree, including collapsed ancestor
// groups and the dedicated search surface. Hidden selections are never cleared.
func VisibleBatchTargets(m Model) []session.SessionBatchTarget {
	var out []session.SessionBatchTarget
	seen := map[string]bool{}
	add := func(id, title string) {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, session.SessionBatchTarget{ID: id, Title: title})
		}
	}
	if m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != "" {
		rows, _ := deriveSessionSearchResults(m.Snapshot, *m.SearchQuery)
		for _, row := range rows {
			add(row.Item.SessionID, row.Item.Title)
		}
		return out
	}
	hidden := map[string]bool{}
	for _, g := range session.BuildWorkspaceGroups(m.Snapshot, sidebarItems(m.Snapshot, m.SearchQuery)) {
		if hidden[g.ParentID] || m.Snapshot.WorkspaceLayout.Collapsed[g.Key] {
			hidden[g.Key] = true
			continue
		}
		for _, it := range g.Items {
			add(it.SessionID, it.Title)
		}
	}
	if !m.HideArchivedList {
		for _, id := range m.Snapshot.ArchivedSessionIDs {
			title := id
			for _, it := range m.Snapshot.Items {
				if it.SessionID == id {
					title = it.Title
					break
				}
			}
			add(id, title)
		}
	}
	return out
}
func batchRow(c *ui.Context, m Model, id, title string, fg ui.Color) bool {
	if !m.Batch.Enabled {
		return false
	}
	mark := "☐ "
	if _, ok := m.Batch.Selected[id]; ok {
		mark = "☑ "
	}
	if ui.Button(c.Key("batch-row-"+id), mark+title).Label("Select session "+id).FillWidth().Height(32).FontSize(theme.InterfaceSize(13)).TextColor(fg).Disabled(m.Batch.Busy || m.Batch.Confirming).Clicked() && m.OnBatchEdit != nil {
		m.OnBatchEdit("select", []session.SessionBatchTarget{{ID: id, Title: title}})
	}
	return true
}
func batchToolbar(c *ui.Context, m Model, fg ui.Color) {
	if m.OnBatchEdit == nil {
		return
	}
	b := m.Batch
	locked := b.Busy || b.Confirming
	label := workspaceText("选择会话", "Select sessions")
	if b.Enabled {
		label = workspaceText("退出多选", "Exit selection")
	}
	if ui.Button(c, label).Disabled(locked).Clicked() {
		m.OnBatchEdit("toggle", nil)
	}
	if !b.Enabled {
		return
	}
	ui.Text(c, fmt.Sprintf(workspaceText("已选择 %d 个会话", "%d sessions selected"), len(b.Selected))).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, workspaceText("选择所有可见", "Select visible")).Disabled(locked).Clicked() {
			m.OnBatchEdit("visible", VisibleBatchTargets(m))
		}
		if ui.Button(c, workspaceText("清除选择", "Clear selection")).Disabled(locked || len(b.Selected) == 0).Clicked() {
			m.OnBatchEdit("clear", nil)
		}
	})
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, workspaceText("批量归档", "Archive selected")).Disabled(locked || len(b.Selected) == 0).Clicked() && m.OnBatchRequest != nil {
			m.OnBatchRequest("archive")
		}
		if ui.Button(c, workspaceText("批量删除", "Delete selected")).Disabled(locked || len(b.Selected) == 0).Clicked() && m.OnBatchRequest != nil {
			m.OnBatchRequest("delete")
		}
	})
	if b.Confirming {
		warning := workspaceText("归档这些会话将停止其中正在进行的工作。之后可从归档列表恢复。", "Archiving these sessions stops their active work. You can restore them from the archive list.")
		if b.Kind == "delete" {
			warning = workspaceText("永久删除以下会话及其记录，无法撤销。只有已归档会话可删除；不符合条件的会话会保留并显示错误。", "Permanently delete these sessions and their history. This cannot be undone. Only archived sessions can be deleted; ineligible sessions remain with an error.")
		}
		ui.Text(c, warning).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		ui.Scroll(c).FillWidth().Height(110).Children(func() {
			for _, t := range b.Targets {
				ui.Text(c, t.Title+" · "+t.ID).FontSize(theme.InterfaceSize(11)).TextColor(fg)
			}
		})
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, workspaceText("确认批量操作", "Confirm batch")).Clicked() && m.OnBatchConfirm != nil {
				m.OnBatchConfirm(b.Token)
			}
			if ui.Button(c, workspaceText("取消", "Cancel batch")).Clicked() && m.OnBatchCancel != nil {
				m.OnBatchCancel(b.Token)
			}
		})
	}
	if b.Busy {
		ui.Text(c, fmt.Sprintf(workspaceText("处理中 %d / %d", "Processing %d / %d"), b.Attempted, len(b.Targets))).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if len(b.Results) > 0 {
		ui.Scroll(c).FillWidth().Height(100).Children(func() {
			for _, r := range b.Results {
				status := workspaceText("完成", "Done")
				if r.Error != "" {
					status = r.Error
				}
				ui.Text(c, r.Title+" · "+status).FontSize(theme.InterfaceSize(11)).TextColor(fg)
			}
		})
		if !b.Busy && ui.Button(c, workspaceText("关闭结果", "Dismiss results")).Clicked() {
			m.OnBatchEdit("dismiss", nil)
		}
	}
}
