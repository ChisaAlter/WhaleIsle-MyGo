package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestQueueEditorFullContentLifecycle(t *testing.T) {
	original := strings.Repeat("完整文本", 80) + "\nsecond block"
	values, _ := json.Marshal(map[string]any{"inbox": map[string]any{"next-turn": []any{
		map[string]any{"id": "q", "source": map[string]any{"kind": "user", "rpcId": "original-request"}, "content": []any{map[string]any{"type": "text", "text": original}}},
		map[string]any{"id": "image", "content": []any{map[string]any{"type": "image", "attachment": map[string]any{"attachmentId": "keep"}}}},
	}}})
	if _, err := queueEditableText(values, "image"); err == nil {
		t.Fatal("attachment edit would discard attachment")
	}
	if _, err := queueEditableText(values, "gone"); err == nil {
		t.Fatal("missing row editable")
	}
	accepted, edits := false, 0
	var sentText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		req := body["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)
		if req["sessionId"] != "A" {
			t.Error("wrong session")
		}
		switch r.URL.Path {
		case "/api/session/projections":
			rpcOK(w, body["rpcId"], map[string]any{"asOfSeq": 10, "values": json.RawMessage(values)})
		case "/api/session/updateQueue":
			edits++
			action := req["action"].(map[string]any)
			if req["itemId"] != "q" || len(req) != 3 || action["kind"] != "edit" {
				t.Errorf("identity changed: %v", req)
			}
			sentText = action["content"].([]any)[0].(map[string]any)["text"].(string)
			rpcOK(w, body["rpcId"], map[string]any{"accepted": accepted})
		default:
			t.Error("unexpected RPC " + r.URL.Path)
		}
	}))
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s := New()
	s.Attach(client, nil)
	s.selectedID = "A"
	s.queueEditor = queueEditor{QueueEditorState: QueueEditorState{ItemID: "q", Token: 1, Loading: true}, sessionID: "A", generation: s.followGeneration}
	s.runLoadQueueEdit(context.Background(), client, "A", 1, "q")
	e := s.Snapshot().QueueEditor
	if !e.Ready || e.Text != original || edits != 0 {
		t.Fatalf("load lost full content: %+v", e)
	}
	changed := original + "\nchanged"
	s.ChangeQueueEdit(1, changed)
	s.runSaveQueueEdit(context.Background(), client, "A", s.Snapshot().QueueEditor)
	e = s.Snapshot().QueueEditor
	if e.Text != changed || e.Error == "" || !e.Ready {
		t.Fatalf("failed save lost draft: %+v", e)
	}
	accepted = true
	s.runSaveQueueEdit(context.Background(), client, "A", e)
	if s.Snapshot().QueueEditor.ItemID != "" || sentText != changed || edits != 2 {
		t.Fatal("save did not preserve complete edited content and close")
	}
	s.queueEditor = queueEditor{QueueEditorState: QueueEditorState{ItemID: "q", Token: 2, Loading: true}, sessionID: "A", generation: s.followGeneration}
	s.CancelQueueEdit(2)
	s.runLoadQueueEdit(context.Background(), client, "A", 2, "q")
	if s.Snapshot().QueueEditor.ItemID != "" || edits != 2 {
		t.Fatal("cancel submitted or late load reopened editor")
	}
	s.applyQueueValuesLocked(values, 10)
	if len(s.planBoard.Queue) != 2 {
		t.Fatal("inbox rows missing")
	}
	s.applyQueueControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"inbox","seq":11,"value":{"next-turn":[]}}`))
	s.applyQueueValuesLocked(values, 10)
	if len(s.planBoard.Queue) != 0 {
		t.Fatal("stale projection resurrected queue")
	}
}
