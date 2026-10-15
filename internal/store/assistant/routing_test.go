package assistant

import "testing"

func TestExplicitDshbotTargetAndAcceptedDraft(t *testing.T) {
	s := New()
	s.SelectDshbot(" bot ", " http://localhost:8787 ")
	got := s.Snapshot()
	if got.DshbotID != "bot" || got.DshbotOrigin != "http://localhost:8787" {
		t.Fatalf("target = %+v", got)
	}
	if s.ConsumeAcceptedDraft("pending") != "pending" {
		t.Fatal("pending draft lost")
	}
	s.AcceptPrompt("sent")
	if s.ConsumeAcceptedDraft("new edit") != "new edit" {
		t.Fatal("newer edit lost")
	}
	if s.ConsumeAcceptedDraft("sent") != "sent" {
		t.Fatal("acknowledgement was reused")
	}
	s.AcceptPrompt("sent")
	if s.ConsumeAcceptedDraft(" sent ") != "" {
		t.Fatal("accepted unchanged draft not cleared")
	}
}
