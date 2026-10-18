package terminal

import "testing"

func TestWorkspaceSessionsKeepSeparateAndCloseExplicitly(t *testing.T) {
	count := 0
	s := Sessions{open: func(root string, dark bool) *Panel { count++; return &Panel{cwd: root} }}
	a := s.Get("/test/a", false)
	b := s.Get("/test/b", false)
	if a == b || s.Get("/test/a", true) != a || count != 2 {
		t.Fatal("workspace switch replaced or shared a session")
	}
	s.CloseWorkspace("/test/a")
	if s.Get("/test/b", false) != b {
		t.Fatal("closing a closed b")
	}
	if s.Get("/test/a", false) == a || count != 3 {
		t.Fatal("closed workspace did not reopen")
	}
	s.Close()
	if len(s.panels) != 0 {
		t.Fatal("close leaked sessions")
	}
}
