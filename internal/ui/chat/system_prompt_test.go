package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSystemPromptExpandsInlineWithBoundedCodeBody(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	body := "system instructions\n" + strings.Repeat("long prompt line\n", 40)
	msg := session.Message{ID: "sys", Kind: session.MessageKindSystemPrompt, Body: body}
	m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Children(func() {
			SystemPromptView(c, m, msg, ui.RGB(0, 0, 0), ui.RGB(90, 90, 90))
			ui.Text(c, "NEXT ROW").Height(30)
		})
	}, 800, 600)
	before, ok := tt.Find("NEXT ROW")
	if !ok {
		t.Fatal("next row missing")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatSystemPrompt)); err != nil {
		t.Fatal(err)
	}
	after, _ := tt.Find("NEXT ROW")
	// Installed SystemPromptRow uses the shared 141px OpaqueBody inline.
	// Four DIP body margin and four DIP flow padding surround that scrollport.
	if after.Y-before.Y != 149 {
		t.Fatalf("inline prompt did not reserve its bounded body: before=%v after=%v", before, after)
	}
	if !tt.HasText("system instructions") {
		t.Fatal("prompt body did not open")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatSystemPrompt)); err != nil {
		t.Fatal(err)
	}
	tt.SetTime(time.Now().Add(time.Second))
	after, _ = tt.Find("NEXT ROW")
	if after.Y != before.Y || tt.HasText("system instructions") {
		t.Fatalf("closing prompt did not restore inline flow: before=%v after=%v", before, after)
	}
}
