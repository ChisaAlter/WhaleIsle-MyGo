package session

import (
	"context"
	"errors"
	"time"
)

// VisualReplyInteractive is a lifetime handle, never a browser/native bridge.
// The opener receives only a durable, authorized HTML snapshot, not file paths,
// URLs, account credentials, or a Host client.
type VisualReplyInteractive interface {
	Done() <-chan struct{}
	Close() error
}

type visualReplyInteraction struct {
	cancel context.CancelFunc
}

// OpenInteractiveVisualReply must run off the GUI thread after an explicit
// per-document gesture. It owns the helper until close, session/Host change,
// replacement, context cancellation or helper exit. A failed launch preserves
// source/save access and never falls back to running HTML in the main process.
func (s *Store) OpenInteractiveVisualReply(ctx context.Context, sessionID, attachmentID string, open func(context.Context, string) (VisualReplyInteractive, error)) {
	if open == nil || ctx.Err() != nil {
		return
	}
	readCtx, stopRead := context.WithTimeout(ctx, 30*time.Second)
	token := s.loadVisualReply(readCtx, sessionID, attachmentID)
	stopRead()
	if token == 0 {
		return
	}
	s.mu.Lock()
	v := &s.visualReply
	if ctx.Err() != nil || s.client == nil || s.selectedID != sessionID || v.SessionID != sessionID || v.AttachmentID != attachmentID || v.generation != s.followGeneration || !v.Ready || v.token != token || s.visualReplyToken != token {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	op := &visualReplyInteraction{cancel: cancel}
	v.interaction = op
	v.InteractiveLoading = true
	v.InteractiveActive = false
	v.InteractiveError = ""
	html := v.HTML
	s.mu.Unlock()
	s.fire()
	defer cancel()

	handle, err := open(ctx, html)
	if err == nil && handle == nil {
		err = errors.New("isolated preview returned no lifetime handle")
	}
	if err == nil {
		err = ctx.Err()
	}
	s.mu.Lock()
	current := s.ownsInteractiveVisualReplyLocked(op, sessionID, token)
	if current {
		s.visualReply.InteractiveLoading = false
		s.visualReply.InteractiveActive = err == nil
		if err != nil {
			s.visualReply.InteractiveError = err.Error()
			s.visualReply.interaction = nil
		}
	}
	s.mu.Unlock()
	if current {
		s.fire()
	}
	if !current || err != nil {
		if handle != nil {
			_ = handle.Close()
		}
		return
	}

	select {
	case <-ctx.Done():
	case <-handle.Done():
	}
	closeErr := handle.Close()
	s.mu.Lock()
	current = s.ownsInteractiveVisualReplyLocked(op, sessionID, token)
	if current {
		s.visualReply.interaction = nil
		s.visualReply.InteractiveActive = false
		s.visualReply.InteractiveLoading = false
		if closeErr != nil {
			s.visualReply.InteractiveError = closeErr.Error()
		}
	}
	s.mu.Unlock()
	if current {
		s.fire()
	}
}

func (s *Store) ownsInteractiveVisualReplyLocked(op *visualReplyInteraction, sessionID string, token uint64) bool {
	v := &s.visualReply
	return s.client != nil && s.selectedID == sessionID && v.SessionID == sessionID && v.generation == s.followGeneration && v.token == token && s.visualReplyToken == token && v.interaction == op
}

func (s *Store) stopInteractiveVisualReplyLocked() {
	if op := s.visualReply.interaction; op != nil {
		op.cancel()
	}
	s.visualReply.interaction = nil
	s.visualReply.InteractiveLoading = false
	s.visualReply.InteractiveActive = false
}

// StopInteractiveVisualReply leaves the authorized source and static preview
// available. Cancellation does not block the GUI waiting for the helper's job.
func (s *Store) StopInteractiveVisualReply() {
	s.mu.Lock()
	s.stopInteractiveVisualReplyLocked()
	s.mu.Unlock()
	s.fire()
}
