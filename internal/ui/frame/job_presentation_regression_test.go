package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const outputFailureSentence = "This task's output could not be read. Try again."

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
	if !tt.HasText(outputFailureSentence) {
		t.Fatalf("the output failure is not in the panel: %q", tt.Texts())
	}
	if tt.HasText("owned output failure") {
		t.Fatalf("the runtime's own error text is the sentence: %q", tt.Texts())
	}
	dismiss, ok := tt.Find("Dismiss")
	if !ok || dismiss.W <= 0 || dismiss.H <= 0 || dismiss.X < 0 || dismiss.Y < 0 || dismiss.X+dismiss.W > 500 || dismiss.Y+dismiss.H > 500 {
		t.Fatalf("failure dismissal is outside the viewport: %+v found=%v", dismiss, ok)
	}
	if err := tt.Click("Details"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("owned output failure") {
		t.Fatalf("the detail disclosure lost the runtime's text: %q", tt.Texts())
	}
	if err := tt.Click("Dismiss"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText(outputFailureSentence) {
		t.Fatalf("dismissed output failure still visible: %q", tt.Texts())
	}
	snap.Observed.Error = "replacement output failure"
	tt.Frame()
	if !tt.HasText(outputFailureSentence) {
		t.Fatal("dismissal hid a later output failure")
	}
	if err := tt.Click("Details"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("replacement output failure") {
		t.Fatalf("the replacement failure's own text is missing: %q", tt.Texts())
	}
}
