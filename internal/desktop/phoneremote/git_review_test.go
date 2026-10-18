package phoneremote

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestGitReviewCallbackAndRevoke(t *testing.T) {
	child := &fixtureChild{}
	r := NewRuntime()
	r.child = child
	r.state.Listening = true
	r.SetGitAuthorizer(func(string, string) bool { return true })
	r.SetGitReviewer(func(ctx context.Context, root string) (map[string]any, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 106*time.Second {
			t.Error("missing bounded deadline")
		}
		return map[string]any{"ok": true, "status": "created"}, nil
	})
	r.reviewGitRequest(child, []byte(`{"id":"1","root":"fixture"}`))
	if !bytes.Contains(child.input.Bytes(), []byte(`"status":"created"`)) {
		t.Fatal(child.input.String())
	}
	entered := make(chan struct{})
	finished := make(chan struct{})
	r.SetGitReviewer(func(ctx context.Context, root string) (map[string]any, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	go func() { r.reviewGitRequest(child, []byte(`{"id":"2","root":"fixture"}`)); close(finished) }()
	<-entered
	r.RevokeGit()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("revocation did not cancel review")
	}
	if !bytes.Contains(child.input.Bytes(), []byte(`"error":"context canceled"`)) {
		t.Fatal(child.input.String())
	}
}
func TestGitReviewUnavailableFailsClosed(t *testing.T) {
	child := &fixtureChild{}
	r := NewRuntime()
	r.child = child
	r.state.Listening = true
	r.reviewGitRequest(child, []byte(`{"id":"1","root":"fixture"}`))
	if !bytes.Contains(child.input.Bytes(), []byte(`"error"`)) {
		t.Fatal("missing review callback was accepted")
	}
}
