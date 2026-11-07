package interaction

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func TestQuestionWizardShowsOnlyCurrentQuestion(t *testing.T) {
	_, tt, _ := questionEditorFixture(t, `[{"id":"first","question":"First"},{"id":"last","question":"Last"}]`)
	if _, ok := tt.Find("Question answer: First"); !ok {
		t.Fatal("first question missing")
	}
	if _, ok := tt.Find("Question answer: Last"); ok {
		t.Fatal("all-question editor still exposes later answers")
	}
}

func TestQuestionWizardPagerMinimizeSummaryAndReopen(t *testing.T) {
	s, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First"},{"id":"last","question":"Last"}]`)
	if err := tt.Click("Question answer: First"); err != nil {
		t.Fatal(err)
	}
	tt.Type("retained answer")
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionNext)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: First"); ok {
		t.Fatal("previous editor remained mounted")
	}
	if _, ok := tt.Find("Question answer: Last"); !ok {
		t.Fatal("next editor missing")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionMinimize)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: Last"); ok {
		t.Fatal("minimize did not remove body")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyChatQuestionPrev)); ok {
		t.Fatal("minimize did not remove footer")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionExpand)); err != nil {
		t.Fatal(err)
	}
	if tt.Focused("Question answer: Last") {
		t.Fatal("expand stole focus from toggle")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionReview)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("retained answer") {
		t.Fatal("summary omitted retained answer")
	}
	if err := tt.Click(fmt.Sprintf(i18n.T(i18n.KeyChatQuestionReviewN), 1)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: First"); !ok {
		t.Fatal("summary could not navigate back")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionHide)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionReopen)); err != nil {
		t.Fatal(err)
	}
	if got := s.SnapshotForAgent("A").QuestionCards["e"]; got.Index != 0 || got.Rows[0].Custom != "retained answer" {
		t.Fatalf("reopen lost progress: %+v", got)
	}
	select {
	case got := <-outcome:
		t.Fatalf("navigation sent an answer: %+v", got)
	default:
	}
}

func TestQuestionWizardSingleChoiceAdvancesAndFinalSkipSends(t *testing.T) {
	s, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First","options":[{"label":"A"},{"label":"B"}]},{"id":"last","question":"Last"}]`)
	if err := tt.Click("B"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: Last"); !ok {
		t.Fatal("single choice did not advance")
	}
	select {
	case got := <-outcome:
		t.Fatalf("choice auto-submitted: %+v", got)
	default:
	}
	if err := tt.Click(i18n.T(i18n.KeyChatQuestionSkip)); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-outcome:
		if got.Kind != "result" {
			t.Fatal(got)
		}
	case <-time.After(time.Second):
		t.Fatal("last skip did not submit")
	}
	before := s.SnapshotForAgent("A").QuestionCards["e"].Index
	_ = tt.Click(i18n.T(i18n.KeyChatQuestionPrev))
	_ = tt.Click(i18n.T(i18n.KeyChatQuestionMinimize))
	if s.SnapshotForAgent("A").QuestionCards["e"].Index != before {
		t.Fatal("Sending navigation changed page")
	}
	if _, ok := tt.Find("Question answer: Last"); !ok {
		t.Fatal("Sending minimized answer")
	}
}

func TestQuestionWizardOptionEnterDoesNotSelectOrAdvance(t *testing.T) {
	s, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First","multiSelect":true,"options":[{"label":"A"},{"label":"B"}]},{"id":"last","question":"Last"}]`)
	if err := tt.Click("A"); err != nil {
		t.Fatal(err)
	}
	// Clicking a multi-select option keeps the page and gives the option focus.
	tt.Key(0, ui.KeyEnter)
	card := s.SnapshotForAgent("A").QuestionCards["e"]
	if card.Index != 1 || len(card.Rows[0].Selected) != 1 || card.Rows[0].Selected[0] != "A" {
		t.Fatal("option Enter toggled choice instead of validating batch")
	}
	select {
	case got := <-outcome:
		t.Fatalf("incomplete option Enter submitted: %+v", got)
	default:
	}
}

func TestQuestionContinuedWaitStatusAndSummaryParity(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })

	s := store.New()
	t.Cleanup(s.Detach)
	colors := fonttheme.Resolve("deepseek.light").Light
	muted := ui.RGB(80, 80, 80)

	i18n.SetLocale("zh")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			questionWaitStatus(c, store.Pending{AgentID: "A", EventID: "c1", Continued: true}, store.QuestionCardView{}, s, colors, muted)
			questionWaitStatus(c, store.Pending{AgentID: "A", EventID: "w1"}, store.QuestionCardView{Wait: &store.QuestionWaitView{RemainingMs: 4200, Running: true, State: "counting"}}, s, colors, muted)
			questionWaitStatus(c, store.Pending{AgentID: "A", EventID: "w2"}, store.QuestionCardView{Wait: &store.QuestionWaitView{RemainingMs: 4200, Running: false, State: "focused"}}, s, colors, muted)
			questionWaitStatus(c, store.Pending{AgentID: "A", EventID: "w3"}, store.QuestionCardView{Wait: &store.QuestionWaitView{State: "waiting"}}, s, colors, muted)
		})
	}, 600, 400)
	for _, want := range []string{"已继续工作，仍可回答", "5 秒后继续工作", "已暂停 · 剩余 5 秒", "会一直等你回答"} {
		if !tt.HasText(want) {
			t.Fatalf("missing zh wait status %q in %q", want, tt.Texts())
		}
	}

	i18n.SetLocale("en")
	tt.Frame()
	for _, want := range []string{"Work continued — you can still answer", "Continuing in 5 seconds", "Paused · 5 seconds remaining", "Will wait for your answer."} {
		if !tt.HasText(want) {
			t.Fatalf("missing en wait status %q in %q", want, tt.Texts())
		}
	}

	rows := []store.QuestionRowView{
		{ID: "q1", Question: "First", Selected: []string{"Fast (recommended)"}},
		{ID: "q2", Question: "Second"},
	}
	i18n.SetLocale("zh")
	if got := questionAnsweredCountText(rows); got != "1/2 已回答" {
		t.Fatalf("zh answered count = %q", got)
	}
	if got := questionStatusText("ask.pendingDetail"); got != "等待中的问题仍可在输入框中回答" {
		t.Fatalf("zh pendingDetail = %q", got)
	}
	if got := questionFeedback("resubmit"); got != "回答未送达，工作已继续，请再提交一次。" {
		t.Fatalf("zh resubmit feedback = %q", got)
	}
	i18n.SetLocale("en")
	if got := questionAnsweredCountText(rows); got != "1/2 answered" {
		t.Fatalf("en answered count = %q", got)
	}
	if got := questionStatusText("ask.pendingDetail"); got != "These pending questions remain answerable from the composer." {
		t.Fatalf("en pendingDetail = %q", got)
	}
	if got := questionFeedback("resubmit"); got != "The answer did not arrive before work continued; submit it again." {
		t.Fatalf("en resubmit feedback = %q", got)
	}

	history := []store.QuestionHistoryView{{
		AgentID: "A", CallID: "call-1", Title: "History", Open: true,
		Rows: []store.QuestionHistoryRow{
			{
				QuestionRowView: store.QuestionRowView{
					ID: "q1", Question: "Pick speed",
					OptionLabels: []string{"Fast (recommended)", "Slow"},
					Recommended:  []bool{true, false},
				},
				Selected: []string{"Fast (recommended)"},
			},
			{
				QuestionRowView: store.QuestionRowView{ID: "q2", Question: "Optional note"},
			},
		},
	}}
	i18n.SetLocale("zh")
	histTester := ui.NewTester(func(c *ui.Context) {
		questionHistory(c, history, s, colors, ui.RGB(0, 0, 0), muted, ui.RGB(0, 80, 160))
	}, 600, 400)
	if !histTester.HasText("1/2 已回答") || !histTester.HasText("Fast") || !histTester.HasText("推荐") || histTester.HasText("Fast (recommended)") {
		t.Fatalf("history did not strip recommended suffix or show count/badge: %q", histTester.Texts())
	}
}
