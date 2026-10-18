package interaction

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
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
	store, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First"},{"id":"last","question":"Last"}]`)
	if err := tt.Click("Question answer: First"); err != nil {
		t.Fatal(err)
	}
	tt.Type("retained answer")
	if err := tt.Click("下一题 / Next question"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: First"); ok {
		t.Fatal("previous editor remained mounted")
	}
	if _, ok := tt.Find("Question answer: Last"); !ok {
		t.Fatal("next editor missing")
	}
	if err := tt.Click("收起问题 / Minimize question"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: Last"); ok {
		t.Fatal("minimize did not remove body")
	}
	if _, ok := tt.Find("上一题 / Previous question"); ok {
		t.Fatal("minimize did not remove footer")
	}
	if err := tt.Click("展开问题 / Expand question"); err != nil {
		t.Fatal(err)
	}
	if tt.Focused("Question answer: Last") {
		t.Fatal("expand stole focus from toggle")
	}
	if err := tt.Click("查看答案摘要 / Review answers"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("retained answer") {
		t.Fatal("summary omitted retained answer")
	}
	if err := tt.Click("回看第 1 题 / Review question 1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Question answer: First"); !ok {
		t.Fatal("summary could not navigate back")
	}
	if err := tt.Click("暂时隐藏 / Hide for now"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("重新打开问题 / Reopen questions"); err != nil {
		t.Fatal(err)
	}
	if got := store.SnapshotForAgent("A").QuestionCards["e"]; got.Index != 0 || got.Rows[0].Custom != "retained answer" {
		t.Fatalf("reopen lost progress: %+v", got)
	}
	select {
	case got := <-outcome:
		t.Fatalf("navigation sent an answer: %+v", got)
	default:
	}
}

func TestQuestionWizardSingleChoiceAdvancesAndFinalSkipSends(t *testing.T) {
	store, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First","options":[{"label":"A"},{"label":"B"}]},{"id":"last","question":"Last"}]`)
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
	if err := tt.Click("跳过此题 / Skip question"); err != nil {
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
	before := store.SnapshotForAgent("A").QuestionCards["e"].Index
	_ = tt.Click("上一题 / Previous question")
	_ = tt.Click("收起问题 / Minimize question")
	if store.SnapshotForAgent("A").QuestionCards["e"].Index != before {
		t.Fatal("Sending navigation changed page")
	}
	if _, ok := tt.Find("Question answer: Last"); !ok {
		t.Fatal("Sending minimized answer")
	}
}

func TestQuestionWizardOptionEnterDoesNotSelectOrAdvance(t *testing.T) {
	store, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First","multiSelect":true,"options":[{"label":"A"},{"label":"B"}]},{"id":"last","question":"Last"}]`)
	if err := tt.Click("A"); err != nil {
		t.Fatal(err)
	}
	// Clicking a multi-select option keeps the page and gives the option focus.
	tt.Key(0, ui.KeyEnter)
	card := store.SnapshotForAgent("A").QuestionCards["e"]
	if card.Index != 1 || len(card.Rows[0].Selected) != 1 || card.Rows[0].Selected[0] != "A" {
		t.Fatal("option Enter toggled choice instead of validating batch")
	}
	select {
	case got := <-outcome:
		t.Fatalf("incomplete option Enter submitted: %+v", got)
	default:
	}
}
