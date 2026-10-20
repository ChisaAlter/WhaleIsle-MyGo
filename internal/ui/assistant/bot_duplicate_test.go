package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotDuplicateUIRequiresIndependentGoalConsent(t *testing.T) {
	var item pluginrpc.DshbotBotItem
	raw := []byte(`{"id":"source","kind":"bot","sessionId":"source-session","name":"Source","title":"Display title","sectionId":"","description":"Full preserved description","avatar":{"kind":"blob","shape":"circle","color":"blue"},"workspaceId":"workspace-fixture","model":{"provider":"provider","model":"model","reasoningEffort":"high"},"pinned":true,"hidden":false,"pinOrder":7,"allowedSenderIds":[],"memberBotIds":[],"objective":"Complete this entire objective","objectiveMaxRounds":23,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"selected","names":["read"]},"skills":{"mode":"all","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":2,"maxSpeaks":7,"inbox":[{"id":"old-message"}],"readInitialized":true,"lastSeenSeq":8,"createdAt":1,"updatedAt":2}`)
	json.Unmarshal(raw, &item.DshbotCatalogItem)
	json.Unmarshal(raw, &item.Profile)
	cmd := pluginrpc.DshbotBotDuplicateCommand{SourceID: "source", Name: "Chosen copy name", ScratchCwd: "/reviewed/cwd", Expected: pluginrpc.DshbotBotCatalog{Revision: 7, Items: []pluginrpc.DshbotBotItem{item}}}
	state := dshbotstore.BotDuplicateSnapshot{Scope: "fixture", Loaded: true, Writable: true, Catalog: cmd.Expected, Draft: &dshbotstore.BotDuplicateDraft{}, Review: &dshbotstore.BotDuplicateReview{Command: cmd}}
	saves, opens := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		botDuplicateView(c, BotDuplicateModel{State: state, HostLabel: "https://fixture.invalid", SourceID: "source", OnRefresh: func() {}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnAcknowledgeExecution: func(v bool) { state.Review.ExecutionAcknowledged = v }, OnConfirm: func() { saves++ }, OnOpen: func(string) { opens++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 3300)
	if saves != 0 || opens != 0 {
		t.Fatal("render copied/opened")
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 7", "source-session", "Chosen copy name", "/reviewed/cwd", "Complete this entire objective", "23", "Full preserved description", "provider", "memoryReview", "allowedSenderIds", "bot 记忆", "user 记忆", "首次 idle", "费用", "不自动打开"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing complete value/effect", term)
		}
	}
	tt.Click(botDuplicateConfirmLabel())
	if saves != 0 {
		t.Fatal("no ack")
	}
	if err := tt.Click(botDuplicateAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	tt.Click(botDuplicateConfirmLabel())
	if saves != 0 {
		t.Fatal("no independent goal ack")
	}
	if err := tt.Click(botDuplicateExecutionAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(botDuplicateConfirmLabel()); err != nil {
		t.Fatal(err)
	}
	if saves != 1 || opens != 0 {
		t.Fatal("duplicate must never auto-open", saves, opens)
	}
	state.Review = nil
	state.Uncertain = true
	state.Pending = &cmd
	state.Candidates = []pluginrpc.DshbotBotItem{item}
	tt.Frame()
	text = strings.Join(tt.Texts(), "\n")
	if !strings.Contains(text, "不能证明") || !strings.Contains(text, "保持未知") {
		t.Fatal("candidate presented as causal success")
	}
}
