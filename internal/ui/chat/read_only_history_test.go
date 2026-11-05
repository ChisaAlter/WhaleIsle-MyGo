package chat

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// A single synthetic pure projection/render fixture. No carrier or real Host.
func TestAgentReadOnlyRichHistoryFixture(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)
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
	if output := toolLifecycleOutput(projected[0].Tool); !strings.Contains(output, "child-image") || !strings.Contains(output, "child output") || !strings.Contains(output, "image caption") {
		t.Fatalf("rich fallback lost durable evidence: %q", output)
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
	// A canonical typed read keeps its real content card and copy action in the
	// child, rather than restoring the obsolete TextOnly IN/OUT copy button.
	current = []session.Message{{ID: "typed-read", Kind: session.MessageKindTool, Tool: toolReadViewCall(2)}}
	tt.Frame()
	if err := tt.Click(toolRowTitle("read")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Read content card") || !tt.HasText("recorded-line-01") {
		t.Fatalf("child lost typed read card: %q", tt.Texts())
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "recorded-line-00\nrecorded-line-01" {
		t.Fatalf("typed child copy lost content: %q", tt.Clipboard())
	}
	if os.Getenv("CHAT_GALLERY_DIR") != "" {
		tt.SetTime(time.Now().Add(time.Second))
		tt.SetSize(800, 360)
		writeGallery(t, galleryDir(t), "child-typed-read-800.png", tt)
		tt.SetSize(400, 360)
		writeGallery(t, galleryDir(t), "child-typed-read-400.png", tt)
		tt.SetSize(800, 1200)
	}
	current = []session.Message{{ID: "assistant", Kind: session.MessageKindText, Role: "assistant", Body: "**bold** and `code`\n\n```go\nfmt.Println(\"child\")\n```\n[report](relative/child.txt) ![photo](attachment:child-image)", Streaming: true}}
	tt.Frame()
	for _, want := range []string{"bold and code", "fmt.Println", "relative/child.txt", "attachment:child-image"} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q: %q", want, tt.Texts())
		}
	}
	if tt.HasText("Assistant message actions") {
		t.Fatal("streaming child exposed settled message actions")
	}
	current[0].Streaming = false
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyChatCopy)); err != nil {
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
	// Paging retains full message content, valid UTF-8, and never edits inputs.
	huge := strings.Repeat("界", readOnlyMessageByteLimit)
	input := []session.Message{{ID: "large", Kind: session.MessageKindText, Role: "assistant", Body: huge}}
	bounded, limited := readOnlyHistoryMessages(input)
	if limited || len(bounded) != 1 || !utf8.ValidString(bounded[0].Body) || bounded[0].Body != huge || input[0].Body != huge {
		t.Fatal("full long child message was lost")
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
	if !limited || len(bounded) != projectflow.ProcessRecordLimit || bytes < projectflow.ProcessRecordLimit*2048 || bounded[len(bounded)-1].ID != many[len(many)-1].ID {
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
