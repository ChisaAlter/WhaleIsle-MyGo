package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestOpenUIFenceStageUI(t *testing.T) {
	store := session.New()
	msg := session.Message{ID: "m", Role: "assistant", Streaming: true}
	m := Model{FeedbackStore: store, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}}
	tt := ui.NewTester(func(c *ui.Context) {
		if !openUIFence(c, m, msg, "openui", "main=Text(\"real program\")", 20) {
			t.Error("fence not recognized")
		}
	}, 800, 400)
	if !tt.HasText("生成完成后可打开交互窗口 / Available when generation completes") {
		t.Fatal("streaming limitation hidden")
	}
	if err := tt.Click("源码 / Source"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("main=Text(\"real program\")") {
		t.Fatal("source inaccessible")
	}
}
