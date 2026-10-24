package main

import "testing"

// The frame reads the generation fields while holding state.mu. These helpers
// must therefore never lock: a re-entrant Lock hung the main window's first
// paint (see the WHALEISLE_STACK_DUMP goroutine dump in the round doc).
func TestGeneratedCommitAccessorsRunUnderTheFrameLock(t *testing.T) {
	var s mainState
	text := "feat: test"
	s.gitCommitGenPending = &text
	s.gitCommitGenerating = true
	s.gitCommitGenErr = "boom"

	s.mu.Lock()
	defer s.mu.Unlock()

	if got, ok := s.takeGeneratedCommitMessageLocked(); !ok || got != text {
		t.Fatalf("take = %q, %v; want %q, true", got, ok, text)
	}
	if s.gitCommitGenPending != nil {
		t.Fatal("pending copy was not consumed")
	}
	if got, ok := s.takeGeneratedCommitMessageLocked(); ok || got != "" {
		t.Fatalf("second take = %q, %v; want empty, false", got, ok)
	}
	if generating, failure := s.gitCommitGenerationStateLocked(); !generating || failure != "boom" {
		t.Fatalf("state = %v, %q; want true, boom", generating, failure)
	}
}

func TestGeneratedCommitAccessorsTolerateNilState(t *testing.T) {
	var s *mainState
	if got, ok := s.takeGeneratedCommitMessageLocked(); ok || got != "" {
		t.Fatalf("nil take = %q, %v", got, ok)
	}
	if generating, failure := s.gitCommitGenerationStateLocked(); generating || failure != "" {
		t.Fatalf("nil state = %v, %q", generating, failure)
	}
}
