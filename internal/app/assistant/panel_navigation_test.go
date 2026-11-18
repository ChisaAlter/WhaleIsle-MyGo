package assistantctl

import "testing"

func TestPanelCloseRevealsCanonicalConversationAndKeepsDraft(t *testing.T) {
	revealed := 0
	s := NewService(HostDeps{RevealMainConversation: func() { revealed++ }})
	t.Cleanup(s.SuspendAll)
	s.SetOpen(true)
	m, _, _ := s.BuildPanelModel(PanelBuildOptions{})
	*m.Draft = "unsent assistant draft"
	m.OnClose()
	if revealed != 1 || s.Open() || s.Draft() != "unsent assistant draft" {
		t.Fatalf("close did not commit navigation and draft: reveals=%d open=%v draft=%q", revealed, s.Open(), s.Draft())
	}
}
