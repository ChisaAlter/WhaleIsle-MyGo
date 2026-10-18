package phoneremote

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type GitReviewer func(context.Context, string) (map[string]any, error)

func (r *Runtime) SetGitReviewer(review GitReviewer) {
	r.mu.Lock()
	r.reviewGit = review
	r.mu.Unlock()
}
func (r *Runtime) reviewGitRequest(child Child, data []byte) {
	var e struct{ ID, Root string }
	if json.Unmarshal(data, &e) != nil || e.ID == "" || len(e.ID) > 64 {
		return
	}
	result := map[string]any{"action": "git-review-result", "id": e.ID}
	r.mu.Lock()
	review, auth := r.reviewGit, r.authorizeGit
	if r.child != child || review == nil || auth == nil || r.reviewCancel != nil {
		r.mu.Unlock()
		result["error"] = "Desktop PR review unavailable or already active"
		_ = r.sendControl(child, result)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 105*time.Second)
	r.reviewCancel = cancel
	r.mu.Unlock()
	defer func() { cancel(); r.mu.Lock(); r.reviewCancel = nil; r.mu.Unlock() }()
	valid := func() bool {
		r.mu.Lock()
		current := r.child == child && r.state.Listening
		r.mu.Unlock()
		return current && auth(e.Root, "gitCreateChangeRequest")
	}
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !valid() {
					cancel()
					return
				}
			}
		}
	}()
	var value map[string]any
	var err error
	if !valid() {
		err = errors.New("Mobile Git desktop session permission expired")
	} else {
		value, err = review(ctx, e.Root)
	}
	if err == nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["value"] = value
	}
	_ = r.sendControl(child, result)
}
