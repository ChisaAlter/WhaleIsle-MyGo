package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDraftWriteOnlyStagedFields(t *testing.T) {
	ns := Namespace{
		Fields: []Field{{Key: "baseURL", Kind: FieldString}, {Key: "maxUses", Kind: FieldNumber}, {Key: "enabled", Kind: FieldBool}},
		Value: json.RawMessage(`{"baseURL":"https://old.example","enabled":true}`),
	}
	patch, ops, submitted, err := draftWrite(ns,
		map[string]string{"baseURL": "https://new.example", "maxUses": "", "enabled": "true"},
		map[string]bool{"baseURL": true}, nil)
	if err != nil || len(ops) != 0 || len(patch) != 1 || len(submitted) != 1 || string(patch["baseURL"]) != `"https://new.example"` {
		t.Fatalf("patch=%s ops=%s submitted=%v err=%v", patch, ops, submitted, err)
	}
}

func TestDraftWriteNestedJSONPreservesUnseenSecrets(t *testing.T) {
	ns := Namespace{
		Fields: []Field{{Key: "providers", Kind: FieldJSON}},
		Value: json.RawMessage(`{"providers":{"custom":{"baseURL":"https://old.example","timeoutMs":1000},"other":{"models":[{"id":"keep"}]}}}`),
	}
	patch, ops, _, err := draftWrite(ns,
		map[string]string{"providers": `{"custom":{"baseURL":"https://new.example","timeoutMs":1000},"other":{"models":[{"id":"keep"}]}}`},
		map[string]bool{"providers": true}, nil)
	if err != nil || len(patch) != 0 || len(ops) != 1 {
		t.Fatalf("patch=%s ops=%s err=%v", patch, ops, err)
	}
	want := fieldOperation("set", []string{"providers", "custom", "baseURL"}, json.RawMessage(`"https://new.example"`))
	if !sameJSON(ops[0], want) {
		t.Fatalf("ops=%s want=%s", ops[0], want)
	}
}

func TestDraftWriteClearsNumericOverrideWithoutNull(t *testing.T) {
	ns := Namespace{Fields: []Field{{Key: "timeoutMs", Kind: FieldNumber}}}
	patch, ops, _, err := draftWrite(ns, map[string]string{"timeoutMs": ""}, map[string]bool{"timeoutMs": true}, nil)
	if err != nil || len(patch) != 0 || len(ops) != 1 || strings.Contains(string(ops[0]), "null") || !strings.Contains(string(ops[0]), "unset") {
		t.Fatalf("patch=%s ops=%s err=%v", patch, ops, err)
	}
}

func TestReplacementUsesOnlyStoredUserLayer(t *testing.T) {
	ns := Namespace{
		Fields: []Field{{Key: "maxDepth", Kind: FieldNumber}, {Key: "maxActiveSubagents", Kind: FieldNumber}},
		Value: json.RawMessage(`{"maxDepth":3,"maxActiveSubagents":8}`),
		Base: json.RawMessage(`{"maxDepth":3,"maxActiveSubagents":8}`),
		User: json.RawMessage(`{"maxDepth":4}`),
	}
	section, err := replacementFromDrafts(ns,
		map[string]string{"maxDepth": "5", "maxActiveSubagents": "8"}, map[string]bool{"maxDepth": true}, nil)
	if err != nil || len(section) != 1 || string(section["maxDepth"]) != "5" {
		t.Fatalf("section=%s err=%v", section, err)
	}
	ns.HasSecrets = true
	if _, err := replacementFromDrafts(ns, nil, nil, nil); err == nil {
		t.Fatal("redacted sections must not be reconstructed for full replacement")
	}
}

func TestResetDraftSurvivesLateAcknowledgement(t *testing.T) {
	s := New()
	ns := Namespace{NS: "subagent", Fields: []Field{{Key: "maxDepth", Kind: FieldNumber}}, Value: json.RawMessage(`{"maxDepth":4}`), Base: json.RawMessage(`{"maxDepth":3}`)}
	s.namespaces = []Namespace{ns}
	s.writable = true
	s.Select(ns.NS)
	s.ResetDraft("maxDepth")
	snapshot := s.Snapshot()
	if !snapshot.Dirty || !snapshot.Resets["maxDepth"] || snapshot.Drafts["maxDepth"] != "3" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	// The same visible text can be a new explicit override while unset is in flight.
	s.SetDraft("maxDepth", "3")
	s.acknowledgeDraftsLocked(ns, map[string]string{"maxDepth": "3"}, map[string]bool{"maxDepth": true})
	if !s.Snapshot().Dirty {
		t.Fatal("late reset acknowledgement erased a newer explicit override")
	}
}

func TestPatchStringPreservesInstructionsWhitespace(t *testing.T) {
	raw, err := PatchValue(FieldString, "  preserve\n")
	if err != nil || string(raw) != `"  preserve\n"` {
		t.Fatalf("raw=%s err=%v", raw, err)
	}
}
