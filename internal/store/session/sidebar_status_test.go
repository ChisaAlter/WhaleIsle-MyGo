package session

import "testing"

func TestSidebarCompletionUnread(t *testing.T) {
	s := New()
	s.selectedID = "main"
	s.observeSidebarRunningLocked("other", false, false)
	if s.completionUnread["other"] {
		t.Fatal("initial idle invented completion")
	}
	s.observeSidebarRunningLocked("other", true, false)
	s.observeSidebarRunningLocked("other", false, false)
	if !s.completionUnread["other"] {
		t.Fatal("background completion not unread")
	}
	s.observeSidebarRunningLocked("main", true, false)
	s.observeSidebarRunningLocked("main", false, false)
	if s.completionUnread["main"] {
		t.Fatal("visible main marked unread")
	}
	s.MarkMainConversationViewed("other")
	if s.completionUnread["other"] {
		t.Fatal("main read did not clear completion")
	}
	s.observeSidebarRunningLocked("other", true, false)
	if s.completionUnread["other"] {
		t.Fatal("restart retained completion")
	}
}
func TestSidebarCompletionBaselineAndOwner(t *testing.T) {
	s := New()
	s.selectedID = "root"
	s.reconcileSidebarStatusesLocked([]Item{{SessionID: "root", Running: true}, {SessionID: "background", Running: true}})
	s.MarkMainConversationViewed("child")
	s.reconcileSidebarStatusesLocked([]Item{{SessionID: "root", Running: false}, {SessionID: "background", Running: false}})
	if !s.completionUnread["root"] || !s.completionUnread["background"] {
		t.Fatal("hidden parent/background completion omitted")
	}
	s.selectSessionLocked("root")
	if s.completionUnread["root"] {
		t.Fatal("select existing root did not clear unread")
	}
	s.reconcileSidebarStatusesLocked([]Item{{SessionID: "root"}})
	if s.completionUnread["background"] {
		t.Fatal("removed session retained unread")
	}
	s.observeSidebarRunningLocked("early", false, true)
	if !s.completionUnread["early"] {
		t.Fatal("actual stop before first baseline omitted")
	}
	s.resetSidebarStatusesLocked()
	if len(s.completionUnread) != 0 || len(s.sidebarRunning) != 0 {
		t.Fatal("Host reset leaked attention")
	}
}
