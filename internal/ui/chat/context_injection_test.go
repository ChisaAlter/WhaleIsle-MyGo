package chat

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestInjectedContextUsesReadOnlyDisclosure(t *testing.T) {
	msg := session.Message{ID: "ctx", Seq: 1, Kind: session.MessageKindContext, Role: "context", Body: "model-facing instructions", Raw: json.RawMessage(`{"type":"user/message","data":{"source":{"kind":"agent-instructions","form":"instructions","changes":[{"path":"AGENTS.md","action":"add"}]}}}`)}
	snap := session.ViewState{SelectedID: "one", Messages: []session.Message{msg}}
	tt := ui.NewTester(func(c *ui.Context) { ContextInjectionView(c, snap.Messages[0], snap.SelectedID, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 800, 1000)
	if _, ok := tt.Find("User message bubble"); ok {
		t.Fatal("context uses human bubble")
	}
	if tt.HasText(msg.Body) {
		t.Fatal("context not initially collapsed")
	}
	if err := tt.Click("Context details ctx"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(msg.Body) || !tt.HasText("AGENTS.md") || !tt.HasText("changes") {
		t.Fatal("context content/producer metadata missing")
	}
	if showMessageCopy(msg) || showMessageEdit(msg) {
		t.Fatal("context carries human/assistant mutation actions")
	}
	snap.SelectedID = "two"
	tt.Frame()
	if tt.HasText(msg.Body) {
		t.Fatal("disclosure state crossed session")
	}
}

func TestContextDisplayBoundDoesNotChangeRetainedContent(t *testing.T) {
	body := strings.Repeat("A", 70<<10) + "END"
	msg := session.Message{ID: "long", Kind: session.MessageKindContext, Role: "context", Body: body}
	tt := ui.NewTester(func(c *ui.Context) { ContextInjectionView(c, msg, "host:session", ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 800, 1000)
	if err := tt.Click("Context details long"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("END") {
		t.Fatal("display budget ignored")
	}
	if session.ContextMessageDetails(msg).Body != body {
		t.Fatal("raw body changed by display clipping")
	}
}
