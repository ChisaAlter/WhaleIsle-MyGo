package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/workflowrun"
)

func TestWorkflowPhaseSummaryMatchesActiveAndInterruptedHistory(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, test := range []struct {
		name, want string
		statuses   []string
	}{
		{"empty", "Completed 0", nil},
		{"clean", "Completed 2", []string{"completed", "completed"}},
		{"running suppresses clean", "Running 1", []string{"completed", "running"}},
		{"failure suppresses clean", "Failed 1", []string{"failed", "completed"}},
		{"cancelled suppresses clean", "Cancelled 1", []string{"completed", "cancelled"}},
		{"interruption preserves clean history", "Completed 1 · Running 1 · Failed 1 · Cancelled 1 · Interrupted 2", []string{"interrupted", "completed", "failed", "running", "interrupted", "cancelled"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			members := make([]workflowrun.Member, len(test.statuses))
			for i, status := range test.statuses {
				members[i].Status = status
			}
			if got := workflowPhaseSummary(members); got != test.want {
				t.Fatalf("phase history summary = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWorkflowWholeMemberRowRetainsFocusWithoutNavigatingFinishedChild(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	member := workflowrun.Member{Seq: 4, Label: "Owned child task", Status: "running"}
	available, opened := true, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			paintWorkflowMember(c, member, member.Label, available, ToolCardStyle{
				FG: c.Theme().Text, Muted: c.Theme().TextMuted, Accent: c.Theme().Accent,
			}, func() { opened++ })
			ui.Button(c, "Outside workflow member")
		})
	}, 400, 150)
	row, ok := tt.Find("Open Owned child task")
	if !ok || row.W != 400 || row.H != 24 {
		t.Fatalf("member action must occupy its complete 24-DIP content row: %+v", row)
	}
	if tt.HasText("查看过程 / View process") {
		t.Fatal("separate navigation button retained instead of the original whole-row action")
	}
	if err := tt.Click("Open Owned child task"); err != nil || opened != 1 {
		t.Fatalf("whole-row navigation callback: opened=%d err=%v", opened, err)
	}
	member.Status, available = "completed", false
	for range 3 {
		tt.Frame()
		if !tt.Focused("Owned child task") {
			t.Fatal("finishing the child dropped its focused row before departure")
		}
	}
	tt.Key(0, ui.KeyEnter)
	if opened != 1 {
		t.Fatal("retained focus navigated a no-longer-available member")
	}
	if err := tt.Click("Outside workflow member"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if tt.Focused("Owned child task") {
		t.Fatal("a completed, blurred member remained a keyboard navigation stop")
	}
	if _, ok := tt.Find("Open Owned child task"); ok {
		t.Fatal("completed member retained a live-navigation accessible label")
	}
}
