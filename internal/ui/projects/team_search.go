package projects

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/projectflow"
	fonttheme "whaleisle/internal/ui/theme"
)

func teamMatches(query string, values ...string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
func teamTaskMatches(task projectflow.TeamTask, query string) bool {
	return teamMatches(query, task.Subject, task.Description, task.OwnerName, task.Status)
}
func teamWorkMatches(work projectflow.Workstream, query string) bool {
	values := []string{work.Title, work.Status, work.BlockedReason, waitingReasonLabel(work.Delegation.WaitingReason)}
	if work.SummaryFailure != nil {
		values = append(values, work.SummaryFailure.Error.Message)
	}
	return teamMatches(query, values...)
}
func teamMemberMatches(o *projectflow.Overview, member projectflow.TeamMember, query string) bool {
	if teamMatches(query, member.Name, member.Description, member.Role, member.Error, o.MemberState(member), stateLabel(o.MemberState(member))) {
		return true
	}
	for _, worker := range o.Workers {
		if worker.SessionID == member.ID && teamMatches(query, worker.Error) {
			return true
		}
	}
	for _, work := range o.Workstreams {
		if work.WorkerSessionID != member.ID {
			continue
		}
		if teamWorkMatches(work, query) {
			return true
		}
		for _, task := range o.Team.Tasks {
			if task.ID == work.ID && teamTaskMatches(task, query) {
				return true
			}
		}
	}
	for _, task := range o.Team.Tasks {
		if member.Name != "" && task.OwnerName == member.Name && teamTaskMatches(task, query) {
			return true
		}
	}
	return false
}
func teamSearchControls(c *ui.Context, m *Model) {
	label := tr("查找成员、任务或工作", "Find members, tasks or work")
	if ui.TextInput(c, &m.TeamQuery).FillWidth().Label(label).Placeholder(label).Changed() && m.OnTeamQuery != nil {
		m.OnTeamQuery(m.TeamQuery)
	}
	if m.TeamQuery != "" && ui.Button(c, tr("清除查找", "Clear search")).Clicked() && m.OnTeamQuery != nil {
		m.TeamQuery = ""
		m.OnTeamQuery("")
	}
	if strings.TrimSpace(m.TeamQuery) != "" {
		ui.Text(c, tr("查询全部已加载成员与 Team 任务；工作流按当前页查找。", "Search all loaded members and Team tasks; workstreams are searched on the current page.")).FontSize(fonttheme.InterfaceSize(11))
	}
}
