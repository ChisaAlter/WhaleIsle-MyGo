package diffsurface

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/webview"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type remoteWorkflowState struct {
	Mode, Title, Body string
	Draft             bool
	Plan              *git.StackPlan
	PR                *git.PullRequestPlan
	Push              git.PushPlan
	Task              *remotePreparation
	Generation        *prGenerationTask
	Generated         *git.GeneratedPRContent
	GenerationError   string
}
type remotePreparation struct {
	root   string
	cancel context.CancelFunc
	done   chan remotePrepared
}
type remotePrepared struct {
	plan git.StackPlan
	pr   *git.PullRequestPlan
	push git.PushPlan
	err  error
}

func (w *Workbench) cancelRemotePreparation() {
	w.cancelPRGeneration()
	if w.Remote != nil && w.Remote.Task != nil {
		w.Remote.Task.cancel()
		w.Remote.Task = nil
		if w.Progress != nil {
			w.Progress.Finish(word("准备已取消", "Preparation canceled"), false)
		}
	}
}
func (w *Workbench) RequestPR(root string) { w.requestRemote(root, "pr") }
func (w *Workbench) requestRemote(root, mode string) {
	w.cancelCommitPreparation()
	w.cancelRemotePreparation()
	if w.Root != root {
		w.cancelBranchList()
		w.cancelRepositoryStatus()
		*w = Workbench{Operation: w.Operation, invalidate: w.invalidate, AgainstHEAD: true}
	}
	w.reload(root)
	w.Remote = &remoteWorkflowState{Mode: mode, Draft: true}
	if mode == "pr" {
		w.Remote.Title = strings.SplitN(w.Message, "\n", 2)[0]
	}
}
func paintPRReview(c *ui.Context, p git.PullRequestPlan, fg, muted ui.Color) {
	ui.Text(c, word("PR 收件仓库：", "PR recipient: ")+p.Host+"/"+p.Repository).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	ui.Text(c, p.Head+" → "+p.Base).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	ui.Text(c, p.Title).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	ui.Text(c, p.Body).FontSize(fonttheme.InterfaceSize(10)).TextColor(fg)
	ui.Text(c, pullRequestModeLabel(p.Draft)+word("；如已有匹配的开放 PR 则打开已有项，不改变其状态。", "; an existing matching open PR is reused without changing its state.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
}
func beginRemotePreparation(c *ui.Context, m Model) {
	if !captionAdmitted(m) {
		return
	}
	w := m.Workbench
	s := w.Remote
	if s == nil || s.Task != nil || s.Generation != nil || w.Operation != nil {
		return
	}
	root, mode, title, body, draft := m.WorkspaceRoot, s.Mode, s.Title, s.Body, s.Draft
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	task := &remotePreparation{root: root, cancel: cancel, done: make(chan remotePrepared, 1)}
	s.Task = task
	s.Plan = nil
	s.PR = nil
	w.Status = ""
	invalidate := c.Services().Invalidate
	progress := git.NewProgress(root, word("正在准备后续操作…", "Preparing next action…"), invalidate)
	w.Progress = progress
	ctx = git.WithProgress(ctx, progress)
	go func() {
		r := remotePrepared{}
		r.push, r.err = git.PreparePush(root)
		if r.err == nil && mode == "pr" {
			var p git.PullRequestPlan
			p, r.err = git.PreparePullRequestMode(ctx, root, r.push, title, body, draft)
			r.pr = &p
		}
		if r.err == nil {
			var push *git.PushPlan
			if mode == "push" {
				push = &r.push
			}
			r.plan, r.err = git.PrepareStack(ctx, root, nil, "", push, r.pr)
		}
		phase := word("核对内容已准备好", "Review ready")
		if r.err != nil {
			phase = word("准备失败，请查看详情", "Preparation failed; review details")
		}
		progress.Finish(phase, r.err != nil)
		task.done <- r
		if invalidate != nil {
			invalidate()
		}
	}()
}
func finishRemotePreparation(m Model) {
	w := m.Workbench
	s := w.Remote
	if s == nil || s.Task == nil {
		return
	}
	task := s.Task
	if task.root != m.WorkspaceRoot {
		w.cancelRemotePreparation()
		return
	}
	select {
	case r := <-task.done:
		task.cancel()
		s.Task = nil
		if r.err != nil {
			w.Status = r.err.Error()
			return
		}
		s.Plan = &r.plan
		s.PR = r.pr
		s.Push = r.push
	default:
	}
}
func paintRemoteWorkflow(c *ui.Context, m Model, fg, muted ui.Color) bool {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	finishRemotePreparation(m)
	s := w.Remote
	if s == nil {
		return false
	}
	if s.Task != nil {
		ui.Text(c, word("正在准备后续操作…", "Preparing next action…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	} else {
		if s.Mode == "pr" {
			oldTitle, oldBody, oldDraft := s.Title, s.Body, s.Draft
			ui.Checkbox(c, &s.Draft, word("创建草稿 PR（取消勾选为普通 PR）", "Create draft PR (uncheck for ready PR)"))
			ui.TextInput(c, &s.Title).FillWidth().Placeholder(word("PR 标题", "PR title"))
			ui.TextArea(c, &s.Body).FillWidth().Lines(2, 6).Placeholder(word("PR 正文", "PR body"))
			if oldTitle != s.Title || oldBody != s.Body || oldDraft != s.Draft {
				s.Plan = nil
				s.PR = nil
			}
			paintPRGeneration(c, m, fg, muted)
		}
		if s.Plan == nil {
			if widgets.Button(c, colors, word("准备核对", "Prepare review"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(s.Generation != nil).Clicked() {
				beginRemotePreparation(c, m)
			}
		} else {
			if s.Mode == "push" {
				ui.Text(c, pushReviewText(s.Push)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
			} else if s.PR != nil {
				paintPRReview(c, *s.PR, fg, muted)
			}
			caption := word("确认仅推送（不重复提交）", "Confirm push only (no new commit)")
			if s.Mode == "pr" {
				caption = word("确认创建", "Confirm create ") + pullRequestModeLabel(s.Draft)
			}
			if ui.Button(c, caption).Clicked() {
				if !captionAdmitted(m) {
					return true
				}
				root, plan := m.WorkspaceRoot, *s.Plan
				w.Remote = nil
				beginGitOperation(c, w, word("正在执行后续操作", "Running next action"), func(ctx context.Context) gitOperationResult {
					result, e := git.RunCommitStack(ctx, root, plan)
					return gitOperationResult{text: result.Output, err: e, stack: &result}
				})
			}
		}
	}
	if widgets.Button(c, colors, word("取消后续操作", "Cancel next action"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
		w.cancelRemotePreparation()
		w.Remote = nil
	}
	if w.Status != "" {
		ui.Text(c, w.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	return true
}
func paintGitFollowups(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w.LastStack == nil {
		return
	}
	r := *w.LastStack
	if r.Committed {
		ui.Text(c, word("已提交：", "Committed: ")+r.CommitSHA).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
	}
	if r.Pushed {
		ui.Text(c, word("推送已完成", "Push completed")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
	}
	switch r.NextAction() {
	case "push":
		if widgets.Button(c, colors, word("核对后仅推送…", "Review push only…"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			w.requestRemote(m.WorkspaceRoot, "push")
		}
	case "create_pr":
		if widgets.Button(c, colors, word("核对 PR…", "Review PR…"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			w.RequestPR(m.WorkspaceRoot)
		}
	case "open_pr":
		if widgets.Button(c, colors, word("查看 PR", "View PR"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			if !git.ValidPullRequestURLForHost(r.PR.URL, r.PR.Host, r.PR.Repository) {
				w.Status = "Unexpected PR URL"
			} else if e := webview.OpenExternal(r.PR.URL); e != nil {
				w.Status = e.Error()
			}
		}
	case "check_pr":
		ui.Text(c, word("PR 创建结果尚不确定，先检查已有项，勿重复创建。", "PR outcome is unknown. Check existing PRs before creating again.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(fg)
		if r.PRPlan != nil && widgets.Button(c, colors, word("检查已有 PR（只读）", "Check existing PR (read-only)"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			root, plan := m.WorkspaceRoot, *r.PRPlan
			beginGitOperation(c, w, word("正在检查已有 PR", "Checking existing PR"), func(ctx context.Context) gitOperationResult {
				found, e := git.LookupPullRequestReviewed(ctx, root, plan, r.CommitSHA)
				if e == nil && found.URL != "" {
					r.PR = found
				} else if e == nil {
					e = errors.New("no matching open PR found; inspect GitHub before preparing a fresh create review")
				}
				return gitOperationResult{text: r.Output, err: e, stack: &r}
			})
		}
	}
}

// Retain completed stages when a fresh review retries only the remaining step.
// The operation result's committed flag still describes this run only.
func mergeStackHistory(previous, next *git.StackResult) *git.StackResult {
	if next == nil {
		return previous
	}
	if next.CommitSHA == "" && !next.Committed && !next.Pushed && next.PR.URL == "" && !next.PR.Uncertain {
		return previous
	}
	result := *next
	if previous != nil && previous.CommitSHA == next.CommitSHA {
		result.Committed = result.Committed || previous.Committed
		if previous.PushPlan != nil && result.PushPlan != nil && *previous.PushPlan == *result.PushPlan {
			result.Pushed = result.Pushed || previous.Pushed
		}
		if result.CreatedBranch == "" {
			result.CreatedBranch = previous.CreatedBranch
		}
	}
	return &result
}

func pullRequestModeLabel(draft bool) string {
	if draft {
		return word("草稿 PR", "draft PR")
	}
	return word("普通 PR（准备审阅）", "ready-for-review PR")
}
