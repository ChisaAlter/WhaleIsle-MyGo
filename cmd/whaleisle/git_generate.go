package main

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo"

	desktopgit "whaleisle/internal/desktop/git"
)

// gitCommitGenerationTimeout bounds the title-bar generation. The original
// request budget is 45s (git-generate.js requestGeneratedJSON); the extra room
// covers the local Git reads that build its input.
const gitCommitGenerationTimeout = 60 * time.Second

// gitCommitGenerationState reports the current generation for one frame.
func (s *mainState) gitCommitGenerationState() (generating bool, failure string) {
	if s == nil {
		return false, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gitCommitGenerating, s.gitCommitGenErr
}

// takeGeneratedCommitMessage consumes the generated copy exactly once. The view
// applies it to its own draft, so a worker finishing mid-frame cannot be
// overwritten by that frame's draft save.
func (s *mainState) takeGeneratedCommitMessage() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gitCommitGenPending == nil {
		return "", false
	}
	text := *s.gitCommitGenPending
	s.gitCommitGenPending = nil
	return text, true
}

// runGitCommitMessageGeneration produces copy for the current uncommitted
// change and hands it to the draft. A failure keeps the user's text and reports
// why: the heuristic fallback and manual entry both stay available.
func (s *mainState) runGitCommitMessageGeneration(win *mygo.Window, root string) {
	if s == nil || strings.TrimSpace(root) == "" {
		return
	}
	s.mu.Lock()
	if s.gitCommitGenerating || s.quitting {
		s.mu.Unlock()
		return
	}
	s.gitCommitGenerating = true
	s.gitCommitGenErr = ""
	s.mu.Unlock()
	if win != nil && !win.IsDestroyed() {
		win.Update(func() {})
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), gitCommitGenerationTimeout)
		defer cancel()
		generated, err := desktopgit.GenerateWorktreeCommitMessage(ctx, root)
		s.mu.Lock()
		s.gitCommitGenerating = false
		if err != nil {
			s.gitCommitGenErr = err.Error()
		} else if text := strings.TrimSpace(generated.Text()); text != "" {
			s.gitCommitGenPending = &text
			s.gitCommitGenErr = ""
		}
		s.mu.Unlock()
		if win != nil && !win.IsDestroyed() {
			win.Update(func() {})
		}
	}()
}
