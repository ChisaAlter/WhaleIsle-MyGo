package agents

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

func TestViewListsChildren(t *testing.T) {
	st := session.ViewState{
		SelectedID: "root",
		Items: []session.Item{
			{SessionID: "root", Title: "Main"},
			{SessionID: "c1", ParentID: "root", Title: "Worker", Running: true},
			{SessionID: "c2", ParentID: "root", Title: "Reviewer"},
			{SessionID: "other", Title: "Unrelated"},
		},
	}
	draft := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Snapshot:    st,
			ThemeID:     "system",
			PromptDraft: &draft,
			OnStop:      func(string) {},
			OnInterrupt: func(string, string) {},
			OnPrompt:    func(string, string, string) {},
		})
	}, 360, 520)
	for _, want := range []string{"智能体团队", "Main", "Worker", "Reviewer", "停止", "中断", "全部中断", "发送"} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q; texts=%q", want, tt.Texts())
		}
	}
	if tt.HasText("Unrelated") {
		t.Fatal("unrelated session must not appear")
	}
}
