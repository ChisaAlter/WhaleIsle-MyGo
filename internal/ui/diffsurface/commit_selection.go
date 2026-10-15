package diffsurface

import (
	"context"
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/desktop/git"
)

type commitSelectionState struct {
	Excluded                      map[string]bool
	NewBranch                     string
	MakeBranch, LargeAcknowledged bool
	LargeFiles                    []git.LargeFile
	ScanError                     string
	Plan                          *git.SelectedCommitPlan
	MessageSnapshot               string
	Task                          *commitPreparation
	Push                          bool
	PushPlan                      git.PushPlan
	Source                        string
	WantPR                        bool
	PRTitle, PRBody               string
	PRPlan                        *git.PullRequestPlan
	StackPlan                     *git.StackPlan
}
type commitPrepared struct {
	plan            git.SelectedCommitPlan
	message, source string
	push            git.PushPlan
	pr              *git.PullRequestPlan
	stack           git.StackPlan
	err             error
}
type commitPreparation struct {
	root   string
	cancel context.CancelFunc
	done   chan commitPrepared
	push   bool
}

func (w *Workbench) commitState() *commitSelectionState {
	if w.Commit == nil {
		w.Commit = &commitSelectionState{Excluded: map[string]bool{}}
	}
	return w.Commit
}
func (w *Workbench) invalidateCommitReview() {
	if w.Commit != nil {
		w.Commit.Plan = nil
		w.Commit.PRPlan = nil
		w.Commit.StackPlan = nil
		w.Commit.Source = ""
	}
	w.ReviewToken = ""
	w.ReviewText = ""
}
func (w *Workbench) cancelCommitPreparation() {
	if w.Commit != nil && w.Commit.Task != nil {
		w.Commit.Task.cancel()
		w.Commit.Task = nil
		if w.Progress != nil {
			w.Progress.Finish(word("准备已取消", "Preparation canceled"), false)
		}
	}
	w.invalidateCommitReview()
}
func (w *Workbench) selectedCommitPaths() []string {
	s := w.commitState()
	var paths []string
	for _, c := range w.Changes {
		if !s.Excluded[c.Path] {
			paths = append(paths, c.Path)
		}
	}
	return paths
}
func (w *Workbench) scanCommitFiles() {
	s := w.commitState()
	s.LargeFiles = nil
	s.ScanError = ""
	files, e := git.CheckLargeFiles(w.Root)
	if e != nil {
		s.ScanError = e.Error()
	} else {
		s.LargeFiles = files
	}
}

// RequestCommit routes all native entry points to the same selection/review flow.
func (w *Workbench) RequestCommit(root, message string) {
	w.cancelCommitPreparation()
	w.cancelRemotePreparation()
	w.Remote = nil
	if w.Root != root {
		*w = Workbench{Operation: w.Operation}
	}
	w.reload(root)
	w.Message = message
	w.Status = word("选择本次提交的文件并核对实际 diff。", "Select the files for this commit, then review the exact diff.")
}
func finishCommitPreparation(m Model) {
	w := m.Workbench
	if w == nil || w.Commit == nil || w.Commit.Task == nil {
		return
	}
	s := w.Commit
	task := s.Task
	if task.root != m.WorkspaceRoot {
		w.cancelCommitPreparation()
		return
	}
	select {
	case result := <-task.done:
		task.cancel()
		s.Task = nil
		if result.err != nil {
			w.Status = result.err.Error()
			return
		}
		s.Plan = &result.plan
		s.StackPlan = &result.stack
		s.PRPlan = result.pr
		if result.pr != nil {
			s.PRTitle, s.PRBody = result.pr.Title, result.pr.Body
		}
		s.MessageSnapshot = result.message
		s.Source = result.source
		s.Push, s.PushPlan = task.push, result.push
		w.Message = result.message
		w.ReviewText = result.plan.Patch
		w.ReviewToken = "selected"
		w.CancelComparison()
		w.BaseRef = ""
		w.Status = word("核对下方所选文件的完整 diff，再确认。未选文件的暂存内容保持不变。", "Review the selected files' complete diff below, then confirm. Unselected staging is preserved.")
	default:
	}
}
func beginCommitPreparation(c *ui.Context, m Model, push, createPR bool) {
	w := m.Workbench
	s := w.commitState()
	if s.Task != nil || w.Operation != nil {
		return
	}
	paths := w.selectedCommitPaths()
	if len(paths) == 0 {
		w.Status = word("至少选择一个文件", "Select at least one file")
		return
	}
	if s.ScanError != "" {
		w.Status = s.ScanError
		return
	}
	for _, f := range s.LargeFiles {
		if !s.Excluded[f.Path] && !s.LargeAcknowledged {
			w.Status = word("所选文件超过 100 MiB；请排除或明确保留后重试。", "A selected file exceeds 100 MiB. Exclude it or explicitly keep large files before reviewing.")
			return
		}
	}
	branch := ""
	if s.MakeBranch {
		branch = strings.TrimSpace(s.NewBranch)
		if branch == "" {
			w.Status = word("请填写新分支名称", "Enter a new branch name")
			return
		}
		if push {
			w.Status = word("先在新分支提交，再单独核对推送目标。", "Commit on the new branch first, then review its push destination.")
			return
		}
	}
	largeAcknowledged := s.LargeAcknowledged
	root, message := m.WorkspaceRoot, w.Message
	prTitle, prBody := s.PRTitle, s.PRBody
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	task := &commitPreparation{root: root, cancel: cancel, done: make(chan commitPrepared, 1), push: push}
	w.invalidateCommitReview()
	s.Task = task
	w.Status = ""
	invalidate := c.Services().Invalidate
	progress := git.NewProgress(root, word("正在准备所选文件…", "Preparing selected files…"), invalidate)
	w.Progress = progress
	ctx = git.WithProgress(ctx, progress)
	go func() {
		r := commitPrepared{message: message}
		r.plan, r.err = git.PrepareSelectedCommit(ctx, root, paths, branch)
		if r.err == nil && !safelyAcknowledgesLarge(r.plan, largeAcknowledged) {
			r.err = errors.New("selected file size changed; refresh and review the large-file warning")
		}
		if r.err == nil && strings.TrimSpace(message) == "" {
			progress.Phase(word("正在生成提交说明…", "Generating commit message…"))
			var generated git.GeneratedCommitMessage
			generated, r.err = git.GenerateSelectedCommitMessage(ctx, root, r.plan)
			r.message, r.source = generated.Text(), generated.Source
		}
		if r.err == nil && push {
			r.push, r.err = git.PreparePush(root)
		}
		if r.err == nil && createPR {
			if strings.TrimSpace(prTitle) == "" {
				prTitle = strings.SplitN(r.message, "\n", 2)[0]
			}
			if strings.TrimSpace(prBody) == "" {
				prBody = r.plan.Summary
			}
			var pr git.PullRequestPlan
			pr, r.err = git.PreparePullRequest(ctx, root, r.push, prTitle, prBody)
			r.pr = &pr
		}
		if r.err == nil {
			var pushPlan *git.PushPlan
			if push {
				pushPlan = &r.push
			}
			r.stack, r.err = git.PrepareStack(ctx, root, &r.plan, r.message, pushPlan, r.pr)
		}
		phase := word("核对内容已准备好", "Review ready")
		if r.err != nil {
			phase = word("准备失败，可重试或手写说明", "Preparation failed; retry or enter a message")
		}
		progress.Finish(phase, r.err != nil)
		task.done <- r
		if invalidate != nil {
			invalidate()
		}
	}()
}

// A preparation captures this value before any worker begins; no widget state
// or mutable selection is accessed by the worker.
func safelyAcknowledgesLarge(p git.SelectedCommitPlan, ack bool) bool {
	return len(p.LargeFiles) == 0 || ack
}
func paintCommitPreparation(c *ui.Context, m Model, fg, muted ui.Color) bool {
	finishCommitPreparation(m)
	w := m.Workbench
	if w.Commit == nil || w.Commit.Task == nil {
		return false
	}
	ui.Text(c, word("正在准备所选文件 / 生成说明…", "Preparing selected files / generating message…")).FontSize(11).TextColor(fg)
	ui.Text(c, word("生成失败可重试或手写说明；此时还未创建提交。", "If generation fails, retry or write a message. No commit has been created yet.")).FontSize(10).TextColor(muted)
	if ui.Button(c, word("取消准备", "Cancel preparation")).Clicked() {
		w.cancelCommitPreparation()
		w.Status = word("已取消，可修改选择或手写说明。", "Canceled. Change the selection or enter a message.")
	}
	return true
}
func paintCommitCheckbox(c *ui.Context, w *Workbench, path string) {
	s := w.commitState()
	selected := !s.Excluded[path]
	ui.Checkbox(c, &selected, "")
	if selected == s.Excluded[path] {
		s.Excluded[path] = !selected
		s.LargeAcknowledged = false
		w.invalidateCommitReview()
	}
}
func paintCommitSelection(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	s := w.commitState()
	ui.Text(c, word("勾选文件将提交其已保存的完整工作副本（含该文件未暂存修改）；未勾选的暂存保持不变。", "Checked files commit their complete saved working copies, including unstaged edits. Unchecked staging is preserved.")).FontSize(10).TextColor(muted)
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, word("全选", "Select all")).Clicked() {
			s.Excluded = map[string]bool{}
			s.LargeAcknowledged = false
			w.invalidateCommitReview()
		}
		if ui.Button(c, word("全不选", "Select none")).Clicked() {
			for _, ch := range w.Changes {
				s.Excluded[ch.Path] = true
			}
			s.LargeAcknowledged = false
			w.invalidateCommitReview()
		}
	})
	if s.ScanError != "" {
		ui.Text(c, word("大文件扫描失败：", "Large-file scan failed: ")+s.ScanError).FontSize(10).TextColor(fg)
	}
	if len(s.LargeFiles) > 0 {
		ui.Text(c, word("警告：以下文件超过 100 MiB，远端可能拒绝。", "Warning: these files exceed 100 MiB and may be rejected by the remote.")).FontSize(11).TextColor(fg)
		for _, file := range s.LargeFiles {
			f := file
			ui.Row(c).Gap(6).Children(func() {
				ui.Text(c, fmt.Sprintf("%s · %.2f MiB", f.Path, float64(f.Size)/(1024*1024))).FontSize(10).TextColor(fg)
				caption := word("排除此文件", "Exclude file")
				if s.Excluded[f.Path] {
					caption = word("已排除（点此包含）", "Excluded (include)")
				}
				if ui.Button(c, caption).FontSize(10).Clicked() {
					s.Excluded[f.Path] = !s.Excluded[f.Path]
					s.LargeAcknowledged = false
					w.invalidateCommitReview()
				}
			})
		}
		ui.Checkbox(c, &s.LargeAcknowledged, word("保留仍被勾选的大文件；我已了解可能被拒绝", "Keep checked large files; I understand they may be rejected"))
	}
	beforeMessage, beforeBranch, beforeMake := w.Message, s.NewBranch, s.MakeBranch
	ui.TextInput(c, &w.Message).FillWidth().Placeholder(word("提交说明（留空自动生成，可修改后确认）", "Commit message (leave empty to generate, then review)"))
	ui.Checkbox(c, &s.MakeBranch, word("在新分支提交", "Commit on a new branch"))
	if s.MakeBranch {
		ui.TextInput(c, &s.NewBranch).FillWidth().Placeholder("feature/my-change")
	}
	if beforeMessage != w.Message || beforeBranch != s.NewBranch || beforeMake != s.MakeBranch {
		w.invalidateCommitReview()
	}
	beforePR, beforeTitle, beforeBody := s.WantPR, s.PRTitle, s.PRBody
	ui.Checkbox(c, &s.WantPR, word("准备提交、推送并创建草稿 PR", "Prepare commit, push and draft PR"))
	if s.WantPR {
		ui.TextInput(c, &s.PRTitle).FillWidth().Placeholder(word("PR 标题（空白时取提交标题）", "PR title (defaults to commit subject)"))
		ui.TextArea(c, &s.PRBody).FillWidth().Lines(2, 5).Placeholder(word("PR 正文（空白时取文件摘要）", "PR body (defaults to file summary)"))
		ui.Text(c, word("PR 使用手写文案；会展示收件仓库和目标分支，创建前要求仓库干净。", "PR text is editable. Review the recipient and base; PR creation requires a clean repository.")).FontSize(10).TextColor(muted)
	}
	if beforePR != s.WantPR || beforeTitle != s.PRTitle || beforeBody != s.PRBody {
		w.invalidateCommitReview()
	}
	if s.Plan == nil {
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, word("核对所选文件并提交…", "Review selected files to commit…")).Clicked() {
				beginCommitPreparation(c, m, false, false)
			}
			if ui.Button(c, word("核对提交并推送…", "Review commit and push…")).Disabled(s.MakeBranch).Clicked() {
				beginCommitPreparation(c, m, true, false)
			}
		})
		if s.WantPR && ui.Button(c, word("核对提交、推送及草稿 PR…", "Review commit, push and draft PR…")).Disabled(s.MakeBranch).Clicked() {
			beginCommitPreparation(c, m, true, true)
		}
		return
	}
	plan := *s.Plan
	branch := plan.Branch
	if plan.NewBranch != "" {
		branch += " → " + plan.NewBranch
	}
	ui.Text(c, word("提交分支：", "Commit branch: ")+branch).FontSize(11).TextColor(fg)
	ui.Text(c, plan.Summary).FontSize(10).TextColor(fg)
	ui.Text(c, s.MessageSnapshot).FontSize(11).TextColor(fg)
	if s.Source != "" {
		source := word("按文件列表生成", "Generated from file list")
		if s.Source == "model" {
			source = word("由已配置模型生成", "Generated by configured model")
		}
		ui.Text(c, source).FontSize(10).TextColor(muted)
	}
	if s.Push {
		ui.Text(c, pushReviewText(s.PushPlan)).FontSize(11).TextColor(fg)
	}
	if s.PRPlan != nil {
		paintPRReview(c, *s.PRPlan, fg, muted)
	}
	ui.Row(c).Gap(6).Children(func() {
		caption := word("确认提交所选文件", "Confirm selected commit")
		if s.Push {
			caption = word("确认所选提交并推送", "Confirm selected commit and push")
		}
		if s.PRPlan != nil {
			caption = word("确认提交、推送并创建草稿 PR", "Confirm commit, push and draft PR")
		}
		if ui.Button(c, caption).Clicked() {
			confirmSelectedCommit(c, m)
		}
		if ui.Button(c, word("取消核对", "Cancel review")).Clicked() {
			w.invalidateCommitReview()
			w.Status = ""
		}
	})
}
func confirmSelectedCommit(c *ui.Context, m Model) {
	w := m.Workbench
	s := w.commitState()
	if s.Plan == nil || s.StackPlan == nil {
		return
	}
	if w.Message != s.MessageSnapshot {
		w.invalidateCommitReview()
		w.Status = word("说明已更改，请重新核对。", "Message changed; review again.")
		return
	}
	root, plan := m.WorkspaceRoot, *s.StackPlan
	w.invalidateCommitReview()
	beginGitOperation(c, w, word("正在执行已核对的 Git 操作", "Running reviewed Git actions"), func(ctx context.Context) gitOperationResult {
		result, e := git.RunCommitStack(ctx, root, plan)
		return gitOperationResult{text: result.Output, err: e, committed: result.Committed, stack: &result}
	})
}
