package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTurnAnchorsCountDurablePrompts(t *testing.T) {
	messages := []session.Message{
		{ID: "a", Role: "user", Kind: session.MessageKindText, Seq: 2, Body: "first prompt"},
		{ID: "b", Role: "assistant", Kind: session.MessageKindText, Seq: 3, Body: "answer"},
		{ID: "c", Role: "user", Kind: session.MessageKindText, Seq: 4, Body: "[image] C:\\tmp\\a.png\nsecond   prompt"},
		{ID: "d", Role: "user", Kind: session.MessageKindTool, Seq: 5, Body: "not a prompt"},
		{ID: "e", Role: "user", Kind: session.MessageKindText, Seq: 6, Body: strings.Repeat("long ", 40)},
	}
	anchors := turnAnchors(messages)
	if len(anchors) != 3 {
		t.Fatalf("anchors = %d, want 3 durable prompts", len(anchors))
	}
	if anchors[0].Seq != 2 || anchors[1].Seq != 4 || anchors[2].Seq != 6 {
		t.Fatalf("anchor seqs = %v", []float64{anchors[0].Seq, anchors[1].Seq, anchors[2].Seq})
	}
	if anchors[1].Preview != "second prompt" {
		t.Fatalf("preview kept placeholders or spaces: %q", anchors[1].Preview)
	}
	if !strings.HasSuffix(anchors[2].Preview, "…") || len([]rune(anchors[2].Preview)) > 73 {
		t.Fatalf("long preview not elided: %q", anchors[2].Preview)
	}
}

func TestTurnRailTicksAndNarrowFallback(t *testing.T) {
	messages := []session.Message{
		{Role: "user", Kind: session.MessageKindText, Seq: 1, Body: "one"},
		{Role: "user", Kind: session.MessageKindText, Seq: 2, Body: "two"},
		{Role: "user", Kind: session.MessageKindText, Seq: 3, Body: "three"},
	}
	jumped := float64(0)
	tt := ui.NewTester(func(c *ui.Context) {
		TurnRailView(c, Model{Snapshot: session.ViewState{Messages: messages}}, 900, func(seq float64) { jumped = seq })
	}, 900, 400)
	tick, ok := tt.Find(i18n.T(i18n.KeyChatTurnRailJump) + " 2")
	if !ok {
		t.Fatalf("rail tick missing: %q", tt.Texts())
	}
	if tick.W != turnRailWidth {
		t.Fatalf("tick seat width %v, want the %v DIP rail column", tick.W, turnRailWidth)
	}
	if err := tt.Click(i18n.T(i18n.KeyChatTurnRailJump) + " 2"); err != nil {
		t.Fatal(err)
	}
	if jumped != 2 {
		t.Fatalf("jump target %v, want seq 2", jumped)
	}

	narrow := ui.NewTester(func(c *ui.Context) {
		TurnRailView(c, Model{Snapshot: session.ViewState{Messages: messages}}, 400, nil)
	}, 400, 400)
	if _, ok := narrow.Find(i18n.T(i18n.KeyChatTurnRailLabel) + " 3"); !ok {
		t.Fatalf("narrow fallback missing: %q", narrow.Texts())
	}
	if _, ok := narrow.Find(i18n.T(i18n.KeyChatTurnRailJump) + " 1"); ok {
		t.Fatal("narrow width kept the floating rail")
	}
}

func TestTurnRailSkipsSingleTurnTranscripts(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		TurnRailView(c, Model{Snapshot: session.ViewState{Messages: []session.Message{{Role: "user", Kind: session.MessageKindText, Seq: 1, Body: "only"}}}}, 900, nil)
	}, 900, 300)
	if _, ok := tt.Find(i18n.T(i18n.KeyChatTurnRailJump) + " 1"); ok {
		t.Fatal("one-turn transcript mounted a rail")
	}
}

func TestQueueDockShowsPreviewAndIconActions(t *testing.T) {
	edited, steered, removed := "", "", ""
	m := PlanBoardModel{
		ThemeID: "deepseek.light",
		Board: session.PlanBoard{
			Queue:        []session.PlanQueueItem{{ID: "q1", Preview: "next prompt to send"}},
			QueueActions: session.QueueActionState{Mutable: true, Running: true},
		},
		OnQueueEdit:   func(id, preview string) { edited = id + ":" + preview },
		OnQueueSteer:  func(id string) { steered = id },
		OnQueueRemove: func(id string) { removed = id },
	}
	tt := ui.NewTester(func(c *ui.Context) { QueueDockView(c, m) }, 640, 240)
	if !tt.HasText(i18n.T(i18n.KeyChatQueueTitle)) {
		t.Fatalf("dock title missing: %q", tt.Texts())
	}
	if !tt.HasText("next prompt to send") {
		t.Fatalf("preview missing: %q", tt.Texts())
	}
	edit := i18n.T(i18n.KeyPlanBoardQueueEdit)
	steer := i18n.T(i18n.KeyPlanBoardQueueSteer)
	remove := i18n.T(i18n.KeyPlanBoardQueueRemove)
	for _, label := range []string{edit, steer, remove} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("icon action %q missing", label)
		}
	}
	if err := tt.Click(edit); err != nil {
		t.Fatal(err)
	}
	if edited != "q1:next prompt to send" {
		t.Fatalf("edit callback = %q", edited)
	}
	if err := tt.Click(steer); err != nil {
		t.Fatal(err)
	}
	if steered != "q1" {
		t.Fatalf("steer callback = %q", steered)
	}
	if err := tt.Click(remove); err != nil {
		t.Fatal(err)
	}
	if removed != "q1" {
		t.Fatalf("remove callback = %q", removed)
	}
}

func TestQueueDockStaysMountedForEmptyPreview(t *testing.T) {
	m := PlanBoardModel{ThemeID: "deepseek.light", Board: session.PlanBoard{
		Queue:        []session.PlanQueueItem{{ID: "q2"}},
		QueueActions: session.QueueActionState{Mutable: true},
	}}
	tt := ui.NewTester(func(c *ui.Context) { QueueDockView(c, m) }, 640, 240)
	if !tt.HasText(i18n.T(i18n.KeyChatQueueEmptyPreview)) {
		t.Fatalf("empty preview placeholder missing: %q", tt.Texts())
	}
	empty := ui.NewTester(func(c *ui.Context) { QueueDockView(c, PlanBoardModel{ThemeID: "deepseek.light"}) }, 640, 240)
	if empty.HasText(i18n.T(i18n.KeyChatQueueTitle)) {
		t.Fatal("dock painted without queue rows")
	}
}
