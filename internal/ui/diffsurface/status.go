package diffsurface

import (
	"context"
	"time"

	"whaleisle/internal/desktop/git"
)

type repositoryStatusResult struct {
	status git.RepositoryStatus
	err    error
}
type repositoryStatusRequest struct {
	root   string
	cancel context.CancelFunc
	done   chan repositoryStatusResult
}

func (w *Workbench) cancelRepositoryStatus() {
	if w.repositoryStatusRequest != nil {
		w.repositoryStatusRequest.cancel()
		w.repositoryStatusRequest = nil
	}
}
func (w *Workbench) reloadRepositoryStatus() {
	w.cancelRepositoryStatus()
	w.RepositoryStatus = nil
	w.RepositoryStatusError = ""
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	request := &repositoryStatusRequest{root: w.Root, cancel: cancel, done: make(chan repositoryStatusResult, 1)}
	w.repositoryStatusRequest = request
	root, invalidate := w.Root, w.invalidate
	go func() {
		defer cancel()
		status, err := git.ReadRepositoryStatus(ctx, root)
		request.done <- repositoryStatusResult{status: status, err: err}
		if invalidate != nil {
			invalidate()
		}
	}()
}
func (w *Workbench) finishRepositoryStatus() {
	request := w.repositoryStatusRequest
	if request == nil {
		return
	}
	if request.root != w.Root {
		w.cancelRepositoryStatus()
		return
	}
	select {
	case result := <-request.done:
		w.repositoryStatusRequest = nil
		if result.err != nil {
			w.RepositoryStatusError = result.err.Error()
		} else {
			w.RepositoryStatus = &result.status
		}
	default:
	}
}

func branchUnsupportedReason() string {
	return word("名称包含桌面无法安全传给 git 的字符，无法从这里切换。", "Name contains characters the desktop cannot pass to git safely.")
}

func (w *Workbench) isDirty() bool {
	if w == nil {
		return false
	}
	if len(w.Changes) > 0 {
		return true
	}
	if w.RepositoryStatus != nil {
		return w.RepositoryStatus.Dirty
	}
	return false
}

func (w *Workbench) statusBranchLabel() string {
	if w == nil || w.RepositoryStatus == nil || w.RepositoryStatus.Branch == "" {
		return word("（分离 HEAD）", "(detached HEAD)")
	}
	return w.RepositoryStatus.Branch
}

func (w *Workbench) statusNotice() string {
	if w == nil || w.RepositoryStatus == nil {
		return ""
	}
	s := *w.RepositoryStatus
	if s.Branch == "" {
		return word("处于分离 HEAD 状态：请先创建并检出分支，才能执行推送和拉取请求操作。", "Detached HEAD: create and checkout a branch to enable push and pull request actions.")
	}
	if !w.isDirty() && s.Behind > 0 && s.Ahead == 0 {
		return word("落后于上游，请先拉取或变基。", "Behind upstream. Pull/rebase first.")
	}
	if !w.isDirty() && s.Behind > 0 && s.Ahead > 0 {
		return word("分支与上游存在分叉，请先拉取或变基后再推送。", "Branch has diverged from upstream. Pull/rebase before pushing.")
	}
	return ""
}

func (w *Workbench) commitDisabledReason() string {
	if w == nil || !w.Repository {
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	if w.Operation != nil {
		return word("Git 操作进行中。", "Git action in progress.")
	}
	if !w.isDirty() {
		return word("工作区干净，请先修改文件后再提交。", "Worktree is clean. Make changes before committing.")
	}
	return ""
}

func (w *Workbench) pushDisabledReason() string {
	if w == nil || !w.Repository {
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	if w.Operation != nil {
		return word("Git 操作进行中。", "Git action in progress.")
	}
	if w.RepositoryStatus == nil {
		if w.isDirty() {
			return word("推送前请先提交或暂存本地更改。", "Commit or stash local changes before pushing.")
		}
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	s := *w.RepositoryStatus
	if s.Branch == "" {
		return word("处于分离 HEAD 状态：请先检出分支后再推送。", "Detached HEAD: checkout a branch before pushing.")
	}
	if w.isDirty() {
		return word("推送前请先提交或暂存本地更改。", "Commit or stash local changes before pushing.")
	}
	if s.Behind > 0 {
		return word("分支落后于上游，请先拉取或变基后再推送。", "Branch is behind upstream. Pull/rebase before pushing.")
	}
	if !s.HasUpstream && !s.HasOrigin {
		return word("推送前请先添加 \"origin\" 远端。", "Add an \"origin\" remote before pushing.")
	}
	if s.Ahead == 0 && !s.AheadUnreliable {
		return word("没有可推送的本地提交。", "No local commits to push.")
	}
	return ""
}

func (w *Workbench) createPRDisabledReason() string {
	if w == nil || !w.Repository {
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	if w.Operation != nil {
		return word("Git 操作进行中。", "Git action in progress.")
	}
	if w.RepositoryStatus == nil {
		if w.isDirty() {
			return word("创建拉取请求前请先提交本地更改。", "Commit local changes before creating a pull request.")
		}
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	s := *w.RepositoryStatus
	if s.Branch == "" {
		return word("处于分离 HEAD 状态：请先检出分支后再创建拉取请求。", "Detached HEAD: checkout a branch before creating a pull request.")
	}
	if w.isDirty() {
		return word("创建拉取请求前请先提交本地更改。", "Commit local changes before creating a pull request.")
	}
	if !s.HasUpstream && !s.HasOrigin {
		return word("创建拉取请求前请先添加 \"origin\" 远端。", "Add an \"origin\" remote before creating a pull request.")
	}
	if s.Behind > 0 {
		return word("分支落后于上游，请先拉取或变基后再创建拉取请求。", "Branch is behind upstream. Pull/rebase before creating a pull request.")
	}
	if s.AheadOfDefault == 0 && s.Ahead == 0 && !s.AheadUnreliable {
		return word("没有可包含在拉取请求中的本地提交。", "No local commits to include in a pull request.")
	}
	return ""
}

func (w *Workbench) publishDisabledReason() string {
	if w == nil || !w.Repository || w.RepositoryStatus == nil {
		return word("Git 状态不可用。", "Git status is unavailable.")
	}
	if w.Operation != nil {
		return word("Git 操作进行中。", "Git action in progress.")
	}
	if w.RepositoryStatus.HasOrigin {
		return word("发布仓库不可用。", "Publish repository is unavailable.")
	}
	return ""
}
