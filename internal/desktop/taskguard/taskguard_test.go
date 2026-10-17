package taskguard

import "testing"

func TestConfirmQuitSkipsWhenIdle(t *testing.T) {
	if !ConfirmQuit(nil, 0, "t", "%d running", "Quit", "Cancel") {
		t.Fatal("expected proceed when no running sessions")
	}
}
