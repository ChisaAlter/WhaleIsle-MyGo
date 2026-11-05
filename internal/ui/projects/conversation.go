package projects

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/theme"
)

func ConversationProject(m Model, sessionID, parentSessionID string) (projectflow.Project, bool, bool) {
	for _, p := range m.Listing.Projects {
		if p.CoordinatorSessionID == sessionID {
			return p, true, false
		}
		if parentSessionID != "" && p.CoordinatorSessionID == parentSessionID && sessionID != parentSessionID {
			return p, true, true
		}
	}
	return projectflow.Project{}, false, false
}

// ConversationComposer returns true only when Project owns the input seat:
// a read-only member or an archived/archiving Lead. A ready Lead keeps the
// ordinary native composer, including its existing Host project-level cancel.
func ConversationComposer(c *ui.Context, m Model, sessionID, parentSessionID string) bool {
	p, known, member := ConversationProject(m, sessionID, parentSessionID)
	if !known || !member && p.Lifecycle != "archived" && !p.Archiving {
		return false
	}
	ui.Column(c).FillWidth().Padding(12).Gap(8).Children(func() {
		if member {
			ui.Text(c, tr("这是项目成员的只读执行记录。请回到 Lead 分配工作。", "This Project member record is read-only. Return to the Lead to assign work.")).FontSize(theme.InterfaceSize(13))
			if ui.Button(c, tr("返回 Lead", "Back to Lead")).Disabled(m.Busy || m.OnOpen == nil).Clicked() {
				m.OnOpen(p.ID)
			}
			ProjectStopButton(c, m, p.CoordinatorSessionID)
		} else {
			label := tr("项目已归档；工作目录、对话和成果均保留。", "This Project is archived. Its directory, conversations and results are retained.")
			if p.Archiving {
				label = tr("正在停止后台工作并归档…", "Stopping background work and archiving…")
			}
			ui.Text(c, label).FontSize(theme.InterfaceSize(13))
			if ui.Button(c, tr("恢复项目", "Restore Project")).Disabled(m.Busy || p.Archiving || m.OnRestoreProject == nil).Clicked() {
				m.OnRestoreProject(p.ID)
			}
		}
		if m.Error != "" {
			ui.Text(c, m.Error).FontSize(theme.InterfaceSize(12))
		}
	})
	return true
}

// ProjectStopButton is shared by the input-right and approval action seats.
func ProjectStopButton(c *ui.Context, m Model, leadSessionID string) bool {
	p, known, _ := ConversationProject(m, leadSessionID, "")
	if !known || p.Lifecycle != "ready" {
		return false
	}
	active := p.Activity.CoordinatorRunning || p.Activity.Running > 0 || p.Activity.Queued > 0 || p.Activity.Provisioning > 0 || p.Activity.RunningJobs > 0
	if !active {
		return false
	}
	if ui.Button(c, tr("停止项目工作", "Stop Project work")).Disabled(m.Busy || p.Archiving || m.OnStopProject == nil).Clicked() {
		m.OnStopProject(p.ID)
	}
	return true
}
