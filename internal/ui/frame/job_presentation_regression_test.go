package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettledJobShowsProgressAndOutputErrorCanBeDismissed(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	controller := jobflow.New(nil)
	job := jobflow.Job{ID: "owned", Label: "Synthetic task", Status: "failed", Progress: "last settled progress", Detail: "fallback detail"}
	snap := jobflow.Snapshot{SelectedID: job.ID, Observed: jobflow.Observation{Error: "owned output failure"}}
	tt := ui.NewTester(func(c *ui.Context) { jobRowView(c, controller, snap, job, theme.Resolve("deepseek.light").Light) }, 500, 500)
	if !tt.HasText("last settled progress") || tt.HasText("fallback detail") {
		t.Fatal("settled job did not prefer its last progress")
	}
	if _, ok := tt.Find("Job state: failed"); !ok {
		t.Fatal("settled error state has no status marker")
	}
	if err := tt.Click("Dismiss"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("owned output failure") {
		t.Fatal("dismissed output failure still visible")
	}
	snap.Observed.Error = "replacement output failure"
	tt.Frame()
	if !tt.HasText("replacement output failure") {
		t.Fatal("dismissal hid a later output failure")
	}
}
