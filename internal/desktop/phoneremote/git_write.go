package phoneremote

import (
	"context"

	desktopgit "whaleisle/internal/desktop/git"
)

type gitWriteAdmission struct {
	child   Child
	cancel  context.CancelFunc
	release func()
}

func phoneGitWrite(action string) bool {
	switch action {
	case "gitFetchForStatus", "gitInit", "gitCommit", "gitPush", "gitPull", "gitCreateChangeRequest", "gitPublishRepository", "gitSwitchBranch", "gitCreateBranch":
		return true
	}
	return false
}

// The private child transport requests ownership only after its existing
// workspace authorization. Recheck on both sides of a queued admission.
func (r *Runtime) acquireGitWrite(child Child, id, root, action string) {
	if child == nil || !validHostActionID(id) || !phoneGitWrite(action) {
		return
	}
	r.mu.Lock()
	if child == nil || r.child != child || r.closing || len(r.gitWrites) >= 32 || r.gitWrites[id] != nil {
		r.mu.Unlock()
		_ = r.sendControl(child, map[string]any{"action": "git-write-result", "id": id, "allowed": false})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	request := &gitWriteAdmission{child: child, cancel: cancel}
	if r.gitWrites == nil {
		r.gitWrites = make(map[string]*gitWriteAdmission)
	}
	r.gitWrites[id] = request
	authorize := r.authorizeGit
	r.mu.Unlock()
	go func() {
		var release func()
		if authorize != nil && authorize(root, action) {
			_, release, _ = desktopgit.AcquireWrite(ctx, root, desktopgit.ReferenceWrite)
		}
		allowed := release != nil && ctx.Err() == nil && authorize != nil && authorize(root, action)
		r.mu.Lock()
		allowed = allowed && r.child == child && !r.closing && r.gitWrites[id] == request
		if allowed {
			request.release = release
		} else if r.gitWrites[id] == request {
			delete(r.gitWrites, id)
		}
		r.mu.Unlock()
		if !allowed && release != nil {
			release()
		}
		cancel()
		if err := r.sendControl(child, map[string]any{"action": "git-write-result", "id": id, "allowed": allowed}); err != nil && allowed {
			r.releaseGitWrite(child, id)
		}
	}()
}

func (r *Runtime) releaseGitWrite(child Child, id string) {
	r.mu.Lock()
	request := r.gitWrites[id]
	if request != nil && request.child == child {
		delete(r.gitWrites, id)
	} else {
		request = nil
	}
	r.mu.Unlock()
	if request != nil {
		request.cancel()
		if request.release != nil {
			request.release()
		}
	}
}

// Revocation cancels queued admission. Already running writes retain ownership
// until Node reports completion or that owned bridge has actually exited.
func (r *Runtime) cancelPendingGitWrites(child Child) {
	r.mu.Lock()
	for _, request := range r.gitWrites {
		if request.child == child && request.release == nil {
			request.cancel()
		}
	}
	r.mu.Unlock()
}

func (r *Runtime) releaseChildGitWrites(child Child) {
	r.mu.Lock()
	var pending []*gitWriteAdmission
	for id, request := range r.gitWrites {
		if request.child == child {
			pending = append(pending, request)
			delete(r.gitWrites, id)
		}
	}
	r.mu.Unlock()
	for _, request := range pending {
		request.cancel()
		if request.release != nil {
			request.release()
		}
	}
}
