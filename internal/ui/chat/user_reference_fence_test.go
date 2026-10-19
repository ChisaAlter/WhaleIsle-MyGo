package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

func TestHistoricalReferenceFencesPreserveEveryBusinessToken(t *testing.T) {
	for _, fence := range []string{"```", "~~~~"} {
		code := fence + "diff\r\n@@ -1 +1 @@\r\n@inside.md /review @[Alpha](dsh-session:ImEi)\r\n" + fence + "\r\n"
		text := "@before.md\n" + code + "@after.md /review"
		facts := userReferenceFacts{Labels: []string{"Alpha"}, Skills: []string{"review"}}
		var files []string
		literal, skills := false, 0
		for _, run := range userReferenceRuns(text, facts) {
			if run.Kind == "" && run.Text == code {
				literal = true
			}
			if run.Kind == "file" {
				files = append(files, run.Path)
			}
			if run.Kind == "skill" {
				skills++
			}
		}
		if !literal || len(files) != 2 || files[0] != "before.md" || files[1] != "after.md" || skills != 1 {
			t.Fatalf("fence altered literal code or adjacent references: %q %v %d", code, files, skills)
		}
	}
	// Neither a shorter fence nor the other marker closes a longer fence.
	unclosed := "````text\n@inside.md\n```\n~~~\n/review"
	runs := userReferenceRuns(unclosed, userReferenceFacts{Skills: []string{"review"}})
	if len(runs) != 1 || runs[0].Kind != "" || runs[0].Text != unclosed {
		t.Fatal("unclosed fence produced a reference")
	}
}

// The Store obtains a real history binding through a loopback fake mux. All
// navigation callbacks are local probes: no Host, model, file or native open.
func TestUserMessageFullModelKeepsFencedTextAndAdjacentReferenceActivation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			var request struct {
				RPCID string `json:"rpcId"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": nil}})
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			if frame.Type == dsh.FrameOpen && frame.Endpoint == "session/follow" {
				if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: json.RawMessage(`{"type":"snapshot","records":[],"hasMore":false}`)}) != nil {
					return
				}
			}
		}
	}))
	defer server.Close()
	store := session.New()
	store.Select("history")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	store.Attach(client, nil)
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	store.AttachMux(mux)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store.FollowSelected(ctx)
	for store.Snapshot().TurnFiles.Scope.Binding == "" {
		select {
		case <-ctx.Done():
			t.Fatal("fake history binding was not loaded")
		case <-time.After(time.Millisecond):
		}
	}
	code := "```diff\n@@ -1 +1 @@\n@inside.md /review @[Alpha](dsh-session:ImEi)\n```"
	body := code + "\nOpen @outside.md"
	message := presentationMessage(body)
	snapshot := store.Snapshot()
	snapshot.Messages = []session.Message{message}
	opened := ""
	model := Model{HistoryStore: store, Snapshot: snapshot, OnOpenDeliverable: func(path string) { opened = path }, OnOpenHistoryReference: func(path string, complete func(error)) { opened = path; complete(nil) }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 1000, 700)
	if !tt.HasText(code) {
		t.Fatalf("complete interactive Model altered literal code: %q", tt.Texts())
	}
	if opened != "" {
		t.Fatal("painting literal code opened a reference")
	}
	if err := tt.Click("outside.md"); err != nil {
		t.Fatal(err)
	}
	if opened != "outside.md" {
		t.Fatalf("adjacent ordinary reference did not open: %q", opened)
	}
	for _, run := range userReferenceRuns(body, userReferenceFacts{Labels: []string{"Alpha"}, Skills: []string{"review"}}) {
		if run.Kind != "" && strings.Contains(run.Raw, "inside") {
			t.Fatal("code retained business activation")
		}
	}
}
