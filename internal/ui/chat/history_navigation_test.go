package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestNativeHistoryTurnNavigationViewFixture(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale("zh")
	messages := []session.Message{{ID: "m2", Seq: 2, TurnSeq: 1, Kind: session.MessageKindText, Role: "user", Body: "first"}, {ID: "m81", Seq: 81, TurnSeq: 80, Kind: session.MessageKindText, Role: "user", Body: "middle"}}
	if session.HistoryTurnMessageIndex(messages, session.HistoryTurn{Turn: 2, Seq: 80}) != 1 {
		t.Fatal("turn did not resolve to final row")
	}
	list := &ui.ListState{FollowEnd: true}
	s := session.New()
	m := Model{HistoryStore: s, List: list, Snapshot: session.ViewState{SelectedID: "main", ClientReady: true, PlanHistoryBinding: "host/main", Messages: messages, HistoryNavigation: session.HistoryNavigationView{Turns: []session.HistoryTurn{{Turn: 1, Seq: 1, Prompt: "first", Loaded: true}, {Turn: 2, Seq: 80, Prompt: "middle", Loaded: true}}, Busy: true, HasTarget: true, TargetSeq: 1}}}
	prepareHistoryNavigation(m, messages, list)
	key := list.Key(1)
	prepended := append([]session.Message{{ID: "older", Seq: 1}}, messages...)
	prepareHistoryNavigation(m, prepended, list)
	if key != list.Key(2) {
		t.Fatal("prepend changed a stable displayed identity")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { HistoryNavigationView(c, m, list, c.Theme().TextMuted) })
	}, 640, 180)
	for _, label := range []string{"Turn directory", "Cancel jump", "Latest"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("missing native action %q", label)
		}
	}
	var selected float64
	var count int
	picker := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Children(func() {
				if seq, ok := historyTurnPicker(c, m.Snapshot.HistoryNavigation, true); ok {
					selected = seq
					count++
				}
			})
		})
	}, 640, 360)
	for range 2 {
		if err := picker.Click("Turn directory"); err != nil {
			t.Fatal(err)
		}
		if _, ok := picker.Find("Turn 2 · middle"); !ok {
			t.Fatalf("directory did not open: hasText=%v bounds=%v", picker.HasText("Turn 2 · middle"), func() ui.Rect { r, _ := picker.Find("Turn directory"); return r }())
		}
		if err := picker.Click("Turn 2 · middle"); err != nil {
			t.Fatal(err)
		}
	}
	if selected != 80 || count != 2 {
		t.Fatalf("repeated explicit turn choice: seq=%v count=%d", selected, count)
	}

}
