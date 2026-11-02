package chat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"whaleisle/internal/store/session"
)

// This is the normal follow/Snapshot message shape of native run02: five
// events per completed turn, including Files on the durable presented row.
// Delivery cards are derived later by the chat projection, not input metadata.
func filesMemoNativeSnapshot(turns int) session.ViewState {
	messages := make([]session.Message, 0, turns*5)
	for turn := 1; turn <= turns; turn++ {
		appendEvent := func(kind, role, body string, data any, files []string) {
			seq := len(messages) + 1
			raw, _ := json.Marshal(struct {
				Type string `json:"type"`
				Seq  int    `json:"seq"`
				Time int    `json:"time"`
				Data any    `json:"data"`
			}{kind, seq, 1000 + seq, data})
			message := session.Message{ID: fmt.Sprintf("event-%d", seq), Kind: session.MessageKindText, Role: role, Body: body, Seq: float64(seq), EventType: kind, Raw: raw, EventTime: float64(1000 + seq), Files: files}
			if kind == "assistant/thinking" {
				message.Kind = session.MessageKindThinking
			}
			if kind == "deliverables/presented" {
				message.Kind = session.MessageKindDeliverables
			}
			messages = append(messages, message)
		}
		appendEvent("user/message", "user", fmt.Sprintf("Synthetic question %03d", turn), map[string]any{"turn": turn, "step": 0, "message": map[string]any{"id": fmt.Sprintf("u%d", turn), "content": []map[string]any{{"type": "text", "text": fmt.Sprintf("Synthetic question %03d", turn)}}}}, nil)
		appendEvent("assistant/thinking", "thinking", "Synthetic process", map[string]any{"turn": turn, "step": 0, "text": "Synthetic process"}, nil)
		path := fmt.Sprintf("synthetic-%03d.md", turn)
		appendEvent("deliverables/presented", "assistant", path, map[string]any{"turn": turn, "callId": fmt.Sprintf("present-%d", turn), "files": []map[string]any{{"path": path, "description": fmt.Sprintf("DELIVERY-%03d", turn)}}}, []string{path})
		appendEvent("assistant/message", "assistant", "Synthetic final answer", map[string]any{"turn": turn, "step": 1, "message": map[string]any{"id": fmt.Sprintf("final-%d", turn), "content": []map[string]any{{"type": "text", "text": "Synthetic final answer"}}}, "stream": []any{}}, nil)
		appendEvent("turn/end", "status", "0", map[string]any{"turn": turn, "step": 1, "reason": "completed"}, nil)
	}
	return session.ViewState{SelectedID: "files-native-shape", PlanHistoryBinding: "synthetic", Messages: messages}
}

func TestFilesProjectionMemoNativeShapeWarmHit(t *testing.T) {
	snap := filesMemoNativeSnapshot(400)
	files, deliveries := 0, 0
	for _, message := range snap.Messages {
		if message.Files != nil {
			files++
		}
		if message.TurnDelivery != nil {
			deliveries++
		}
		if message.AutomaticCompaction != nil || message.Retry != nil || message.Workflow != nil || message.Submission != nil || message.Tool != nil || message.Command != nil || message.ConversationTurn != nil || message.ConversationStep != nil || message.Web != nil {
			t.Fatal("native-shape fixture introduced unrelated complex metadata")
		}
	}
	if len(snap.Messages) != 2000 || files != 400 || deliveries != 0 {
		t.Fatalf("wrong native input shape: rows=%d Files=%d TurnDelivery=%d", len(snap.Messages), files, deliveries)
	}
	first := projectChatMessages(snap)
	second := projectChatMessages(snap)
	if len(first) != 2000 || len(second) != 2000 || &first[0] != &second[0] {
		t.Fatal("native-shaped Files metadata rebuilt the warm chat projection")
	}
	for _, message := range second {
		if message.TurnDelivery != nil {
			deliveries++
		}
	}
	if deliveries != 400 {
		t.Fatalf("warm projection lost native delivery cards: %d", deliveries)
	}
}

func TestFilesProjectionMemoOwnsSourceAndOutput(t *testing.T) {
	snap := session.ViewState{SelectedID: "files-owned", Messages: []session.Message{{ID: "files", Role: "assistant", Kind: session.MessageKindText, Raw: json.RawMessage(`{}`), Files: []string{"file-a.txt", "file-b.txt"}}}}
	first := projectChatMessages(snap)
	if &first[0].Files[0] == &snap.Messages[0].Files[0] {
		t.Fatal("memoized output retains caller-owned Files slice")
	}
	cp := chatProjectionOf(snap)
	cp.mu.Lock()
	plainSourceClear := len(cp.source) == 1 && cp.source[0].message.Files == nil
	cp.mu.Unlock()
	if !plainSourceClear {
		t.Fatal("plain source value retains caller-owned Files slice")
	}
	snap.Messages[0].Files[0] = "file-c.txt"
	if first[0].Files[0] != "file-a.txt" {
		t.Fatal("caller edit changed a previously returned Files projection")
	}
	current := projectChatMessages(snap)
	if &current[0] == &first[0] || current[0].Files[0] != "file-c.txt" {
		t.Fatal("same-length in-place Files edit failed to invalidate")
	}
	if again := projectChatMessages(snap); &again[0] != &current[0] {
		t.Fatal("unchanged Files edit did not stay warm")
	}
}

func TestFilesProjectionMemoNilEmptyOrderAndEqualReplacement(t *testing.T) {
	snap := session.ViewState{SelectedID: "files-states", Messages: []session.Message{{ID: "files", Role: "assistant", Raw: json.RawMessage(`{}`)}}}
	var previous []session.Message
	states := [][]string{nil, {}, {"file-a.txt", "file-b.txt"}, {"file-b.txt", "file-a.txt"}, {"file-c.txt", "file-a.txt"}, {"file-a.txt"}, nil}
	for index, files := range states {
		snap.Messages[0].Files = files
		current := projectChatMessages(snap)
		if !reflect.DeepEqual(current[0].Files, files) {
			t.Fatalf("Files state %d lost nil/empty/order semantics", index)
		}
		if previous != nil && &previous[0] == &current[0] {
			t.Fatalf("changed Files state %d did not invalidate", index)
		}
		if files != nil {
			snap.Messages[0].Files = make([]string, len(files))
			copy(snap.Messages[0].Files, files)
		}
		if again := projectChatMessages(snap); &again[0] != &current[0] {
			t.Fatalf("equal separately owned Files state %d rebuilt", index)
		}
		previous = current
	}
}

func TestFilesProjectionMemoRetainsRawRoleAndTimingGuards(t *testing.T) {
	stamp := 1.0
	snap := session.ViewState{SelectedID: "files-base-fields", Messages: []session.Message{{ID: "files", Role: "assistant", Raw: json.RawMessage(`{"x":"a"}`), Files: []string{"file-a.txt"}, TrajectoryTime: &session.TrajectoryTiming{StartedAt: &stamp}}}}
	first := projectChatMessages(snap)
	copy(snap.Messages[0].Raw, `{"x":"b"}`)
	current := projectChatMessages(snap)
	if &current[0] == &first[0] {
		t.Fatal("Files support bypassed same-length raw identity")
	}
	first = current
	snap.Messages[0].Role = "user"
	current = projectChatMessages(snap)
	if &current[0] == &first[0] {
		t.Fatal("Files support bypassed role identity")
	}
	first = current
	stamp = 2
	current = projectChatMessages(snap)
	if &current[0] == &first[0] {
		t.Fatal("Files support bypassed mutable timing identity")
	}
}

func TestFilesProjectionMemoConcurrentIndependentReadOnlyInputs(t *testing.T) {
	var workers sync.WaitGroup
	for index := 0; index < 8; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			path := fmt.Sprintf("worker-%d.txt", index)
			snap := session.ViewState{SelectedID: "files-concurrent", Messages: []session.Message{{ID: "files", Role: "assistant", Raw: json.RawMessage(`{}`), Files: []string{path}}}}
			for range 16 {
				out := projectChatMessages(snap)
				if len(out) != 1 || len(out[0].Files) != 1 || out[0].Files[0] != path || &out[0].Files[0] == &snap.Messages[0].Files[0] {
					t.Errorf("worker %d received foreign or caller-owned Files", index)
					return
				}
			}
		}(index)
	}
	workers.Wait()
}

var filesMemoProjectionSink []session.Message

func BenchmarkFilesProjectionMemoNativeShape(b *testing.B) {
	for _, turns := range []int{400, 1600} {
		snap := filesMemoNativeSnapshot(turns)
		b.Run(fmt.Sprintf("rows_%d_files_%d", len(snap.Messages), turns), func(b *testing.B) {
			projectChatMessages(snap)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				filesMemoProjectionSink = projectChatMessages(snap)
			}
		})
	}
}
