package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
)

func TestWhaleSettingsNativeFormRequiresExplicitClear(t *testing.T) {
	var editor assiststore.WhaleSettingsEditor
	editor.Bind("synthetic")
	q, _ := editor.BeginLoad("synthetic")
	editor.FinishLoad(q, pluginrpc.WhaleSettings{HomeDir: "/fixture", SessionID: "whale-fixture", Name: "Whale", Memory: "saved memory", PersonaText: "kind"}, nil)
	editor.Edit("synthetic", "memory", "unsaved draft")
	confirmed := 0
	saves := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeWhale, HarnessReady: true, WhaleEnabled: true, WhaleSettings: WhaleSettingsModel{State: editor.Snapshot(), OnRefresh: func() {}, OnEdit: func(f, v string) { editor.Edit("synthetic", f, v) }, OnRestore: func(f string) { editor.Restore("synthetic", f) }, OnSave: func(a string) { saves = append(saves, a) }, OnReviewClear: func() { editor.ReviewClear("synthetic") }, OnCancelClear: func() { editor.CancelClear("synthetic") }, OnConfirmClear: func() {
			if _, _, ok := editor.BeginWrite("synthetic", "clear"); ok {
				confirmed++
			}
		}}}
		whaleSettingsBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.RGB(60, 130, 240))
	}, 1100, 2200)
	for _, text := range []string{i18n.T(i18n.KeyMiscAssistantSaveMemory), i18n.T(i18n.KeyMiscAssistantSavePersona)} {
		if !tt.HasText(text) {
			t.Fatalf("missing native control %q", text)
		}
	}
	if tt.HasText(i18n.T(i18n.KeyMiscAssistantConfirmClearMemory)) || confirmed != 0 {
		t.Fatal("clear confirmation initially open")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantClearMemory)); err != nil {
		t.Fatal(err)
	}
	if confirmed != 0 {
		t.Fatal("first click sent clear")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantCancelClear)); err != nil {
		t.Fatal(err)
	}
	if editor.Snapshot().Memory != "unsaved draft" {
		t.Fatal("cancel lost draft")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantSavePersona)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 || saves[0] != "persona" {
		t.Fatal(saves)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantClearMemory)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmClearMemory)); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("confirmation did not submit exactly once")
	}
}
