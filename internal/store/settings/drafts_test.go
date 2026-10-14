package settings

import (
	"encoding/json"
	"testing"
)

func TestDraftSurvivesRefreshSwitchAndLateSave(t *testing.T) {
	s := New()
	n := Namespace{NS: "a", Fields: []Field{{Key: "name", Kind: FieldString}}, Value: json.RawMessage(`{"name":"old"}`)}
	s.namespaces = []Namespace{n, {NS: "b"}}
	s.Select("a")
	s.SetDraft("name", "draft")
	s.Select("b")
	s.Select("a")
	n.Value = json.RawMessage(`{"name":"remote"}`)
	s.mu.Lock()
	s.seedDraftsLocked(n)
	s.mu.Unlock()
	if s.Snapshot().Drafts["name"] != "draft" {
		t.Fatal("refresh/switch lost draft")
	}
	submitted := map[string]string{"name": "draft"}
	s.SetDraft("name", "newer")
	n.Value = json.RawMessage(`{"name":"draft"}`)
	s.mu.Lock()
	s.acknowledgeDraftsLocked(n, submitted)
	s.mu.Unlock()
	if s.Snapshot().Drafts["name"] != "newer" {
		t.Fatal("late save lost newer edit")
	}
}
