package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// oneLineHeight is the Original 24px disclosure seat.
const oneLineHeight = 24

func TestDisclosureRowIsOneLineAndExpandsOnClick(t *testing.T) {
	opened := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(8).Children(func() {
			paintDisclosureRow(c, disclosureRow{
				Key:        "row",
				Title:      "Title",
				Summary:    "summary",
				Status:     statusDone,
				Open:       opened,
				Expandable: true,
				OnToggle:   func() { opened = !opened },
			}, disclosureStyle{FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), Hover: ui.RGB(240, 240, 240), Accent: ui.RGB(40, 90, 200), Size: 13})
			if opened {
				ui.Text(c, "expanded body")
			}
		})
	}, 600, 300)
	row, ok := tt.Find("Title summary")
	if !ok {
		t.Fatalf("row label missing: %q", tt.Texts())
	}
	if row.H > oneLineHeight+0.5 {
		t.Fatalf("collapsed row height %v, want the 24px seat", row.H)
	}
	if tt.HasText("expanded body") {
		t.Fatal("body rendered before opening")
	}
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("expanded body") {
		t.Fatal("row did not expand")
	}
	if err := tt.Click("Title"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("expanded body") {
		t.Fatal("row did not collapse")
	}
}

func TestToolRowCollapsedKeepsOneLineAndStructuredBody(t *testing.T) {
	msg := session.Message{
		ID:   "call-1",
		Kind: session.MessageKindTool,
		Tool: &session.ToolCallView{
			CallID:    "call-1",
			Name:      "bash",
			Arguments: `{"command":"npm test"}`,
			State:     "ok",
		},
	}
	st := DefaultToolCardStyle(ui.RGB(0, 0, 0), ui.RGB(90, 90, 90), ui.RGB(220, 220, 220), ui.RGB(245, 245, 245), ui.RGB(40, 90, 200))
	tt := ui.NewTester(func(c *ui.Context) { ToolMessageView(c, msg, st, ToolImageGalleryOptions{ID: msg.ID}) }, 800, 500)
	if !tt.HasText(toolKindTitle(ToolKindBash)) {
		t.Fatalf("localized tool title missing: %q", tt.Texts())
	}
	if !tt.HasText("npm test") {
		t.Fatalf("argument summary missing: %q", tt.Texts())
	}
	before, ok := tt.Find(toolKindTitle(ToolKindBash))
	if !ok {
		t.Fatal("row title seat missing")
	}
	if before.H > oneLineHeight+0.5 {
		t.Fatalf("collapsed tool row height %v, want one 24px line", before.H)
	}
	if err := tt.Click(toolKindTitle(ToolKindBash)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	// The terminal body mounts with the row: command label + output block.
	if !tt.HasText(i18n.T(i18n.KeyChatToolCommand)) {
		t.Fatalf("expanded command body missing: %q", tt.Texts())
	}
}

func TestInternalStatusPhrasesStayOutOfTranscript(t *testing.T) {
	for _, internal := range []string{"workspace ready: Whale Isle", "workspace ready", "session ready: abc"} {
		if got := displayableAction(internal); got != "" {
			t.Fatalf("internal phrase %q surfaced as %q", internal, got)
		}
	}
	if got := displayableAction("  已重新连接  "); got != "已重新连接" {
		t.Fatalf("user-facing status lost: %q", got)
	}
	if displayableAction("   ") != "" {
		t.Fatal("blank status rendered")
	}
}

func TestTurnEndRowShowsStatusAndElapsed(t *testing.T) {
	cases := []struct {
		body string
		raw  string
		want string
	}{
		{body: "45000", raw: `{"data":{"reason":{"kind":"completed"}}}`, want: "已完成，用时 45 秒"},
		{body: "78000", raw: `{"data":{"reason":{"kind":"max-tokens"}}}`, want: turnStatusLabel(session.Message{Raw: []byte(`{"data":{"reason":{"kind":"max-tokens"}}}`)})},
	}
	for _, tc := range cases {
		msg := session.Message{ID: "end", Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Seq: 9, Body: tc.body, Raw: []byte(tc.raw)}
		tt := messageTester(msg, 800, nil)
		if !tt.HasText(tc.want) {
			t.Fatalf("turn/end row missing %q: %q", tc.want, tt.Texts())
		}
	}
}
