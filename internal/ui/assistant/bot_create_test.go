package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotCreateUIFullReviewExplicitAction(t *testing.T) {
	cmd := pluginrpc.DshbotBotCreateCommand{ID: "bot-fixture", SessionID: "canonical-fixture", Expected: pluginrpc.DshbotBotCatalog{Revision: 12}, Draft: pluginrpc.DshbotBotDraft{Name: "Stable name", Title: "Display title", Description: "Full description untruncated", WorkspaceID: "workspace-fixture", ScratchCwd: "/reviewed/cwd", Provider: "provider-id", Model: "model-id", ReasoningEffort: "high", MemoryReview: true, Notify: true}}
	state := dshbotstore.BotCreateSnapshot{Scope: "fixture", Loaded: true, Writable: true, Catalog: cmd.Expected, Draft: &dshbotstore.BotCreateDraft{}, Review: &dshbotstore.BotCreateReview{Command: cmd}}
	saves := 0
	tt := ui.NewTester(func(c *ui.Context) {
		botCreateView(c, BotCreateModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() {}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 3000)
	if saves != 0 {
		t.Fatal("render mutated")
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 12", "Stable name", "Display title", "Full description untruncated", "workspace-fixture", "/reviewed/cwd", "bot-fixture", "canonical-fixture", "provider-id", "model-id", "high", "circle/blue", "objective", "memoryReview: true", "notify: true", "全部", "费用", "不自动重试"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing reviewed field/effect", term)
		}
	}
	tt.Click(botCreateConfirmLabel)
	if saves != 0 {
		t.Fatal("no acknowledgement")
	}
	if err := tt.Click(botCreateAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(botCreateConfirmLabel); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit create missing")
	}
}
