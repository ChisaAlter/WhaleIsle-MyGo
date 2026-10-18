package session

import (
	"testing"
	"whaleisle/internal/dsh"
)

func TestOpenUIOwnerTicket(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "session"
	s.planHistoryBinding = "binding"
	s.messages = []Message{{ID: "message", Role: "assistant", Body: "```openui\nmain=Text(\"hello\")\n```"}}
	target, valid, err := s.CaptureOpenUI("binding", "session", "message", "main=Text(\"hello\")")
	if err != nil || !valid() || target.mode != PromptModeQueue {
		t.Fatal("valid owner", err)
	}
	if _, _, err = s.CaptureOpenUI("other", "session", "message", "main=Text(\"hello\")"); err == nil {
		t.Fatal("foreign host admitted")
	}
	s.selectedID = "other"
	if valid() {
		t.Fatal("navigation did not revoke")
	}
	s.selectedID = "session"
	s.messages = nil
	if valid() {
		t.Fatal("deleted message did not revoke")
	}
}
