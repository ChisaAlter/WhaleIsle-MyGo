package chat

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestParitySecretMaskAndReveal(t *testing.T) {
	values := []string{"sk-" + strings.Repeat("a", 20) + "1", "ghp_" + strings.Repeat("a", 36), "AIza" + strings.Repeat("a", 35), "AKIA" + strings.Repeat("A", 16), "eyJ" + strings.Repeat("a", 8) + ".eyJ" + strings.Repeat("b", 8) + "." + strings.Repeat("c", 8)}
	for _, value := range values {
		masked := maskedUserText("before " + value + " after")
		if strings.Contains(masked, value) || !strings.Contains(masked, "*") {
			t.Fatal("credential not masked", value)
		}
	}
	if got := maskedUserText("https://user:password@host/path"); got != "https://user:********@host/path" {
		t.Fatal(got)
	}
	private := "-----BEGIN PRIVATE KEY-----\nbody\n-----END PRIVATE KEY-----"
	if got := maskedUserText(private); got != "-----BEGIN PRIVATE KEY-----\n****\n-----END PRIVATE KEY-----" {
		t.Fatal(got)
	}
	harmless := "sk-" + strings.Repeat("a", 30)
	if maskedUserText(harmless) != harmless {
		t.Fatal("digit-free ordinary text masked")
	}
	if len(userSecrets("eyJ"+strings.Repeat("a", 2049)+".eyJ"+strings.Repeat("b", 8)+"."+strings.Repeat("c", 8))) != 0 {
		t.Fatal("JWT first-segment budget widened")
	}
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	raw := values[0]
	tt := ui.NewTester(func(c *ui.Context) {
		userReferenceText(c, raw, 14, c.Theme().Text, c.Theme().TextMuted, nil, userReferenceFacts{Interactive: true})
	}, 700, 180)
	if tt.HasText(raw) {
		t.Fatal("raw secret exposed before reveal")
	}
	if err := tt.Click("Reveal credential"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(raw) {
		t.Fatal("reveal not rendered")
	}
	raw = values[1]
	tt.Frame()
	if tt.HasText(raw) {
		t.Fatal("new credential inherited old reveal")
	}
}

func TestParityGoalDockAndNativeRenderEvidence(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	edited := ""
	paused := 0
	board := session.PlanBoard{Goal: &session.PlanGoal{ID: "active", Revision: 1, Phase: "active", Activation: "armed", Objective: "Ship the native conversation"}, Queue: []session.PlanQueueItem{{ID: "q", Preview: "Review the next change"}}, QueueActions: session.QueueActionState{Mutable: true, Running: true}}
	expanded := map[string]bool{"reasoning": true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(12).Background(ui.RGB(245, 246, 248)).Children(func() {
			GoalDockView(c, PlanBoardModel{Board: board, ThemeID: "deepseek.light", OnEditGoal: func(v string) { edited = v }, OnPauseGoal: func() { paused++ }, OnCompleteGoal: func() {}, OnClearGoal: func() {}})
			QueueDockView(c, PlanBoardModel{Board: board, ThemeID: "deepseek.light"})
			userReferenceText(c, "Connect using sk-abcdefghijklmnopqrst1234 and @notes.txt", 14, ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), nil, userReferenceFacts{})
			ThinkingBlockView(c, "reasoning", "**Native reasoning** preserves Markdown and code.\n\n```go\nanswer := 42\n```", &expanded, ThinkingStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(90, 90, 90), Field: ui.RGB(235, 236, 239), Size: 13})
		})
	}, 900, 560)
	if err := tt.Click("Edit goal"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Save goal"); err != nil {
		t.Fatal(err)
	}
	if edited != board.Goal.Objective {
		t.Fatal("goal editor did not seed accepted objective")
	}
	if err := tt.Click("Pause goal"); err != nil {
		t.Fatal(err)
	}
	if paused != 1 {
		t.Fatal("pause unreachable")
	}
	dir := os.Getenv("WHALEISLE_CHAT_RENDER_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, fixture *ui.Tester) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, fixture.Image()); err != nil {
			t.Fatal(err)
		}
	}
	write("native-chat-actions.png", tt)
	selected, tab := "reply", "overview"
	trajectory := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{PaneWidth: 1100, Snapshot: session.ViewState{SelectedID: "synthetic", Messages: []session.Message{{ID: "user", Seq: 1, Role: "user", Kind: session.MessageKindText, Body: "Explain the patch"}, {ID: "reply", Seq: 2, Role: "assistant", Kind: session.MessageKindText, Body: "## The native path\n\nThis **Markdown preview** keeps structure.\n\n```go\nresult := 42\n```"}}}, SelectedID: &selected, DetailTab: &tab})
	}, 1100, 650)
	write("native-trajectory-preview.png", trajectory)
}
func TestParityTrajectoryFoldsRetainSummaries(t *testing.T) {
	messages := []session.Message{{ID: "u", Role: "user", Kind: session.MessageKindText, Body: "input"}, {ID: "a", Role: "assistant", Kind: session.MessageKindText, Body: "answer"}, {ID: "t", Kind: session.MessageKindTool, Body: "read", Args: `{"path":"unique"}`}, {ID: "u2", Role: "user", Kind: session.MessageKindText, Body: "next"}}
	state := trajectoryFoldState{}
	all := trajectoryAllRows(messages, "")
	folded := state.rows(all, "", true, true)
	if len(folded) != 2 || folded[0].FoldKey != "turn:"+all[0].TurnKey || all[0].TurnKey == "" {
		t.Fatalf("turn summary lost %#v", folded)
	}
	state.toggle(folded[0])
	rows := state.rows(all, "", true, true)
	if len(rows) != 3 || rows[1].Role != "assistant-group" || !strings.Contains(rows[1].Full.Tools, "read") {
		t.Fatalf("assistant summary lost %#v", rows)
	}
	state.toggle(rows[1])
	rows = state.rows(all, "", true, true)
	if len(rows) != 4 {
		t.Fatalf("local call expansion lost %#v", rows)
	}
	if found := state.rows(all, "unique", true, true); len(found) != 1 || found[0].ID != "t" {
		t.Fatal("folds discarded full search data")
	}
}
func TestParityCollapseTiming(t *testing.T) {
	raw := func(turn int) json.RawMessage {
		return json.RawMessage(`{"data":{"turn":` + string(rune('0'+turn)) + `,"step":1,"reason":{"kind":"completed"}}}`)
	}
	msgs := []session.Message{{ID: "u", Seq: 1, Role: "user", Kind: session.MessageKindText, Raw: raw(1)}, {ID: "think", Seq: 2, Role: "assistant", Kind: session.MessageKindThinking, Body: "thought", Raw: raw(1)}, {ID: "answer", Seq: 3, Role: "assistant", Kind: session.MessageKindText, Body: "answer", Raw: raw(1)}, {ID: "end", Seq: 4, Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: raw(1)}}
	m := Model{Preferences: settings.ChatPreferences{TranscriptView: "standard", CollapseTiming: "next-input"}}
	rows, _ := projectTranscript(msgs, m)
	has := func(rows []session.Message, id string) bool {
		for _, r := range rows {
			if r.ID == id {
				return true
			}
		}
		return false
	}
	if !has(rows, "think") {
		t.Fatal("next-input folded at completion")
	}
	msgs = append(msgs, session.Message{ID: "u2", Seq: 5, Role: "user", Kind: session.MessageKindText, Raw: raw(2)})
	rows, _ = projectTranscript(msgs, m)
	if has(rows, "think") {
		t.Fatal("next input did not fold completed turn")
	}
}
func TestParityTerminalReceipts(t *testing.T) {
	args := map[string]json.RawMessage{"sessionId": json.RawMessage(`"term-1"`)}
	for _, tc := range []struct{ name, text string }{{"terminal_open", "started terminal session term-1 (build) [type: pwsh]\nready"}, {"terminal_read", "output\n[lines: 2-8 of 9]\n[output truncated]"}, {"terminal_signal", "delivered SIGINT to foreground process group 42"}, {"terminal_close", "closed terminal session term-1"}, {"terminal_close", "terminal session term-1 was already closing"}} {
		model := toolDetailsTerminal(tc.name, args, tc.text)
		if model == nil || len(model.Items) != 1 {
			t.Fatalf("missing %s %#v", tc.name, model)
		}
		if (tc.name == "terminal_open" || tc.name == "terminal_read") && model.Items[0].Code == nil {
			t.Fatal("output lacks code region")
		}
	}
	if toolDetailsTerminal("terminal_signal", args, "malformed") != nil {
		t.Fatal("malformed receipt invented fields")
	}
}

func TestParityTrajectoryNativeProgramUsageAndHistory(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	program := "const result = await tools.read({path: 'file.txt'});\nconsole.log(result)"
	args, _ := json.Marshal(map[string]string{"code": program})
	selected, tab := "code", "program"
	snap := session.ViewState{Messages: []session.Message{{ID: "code", Seq: 9, Kind: session.MessageKindTool, Body: "run_code", Args: string(args), EventType: "tool/call"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: snap, SelectedID: &selected, DetailTab: &tab})
	}, 1100, 720)
	if err := tt.Click("Copy program"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != program {
		t.Fatal("program copy returned JSON envelope")
	}
	if err := tt.Click("JSON arguments"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Structured JSON") {
		t.Fatal("recorded JSON mode not available")
	}
	snap.Messages = []session.Message{{ID: "old", Seq: 1, Kind: session.MessageKindSystemPrompt, Body: "old policy"}, {ID: "new", Seq: 2, Kind: session.MessageKindSystemPrompt, Body: "new policy"}}
	selected, tab = "new", "history-diff"
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "policy") {
		t.Fatal("system history diff empty")
	}
	snap.TrajectoryRequests = []session.TrajectoryRequest{{ID: "one", StartSeq: 1, Usage: json.RawMessage(`{"inputTokens":10,"outputTokens":2}`)}, {ID: "two", StartSeq: 2, Usage: json.RawMessage(`{"inputTokens":20,"outputTokens":3}`)}}
	selected, tab = "request:two", "usage"
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "30") {
		t.Fatal("request cumulative usage missing")
	}
	selected = "new"
	tt.Frame()
	if tab != "overview" {
		t.Fatal("changing record left an unavailable request-only tab selected")
	}
}
