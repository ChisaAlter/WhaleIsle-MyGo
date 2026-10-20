package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestQuestionReplyDisclosureCopyAndSessionIsolation(t *testing.T) {
	body := `{"questions":[{"id":"q","header":"Header","question":"Pick","detail":"Full detail","options":[{"label":"Yes","description":"Description"}]}],"answers":[{"id":"q","selected":["Yes"],"custom":"custom"}]}`
	msg := session.Message{ID: "reply", Kind: session.MessageKindQuestionReply, Body: body, EventTime: 1000}
	scope := "one"
	tt := ui.NewTester(func(c *ui.Context) { QuestionReplyView(c, msg, scope, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 800, 700)
	if tt.HasText("Full detail") {
		t.Fatal("not collapsed")
	}
	tt.Click("Question reply details reply")
	for _, text := range []string{"Header", "Full detail", "Yes — Description", i18n.T(i18n.KeyChatQuestionAnswer) + ": Yes, custom"} {
		if !tt.HasText(text) {
			t.Fatalf("missing %s", text)
		}
	}
	tt.Click("Copy question reply reply")
	if tt.Clipboard() != "Header — Pick\n" + i18n.T(i18n.KeyChatQuestionAnswer) + ": Yes, custom" {
		t.Fatal(tt.Clipboard())
	}
	scope = "two"
	tt.Frame()
	if tt.HasText("Full detail") {
		t.Fatal("disclosure crossed session")
	}
	msg.Body = strings.Repeat("fallback ", 100) + "TAIL"
	tt.Frame()
	tt.Click("Copy question reply reply")
	if tt.Clipboard() != msg.Body {
		t.Fatal("raw fallback copy clipped")
	}
	if showMessageEdit(msg) || showMessageCopy(msg) {
		t.Fatal("reply gained generic human actions")
	}
}
