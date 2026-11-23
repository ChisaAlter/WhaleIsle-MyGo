package chat

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	nativeui "whaleisle/internal/ui/openui"
)

func TestOpenUIFenceStageUI(t *testing.T) {
	store := session.New()
	msg := session.Message{ID: "m", Role: "assistant", Streaming: true}
	m := Model{FeedbackStore: store, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}}
	tt := ui.NewTester(func(c *ui.Context) {
		if !openUIFence(c, m, msg, "openui", "main=Text(\"real program\")", 20) {
			t.Error("fence not recognized")
		}
	}, 800, 1000)
	if !tt.HasText(i18n.T(i18n.KeyChatIsolatedSource)) {
		t.Fatal("streaming native block missing")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatIsolatedSource)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("main=Text(\"real program\")") {
		t.Fatal("source inaccessible")
	}
}

func TestOpenUIFenceRendersNatively(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	store := session.New()
	msg := session.Message{ID: "native", Role: "assistant"}
	m := Model{FeedbackStore: store, Snapshot: session.ViewState{SelectedID: "s-native", PlanHistoryBinding: "h"}}
	program := "root = Col([Text(\"Native hello\"), Button(\"Inc\", @Set($n, $n + 1)), Text(\"n=\" + $n), HtmlView(\"<b>x</b>\", 100, \"Embedded page\"), Button(\"Ask\", @ToAssistant(\"hi\"))])\n$n = 0"
	tt := ui.NewTester(func(c *ui.Context) {
		openUIFence(c, m, msg, "whale-ui", program, 3)
	}, 800, 1000)
	for _, s := range []string{"Native hello", "n=0", "Embedded page", nativeui.OpenCompatLabel()} {
		if !tt.HasText(s) {
			t.Fatalf("%q missing: %q", s, tt.Texts())
		}
	}
	if err := tt.Click("Inc"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("n=1") {
		t.Fatalf("local action did not apply: %q", tt.Texts())
	}
	// No Host owner: the submit fails with a sentence, never the raw error.
	if err := tt.Click("Ask"); err != nil {
		t.Fatal(err)
	}
	want := errcopy.Sentence(errCardOwner)
	deadline := time.Now().Add(5 * time.Second)
	for !tt.HasText(want) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText(want) || tt.HasText("OpenUI block owner is no longer active") {
		t.Fatalf("failure copy: %q", tt.Texts())
	}
	if openUICompatFor(openui.Identity("h", "s-native", "native", 3)) != nil {
		t.Fatal("the workbench opened without a click")
	}
}
