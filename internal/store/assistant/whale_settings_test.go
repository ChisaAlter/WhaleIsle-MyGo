package assistant

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func whaleFixture() pluginrpc.WhaleSettings {
	return pluginrpc.WhaleSettings{Name: "Whale", UserTitle: "you", PersonaText: "kind", Memory: "old memory", HomeDir: "/synthetic/whale", SessionID: "whale-a"}
}
func loadWhale(t *testing.T, s *WhaleSettingsEditor, host string) {
	t.Helper()
	s.Bind(host)
	q, ok := s.BeginLoad(host)
	if !ok {
		t.Fatal("load denied")
	}
	if !s.FinishLoad(q, whaleFixture(), nil) {
		t.Fatal("load ignored")
	}
}
func TestWhaleEditorClearConfirmationAndFailure(t *testing.T) {
	var s WhaleSettingsEditor
	loadWhale(t, &s, "host-a")
	s.Edit("host-a", "memory", "unsaved")
	if _, _, ok := s.BeginWrite("host-a", "clear"); ok {
		t.Fatal("clear bypassed explicit confirmation")
	}
	s.ReviewClear("host-a")
	s.CancelClear("host-a")
	if v := s.Snapshot(); v.Memory != "unsaved" || v.ConfirmClear {
		t.Fatalf("cancel lost draft: %+v", v)
	}
	s.ReviewClear("host-a")
	q, cmd, ok := s.BeginWrite("host-a", "clear")
	if !ok || cmd.Action != "clear" {
		t.Fatal("confirmed clear unavailable")
	}
	if _, _, ok := s.BeginWrite("host-a", "clear"); ok {
		t.Fatal("duplicate clear allowed")
	}
	s.FinishWrite(q, cmd, nil, false, errors.New("synthetic timeout"))
	v := s.Snapshot()
	if v.Memory != "unsaved" || v.ConfirmClear || v.Loaded || v.Error == "" {
		t.Fatalf("uncertain clear handling: %+v", v)
	}
	if _, _, ok := s.BeginWrite("host-a", "clear"); ok {
		t.Fatal("uncertain clear blindly retried")
	}
	q, _ = s.BeginLoad("host-a")
	actual := whaleFixture()
	actual.Memory = "default template"
	s.FinishLoad(q, actual, nil)
	if s.Snapshot().Memory != "unsaved" {
		t.Fatal("refresh overwrote draft")
	}
	s.ReviewClear("host-a")
	q, cmd, ok = s.BeginWrite("host-a", "clear")
	if !ok {
		t.Fatal("explicit reconfirmation failed")
	}
	s.FinishWrite(q, cmd, &actual, true, nil)
	if v = s.Snapshot(); v.Memory != "default template" || v.Notice == "" || v.Error != "" {
		t.Fatalf("clear did not use actual Host text: %+v", v)
	}
}
func TestWhaleEditorOwnerRevisionsAndDirtyRefresh(t *testing.T) {
	var s WhaleSettingsEditor
	loadWhale(t, &s, "host-a")
	s.Edit("host-a", "memory", "draft-a")
	q, _ := s.BeginLoad("host-a")
	server := whaleFixture()
	server.Memory = "new server"
	s.FinishLoad(q, server, nil)
	if s.Snapshot().Memory != "draft-a" {
		t.Fatal("dirty reload discarded draft")
	}
	q, cmd, ok := s.BeginWrite("host-a", "memory")
	if !ok {
		t.Fatal("save denied")
	}
	s.Edit("host-a", "memory", "newer edit")
	server.Memory = "draft-a"
	s.FinishWrite(q, cmd, &server, true, nil)
	if s.Snapshot().Memory != "newer edit" {
		t.Fatal("late success discarded newer edit")
	}
	q, _ = s.BeginLoad("host-a")
	s.Bind("host-b")
	if s.FinishLoad(q, server, nil) || s.Snapshot().Loaded || s.Snapshot().Memory != "" {
		t.Fatal("old Host result crossed identity")
	}
	loadWhale(t, &s, "host-b")
	s.Edit("host-b", "memory", "identity-a draft")
	q, _ = s.BeginLoad("host-b")
	server.SessionID = "whale-b"
	server.Memory = "identity-b memory"
	s.FinishLoad(q, server, nil)
	if s.Snapshot().Memory != "identity-b memory" {
		t.Fatal("Whale identity mixed drafts")
	}
	q, cmd, ok = s.BeginWrite("host-b", "memory")
	if !ok {
		t.Fatal("save unavailable")
	}
	s.Suspend()
	if s.FinishWrite(q, cmd, &server, true, nil) {
		t.Fatal("closed view accepted stale write")
	}
}
func TestWhaleEditorAcceptedButUnreadAndFieldIsolation(t *testing.T) {
	var s WhaleSettingsEditor
	loadWhale(t, &s, "host")
	s.Edit("host", "memory", "draft memory")
	s.Edit("host", "persona", "draft persona")
	s.Edit("host", "name", "New Whale")
	q, cmd, _ := s.BeginWrite("host", "persona")
	s.FinishWrite(q, cmd, nil, true, errors.New("read unavailable"))
	v := s.Snapshot()
	if v.PersonaText != "draft persona" || v.Memory != "draft memory" || v.Name != "New Whale" || v.Loaded || v.Notice == "" || v.Error == "" {
		t.Fatalf("accepted/read failure corrupt state: %+v", v)
	}
	q, _ = s.BeginLoad("host")
	server := whaleFixture()
	server.PersonaText = "draft persona"
	s.FinishLoad(q, server, nil)
	if s.Snapshot().Memory != "draft memory" {
		t.Fatal("persona refresh touched memory draft")
	}
	s.ReviewClear("host")
	s.Edit("host", "memory", "later")
	if s.Snapshot().ConfirmClear {
		t.Fatal("editing must invalidate clear review")
	}
}

func TestWhaleEditorReadBackPreservesRevertedAndConflictingDraft(t *testing.T) {
	var s WhaleSettingsEditor
	loadWhale(t, &s, "host")
	s.Edit("host", "memory", "submitted")
	q, cmd, _ := s.BeginWrite("host", "memory")
	current := whaleFixture()
	current.Memory = "changed elsewhere"
	if cmd.Check(current) == nil {
		t.Fatal("preflight missed concurrent edit")
	}
	current = whaleFixture()
	if cmd.Check(current) != nil {
		t.Fatal("unchanged preflight refused")
	}
	s.Edit("host", "memory", "old memory")
	current.Memory = "submitted"
	s.FinishWrite(q, cmd, &current, true, nil)
	if s.Snapshot().Memory != "old memory" {
		t.Fatal("newer deliberate revert consumed by readback")
	}
	q, cmd, _ = s.BeginWrite("host", "memory")
	current.Memory = "changed again"
	s.FinishWrite(q, cmd, &current, true, nil)
	v := s.Snapshot()
	if v.Memory != "old memory" || v.Server.Memory != "changed again" || v.Error == "" {
		t.Fatalf("conflicting receipt lost draft: %+v", v)
	}
}
