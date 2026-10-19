package projects

import (
	"fmt"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// RequestCardActions keeps every card operation on the exact Host-admitted
// request revision, independently of other expanded cards.
type RequestCardActions struct {
	Known                   bool
	OpenFile                func(projectflow.Artifact, func(string))
	WorkDiff, WorkDetail    func(string)
	HandoverRetry, Activity func()
}

func requestCardActions(m Model, card projectflow.RequestCard) RequestCardActions {
	if m.RequestCardActions != nil {
		return m.RequestCardActions(card.ID, card.Revision)
	}
	return RequestCardActions{}
}

// Mode belongs to the observer/request identity, not its position in a page.
// Completion changes reset it just as the source observer's completed-bit guard.
func (o *ObserverState) requestMode(card projectflow.RequestCard, initial string) RequestMode {
	if o.RequestModes == nil {
		o.RequestModes = map[string]RequestMode{}
	}
	mode, ok := o.RequestModes[card.ID]
	if !ok || mode.Completed != (card.Status == "completed") {
		mode = RequestMode{Mode: initial, Completed: card.Status == "completed", Tab: "overview"}
		o.RequestModes[card.ID] = mode
	}
	if mode.Tab == "" {
		mode.Tab = "overview"
		o.RequestModes[card.ID] = mode
	}
	return mode
}

func observerRequests(c *ui.Context, m Model, o *ObserverState, p projectflow.Project) {
	queue := m.RequestQueue
	if queue == nil {
		queue = p.Queue
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		for _, tab := range []struct{ id, label string }{{"current", tr("当前", "Current")}, {"completed", tr("已交付", "Delivered")}} {
			label := tab.label
			if m.RequestKind == tab.id {
				label = "✓ " + label
			}
			if ui.Button(c, label).Height(28).FontSize(theme.InterfaceSize(12)).Disabled(m.Busy).Clicked() && m.OnRequestList != nil {
				m.OnRequestList(tab.id)
			}
		}
		ui.Button(c, tr("任务历史", "Task history")).Height(28).FontSize(theme.InterfaceSize(12)).Menu(func(menu *ui.Menu) {
			for _, entry := range []struct{ id, label string }{{"cancelled", tr("已取消", "Cancelled")}, {"archived", tr("已归档", "Archived")}} {
				if menu.Item(entry.label).Disabled(m.Busy || m.OnRequestList == nil).Chosen() && m.OnRequestList != nil {
					m.OnRequestList(entry.id)
				}
			}
		})
	})
	if m.RequestLoaded && len(m.RequestPage.Items) == 0 && (queue == nil || queue.Active == nil) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(14, 16).Gap(8).Children(func() {
			ui.Text(c, tr("任务会显示在这里", "Your tasks appear here")).FontSize(theme.InterfaceSize(14)).FontWeight(600)
			ui.Text(c, tr("在 Lead 对话中描述目标，查看任务进展与交付。", "Describe a goal in the Lead conversation to follow its progress and delivery.")).FontSize(theme.InterfaceSize(13))
		})
	}
	activeID := ""
	if queue != nil && queue.Active != nil {
		activeID = queue.Active.ID
	}
	if activeID != "" {
		active := *queue.Active
		for _, item := range m.RequestPage.Items {
			if item.ID == activeID {
				active = item
				break
			}
		}
		if active.ProjectID == "" {
			active.ProjectID = p.ID
		}
		observerTaskCard(c.Key(p.ID+"/"+active.ID), m, o, p, active, "card", queue)
	}
	for _, item := range m.RequestPage.Items {
		if item.ID == activeID {
			continue
		}
		initial := "strip"
		if item.ID == activeID {
			initial = "card"
		}
		observerTaskCard(c.Key(p.ID+"/"+item.ID), m, o, p, item, initial, queue)
	}
	if m.RequestPageNumber > 1 || m.RequestPage.NextCursor != nil {
		ui.Row(c).FillWidth().Wrap().Gap(4).Children(func() {
			if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || m.RequestPageNumber <= 1).Clicked() && m.OnRequestPage != nil {
				m.OnRequestPage(-1)
			}
			if ui.Button(c, tr("更多任务", "More tasks")).Disabled(m.Busy || m.RequestPage.NextCursor == nil).Clicked() && m.OnRequestPage != nil {
				m.OnRequestPage(1)
			}
		})
	}
}

func observerTaskCard(c *ui.Context, m Model, o *ObserverState, p projectflow.Project, preview projectflow.RequestCard, initial string, queue *projectflow.RequestQueue) {
	mode := o.requestMode(preview, initial)
	if o.requested == nil {
		o.requested = map[string]string{}
	}
	revision := fmt.Sprintf("%d/%g/%s", preview.Revision, preview.UpdatedAt, preview.Status)
	if mode.Mode != "strip" && o.requested[preview.ID] != revision && o.reading == "" && !m.Busy && m.OnRequestGet != nil {
		o.requested[preview.ID] = revision
		o.reading = preview.ID
		m.OnRequestGet(preview.ID)
	}
	card := preview
	loaded := false
	if saved, ok := o.requests[preview.ID]; ok && saved.Revision == preview.Revision {
		card = saved
		loaded = true
	}
	setMode := func(value string) {
		mode.Mode = value
		o.RequestModes[preview.ID] = mode
		if value != "strip" && m.OnRequestGet != nil {
			delete(o.requested, preview.ID)
		}
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	state, executor, reason, next := taskCardStatus(p, card, m.Team, queue)
	box := ui.Column(c).Label("Task card: "+card.ID).FillWidth().MinWidth(0).Gap(8).Radius(14).Border(1, chrome.Color(colors.Border)).Background(chrome.Color(colors.Field)).TextColor(chrome.Color(colors.FG))
	if mode.Mode == "strip" {
		box.Height(44).Padding(0, 4).Children(func() {
			ui.Row(c).FillWidth().Height(44).MinWidth(0).AlignItems(ui.Center).Children(func() {
				open := ui.Button(c, "").Label("Expand task: "+card.Title).Grow(1).MinWidth(0).Height(44).Padding(0, 8).BorderWidth(0).Background(ui.Transparent)
				open.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
						taskStateDot(c, card.Status, chrome.Color(colors.Accent))
						ui.Text(c, card.Title).Grow(1).MinWidth(0).SingleLine().FontSize(theme.InterfaceSize(13))
						executorName, _, _ := strings.Cut(executor, " · ")
						small := state + " · " + executorName
						if card.Queue != nil && card.Queue.State == "queued" {
							small = state
						} else if card.Status == "completed" && card.Delivery != nil {
							count := max(card.Delivery.ArtifactCount, len(card.Delivery.Artifacts))
							if count > 0 {
								small = fmt.Sprintf("%s · %d %s", state, count, tr("个文件", "files"))
							}
						}
						ui.Text(c, small).MaxWidthPercent(38).MinWidth(0).SingleLine().FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
						projectIcon(c, "down", 14, chrome.Color(colors.Muted), false)
					})
				})
				if open.Clicked() {
					setMode("card")
				}
				taskCardMenu(c, m, p, card, queue)
			})
		})
		return
	}
	box.Padding(14, 16, 8, 16).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
			ui.Text(c, card.Title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).FontWeight(600)
			taskCardMenu(c, m, p, card, queue)
			if projectIconButton(c, "collapse", "Collapse task: "+card.Title, "up", chrome.Color(colors.Muted)).Clicked() {
				setMode("strip")
			}
		})
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, state).FontSize(theme.InterfaceSize(12)).Padding(3, 7).Radius(6).Background(chrome.Color(colors.Hover))
			ui.Text(c, executor).MinWidth(0).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
		})
		if mode.Mode == "card" {
			if reason != "" {
				ui.Text(c, reason).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.Muted))
			}
			if next != "" {
				ui.Text(c, tr("下一步：", "Next: ")+next).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.Muted))
			}
			ui.Divider(c)
			if card.Status == "completed" && card.Delivery != nil {
				summary := card.Delivery.Summary
				if summary == "" {
					summary = card.Delivery.SummaryPreview
				}
				ui.Text(c, summary).MaxLines(3).FontSize(theme.InterfaceSize(13))
			}
			taskDeliveryFiles(c, m, card, 4)
			taskDeliveryChangeActions(c, m, card)
			steps := taskCardSteps(card, m.Team)
			if len(steps) > 0 {
				completed := 0
				for _, step := range steps {
					if step.Status == "completed" {
						completed++
					}
				}
				ui.Row(c).FillWidth().Children(func() {
					ui.Text(c, tr("任务进度", "Task progress")).Grow(1).FontSize(theme.InterfaceSize(12)).FontWeight(600)
					ui.Text(c, fmt.Sprintf("%d / %d %s", completed, len(steps), tr("项工作", "work items"))).FontSize(theme.InterfaceSize(12))
				})
				ui.Progress(c, float64(completed)/float64(len(steps))).Label("Task progress: " + card.ID).FillWidth().Height(3)
				for _, step := range steps[:min(3, len(steps))] {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(6, 0).Children(func() {
						taskStateDot(c, step.Status, chrome.Color(colors.Accent))
						ui.Text(c, step.Title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13))
						ui.Text(c, step.MemberName+" · "+stateLabel(step.Status)).MaxWidthPercent(46).MinWidth(0).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
					})
				}
			} else if card.Status != "completed" && reason == "" && next == "" {
				ui.Text(c, tr("尚未分配具体工作。", "No work assigned yet.")).FontSize(theme.InterfaceSize(13))
			}
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				if loaded && !taskCardCurrent(m, card) && m.OnRequestGet != nil && ui.Button(c, tr("刷新任务详情", "Refresh task details")).Height(28).Disabled(m.Busy).Clicked() {
					m.OnRequestGet(card.ID)
				}
				for _, action := range taskQueueActions(m, p, card, queue) {
					if action.id == "enqueue" || action.id == "resume" {
						if ui.Button(c, action.label).Height(28).Disabled(action.disabled).Clicked() && m.OnRequestQueue != nil {
							m.OnRequestQueue(card.ID, action.id)
						}
						break
					}
				}
				// The compact original card exposes undo beside its primary
				// action for discussing/queued requests, as well as in More.
				if card.Queue != nil && (card.Queue.State == "discussing" || card.Queue.State == "queued") {
					for _, action := range taskQueueActions(m, p, card, queue) {
						if action.id == "cancel" && ui.Button(c, tr("撤回", "Undo")).Height(28).Disabled(action.disabled).Clicked() && m.OnRequestQueue != nil {
							m.OnRequestQueue(card.ID, action.id)
						}
					}
				}
				ui.Box(c).Grow(1)
				if ui.Button(c, tr("任务详情", "Task details")).Height(28).FontSize(theme.InterfaceSize(12)).Clicked() {
					setMode("full")
				}
			})
		} else {
			_, height := c.Size()
			bodyScroll := ui.Local(box, "full-scroll", func() ui.ScrollState { return ui.ScrollState{} })
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
				for _, tab := range []struct{ id, label string }{{"overview", tr("概览", "Overview")}, {"team", tr("团队", "Team")}, {"results", tr("成果", "Results")}, {"activity", tr("活动", "Activity")}} {
					label := tab.label
					if mode.Tab == tab.id {
						label = "✓ " + label
					}
					if ui.Button(c, label).Height(28).FontSize(theme.InterfaceSize(12)).Clicked() {
						mode.Tab = tab.id
						o.RequestModes[card.ID] = mode
						bodyScroll.Y = 0
					}
				}
			})
			ui.Scroll(c.Key("full/" + mode.Tab)).Label("Task detail body: " + card.ID).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(48, min(420, height-240))).TrackScroll(bodyScroll).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					if !loaded {
						ui.Text(c, tr("正在加载详情…", "Loading details…")).FontSize(theme.InterfaceSize(13))
						if m.Error != "" && ui.Button(c, tr("重试读取详情", "Retry task details")).Disabled(m.Busy || m.OnRequestGet == nil).Clicked() && m.OnRequestGet != nil {
							m.OnRequestGet(card.ID)
						}
					} else {
						taskCardDetail(c, m, card, mode.Tab)
					}
				})
			})
			if ui.Button(c, tr("收起详情", "Compact card")).Height(28).FontSize(theme.InterfaceSize(12)).Clicked() {
				setMode("card")
			}
		}
	})
}

func taskStateDot(c *ui.Context, state string, ink ui.Color) {
	if state == "completed" {
		ui.Text(c, "✓").FontSize(14).TextColor(ink).Shrink(0)
	} else {
		ui.Box(c).Width(6).Height(6).Radius(3).Margin(4, 4).Background(ink).Shrink(0)
	}
}

func taskCardSteps(card projectflow.RequestCard, team *projectflow.Overview) []projectflow.RequestWork {
	steps := append([]projectflow.RequestWork(nil), card.Work...)
	if card.Work == nil {
		steps = append(steps, card.WorkPreview...)
	}
	for i := range steps {
		s := &steps[i]
		s.MemberName = strings.TrimSpace(s.MemberName)
		if generatedProjectMemberName(s.MemberName, s.WorkerSessionID) || s.MemberName == s.Title {
			s.MemberName = tr("团队成员", "Team member")
		}
		role := ""
		switch s.MemberRole {
		case "development":
			role = tr("开发", "Development")
		case "readonly":
			role = tr("评审", "Review")
		case "docs":
			role = tr("文档", "Docs")
		}
		if role != "" {
			if s.MemberName == tr("团队成员", "Team member") {
				s.MemberName = role
			} else {
				s.MemberName = role + " · " + s.MemberName
			}
		}
		if card.Status == "completed" || card.Queue != nil && card.Queue.Released || s.Current != nil && !*s.Current || team == nil || team.Team == nil {
			continue
		}
		for _, task := range team.Team.Tasks {
			if task.ID != s.WorkstreamID {
				continue
			}
			if task.Subject != "" {
				s.Title = task.Subject
			}
			s.Status = task.Status
			// Settled work retains its state even if old dependencies are
			// still present in the current team projection.
			if task.Status == "completed" || task.Status == "deleted" {
				continue
			}
			if task.Status == "in_progress" {
				s.Status = "assigned"
				for _, worker := range team.Workers {
					if worker.SessionID == s.WorkerSessionID && !worker.Stopped && (worker.ActiveRunID != "" || len(worker.Jobs) > 0) {
						s.Status = "running"
					}
				}
			}
			if task.Status == "pending" && !task.Ready {
				s.Status = "blocked"
			}
			for _, id := range task.BlockedBy {
				for _, dep := range team.Team.Tasks {
					if dep.ID == id && dep.Status != "completed" {
						s.Status = "blocked"
						s.BlockedReason = tr("等待依赖：", "Waiting for dependency: ") + dep.Subject
					}
				}
			}
		}
	}
	return steps
}

func taskCardStatus(p projectflow.Project, card projectflow.RequestCard, team *projectflow.Overview, queue *projectflow.RequestQueue) (state, executor, reason, next string) {
	steps := taskCardSteps(card, team)
	state = requestStatus(card.Status)
	executor = "Lead · " + state
	for _, step := range steps {
		if step.Status == "running" {
			executor = step.MemberName + " · " + stateLabel(step.Status)
			break
		}
	}
	if card.Status == "completed" {
		executor = tr("Lead · 已完成", "Lead · Completed")
	}
	if p.Archiving || p.Activity.State == "stopping" {
		executor = tr("Lead · 正在停止", "Lead · Stopping")
	}
	if card.ArchivedAt != nil {
		return tr("已归档", "Archived"), executor, "", ""
	}
	if card.Queue != nil && card.Queue.State == "cancelled" {
		return tr("已取消", "Cancelled"), executor, "", tr("重新加入队列以继续。", "Rejoin the queue to continue.")
	}
	if card.Status == "completed" {
		if card.Changes != nil && card.Changes.RevertedAt != nil {
			return tr("已撤销改动", "Changes reverted"), executor, "", tr("如需再次实现，请在对话中说明。", "Say so in the conversation if you want it implemented again.")
		}
		return tr("已交付", "Delivered"), executor, "", ""
	}
	if card.Status == "needsReview" {
		return tr("需复核", "Needs review"), executor, "", tr("交付依据已变化，请查看详情后重新加入队列。", "Its delivery evidence changed; review details, then queue it again.")
	}
	if card.Queue != nil {
		switch card.Queue.State {
		case "discussing":
			return tr("讨论中", "Discussing"), executor, "", tr("准备好后加入开发队列。", "Add this request to the queue when ready.")
		case "queued":
			state = tr("排队中", "Queued")
			if card.Queue.Position > 0 {
				state += fmt.Sprintf(" · %d", card.Queue.Position)
			}
			next = tr("等待前面的要求交付后自动开始。", "Starts automatically after the earlier request is delivered.")
			for _, step := range steps {
				if step.Status == "running" {
					next = tr("正在准备此要求的工作。", "Preparing work for this request.")
					break
				}
			}
			if queue != nil && queue.Paused {
				next = tr("开发队列已暂停。", "The development queue is paused.")
			}
			return
		}
	}
	if card.Status == "stopping" || p.Activity.State == "stopping" {
		return tr("正在停止", "Stopping"), executor, "", ""
	}
	if p.Paused || card.Status == "paused" {
		return tr("已暂停", "Paused"), executor, "", tr("选择继续以恢复。", "Choose Continue to resume.")
	}
	if card.AuthorityError != "" {
		return tr("受阻", "Blocked"), executor, card.AuthorityError, tr("核对原始要求及其授权。", "Review the original request and its authorization.")
	}
	if card.Status == "summaryFailed" || card.HandoverError != "" {
		return tr("受阻", "Blocked"), executor, card.HandoverError, tr("重新整理已保存的成果。", "Retry summarizing the saved results.")
	}
	if card.Status == "blocked" && card.Delivery != nil && card.Delivery.Outcome == "partial" {
		return tr("部分交付", "Partial delivery"), executor, "", tr("在对话里补充，或请协调者继续未完成的部分。", "Add details in the conversation, or ask the coordinator to finish the rest.")
	}
	for _, step := range steps {
		if step.WaitingReason != nil {
			reason = waitingReasonLabel(step.WaitingReason)
			switch step.WaitingReason.Kind {
			case "dependencies":
				next = tr("等待前置工作完成。", "Wait for prerequisite work to finish.")
			case "directory":
				next = tr("处理工作目录后继续。", "Resolve the working directory to continue.")
			case "capacity":
				next = tr("等待可用的执行名额。", "Wait for execution capacity.")
			}
			break
		}
		if step.BlockedReason != "" {
			reason = step.BlockedReason
			break
		}
	}
	if card.Status == "blocked" {
		state = tr("受阻", "Blocked")
		if next == "" {
			next = tr("查看详情并处理阻碍。", "Review the details and resolve the blocker.")
		}
		return
	}
	state = tr("进行中", "Active")
	if len(steps) == 0 && card.Queue != nil && card.Queue.State == "active" && !card.Queue.Released && card.Delivery == nil && card.Status != "pendingDelivery" {
		executor = tr("Lead · 正在安排", "Lead · Arranging")
		reason = tr("协调者正在安排团队工作。", "The coordinator is arranging development.")
	}
	return
}

type taskQueueAction struct {
	id, label string
	disabled  bool
}

func taskQueueActions(m Model, p projectflow.Project, card projectflow.RequestCard, queue *projectflow.RequestQueue) []taskQueueAction {
	if queue == nil || card.Queue == nil || card.Status == "completed" || card.ArchivedAt != nil || p.Lifecycle != "ready" || p.Archiving {
		return nil
	}
	disabled := m.Busy || !m.Listing.Available || !m.RequestMutable || m.OnRequestQueue == nil
	var out []taskQueueAction
	add := func(id, label string, blocked bool) {
		out = append(out, taskQueueAction{id, label, disabled || blocked})
	}
	state := card.Queue.State
	index := -1
	if queue != nil {
		for i, item := range queue.Queued {
			if item.ID == card.ID {
				index = i
				break
			}
		}
	}
	if state == "discussing" {
		add("enqueue", tr("加入队列", "Add to queue"), false)
	} else if state == "cancelled" || card.Queue.Released {
		add("enqueue", tr("重新加入队列", "Rejoin queue"), false)
	}
	if queue != nil && ((queue.Active != nil && queue.Active.ID == card.ID && !card.Queue.Released && (queue.Paused || card.Status == "paused")) || (state == "queued" && queue.Paused && queue.Active == nil && index == 0)) {
		add("resume", tr("继续任务", "Resume request"), false)
	}
	if state == "queued" {
		add("moveUp", tr("上移", "Move up"), index <= 0)
		add("moveDown", tr("下移", "Move down"), queue == nil || index < 0 || index >= len(queue.Queued)-1)
	}
	if state == "discussing" || state == "queued" || state == "active" {
		add("cancel", tr("取消任务", "Cancel request"), false)
	}
	return out
}
func taskCardMenu(c *ui.Context, m Model, p projectflow.Project, card projectflow.RequestCard, queue *projectflow.RequestQueue) {
	actions := taskQueueActions(m, p, card, queue)
	archive := p.Lifecycle == "ready" && !p.Archiving && (card.Status == "completed" || card.ArchivedAt != nil)
	if len(actions) == 0 && !archive {
		return
	}
	button := projectIconButton(c, "task-more", "More task actions: "+card.Title, "more", chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Muted))
	button.Menu(func(menu *ui.Menu) {
		for _, action := range actions {
			if menu.Item(action.label).Disabled(action.disabled).Chosen() && m.OnRequestQueue != nil {
				m.OnRequestQueue(card.ID, action.id)
			}
		}
		if archive {
			label := tr("归档已交付要求", "Archive delivered request")
			if card.ArchivedAt != nil {
				label = tr("恢复到任务列表", "Restore to task list")
			}
			if menu.Item(label).Disabled(m.Busy || !m.Listing.Available || !m.RequestMutable || m.OnRequestArchive == nil).Chosen() && m.OnRequestArchive != nil {
				m.OnRequestArchive(card.ID, card.ArchivedAt == nil)
			}
		}
	})
}

func taskDeliveryFiles(c *ui.Context, m Model, card projectflow.RequestCard, limit int) {
	actions := requestCardActions(m, card)
	if card.Delivery == nil || (card.Status != "completed" && card.Delivery.Outcome != "partial") {
		return
	}
	files := card.Delivery.Artifacts
	if len(files) == 0 {
		return
	}
	ui.Text(c, fmt.Sprintf(tr("交付文件（%d）", "Delivered files (%d)"), len(files))).FontSize(theme.InterfaceSize(12)).FontWeight(600)
	for _, file := range files[:min(limit, len(files))] {
		label := file.Title
		if label == "" {
			label = path.Base(strings.ReplaceAll(file.Path, "\\", "/"))
		}
		if ui.Button(c, tr("打开文件：", "Open file: ")+label).FillWidth().MinWidth(0).Height(32).Disabled(m.Busy || !m.Listing.Available || !actions.Known || actions.OpenFile == nil || m.OnResolvedFile == nil).Clicked() && actions.OpenFile != nil {
			actions.OpenFile(file, m.OnResolvedFile)
		}
	}
}

func taskCardDetail(c *ui.Context, m Model, card projectflow.RequestCard, tab string) {
	actions := requestCardActions(m, card)
	current := taskCardCurrent(m, card)
	if !current && m.OnRequestGet != nil && ui.Button(c, tr("刷新任务详情", "Refresh task details")).Disabled(m.Busy).Clicked() {
		m.OnRequestGet(card.ID)
	}
	switch tab {
	case "team":
		if len(card.Work) == 0 {
			ui.Text(c, tr("尚无关联成员工作。", "No associated member work."))
		}
		for _, work := range card.Work {
			ui.Text(c, work.Title+" · "+stateLabel(work.Status)).FontSize(theme.InterfaceSize(14)).FontWeight(500)
			ui.Text(c, work.MemberName).FontSize(theme.InterfaceSize(12))
			if work.Summary != "" {
				ui.Text(c, work.Summary).FontSize(theme.InterfaceSize(13))
			}
			if work.BlockedReason != "" {
				ui.Text(c, work.BlockedReason)
			}
			if ui.Button(c, tr("查看工作详情", "View work details")).Disabled(m.Busy || !m.Listing.Available || !current || actions.WorkDetail == nil).Clicked() && actions.WorkDetail != nil {
				actions.WorkDetail(work.WorkstreamID)
			}
			if ui.Button(c, tr("查看当前工作区变更", "View current workspace diff")).Disabled(m.Busy || !m.Listing.Available || !current || actions.WorkDiff == nil).Clicked() && actions.WorkDiff != nil {
				actions.WorkDiff(work.WorkstreamID)
			}
		}
	case "results":
		taskDeliveryChangeActions(c, m, card)
		if card.Delivery != nil {
			pageModel := m
			pageModel.RequestCard = &card
			renderVisualPages(c, pageModel, "request:"+card.ID, card.Delivery.Pages)
		}
		if card.Delivery == nil {
			ui.Text(c, tr("尚无最终交付。", "No final delivery yet."))
		} else {
			taskDeliveryFiles(c, m, card, len(card.Delivery.Artifacts))
			ui.Text(c, card.Delivery.Summary).FontSize(theme.InterfaceSize(13))
			for _, text := range card.Delivery.Evidence {
				ui.Text(c, tr("佐证：", "Evidence: ")+text)
			}
			for _, text := range card.Delivery.RemainingIssues {
				ui.Text(c, tr("剩余问题：", "Remaining issue: ")+text)
			}
		}
	case "activity":
		if ui.Button(c, tr("活动时间线", "Activity timeline")).Disabled(m.Busy || !m.Listing.Available || !current || actions.Activity == nil).Clicked() && actions.Activity != nil {
			actions.Activity()
		}
		for _, version := range card.Versions {
			ui.Text(c, fmt.Sprintf("%s %d · %s", tr("要求版本", "Requirement version"), version.Revision, version.Title)).FontSize(theme.InterfaceSize(13))
			ui.Text(c, version.Goal).FontSize(theme.InterfaceSize(13))
		}
		if len(card.Versions) == 0 {
			ui.Text(c, tr("暂无已记录活动。", "No recorded activity yet."))
		}
	default:
		ui.Text(c, tr("要求 → 团队 → 成员报告 → 交付给你", "Request → Team → Member reports → Delivery")).FontSize(theme.InterfaceSize(12))
		if card.Delivery != nil {
			ui.Text(c, card.Delivery.Summary).FontSize(theme.InterfaceSize(13))
		} else {
			ui.Text(c, tr("尚无最终交付。", "No final delivery yet.")).FontSize(theme.InterfaceSize(13))
		}
		ui.Text(c, tr("用户要求", "User request")).FontSize(theme.InterfaceSize(14)).FontWeight(600)
		ui.Text(c, card.Goal).FontSize(theme.InterfaceSize(13))
		for _, text := range card.Conditions {
			ui.Text(c, tr("验收条件：", "Acceptance condition: ")+text).FontSize(theme.InterfaceSize(13))
		}
		for _, text := range card.Constraints {
			ui.Text(c, tr("约束：", "Constraint: ")+text).FontSize(theme.InterfaceSize(13))
		}
		if card.AuthorityError != "" {
			ui.Text(c, card.AuthorityError).FontSize(theme.InterfaceSize(13))
		}
		if card.HandoverError != "" {
			ui.Text(c, card.HandoverError).FontSize(theme.InterfaceSize(13))
			if ui.Button(c, tr("发送交付恢复请求", "Send handover recovery request")).Disabled(m.Busy || !m.Listing.Available || !m.RequestMutable || !current || card.AuthorityError != "" || actions.HandoverRetry == nil).Clicked() && actions.HandoverRetry != nil {
				actions.HandoverRetry()
			}
		}
	}
	ui.Text(c, fmt.Sprintf(tr("要求版本 %d", "Requirement version %d"), card.Revision)).FontSize(theme.InterfaceSize(12))
}

func taskCardCurrent(m Model, card projectflow.RequestCard) bool {
	if m.RequestCardActions != nil {
		return requestCardActions(m, card).Known
	}
	return m.RequestCard != nil && m.RequestCard.ID == card.ID && m.RequestCard.ProjectID == card.ProjectID && m.RequestCard.Revision == card.Revision
}
