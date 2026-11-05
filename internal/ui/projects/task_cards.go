package projects

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// RequestCardActions keeps every card operation on the exact Host-admitted
// request revision, independently of other expanded cards.
type RequestCardActions struct {
	OpenAuthorizedFile      func(projectflow.Artifact, func(projectflow.FileTarget))
	Known                   bool
	OpenFile                func(projectflow.Artifact, func(string))
	WorkDiff, WorkDetail    func(string)
	OpenCurrent             func(projectflow.RequestWork) error
	OpenAssignment          func(projectflow.RequestWork)
	WorkAction              func(string, projectflow.RequestWork)
	Workstreams             []projectflow.Workstream
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
	visible := append([]projectflow.RequestCard(nil), o.Pages["current"].Items...)
	if queue != nil && queue.Active != nil {
		found := false
		for _, card := range visible {
			found = found || card.ID == queue.Active.ID
		}
		if !found {
			visible = append([]projectflow.RequestCard{*queue.Active}, visible...)
		}
	}
	visible = append(visible, o.Pages["completed"].Items...)
	byID := map[string]projectflow.RequestCard{}
	for _, card := range visible {
		byID[card.ID] = card
	}
	ordered := make([]projectflow.RequestCard, 0, len(visible))
	seen := map[string]bool{}
	for _, id := range o.ReadingOrder {
		if card, ok := byID[id]; ok && !seen[id] {
			ordered = append(ordered, card)
			seen[id] = true
		}
	}
	for _, card := range visible {
		if !seen[card.ID] {
			ordered = append(ordered, card)
			seen[card.ID] = true
		}
	}
	o.ReadingOrder = o.ReadingOrder[:0]
	for _, card := range ordered {
		o.ReadingOrder = append(o.ReadingOrder, card.ID)
		initial := "strip"
		if queue != nil && queue.Active != nil && queue.Active.ID == card.ID {
			initial = "card"
		}
		observerTaskCard(c.Key(p.ID+"/"+card.ID), m, o, p, card, initial, queue)
	}
	if len(ordered) == 0 && o.CatalogLoaded["current"] && o.CatalogLoaded["completed"] {
		ui.Text(c, tr("任务会显示在这里。在Lead对话中描述目标，查看进展与交付。", "Tasks appear here. Describe a goal in the Lead conversation to follow progress and delivery.")).FontSize(theme.InterfaceSize(13))
	}
	requirementPageControls(c, m, o, "current")
	if o.HistoryKind != "" {
		kind := o.HistoryKind
		label := tr("已取消", "Cancelled")
		if kind == "archived" {
			label = tr("已归档", "Archived")
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).Grow(1).FontSize(theme.InterfaceSize(13)).FontWeight(600)
			if ui.Button(c, tr("返回任务", "Back to tasks")).Clicked() {
				o.HistoryKind = ""
			}
		})
		for _, card := range o.Pages[kind].Items {
			observerTaskCard(c.Key(p.ID+"/history/"+card.ID), m, o, p, card, "strip", queue)
		}
		requirementPageControls(c, m, o, kind)
	}
	if o.Pages["completed"].Total > 0 {
		ui.Text(c, fmt.Sprintf(tr("已交付 · %d", "Delivered · %d"), o.Pages["completed"].Total)).FontSize(theme.InterfaceSize(12))
		requirementPageControls(c, m, o, "completed")
	}
}
func requirementPageControls(c *ui.Context, m Model, o *ObserverState, kind string) {
	page := o.Pages[kind]
	number := max(1, len(o.Cursors[kind]))
	if number <= 1 && page.NextCursor == nil && o.CatalogErrors[kind] == "" {
		return
	}
	ui.Row(c.Key("requirements-page/" + kind)).FillWidth().Gap(6).Justify(ui.End).Children(func() {
		if o.CatalogErrors[kind] != "" {
			if ui.Button(c, tr("重试读取", "Retry reading")).Disabled(m.Busy || o.OnLoadCatalog == nil).Clicked() {
				delete(o.CatalogErrors, kind)
				o.OnLoadCatalog(kind, 0)
			}
		}
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || number <= 1 || o.OnLoadCatalog == nil).Clicked() {
			o.OnLoadCatalog(kind, -1)
		}
		ui.Text(c, fmt.Sprintf("%d", number)).FontSize(theme.InterfaceSize(12))
		if ui.Button(c, tr("下一页", "Next")).Disabled(m.Busy || page.NextCursor == nil || o.OnLoadCatalog == nil).Clicked() {
			o.OnLoadCatalog(kind, 1)
		}
	})
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
			steps := taskCardSteps(card, m.Team)
			finished, total := 0, len(steps)
			for _, step := range steps {
				if step.Status == "completed" {
					finished++
				}
			}
			if card.Counts != nil {
				finished, total = card.Counts.Completed, card.Counts.Total
			}
			if total > 0 {
				ui.Text(c, fmt.Sprintf("%d / %d", finished, total)).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
			}
			ui.Text(c, executor).MinWidth(0).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
		})
		if reason != "" || next != "" {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				if reason != "" {
					ui.Text(c, reason).FontSize(theme.InterfaceSize(13))
				}
				if next != "" {
					ui.Text(c, tr("下一步：", "Next: ")+next).FontSize(theme.InterfaceSize(13))
				}
			})
		}
		if mode.Tab == "team" {
			mode.Tab = "work"
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			for _, tab := range []struct{ id, label string }{{"overview", tr("概览", "Overview")}, {"plan", tr("计划", "Plan")}, {"work", tr("工作", "Work")}, {"results", tr("成果", "Results")}} {
				active := mode.Tab == tab.id
				button := ui.Button(c, tab.label).Role(ui.RoleTab).Selected(active).Height(28).Padding(0, 10).BorderWidth(0).Radius(6).Background(ui.Transparent).TextColor(chrome.Color(colors.FG)).FontSize(theme.InterfaceSize(12))
				if active {
					button.Background(chrome.Color(colors.Hover))
				}
				if button.Clicked() {
					mode.Tab = tab.id
					o.RequestModes[card.ID] = mode
				}
			}
		})
		if !loaded {
			ui.Text(c, tr("正在加载详情…", "Loading details…")).FontSize(theme.InterfaceSize(13))
			if m.Error != "" && ui.Button(c, tr("重试读取详情", "Retry task details")).Disabled(m.Busy || m.OnRequestGet == nil).Clicked() {
				delete(o.requested, preview.ID)
				o.reading = ""
			}
		} else {
			taskCardDetail(c, m, card, mode.Tab)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			for _, action := range taskQueueActions(m, p, card, queue) {
				if action.id == "enqueue" || action.id == "resume" || action.id == "cancel" && (card.Queue.State == "discussing" || card.Queue.State == "queued") {
					if ui.Button(c, action.label).Height(28).Disabled(action.disabled).Clicked() && m.OnRequestQueue != nil {
						m.OnRequestQueue(card.ID, action.id)
					}
				}
			}
		})
	})
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
		canOpen := actions.OpenAuthorizedFile != nil && m.OnOpenAuthorizedFile != nil || actions.OpenFile != nil && m.OnResolvedFile != nil
		if ui.Button(c, tr("打开文件：", "Open file: ")+label).FillWidth().MinWidth(0).Height(32).Disabled(m.Busy || !m.Listing.Available || !actions.Known || !canOpen).Clicked() {
			if actions.OpenAuthorizedFile != nil && m.OnOpenAuthorizedFile != nil {
				actions.OpenAuthorizedFile(file, m.OnOpenAuthorizedFile)
			} else if actions.OpenFile != nil {
				actions.OpenFile(file, m.OnResolvedFile)
			}
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
	case "plan":
		current := m.RequestQueue != nil && m.RequestQueue.Active != nil && m.RequestQueue.Active.ID == card.ID
		lead := ""
		for _, project := range m.Listing.Projects {
			if project.ID == card.ProjectID {
				lead = project.CoordinatorSessionID
				break
			}
		}
		if current && lead != "" && m.PlanSessionID == lead {
			chat.PlanBoardView(c, chat.PlanBoardModel{Board: m.PlanBoard, HideQueue: true, ThemeID: m.ThemeID, SystemDark: m.SystemDark})
		} else {
			ui.Text(c, tr("仅当前正在执行的需求显示 Lead 计划。", "The Lead plan is shown only for the current active requirement."))
		}
	case "work", "team":
		if len(card.Work) == 0 {
			ui.Text(c, tr("尚无关联成员工作。", "No associated member work."))
		}
		for _, work := range card.Work {
			work := work
			historical := work.Current != nil && !*work.Current
			ui.Column(c.Key("request-work/"+work.WorkstreamID+"/"+work.DelegationRef)).FillWidth().MinWidth(0).Gap(5).Padding(8, 0).Children(func() {
				button := ui.Button(c, "").Label(work.Title + " · " + tr("查看过程", "View process")).FillWidth().MinWidth(0).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(work.WorkerSessionID == "" || m.Busy || !current || historical && actions.OpenAssignment == nil || !historical && actions.OpenCurrent == nil)
				button.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
						taskStateDot(c, work.Status, chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent))
						ui.Text(c, work.Title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13))
						ui.Text(c, work.MemberName+" · "+stateLabel(work.Status)).FontSize(theme.InterfaceSize(12))
					})
				})
				if button.Clicked() {
					if historical {
						actions.OpenAssignment(work)
					} else if err := actions.OpenCurrent(work); err != nil {
						c.AddToast(ui.Toast{ID: "project-member-open", Description: err.Error(), Type: "error", Timeout: -1})
					}
				}
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).Wrap().Children(func() {
					if work.ReportID != "" && ui.Button(c, tr("查看成果", "View result")).Height(28).Disabled(m.Busy || !current || actions.WorkAction == nil).Clicked() {
						actions.WorkAction("report", work)
					}
					changes := card.Status != "completed" && !historical && work.MemberRole != "" && work.MemberRole != "readonly"
					if changes && ui.Button(c, tr("查看变更", "View changes")).Height(28).Disabled(m.Busy || !current || actions.WorkDiff == nil).Clicked() {
						actions.WorkDiff(work.WorkstreamID)
					}
					stoppable := !historical && (work.Status == "running" || work.Status == "queued" || work.Status == "preparing" || work.Status == "provisioning" || work.Status == "stopping")
					paused := false
					for _, p := range m.Listing.Projects {
						if p.ID == card.ProjectID {
							paused = p.Paused
						}
					}
					if stoppable && !paused && ui.Button(c, tr("停止工作", "Stop work")).Height(28).Disabled(m.Busy || work.Status == "stopping" || !current || actions.WorkAction == nil).Clicked() {
						actions.WorkAction("stop", work)
					}
				})
				if work.BlockedReason != "" && work.BlockedReason != "stopped" && work.WaitingReason == nil {
					ui.Text(c, work.BlockedReason).FontSize(theme.InterfaceSize(12))
				}
				for _, stream := range actions.Workstreams {
					if stream.ID != work.WorkstreamID {
						continue
					}
					if stream.PrerequisitesChanged && !historical {
						ui.Text(c, tr("前置工作已变化，请复核依赖。", "Prerequisites changed; review dependencies.")).FontSize(theme.InterfaceSize(12))
					}
					failure := stream.SummaryFailure
					if work.Status != "summaryFailed" || failure == nil || failure.DelegationRef != work.DelegationRef {
						continue
					}
					ui.Text(c, tr("成果已保存，汇总失败：", "Results are saved; summary failed: ")+failure.Error.Message).FontSize(theme.InterfaceSize(12))
					ui.Row(c).Gap(6).Children(func() {
						if failure.Error.Code == "MISSING_CREDENTIAL" && m.OnOpenModelSettings != nil && ui.Button(c, tr("模型设置", "Model settings")).Height(28).Clicked() {
							m.OnOpenModelSettings()
						}
						if ui.Button(c, tr("重新汇总", "Retry summary")).Height(28).Disabled(m.Busy || !current || card.AuthorityError != "" || actions.WorkAction == nil).Clicked() {
							actions.WorkAction("summary", work)
						}
					})
				}
			})
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
		button := ui.Button(c, tr("活动时间线", "Activity timeline")).Disabled(m.Busy || !m.Listing.Available || !current || actions.Activity == nil)
		origin := ui.Local(button, "activity-origin", func() ui.Handle { return ui.Handle{} })
		button.Bind(origin)
		if button.Clicked() && actions.Activity != nil {
			if m.presentation != nil {
				m.presentation.ChildOrigin = &projectChildOrigin{
					route: "activity/" + card.ProjectID + "/" + card.ID,
					host:  m.ObserverHost, session: m.SelectedSession,
					request: projectflow.RequestCard{ID: card.ID, ProjectID: card.ProjectID, Revision: card.Revision},
					handle:  origin,
				}
			}
			actions.Activity()
		}
		for _, version := range card.Versions {
			ui.Text(c, fmt.Sprintf("%s %d · %s", tr("要求版本", "Requirement version"), version.Revision, version.Title)).FontSize(theme.InterfaceSize(13))
			if version.At > 0 {
				ui.Text(c, time.UnixMilli(int64(version.At)).Format(time.RFC3339)).FontSize(theme.InterfaceSize(12))
			}
			ui.Text(c, version.Goal).FontSize(theme.InterfaceSize(13))
			for _, text := range version.Constraints {
				ui.Text(c, tr("约束：", "Constraint: ")+text).FontSize(theme.InterfaceSize(13))
			}
			for _, text := range version.Conditions {
				ui.Text(c, tr("验收条件：", "Acceptance condition: ")+text).FontSize(theme.InterfaceSize(13))
			}
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
		taskRequestVersions(c, m, card)
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

func taskRequestVersions(c *ui.Context, m Model, card projectflow.RequestCard) {
	if len(card.Versions) <= 1 {
		return
	}
	for i := len(card.Versions) - 1; i >= 0; i-- {
		version := card.Versions[i]
		root := ui.Column(c.Key(fmt.Sprintf("request-version/%s/%d", card.ID, version.Revision))).FillWidth().MinWidth(0)
		open := ui.Local(root, "expanded", func() bool { return false })
		stamp := ""
		if version.At > 0 {
			stamp = time.UnixMilli(int64(version.At)).Local().Format("2006-01-02 15:04")
		}
		root.Children(func() {
			widgets.DisclosureRow(c, widgets.DisclosureRowOptions{Title: fmt.Sprintf(tr("要求版本 %d", "Requirement version %d"), version.Revision), Summary: stamp, Open: open, Body: func() {
				projectMarkdown(c, m, version.Goal)
				for _, text := range version.Constraints {
					ui.Text(c, tr("约束：", "Constraint: ")+text).FontSize(theme.InterfaceSize(13))
				}
				for _, text := range version.Conditions {
					ui.Text(c, tr("验收条件：", "Acceptance condition: ")+text).FontSize(theme.InterfaceSize(13))
				}
			}})
		})
	}
}

func taskCardCurrent(m Model, card projectflow.RequestCard) bool {
	if m.RequestCardActions != nil {
		return requestCardActions(m, card).Known
	}
	return m.RequestCard != nil && m.RequestCard.ID == card.ID && m.RequestCard.ProjectID == card.ProjectID && m.RequestCard.Revision == card.Revision
}

// Match the requirements checklist: a completed check, otherwise a static
// six-DIP state dot in the same fourteen-DIP leading slot.
func taskStateDot(c *ui.Context, status string, accent ui.Color) {
	if status == "completed" {
		ui.Icon(c, taskCompletedIcon).Size(14, 14).Shrink(0).TextColor(accent)
		return
	}
	ink := c.Theme().TextMuted
	switch status {
	case "running":
		ink = accent
	case "blocked":
		ink = c.Theme().Warning
	}
	ui.Box(c).Size(14, 14).Shrink(0).Role(ui.RoleNone).Center().Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(ink)
	})
}

var taskCompletedIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m3 8 3 3 7-7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
