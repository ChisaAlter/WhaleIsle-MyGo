package session

import "testing"

func TestRunningCount(t *testing.T) {
	var s Store
	s.mu.Lock()
	s.items = []Item{
		{SessionID: "a", Running: true},
		{SessionID: "b", Running: false},
		{SessionID: "c", Running: true},
	}
	s.mu.Unlock()
	if n := s.RunningCount(); n != 2 {
		t.Fatalf("RunningCount() = %d, want 2", n)
	}
	if (*Store)(nil).RunningCount() != 0 {
		t.Fatal("nil store should report 0")
	}
}
