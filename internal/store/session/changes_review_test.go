package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
)

func TestChangesReviewPreviewRequiresRecordedSummary(t *testing.T) {
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}}
	s.selectedID = "source"
	s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
	s.messages = []Message{{Raw: json.RawMessage(`{"type":"workspace/changes","seq":3,"data":{"turn":1}}`)}, {Raw: json.RawMessage(`{"type":"turn/end","seq":4,"data":{"turn":1}}`)}}
	scope := s.Snapshot().TurnFiles.Scope
	target := TurnFileTarget{Kind: "changes", Turn: 1, Seq: 3, Index: 0}
	if _, ok := s.ChangesReviewPath(scope, target); ok {
		t.Fatal("unloaded summary granted preview")
	}
	base := target
	base.Index = -1
	s.turnFiles.results = map[TurnFileRequest]TurnFileResult{}
	s.turnFiles.results[TurnFileRequest{Target: base, Action: "summary"}] = TurnFileResult{Summary: &TurnChangesSummary{Files: []TurnChangedFile{{Path: "recorded.go"}}}}
	if path, ok := s.ChangesReviewPath(scope, target); !ok || path != "recorded.go" {
		t.Fatal("recorded preview missing")
	}
	foreign := scope
	foreign.SessionID = "other"
	if _, ok := s.ChangesReviewPath(foreign, target); ok {
		t.Fatal("foreign session granted preview")
	}
	s.messages = nil
	if _, ok := s.ChangesReviewPath(scope, target); ok {
		t.Fatal("revoked declaration granted preview")
	}
}
