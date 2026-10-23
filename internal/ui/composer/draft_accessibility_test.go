package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestNativeDraftAccessibleReplacementUsesDraftHistoryAndSend(t *testing.T) {
	for _, hero := range []bool{false, true} {
		name := "conversation"
		if hero {
			name = "hero"
		}
		t.Run(name, func(t *testing.T) {
			draft := "A😀原稿\n尾行"
			client := NewDraftClient(&draft)
			client.SetSelectionUTF16(3, 5)
			originalSelection := client.Selection()
			sent := ""
			m := Model{Draft: &draft, TextInput: client, UseTextInputClient: true, Hero: hero, WorkspaceRoot: "fixture", OnSend: func(text string) { sent = text }}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 700)
			if _, ok := any(&client.input).(ui.AccessibleTextInputClient); !ok {
				t.Fatal("registered native wrapper lost the accessibility contract")
			}
			if info := client.input.TextInputAccessibility(); !info.Multiline || info.Placeholder != composerPlaceholder(m) || info.Password || info.ReadOnly {
				t.Fatalf("composer accessibility metadata: %+v", info)
			}
			if err := tt.SetValue("Composer draft", "新😀草稿\n第二行"); err != nil {
				t.Fatal(err)
			}
			if draft != "新😀草稿\n第二行" || sent != "" {
				t.Fatal("accessible replacement edited only the selection or sent a prompt", draft, sent)
			}
			tt.Command("undo")
			if draft != "A😀原稿\n尾行" || client.Selection() != originalSelection {
				t.Fatal("accessible edit bypassed the draft's undo owner", draft, client.Selection())
			}
			tt.Command("redo")
			if draft != "新😀草稿\n第二行" {
				t.Fatal("redo did not restore the accessible edit", draft)
			}
			tt.Key(ui.Cmd, ui.KeyZ)
			if draft != "A😀原稿\n尾行" {
				t.Fatal("keyboard undo did not share accessible edit history", draft)
			}
			tt.Key(ui.Cmd, ui.KeyY)
			if draft != "新😀草稿\n第二行" {
				t.Fatal("keyboard redo did not share accessible edit history", draft)
			}
			if err := tt.Click(i18n.T(i18n.KeyComposerSend)); err != nil {
				t.Fatal(err)
			}
			if sent != "新😀草稿\n第二行" || draft != "" {
				t.Fatal("Send did not consume the accessible draft once", sent, draft)
			}
			if client.Undo() {
				t.Fatal("sent draft leaked into the next prompt's undo history")
			}
		})
	}
}

func TestNativeDraftAccessibleReplacementClearsMarkedTextAndRejectsInert(t *testing.T) {
	draft := "base"
	client := NewDraftClient(&draft)
	m := Model{Draft: &draft, TextInput: client, UseTextInputClient: true, WorkspaceRoot: "fixture"}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 600, 600)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Compose("候选", 2)
	if err := tt.SetValue("Composer draft", "全新值😀"); err != nil {
		t.Fatal(err)
	}
	if _, marked := client.MarkedRange(); marked || draft != "全新值😀" || client.typing.composing || client.compositionPending != nil {
		t.Fatal("accessible replacement left an active preedit", draft)
	}
	tt.Type("!")
	if draft != "全新值😀!" {
		t.Fatal("typing after replacement used stale native offsets", draft)
	}
	m.Inert = true
	tt.Frame()
	if err := tt.SetValue("Composer draft", "forbidden"); err == nil || draft != "全新值😀!" {
		t.Fatal("inert composer accepted accessible editing", err, draft)
	}
}
