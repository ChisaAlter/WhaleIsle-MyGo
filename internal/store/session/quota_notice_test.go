package session

import (
	"encoding/json"
	"testing"
)

func TestQuotaNoticeOnlyLiveAppendAndHeldUntilReturn(t *testing.T) {
	s := New()
	s.selectedID = "first"
	event := json.RawMessage(`{"type":"turn/end","seq":7,"data":{"turn":2,"reason":{"kind":"error","error":{"code":"ACCOUNT_QUOTA"}}}}`)
	// Snapshot/history uses the publisher directly and must stay quiet.
	s.publishAssistantDurableLocked(event, 7)
	if s.QuotaNotice().ID != 0 {
		t.Fatal("history raised quota notice")
	}
	live := json.RawMessage(`{"type":"turn/end","seq":8,"data":{"turn":3,"reason":{"kind":"error","error":{"code":"ACCOUNT_QUOTA"}}}}`)
	s.acceptAssistantDurableLocked(live)
	n := s.QuotaNotice()
	if n.ID == 0 || n.Code != "ACCOUNT_QUOTA" || n.SessionID != "first" {
		t.Fatalf("missing live notice: %+v", n)
	}
	if !s.HoldQuotaNotice(n.ID) {
		t.Fatal("hold failed")
	}
	s.selectedID = "second"
	s.acceptAssistantDurableLocked(json.RawMessage(`{"type":"turn/end","seq":9,"data":{"turn":1,"reason":{"kind":"error","error":{"code":"QUOTA"}}}}`))
	if s.QuotaNotice() != n {
		t.Fatal("held notice replaced across navigation")
	}
	s.DismissQuotaNotice(n.ID)
	if s.QuotaNotice().ID != 0 {
		t.Fatal("return did not dismiss")
	}
	s.acceptAssistantDurableLocked(live)
	if s.QuotaNotice().ID != 0 {
		t.Fatal("duplicate event reopened dismissed notice")
	}
}
