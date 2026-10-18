package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

func TestTodoPanelViewHiddenWhenEmpty(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		TodoPanelView(c, TodoPanelModel{ThemeID: "system"})
	}, 400, 200)
	if tt.HasText("任务") {
		t.Fatal("empty todos should not paint")
	}
}

func TestTodoPanelViewShowsProgress(t *testing.T) {
	collapsed := false
	tt := ui.NewTester(func(c *ui.Context) {
		TodoPanelView(c, TodoPanelModel{
			Todos: []session.PlanTodo{
				{Content: "a", Status: "completed"},
				{Content: "b", Status: "in_progress"},
				{Content: "c", Status: "pending"},
			},
			ThemeID:   "system",
			Collapsed: &collapsed,
		})
	}, 480, 320)
	joined := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"任务", "已完成", "进行中", "待处理", "a", "b", "c"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}
