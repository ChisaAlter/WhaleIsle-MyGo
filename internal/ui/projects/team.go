package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
)

func stateLabel(state string) string {
	labels := map[string][2]string{
		"ready": {"就绪", "Ready"}, "running": {"运行中", "Running"}, "assigned": {"已分配，未运行", "Assigned, not running"}, "completed": {"工作已完成", "Work completed"}, "delivered": {"已交付", "Delivered"}, "reported": {"已提交报告", "Report submitted"}, "paused": {"已停止", "Stopped"}, "stopping": {"正在停止", "Stopping"}, "provisioning": {"准备中", "Preparing"}, "queued": {"排队中", "Queued"}, "blocked": {"受阻", "Blocked"}, "failed": {"失败", "Failed"}, "summaryFailed": {"汇总失败", "Summary failed"}, "pendingSummary": {"等待汇总", "Awaiting summary"}, "needsReview": {"依赖已改变，需复核", "Dependencies changed; review needed"}, "unavailable": {"运行状态不可用", "Runtime status unavailable"}, "pending": {"待处理", "Pending"}, "in_progress": {"进行中", "In progress"}}
	if pair, ok := labels[state]; ok {
		return tr(pair[0], pair[1])
	}
	return state
}
func teamView(c *ui.Context, m Model) {
	if m.Detail != nil {
		detailView(c, m)
		return
	}
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("返回项目", "Back to Projects")).Clicked() && m.OnCloseTeam != nil {
			m.OnCloseTeam()
		}
		if ui.Button(c, tr("刷新 Team", "Refresh Team")).Disabled(m.Busy).Clicked() && m.OnTeam != nil {
			m.OnTeam(m.TeamID)
		}
	})
	if m.Team == nil {
		if !m.Busy {
			ui.Text(c, tr("Team 当前不可用，请刷新重试。", "Team is unavailable; refresh to retry.")).FontSize(12)
		}
		return
	}
	o := m.Team
	ui.Text(c, o.Project.Title+" · Team").FontSize(13)
	ui.Text(c, tr("最近读取的状态；刷新以获取最新进展。", "Last fetched state; refresh for current progress.")).FontSize(11)
	if m.TeamNote != "" {
		ui.Text(c, m.TeamNote).FontSize(12)
	}
	disabled := m.Busy || !m.Listing.Available || o.Team == nil || o.Team.Failure != "" || o.Project.Lifecycle != "ready"
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("停止全部工作", "Stop all work")).Disabled(disabled).Clicked() && m.OnControl != nil {
			m.OnControl("stop", "")
		}
		if o.Project.Paused && ui.Button(c, tr("解除暂停", "Unpause")).Disabled(disabled).Clicked() && m.OnControl != nil {
			m.OnControl("resume", "")
		}
	})
	if o.Project.Paused {
		ui.Text(c, tr("解除暂停后需在 Lead 对话中发新指令；不会自动重启旧任务。", "After unpausing, send a new instruction in the Lead conversation. Old work is not restarted automatically.")).FontSize(11)
	}
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			if o.Team == nil {
				return
			}
			ui.Text(c, fmt.Sprintf(tr("成员（%d）", "Members (%d)"), len(o.Team.Members))).FontSize(12)
			for _, member := range o.Team.Members {
				member := member
				name := member.Name
				if member.Role == "lead" {
					name = "Lead"
				}
				if name == "" {
					name = member.ID
				}
				ui.Text(c, name+" · "+stateLabel(o.MemberState(member))).FontSize(12)
				if member.Role != "lead" && ui.Button(c, tr("查看成员", "View member")).Disabled(m.Busy).Clicked() && m.OnInspect != nil {
					m.OnInspect("", member.ID)
				}
				if member.Description != "" {
					ui.Text(c, member.Description).FontSize(11)
				}
				if member.Error != "" {
					ui.Text(c, member.Error).FontSize(11)
				}
			}
			ui.Text(c, fmt.Sprintf(tr("Team 任务（%d）", "Team tasks (%d)"), len(o.Team.Tasks))).FontSize(12)
			for _, task := range o.Team.Tasks {
				ui.Text(c, task.Subject+" · "+stateLabel(task.Status)).FontSize(12)
				owner := task.OwnerName
				if owner == "" {
					owner = tr("未分配", "Unassigned")
				}
				ui.Text(c, owner).FontSize(11)
				if task.Description != "" {
					ui.Text(c, task.Description).FontSize(11)
				}
				if len(task.BlockedBy) > 0 {
					ui.Text(c, tr("依赖：", "Dependencies: ")+strings.Join(task.BlockedBy, ", ")).FontSize(11)
				}
				if len(task.WriteScopes) > 0 {
					ui.Text(c, tr("写入范围：", "Write scopes: ")+strings.Join(task.WriteScopes, ", ")).FontSize(11)
				}
				for _, warning := range task.WriteScopeWarnings {
					ui.Text(c, warning).FontSize(11)
				}
			}
			ui.Text(c, fmt.Sprintf(tr("工作流（本页 %d；当前总数 %d）", "Workstreams (%d on page; %d current total)"), len(o.Workstreams), o.WorkPage.CurrentTotal)).FontSize(12)
			for _, work := range o.Workstreams {
				work := work
				if ui.Button(c, tr("查看工作详情", "View work details")).Disabled(m.Busy).Clicked() && m.OnInspect != nil {
					m.OnInspect(work.ID, "")
				}
				ui.Text(c, work.Title+" · "+stateLabel(work.Status)+" · "+work.Group).FontSize(12)
				if work.BlockedReason != "" {
					ui.Text(c, work.BlockedReason).FontSize(11)
				}
				if work.SummaryFailure != nil {
					ui.Text(c, work.SummaryFailure.Error.Message).FontSize(11)
				}
				live := false
				for _, worker := range o.Workers {
					if worker.SessionID == work.WorkerSessionID && !worker.Stopped {
						live = worker.ActiveRunID != "" || len(worker.Jobs) > 0 || worker.Phase == "stopping" || work.Delegation.Phase == "queued" || work.Delegation.Phase == "preparing"
					}
				}
				if live && ui.Button(c, tr("停止这项工作", "Stop this work")).Disabled(disabled).Clicked() && m.OnControl != nil {
					m.OnControl("stop", work.ID)
				}
			}
			if o.WorkPage.CurrentNextCursor != nil && ui.Button(c, tr("下一页当前工作", "Next current-work page")).Disabled(disabled).Clicked() && m.OnNextWork != nil {
				m.OnNextWork(*o.WorkPage.CurrentNextCursor)
			}
		})
	})
}
