package interaction

import (
	"github.com/egoist/mygo/ui"
	"testing"
	store "whaleisle/internal/store/interaction"
	fonttheme "whaleisle/internal/ui/theme"
)

func TestQuestionHistoryFullContext(t *testing.T) {
	records := []store.QuestionHistoryView{{AgentID: "s", CallID: "q", Title: "Title", Open: true, Queued: true, Rows: []store.QuestionHistoryRow{{QuestionRowView: store.QuestionRowView{ID: "1", Header: "Context header", Question: "Pick one", Detail: "Detail", OptionLabels: []string{"chosen", "other"}, OptionDescriptions: []string{"chosen description", "other description"}}, Selected: []string{"chosen"}}}}}
	tt := ui.NewTester(func(c *ui.Context) {
		questionHistory(c, records, store.New(), fonttheme.Resolve("deepseek.light").Light, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(0, 80, 160))
	}, 800, 600)
	for _, s := range []string{"Context header", "chosen description", "other description"} {
		if !tt.HasText(s) {
			t.Fatalf("missing history context %s", s)
		}
	}
}
