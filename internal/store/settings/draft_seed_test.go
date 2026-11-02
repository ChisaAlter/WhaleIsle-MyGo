package settings

import (
	"encoding/json"
	"testing"
)

// Reseeding from a cached value yields what ValueDraft does, follows a changed
// value, still resets unmarked drafts, and never touches dirty fields.
func TestSeedDraftsFollowsValueAndKeepsDirtyFields(t *testing.T) {
	st := New()
	fields := []Field{{Key: "name", Kind: FieldString}, {Key: "size", Kind: FieldNumber}, {Key: "extra", Kind: FieldJSON}}
	value := json.RawMessage(`{"name":"a","size":3,"extra":{"k":[1,2]}}`)
	st.namespaces = []Namespace{{NS: "ns", Fields: fields, Value: value}}
	drafts := func() map[string]string {
		st.mu.Lock()
		defer st.mu.Unlock()
		st.ensureDraftsLocked("ns")
		return copyDrafts(st.drafts["ns"])
	}
	first := drafts()
	for _, f := range fields {
		if want := ValueDraft(value, f.Key, f.Kind); first[f.Key] != want {
			t.Fatalf("%s: %q, want %q", f.Key, first[f.Key], want)
		}
	}
	st.mu.Lock()
	st.drafts["ns"]["name"] = "unmarked edit"
	st.mu.Unlock()
	if got := drafts()["name"]; got != "a" {
		t.Fatalf("unmarked draft not reset from the value: %q", got)
	}
	st.mu.Lock()
	st.drafts["ns"]["size"] = "typed"
	st.dirty = map[string]map[string]bool{"ns": {"size": true}}
	st.namespaces[0].Value = json.RawMessage(`{"name":"b","size":9}`)
	st.mu.Unlock()
	next := drafts()
	if next["name"] != "b" || next["size"] != "typed" || next["extra"] != ValueDraft(json.RawMessage(`{"name":"b","size":9}`), "extra", FieldJSON) {
		t.Fatalf("drafts after a new value: %+v", next)
	}
}
