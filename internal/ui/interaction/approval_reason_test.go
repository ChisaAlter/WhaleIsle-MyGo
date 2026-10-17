package interaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
)

func TestApprovalReasonFollowsLocaleWithoutLosingOriginal(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("zh")
	store := interaction.New()
	t.Cleanup(store.Detach)
	ready := make(chan struct{}, 1)
	store.Attach(func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	outcome := make(chan *dsh.EventResultOutcome, 1)
	go func() {
		out, _ := store.HandleWaterfall(dsh.EventsWaterfall{AgentID: "A", EventID: "approval", Event: interaction.EventApproval,
			Request: json.RawMessage(`{"toolName":"read","reason":"original host reason","displayReason":{"en":"Read project files","zh":"读取项目文件"}}`)})
		outcome <- out
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("approval did not open")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("A"), ScopeAgentID: "A", ThemeID: "deepseek.light"})
		})
	}, 600, 500)
	if !tt.HasText("读取项目文件") || !tt.HasText("original host reason") {
		t.Fatal("approval reason ignores Chinese locale or loses original host reason")
	}
	pending := store.SnapshotForAgent("A").Pending[0]
	if pending.ApprovalOriginalReason != "original host reason" || pending.ApprovalDisplayReason != `{"en":"Read project files","zh":"读取项目文件"}` {
		t.Fatal("original reason payload was not retained intact")
	}
	i18n.SetLocale("en")
	tt.Frame()
	if !tt.HasText("Read project files") || !tt.HasText("original host reason") || tt.HasText("读取项目文件") {
		t.Fatal("pending approval reason did not follow locale change")
	}
	select {
	case <-outcome:
		t.Fatal("language switch answered the approval")
	default:
	}
	store.ResolveDeny("approval")
	select {
	case out := <-outcome:
		if out == nil || string(out.Value) != `"rejected"` {
			t.Fatalf("approval decision changed: %+v", out)
		}
	case <-time.After(time.Second):
		t.Fatal("approval not answerable after locale change")
	}
}

func TestApprovalReasonLocaleFallbackAndVerbatimText(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, tc := range []struct{ locale, display, want string }{
		{"zh", `{"en":"English"}`, "English"},
		{"zh", `{"en":"English","zh":"中文"}`, "中文"},
		{"zh", `{"en":"English","zh-cn":"区域中文","zh":"中文"}`, "区域中文"},
		{"en", `{"en":"English","zh":"中文"}`, "English"},
		{"zh", `{"en":"English","zh":""}`, ""},
		{"zh", `""`, ""},
		{"zh", `"Literal {name}"`, "Literal {name}"},
		{"en", `{"zh":"中文"}`, "original"},
		{"zh", `{malformed`, "original"},
		{"zh", `null`, "original"},
	} {
		i18n.SetLocale(tc.locale)
		p := interaction.Pending{ApprovalOriginalReason: "original", ApprovalDisplayReason: tc.display, Detail: "legacy detail"}
		if got := approvalReasonText(p); got != tc.want {
			t.Errorf("locale %q, display %s: got %q, want %q", tc.locale, tc.display, got, tc.want)
		}
	}
	if got := approvalReasonText(interaction.Pending{Detail: "legacy detail"}); got != "legacy detail" {
		t.Fatalf("legacy detail lost: %q", got)
	}
}
