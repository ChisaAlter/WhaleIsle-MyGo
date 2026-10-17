package interaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
)

func questionEditorFixture(t *testing.T, questions string) (*interaction.Store, *ui.Tester, <-chan dsh.EventResultOutcome) {
	t.Helper()
	store := interaction.New()
	t.Cleanup(store.Detach)
	ready := make(chan struct{}, 1)
	store.Attach(func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	outcome := make(chan dsh.EventResultOutcome, 1)
	go func() {
		out, _ := store.HandleWaterfall(dsh.EventsWaterfall{AgentID: "A", EventID: "e", Event: interaction.EventUserQuestions, Request: json.RawMessage(`{"questions":` + questions + `}`)})
		outcome <- *out
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("question not opened")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() { View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("A"), ThemeID: "deepseek.light"}) })
	}, 600, 900)
	return store, tt, outcome
}

func TestQuestionCustomShiftEnterPreservesNewline(t *testing.T) {
	store, tt, outcome := questionEditorFixture(t, `[{"id":"q","question":"Question"}]`)
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Type("one")
	tt.Key(ui.Shift, ui.KeyEnter)
	tt.Type("two")
	if got := *store.QuestionCustomPtr("e", "q"); got != "one\ntwo" {
		t.Fatalf("multiline answer = %q", got)
	}
	select {
	case got := <-outcome:
		t.Fatalf("Shift+Enter submitted: %+v", got)
	default:
	}
}

func TestQuestionCustomEnterAdvancesThenSubmits(t *testing.T) {
	store, tt, outcome := questionEditorFixture(t, `[{"id":"first","question":"First"},{"id":"last","question":"Last"}]`)
	if err := tt.Click("Question answer: First"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if !tt.Focused("Question answer: First") {
		t.Fatal("blank answer advanced")
	}
	tt.Type("first")
	tt.Key(0, ui.KeyEnter)
	if !tt.Focused("Question answer: Last") {
		t.Fatal("Enter did not focus next question")
	}
	select {
	case got := <-outcome:
		t.Fatalf("intermediate Enter submitted: %+v", got)
	default:
	}
	tt.Type("last")
	tt.Key(0, ui.KeyEnter)
	select {
	case got := <-outcome:
		var answer struct {
			Answers []struct {
				ID     string
				Custom string
			}
		}
		if json.Unmarshal(got.Value, &answer) != nil || got.Kind != "result" || len(answer.Answers) != 2 || answer.Answers[0].Custom != "first" || answer.Answers[1].Custom != "last" {
			t.Fatalf("outcome %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("final Enter did not submit")
	}
	if !store.SnapshotForAgent("A").Pending[0].Sending {
		t.Fatal("answer did not freeze")
	}
}

func TestQuestionCustomIMEAndShiftDoNotSubmit(t *testing.T) {
	_, tt, outcome := questionEditorFixture(t, `[{"id":"q","question":"Question"}]`)
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Compose("ni", 2)
	tt.Key(0, ui.KeyEnter)
	select {
	case got := <-outcome:
		t.Fatalf("composition Enter submitted: %+v", got)
	default:
	}
	tt.Type("你")
	tt.Compose("", 0)
	tt.Key(0, ui.KeyEnter)
	select {
	case got := <-outcome:
		t.Fatalf("commit Enter submitted: %+v", got)
	default:
	}
	tt.Key(ui.Shift, ui.KeyEnter)
	tt.Type("好")
	select {
	case got := <-outcome:
		t.Fatalf("Shift+Enter submitted: %+v", got)
	default:
	}
}

func TestQuestionEditorSendingHasNoLivePointerBindings(t *testing.T) {
	store, tt, outcome := questionEditorFixture(t, `[{"id":"q","question":"Question","multiSelect":true,"options":[{"label":"A"},{"label":"B"}]}]`)
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Type("before")
	if err := tt.Click("A"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	store.ResolveQuestions("e")
	// The UI still has its pre-Sending frame here: an input handler must not
	// mutate a pointer inside the retained draft before the next redraw.
	tt.Type("late")
	tt.Frame()
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Type("forbidden")
	if err := tt.Click("B"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("跳过此题 / Skip question"); err != nil {
		t.Fatal(err)
	}
	if got := *store.QuestionCustomPtr("e", "q"); got != "before" {
		t.Fatalf("busy text mutated: %q", got)
	}
	if *store.QuestionOptionCheckedPtr("e", "q", "B") || store.SnapshotForAgent("A").QuestionCards["e"].Rows[0].Skipped {
		t.Fatal("busy choice/skip mutated")
	}
	select {
	case got := <-outcome:
		if got.Kind != "result" {
			t.Fatal(got)
		}
	case <-time.After(time.Second):
		t.Fatal("answer lost")
	}
}

func TestQuestionEditorKeyGuard(t *testing.T) {
	var keys questionEditorKeys
	advances := 0
	advance := func() { advances++ }
	keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: ui.Shift}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputCompose, Text: "ni"}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputText, Text: "你"}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputKeyUp, Key: ui.KeyEnter}, false, advance)
	keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Repeat: true}, false, advance)
	if advances != 0 {
		t.Fatal("IME, newline or held Enter advanced")
	}
	if !keys.handle(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Mods: ui.Ctrl}, false, advance) || advances != 1 {
		t.Fatal("non-Shift Enter not handled")
	}
}
