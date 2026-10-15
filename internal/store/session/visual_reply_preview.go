package session

import (
	"context"
	"fmt"
	"sync/atomic"
)

var previewSequence atomic.Uint64

type VisualReplyCapture struct {
	PNG                                  []byte
	Width, ContentHeight, CapturedHeight int
}

// PreviewVisualReply only renders HTML already authorized by LoadVisualReply.
// Callbacks receive no path or URL and must enforce the native isolation gate.
func (s *Store) PreviewVisualReply(ctx context.Context, sessionID, attachmentID string, render func(context.Context, string) (VisualReplyCapture, error)) {
	s.mu.Lock()
	v := &s.visualReply
	if render == nil || ctx.Err() != nil || s.client == nil || s.selectedID != sessionID || v.SessionID != sessionID || v.AttachmentID != attachmentID || !v.Ready || v.generation != s.followGeneration || v.PreviewLoading {
		s.mu.Unlock()
		return
	}
	s.stopInteractiveVisualReplyLocked()
	client, generation, html := s.client, s.followGeneration, v.HTML
	s.visualReplyToken++
	token := s.visualReplyToken
	v.token = token
	ctx, cancel := context.WithCancel(ctx)
	v.previewCancel = cancel
	v.PreviewLoading = true
	v.PreviewError = ""
	v.PreviewPNG = nil
	v.PreviewID = 0
	s.mu.Unlock()
	s.fire()
	result, err := render(ctx, html)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && (len(result.PNG) == 0 || result.Width < 240 || result.Width > 1600 || result.CapturedHeight < 1 || result.CapturedHeight > 4000 || result.ContentHeight < result.CapturedHeight) {
		err = fmt.Errorf("Invalid isolated preview result")
	}
	cancel()
	s.mu.Lock()
	if s.client != client || s.selectedID != sessionID || s.followGeneration != generation || s.visualReply.token != token || s.visualReplyToken != token {
		s.mu.Unlock()
		return
	}
	v = &s.visualReply
	v.previewCancel = nil
	v.PreviewLoading = false
	if err != nil {
		v.PreviewError = err.Error()
	} else {
		v.PreviewPNG = append([]byte(nil), result.PNG...)
		v.PreviewWidth = result.Width
		v.PreviewContentHeight = result.ContentHeight
		v.PreviewCapturedHeight = result.CapturedHeight
		v.PreviewID = previewSequence.Add(1)
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) cancelVisualReplyLocked() {
	s.stopInteractiveVisualReplyLocked()
	if s.visualReply.previewCancel != nil {
		s.visualReply.previewCancel()
	}
	s.visualReplyToken++
	s.visualReply = visualReplyRead{}
}
