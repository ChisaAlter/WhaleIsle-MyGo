package chat

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// A single synthetic pure projection/render fixture. No carrier or real Host.
func TestAgentReadOnlyRichHistoryFixture(t *testing.T) {
	var fetches atomic.Int32
	SetAttachmentFetcher(func(context.Context, string) (string, error) { fetches.Add(1); return "", nil })
	defer SetAttachmentFetcher(nil)
	address := session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	scope := ReadOnlyHistoryScope("host-A/root", address, 7)
	style := DefaultToolCardStyle(ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(180, 180, 180), ui.RGB(240, 240, 240), ui.RGB(0, 90, 180))
	event := func(kind string, seq int, data string) string {
		b, _ := json.Marshal(map[string]any{"type": "event", "event": map[string]any{"type": kind, "seq": seq, "time": 123456, "opaqueMetadata": "preserved", "data": json.RawMessage(data)}})
		return string(b)
	}
	raw := `{"type":"snapshot","cursor":9,"records":[` + strings.Join([]string{
		event("tool/call", 1, `{"callId":"root-call","name":"read","arguments":"{\"path\":\"relative/child.txt\"}"}`),
		event("tool/ptc-dispatch-start", 2, `{"subCallId":"nested","parentCallId":"root-call","rootCallId":"root-call","name":"grep","arguments":{"query":"needle"}}`),
		event("tool/ptc-dispatch", 3, `{"subCallId":"nested","parentCallId":"root-call","rootCallId":"root-call","name":"grep","isError":true,"error":{"reason":"nested failure"},"content":[{"type":"text","text":"failure evidence"}]}`),
		event("tool/result", 4, `{"message":{"role":"tool","toolCallId":"root-call","content":[{"type":"text","text":"child output"},{"type":"image","text":"image caption","attachment":{"attachmentId":"child-image","mediaType":"image/png","bytes":100,"width":2,"height":3}}]}}`),
	}, ",") + `],"hasMore":false}`
	process, messages, _, err := session.ProjectAgentNavigationHistory(json.RawMessage(raw), 9, 0)
	if err != nil {
		t.Fatal(err)
	}
	projected, limited := readOnlyHistoryMessages(messages)
	if limited || len(projected) != 1 || projected[0].Tool == nil || projected[0].Tool.State != "ok" || len(projected[0].Tool.Children) != 1 || projected[0].Tool.Children[0].State != "error" {
		t.Fatalf("tool projection: %#v", projected)
	}
	if len(messages) != 4 || messages[0].Tool != nil || messages[0].EventTime != 123456 || !strings.Contains(string(messages[0].Raw), "opaqueMetadata") || len(process.Records) != 4 || process.Cursor != 9 {
		t.Fatal("projection mutated durable source")
	}
	// A parent outside this page remains a visible orphan, not a lost result.
	orphan := messages[1]
	orphan.Raw = json.RawMessage(`{"type":"tool/ptc-dispatch-start","seq":2,"data":{"subCallId":"orphan","parentCallId":"outside-page","name":"read","arguments":{}}}`)
	orphanRows, _ := readOnlyHistoryMessages([]session.Message{orphan})
	if len(orphanRows) != 1 || orphanRows[0].Tool == nil || orphanRows[0].Tool.ParentIssue != "missing parent" {
		t.Fatal("page-boundary tool lost")
	}
	preparing := session.Message{ID: "preparing", Kind: session.MessageKindTool, Streaming: true, Tool: &session.ToolCallView{CallID: "future", Name: "read", Arguments: `{"path":"child"}`, State: "preparing"}}
	previews, _ := readOnlyHistoryMessages([]session.Message{preparing})
	if len(previews) != 1 || previews[0].Tool.State != "preparing" || !previews[0].Streaming {
		t.Fatal("temporary tool preview lost")
	}
	current := messages
	tt := ui.NewTester(func(c *ui.Context) { ReadOnlyHistoryView(c, scope, current, "", style) }, 800, 1200)
	if !tt.HasText(toolRowTitle("read")) || !tt.HasText(toolRowTitle("grep")) {
		t.Fatalf("tree not rendered: %q", tt.Texts())
	}
	if tt.HasText("↳ root-call") {
		t.Fatal("parent identity leaked into the compact tool tree")
	}
	// Tool rows are disclosures: the result evidence mounts with the body.
	if err := tt.Click(toolRowTitle("read")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("child-image") || !tt.HasText("image/png") {
		t.Fatalf("attachment evidence hidden: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyChatToolCopy)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tt.Clipboard(), "child-image") || !strings.Contains(tt.Clipboard(), "child output") || !strings.Contains(tt.Clipboard(), "image caption") {
		t.Fatalf("copy lost evidence: %q", tt.Clipboard())
	}
	for _, target := range []string{"编辑", i18n.T(i18n.KeyChatImageRetry), "批准", "Images (1)", "Branch here"} {
		if tt.HasText(target) {
			t.Fatalf("unsafe action %q", target)
		}
	}
	for _, next := range []string{
		ReadOnlyHistoryScope("host-B/root", address, 7),
		ReadOnlyHistoryScope("host-B/root", session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "other", ChildSessionID: "child", Mode: "continuable"}, 7),
		ReadOnlyHistoryScope("host-B/root", address, 8),
		ReadOnlyHistoryScope("host-B/root", session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "one-shot"}, 8),
	} {
		scope = next
		tt.Frame()
		// "image/png" only appears inside the expanded tool body, so its absence
		// proves the disclosure did not survive the identity/token change.
		if tt.HasText("image/png") {
			t.Fatal("disclosure crossed identity/token")
		}
	}
	current = []session.Message{{ID: "assistant", Kind: session.MessageKindText, Role: "assistant", Body: "**bold** and `code`\n\n```go\nfmt.Println(\"child\")\n```\n[report](relative/child.txt) ![photo](attachment:child-image)", Streaming: true}}
	tt.Frame()
	for _, want := range []string{"bold and code", "fmt.Println", "relative/child.txt", "attachment:child-image", i18n.T(i18n.KeyChatReadOnlyLive)} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q: %q", want, tt.Texts())
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyChatReadOnlyCopy)); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != current[0].Body {
		t.Fatal("message copy not original Markdown")
	}
	current = []session.Message{{ID: "user", Kind: session.MessageKindText, Role: "user", Body: "**literal** `relative/file`"}}
	tt.Frame()
	if !tt.HasText("**literal** `relative/file`") {
		t.Fatal("user text reinterpreted")
	}
	current = []session.Message{{ID: "file", Kind: session.MessageKindDeliverables, Role: "tool", Files: []string{"../child-only.txt"}, Body: "presented"}}
	tt.Frame()
	if !tt.HasText("../child-only.txt") {
		t.Fatal("file identity missing")
	}
	// Pure clipping retains newest data, valid UTF-8, and never edits inputs.
	huge := strings.Repeat("界", readOnlyMessageByteLimit)
	input := []session.Message{{ID: "large", Kind: session.MessageKindText, Role: "assistant", Body: huge}}
	bounded, limited := readOnlyHistoryMessages(input)
	if !limited || len(bounded) != 1 || !utf8.ValidString(bounded[0].Body) || len(bounded[0].Body) > readOnlyMessageByteLimit || input[0].Body != huge {
		t.Fatal("per-row byte bound")
	}
	many := make([]session.Message, projectflow.ProcessRecordLimit+1)
	for i := range many {
		many[i] = session.Message{ID: string(rune(i + 100)), Kind: session.MessageKindText, Body: strings.Repeat("x", 2048)}
	}
	bounded, limited = readOnlyHistoryMessages(many)
	bytes := 0
	for _, m := range bounded {
		bytes += readOnlyMessageBytes(m)
	}
	if !limited || len(bounded) > projectflow.ProcessRecordLimit || bytes > readOnlyHistoryByteLimit || bounded[len(bounded)-1].ID != many[len(many)-1].ID {
		t.Fatal("reader bounds/newest retention")
	}
	for i := range many {
		many[i].Body = "small"
	}
	bounded, limited = readOnlyHistoryMessages(many)
	if !limited || len(bounded) != projectflow.ProcessRecordLimit || bounded[0].ID != many[1].ID {
		t.Fatal("400-row display cap")
	}
	if fetches.Load() != 0 {
		t.Fatal("child rendering used parent attachment resolver")
	}
}
