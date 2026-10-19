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
