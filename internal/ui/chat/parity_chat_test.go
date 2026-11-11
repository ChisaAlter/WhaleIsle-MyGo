package chat

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParityQueueDockKeyboardReorderAndMoveFailed(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		open := true
		moves := [][2]any{}
		allowMove := true
		var m PlanBoardModel
		m = PlanBoardModel{
			ThemeID: "deepseek.light",
			Open:    &open,
			Board: session.PlanBoard{
				Queue: []session.PlanQueueItem{
					{ID: "q1", Preview: "first item"},
					{ID: "q2", Preview: "second item"},
				},
				QueueActions: session.QueueActionState{Mutable: true, Running: true},
			},
			OnQueueMove: func(id string, toIndex int) {
				moves = append(moves, [2]any{id, toIndex})
			},
			OnQueueMoveResult: func(id string, toIndex int) bool {
				return allowMove
			},
			OnQueueEdit:   func(string, string) {},
			OnQueueSteer:  func(string) {},
			OnQueueRemove: func(string) {},
		}

		tt := ui.NewTester(func(c *ui.Context) { QueueDockView(c, m) }, 680, 260)

		wantReorder1 := queueReorderLabel(1, 2)
		wantReorder2 := queueReorderLabel(2, 2)
		wantHint := i18n.T(i18n.KeyParityConvQueueReorderHint)
		if !tt.HasText(wantReorder1) || !tt.HasText(wantReorder2) {
			t.Fatalf("[%s] missing queue.reorder labels (%q, %q) in %v", locale, wantReorder1, wantReorder2, tt.Texts())
		}
		if !tt.HasText(wantHint) {
			t.Fatalf("[%s] missing queue.reorder.hint %q in %v", locale, wantHint, tt.Texts())
		}

		for _, b := range tt.Interactives() {
			if slices.Contains(b.Kinds, "click") && !slices.Contains(b.Kinds, "text") && b.Cursor != ui.CursorPointer {
				t.Fatalf("[%s] clickable queue dock control without CursorPointer: %s", locale, b.String())
			}
		}

		// Tab to focus the first item's grip button (after the collapse toggle button), then press ArrowDown.
		tt.Key(0, ui.KeyTab)
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(wantReorder1) {
			t.Fatalf("[%s] expected first grip (%q) to be focused after Tab", locale, wantReorder1)
		}
		tt.Key(0, ui.KeyDown)
		if len(moves) != 1 || moves[0][0] != "q1" || moves[0][1] != 1 {
			t.Fatalf("[%s] ArrowDown reorder = %+v, want [q1 1]", locale, moves)
		}

		// Reorder the slice so q1 is now at index 1 and press ArrowUp to move it back to index 0.
		m.Board.Queue[0], m.Board.Queue[1] = m.Board.Queue[1], m.Board.Queue[0]
		tt.Frame()
		tt.Key(0, ui.KeyUp)
		if len(moves) != 2 || moves[1][0] != "q1" || moves[1][1] != 0 {
			t.Fatalf("[%s] ArrowUp reorder = %+v, want [q1 0]", locale, moves)
		}

		// Simulate move failure (e.g. message already started sending) and verify queue.moveFailed notice.
		allowMove = false
		tt.Key(0, ui.KeyUp)
		wantMoveFailed := i18n.T(i18n.KeyParityConvQueueMoveFailed)
		if !tt.HasText(wantMoveFailed) {
			t.Fatalf("[%s] missing queue.moveFailed %q in %v", locale, wantMoveFailed, tt.Texts())
		}
	}
}

func TestParityTurnUsageSummaryWithCacheAndMessageActionsCursor(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	cacheRead := 800.0
	cacheWrite := 100.0
	reasoning := 50.0
	msg := session.Message{ID: "a1", SourceMessageID: "a1", Role: "assistant", Kind: session.MessageKindText, Seq: 2, Body: "Answer"}
	endMsg := session.Message{ID: "end", Role: "status", EventType: "turn/end", Kind: session.MessageKindText, Seq: 3}
	m := Model{
		Snapshot: session.ViewState{
			SelectedID:  "s1",
			ClientReady: true,
			Messages: []session.Message{
				{ID: "u1", Role: "user", Kind: session.MessageKindText, Seq: 1, Body: "Prompt"},
				msg,
				endMsg,
			},
			TurnUsage: map[float64]session.TurnUsage{
				3: {
					Input:      200,
					CacheRead:  &cacheRead,
					CacheWrite: &cacheWrite,
					Output:     200,
					Reasoning:  &reasoning,
					Total:      1200,
					Routes:     []string{"deepseek/deepseek-chat"},
				},
			},
		},
		OnQuoteMessage: func(string) {},
		OnForkTurn:     func(float64) {},
		FeedbackStore:  session.New(),
	}

	i18n.SetLocale(i18n.LocaleZH)
	ttZH := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			TurnUsageView(c, m, endMsg, ui.RGB(100, 100, 100))
			View(c, m)
		})
	}, 840, 520)
	if !ttZH.HasText("用量 1.20K · 缓存命中 80%") {
		t.Fatalf("zh missing turnUsage.summaryWithCache in %v", ttZH.Texts())
	}
	if !ttZH.HasText("在新对话中分支") {
		t.Fatalf("zh missing message.branch in %v", ttZH.Texts())
	}
	for _, b := range ttZH.Interactives() {
		if slices.Contains(b.Kinds, "click") && !slices.Contains(b.Kinds, "text") && b.Cursor != ui.CursorPointer {
			t.Fatalf("zh clickable chat element missing CursorPointer: %s", b.String())
		}
	}

	i18n.SetLocale(i18n.LocaleEN)
	ttEN := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			TurnUsageView(c, m, endMsg, ui.RGB(100, 100, 100))
			View(c, m)
		})
	}, 840, 520)
	if !ttEN.HasText("Usage 1.20K · 80% cache hit") || !ttEN.HasText("Usage 1.20K · Cache hit 80%") {
		t.Fatalf("en missing turnUsage.summaryWithCache in %v", ttEN.Texts())
	}
	if !ttEN.HasText("Branch into a new conversation") {
		t.Fatalf("en missing message.branch in %v", ttEN.Texts())
	}
}

func TestParityStepProcessAndCatalogKeys(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	keys := []string{
		i18n.KeyParityChatStepThinking,
		i18n.KeyParityChatStepWebFetch,
		i18n.KeyParityChatStepSubagents,
		i18n.KeyParityChatStepPlan,
		i18n.KeyParityChatStepQuestions,
		i18n.KeyParityChatStepTools,
		i18n.KeyParityChatStepDoneThinking,
		i18n.KeyParityChatStepDoneWebFetch,
		i18n.KeyParityChatStepDoneSubagents,
		i18n.KeyParityChatStepDonePlan,
		i18n.KeyParityChatStepDoneQuestions,
		i18n.KeyParityChatStepDoneTools,
		i18n.KeyParityConvHeroClaudeHeadline,
		i18n.KeyParityConvSessionMoreViews,
		i18n.KeyParityConvPlanApproved,
	}
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		for _, k := range keys {
			if got := i18n.T(k); got == "" || got == k {
				t.Fatalf("[%s] key %s resolved to %q", locale, k, got)
			}
		}
	}
}
