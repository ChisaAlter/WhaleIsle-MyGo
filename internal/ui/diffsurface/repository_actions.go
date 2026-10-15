package diffsurface

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
)

type gitOperationResult struct {
	text      string
	err       error
	committed bool
	stack     *git.StackResult
}

type gitOperation struct {
	root, label string
	done        chan gitOperationResult
}

func beginGitOperation(c *ui.Context, w *Workbench, label string, run func(context.Context) gitOperationResult) {
	if w.Operation != nil {
		return
	}
	operation := &gitOperation{root: w.Root, label: label, done: make(chan gitOperationResult, 1)}
	w.Operation = operation
	w.ReviewToken = ""
	w.ReviewText = ""
	w.Status = ""
	invalidate := c.Services().Invalidate
	progress := git.NewProgress(w.Root, label, invalidate)
	w.Progress = progress
	go func() {
		ctx, cancel := context.WithTimeout(git.WithProgress(context.Background(), progress), git.DefaultCommandTimeout)
		defer cancel()
		result := run(ctx)
		phase := word("操作完成", "Action finished")
		if result.err != nil {
			phase = word("操作失败；保留已完成步骤，请查看详情", "Action failed; completed steps are preserved. Review the details")
		}
		if result.committed && result.err != nil {
			phase = word("本地提交已创建；后续步骤失败", "Local commit created; a later step failed")
		}
		progress.Finish(phase, result.err != nil)
		operation.done <- result
		if invalidate != nil {
			invalidate()
		}
	}()
}

func finishGitOperation(m Model) {
	w := m.Workbench
	operation := w.Operation
	if operation == nil {
		return
	}
	select {
	case result := <-operation.done:
		w.Operation = nil
		sameRoot := operation.root == m.WorkspaceRoot
		if sameRoot {
			if result.stack != nil {
				w.LastStack = mergeStackHistory(w.LastStack, result.stack)
			}
			w.reload(m.WorkspaceRoot)
			if w.BranchesOpen {
				w.reloadBranches()
			}
			refresh(m)
			if result.committed {
				w.Message = ""
			}
			if result.err == nil {
				w.PublishOpen, w.PublishReview = false, false
			}
		}
		w.Status = result.text
		if result.err != nil {
			if w.Status != "" {
				w.Status += "\n"
			}
			w.Status += result.err.Error()
		}
		if w.Status == "" {
			w.Status = word("操作已完成", "Action finished")
		}
		if !sameRoot {
			w.Status = operation.root + "\n" + w.Status
		}
	default:
	}
}

// Returns true while only repository-level controls should be displayed.
func paintRepositoryActions(c *ui.Context, m Model, fg, muted ui.Color) bool {
	w := m.Workbench
	finishGitOperation(m)
	DrawGitProgress(c, w.Progress, fg, muted, func() { w.Progress = nil })
	if w.Operation != nil {
		ui.Text(c, w.Operation.label+" · "+w.Operation.root).FontSize(11).TextColor(fg)
		ui.Text(c, word("操作进行中，请勿重复提交。", "Operation in progress. Do not submit it again.")).FontSize(10).TextColor(muted)
		return true
	}
	paintGitFollowups(c, m, fg, muted)
	if paintRemoteWorkflow(c, m, fg, muted) {
		return true
	}
	if !w.Repository {
		ui.Text(c, word("此目录尚未初始化 Git。初始化会创建本地 main 分支，不提交文件。", "This folder has no Git repository. Initialize creates a local main branch without committing files.")).FontSize(11).TextColor(muted)
		if ui.Button(c, word("初始化 Git", "Initialize Git")).Disabled(strings.TrimSpace(m.WorkspaceRoot) == "").Clicked() {
			root := m.WorkspaceRoot
			beginGitOperation(c, w, word("正在初始化", "Initializing"), func(ctx context.Context) gitOperationResult {
				out, err := git.Initialize(root)
				return gitOperationResult{text: out, err: err}
			})
		}
		if w.Status != "" {
			ui.Text(c, w.Status).FontSize(11).TextColor(fg)
		}
		return true
	}
	if ui.Button(c, word("核对草稿 PR…", "Review draft PR…")).Clicked() {
		w.RequestPR(m.WorkspaceRoot)
	}
	if ui.Button(c, word("发布仓库…", "Publish repository…")).Clicked() {
		w.PublishOpen = !w.PublishOpen
		w.PublishReview = false
	}
	if w.PublishOpen {
		paintPublish(c, m, fg, muted)
	}
	return false
}

func paintPublish(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if !w.PublishReview {
		ui.Text(c, word("填写已有 HTTPS/SSH 远端 URL，或用已登录的 gh 创建 GitHub 仓库。新仓库默认私有。", "Enter an existing HTTPS/SSH remote URL, or create a GitHub repository using your signed-in gh account. New repositories default to private.")).FontSize(10).TextColor(muted)
		ui.TextInput(c, &w.PublishRemoteURL).FillWidth().Placeholder(word("已有远端 URL（可留空）", "Existing remote URL (optional)"))
		ui.TextInput(c, &w.PublishName).FillWidth().Placeholder(word("仓库名称或 owner/name", "Repository name or owner/name"))
		if strings.TrimSpace(w.PublishRemoteURL) == "" {
			ui.Row(c).Gap(6).Children(func() {
				privateLabel := word("私有", "Private")
				publicLabel := word("公开", "Public")
				if w.PublishPublic {
					publicLabel = "▸ " + publicLabel
				} else {
					privateLabel = "▸ " + privateLabel
				}
				if ui.Button(c, privateLabel).Clicked() {
					w.PublishPublic = false
				}
				if ui.Button(c, publicLabel).Clicked() {
					w.PublishPublic = true
				}
			})
		}
		if ui.Button(c, word("核对发布目标", "Review publish target")).Clicked() {
			input, err := git.PreparePublish(m.WorkspaceRoot, git.PublishInput{Name: w.PublishName, RemoteURL: w.PublishRemoteURL, Public: w.PublishPublic})
			if err != nil {
				w.Status = err.Error()
			} else {
				w.ReviewedPublish = input
				w.PublishReview = true
				w.Status = ""
			}
		}
		return
	}
	input := w.ReviewedPublish
	target := input.RemoteURL
	if target == "" {
		visibility := word("私有", "private")
		if input.Public {
			visibility = word("公开，任何人可查看", "public, visible to everyone")
		}
		target = "GitHub " + input.Name + " (" + visibility + ")"
	}
	ui.Text(c, word("发布目标：", "Publish destination: ")+target).FontSize(11).TextColor(fg)
	ui.Text(c, word("将设置 origin；若已有提交，会上传当前分支及其提交历史。空仓库仅设置远端。未提交文件与编辑器草稿不会上传。", "This sets origin and uploads the current branch and its commit history when commits exist. An empty repository gets the remote only. Uncommitted files and editor drafts are excluded.")).FontSize(10).TextColor(muted)
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, word("确认发布仓库", "Confirm publish repository")).Clicked() {
			root := m.WorkspaceRoot
			w.PublishReview = false
			beginGitOperation(c, w, word("正在发布仓库", "Publishing repository"), func(ctx context.Context) gitOperationResult {
				out, err := git.PublishRepository(root, input)
				return gitOperationResult{text: out, err: err}
			})
		}
		if ui.Button(c, word("返回修改", "Back")).Clicked() {
			w.PublishReview = false
		}
		if ui.Button(c, word("取消发布", "Cancel publish")).Clicked() {
			w.PublishOpen, w.PublishReview = false, false
		}
	})
}

func pushReviewText(plan git.PushPlan) string {
	text := fmt.Sprintf("%s → %s/%s\n%s", plan.Branch, plan.Remote, plan.Target, plan.URL)
	if plan.DefaultBranch {
		text += "\n" + word("目标是默认分支。仅在确认要更新它时继续。", "The destination is the default branch. Continue only if you intend to update it.")
	}
	return text
}

func reviewCommitPush(m Model) {
	w := m.Workbench
	if strings.TrimSpace(w.Message) == "" {
		w.Status = word("请填写提交说明", "Enter a commit message")
		return
	}
	plan, err := git.PreparePush(m.WorkspaceRoot)
	if err != nil {
		w.Status = err.Error()
		return
	}
	text, token, err := git.IndexReview(m.WorkspaceRoot)
	if err != nil {
		w.Status = err.Error()
		return
	}
	w.PushAfterCommit = true
	w.ReviewedPush = plan
	w.ReviewToken, w.ReviewText = token, text
	scopeDiff(m, "", true)
	w.Status = word("核对已暂存 diff 和推送目标；确认后先创建本地提交，再普通推送。", "Review the staged diff and push destination. Confirm creates a local commit and then performs a normal push.")
}

func confirmReviewedCommit(c *ui.Context, m Model) {
	w := m.Workbench
	root, message, token, plan := m.WorkspaceRoot, w.Message, w.ReviewToken, w.ReviewedPush
	push := w.PushAfterCommit
	label := word("正在提交", "Committing")
	if push {
		label = word("正在提交并推送", "Committing and pushing")
	}
	beginGitOperation(c, w, label, func(ctx context.Context) gitOperationResult {
		if push {
			result, err := git.CommitPushReviewed(root, message, token, plan)
			return gitOperationResult{text: result.Output, err: err, committed: result.Committed}
		}
		out, err := git.CommitReviewed(root, message, token)
		return gitOperationResult{text: out, err: err, committed: err == nil}
	})
}
