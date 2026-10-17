package session

import (
	"encoding/json"
	"testing"
)

func TestApplyWorkspaceSessionPatch(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.items = []Item{{SessionID: "s1", Title: "Old"}}
	s.mu.Unlock()

	raw, _ := json.Marshal(map[string]any{
		"sessionId": "s1",
		"title":     "From stream",
		"cwd":       "/proj",
		"running":   true,
	})
	if !s.applyWorkspaceSessionPatch(raw) {
		t.Fatal("expected patch applied")
	}
	st := s.Snapshot()
	if len(st.Items) != 1 {
		t.Fatalf("items = %+v", st.Items)
	}
	if st.Items[0].Title != "From stream" || st.Items[0].Cwd != "/proj" || !st.Items[0].Running {
		t.Fatalf("%+v", st.Items[0])
	}
}

func TestApplyWorkspaceSnapshot(t *testing.T) {
	s := New()
	raw, _ := json.Marshal(map[string]any{
		"type": "snapshot",
		"sessions": []map[string]any{{
			"sessionId":      "a",
			"updatedAt":      3.0,
			"running":        false,
			"blank":          false,
			"agentAvailable": true,
			"cwd":            "/ws",
			"projections": map[string]any{
				"kind":    "title",
				"asOfSeq": 1.0,
				"values":  map[string]string{"title": "Alpha"},
			},
		}},
	})
	s.applyWorkspaceFollowFrame(raw)
	st := s.Snapshot()
	if len(st.Items) != 1 || st.Items[0].SessionID != "a" {
		t.Fatalf("items = %+v", st.Items)
	}
	if st.Items[0].Title != "Alpha" || st.Items[0].Cwd != "/ws" {
		t.Fatalf("%+v", st.Items[0])
	}
}

func TestApplyWorkspaceFollowUnknownFrameNoPanic(t *testing.T) {
	s := New()
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"future","payload":{"x":1}}`))
}
