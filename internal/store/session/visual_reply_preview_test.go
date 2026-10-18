package session

import (
	"context"
	"errors"
	"testing"

	"whaleisle/internal/dsh"
)

// Synthetic callback only; no renderer, Host, daemon or untrusted HTML runs.
func TestVisualReplyPreviewAuthorizedSnapshot(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "A"
	s.followGeneration = 7
	calls := 0
	render := func(context.Context, string) (VisualReplyCapture, error) {
		calls++
		return VisualReplyCapture{}, errors.New("isolation unavailable")
	}
	s.PreviewVisualReply(context.Background(), "A", "page", render)
	if calls != 0 {
		t.Fatal("rendered without authorized read")
	}
	s.visualReply = visualReplyRead{VisualReplyState: VisualReplyState{SessionID: "A", AttachmentID: "page", HTML: "fixture", Ready: true}, generation: 7}
	s.PreviewVisualReply(context.Background(), "A", "page", render)
	state := s.VisualReplySnapshot()
	if calls != 1 || !state.Ready || state.HTML != "fixture" || state.PreviewError != "isolation unavailable" {
		t.Fatal("failure lost source fallback", state)
	}
	started, done := make(chan struct{}), make(chan struct{})
	go func() {
		s.PreviewVisualReply(context.Background(), "A", "page", func(ctx context.Context, html string) (VisualReplyCapture, error) {
			close(started)
			<-ctx.Done()
			return VisualReplyCapture{}, ctx.Err()
		})
		close(done)
	}()
	<-started
	s.CloseVisualReply()
	<-done
	if state = s.VisualReplySnapshot(); state.Ready || state.PreviewLoading || len(state.PreviewPNG) > 0 {
		t.Fatal("closed preview survived", state)
	}
}
