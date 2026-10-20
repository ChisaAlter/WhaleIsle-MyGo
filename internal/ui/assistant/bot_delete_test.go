package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotDeleteUIDefaultCancelAndCompleteEffects(t *testing.T) {
	p := pluginrpc.DshbotBotDeletePlan{Catalog: pluginrpc.DshbotDeleteCatalog{Revision: 9}, Target: pluginrpc.DshbotBotItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "exact-id", Name: "Exact name", SessionID: "exact-session"}}, Impact: json.RawMessage(`{"notepad":"dshbot-routine/id.md","task":"Full untruncated task body","group":"complete group members"}`)}
	text := botDeleteReviewText("fixture-host", p)
	for _, want := range []string{"exact-id", "Exact name", "exact-session", "revision 9", "Full untruncated task body", "dshbot-routine/id.md"} {
		if !strings.Contains(text, want) {
			t.Fatal("omitted review", want, text)
		}
	}
	count := 0
	m := BotDeleteModel{HostLabel: "fixture-host", SourceID: "exact-id", State: dshbotstore.BotDeleteSnapshot{Scope: "scope", Loaded: true, Writable: true, Review: &dshbotstore.BotDeleteReview{Plan: p}}, OnConfirm: func() { count++ }, OnCancel: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		botDeleteView(c, m, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 3000)
	tt.Click(botDeleteConfirmLabel())
	if count != 0 {
		t.Fatal("render/default mutated")
	}
	for _, want := range []string{"不可恢复", "notepad", "当前", "原子", "未知", "历史", "memory"} {
		if !strings.Contains(botDeleteEffectsWarning, want) {
			t.Fatal("missing consequence", want)
		}
	}
	if botDeleteReviewConfirmed(m.State.Review) {
		t.Fatal("default destructive acceptance")
	}
	m.State.Review.TypedID = "exact-id"
	m.State.Review.TypedName = "Exact name"
	if botDeleteReviewConfirmed(m.State.Review) {
		t.Fatal("typed fields bypass danger consent")
	}
	m.State.Review.Acknowledged = true
	if !botDeleteReviewConfirmed(m.State.Review) {
		t.Fatal("explicit confirmation rejected")
	}
}
