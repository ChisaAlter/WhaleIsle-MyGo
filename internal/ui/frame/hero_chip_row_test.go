package frame

import (
	"math"
	"testing"
)

// Installed0.3.10 ConversationRoot.heroWorkspaceRow defines20-DIP left padding
// and a two-DIP chip gap. Pin component-relative offsets so changing viewport
// or caption geometry cannot preserve an obsolete absolute-seat measurement.
func TestHeroChipRowKeepsTheNativeInsets(t *testing.T) {
	tt := chatChromeTester(t)
	workspace, ok := tt.Find("Composer workspace")
	if !ok {
		t.Fatalf("no workspace chip; texts=%v", tt.Texts())
	}
	preset, ok := tt.Find("Composer hero preset")
	if !ok {
		t.Fatal("no preset chip")
	}
	row, ok := tt.Find("Composer hero preset row")
	if !ok {
		t.Fatal("no hero row")
	}
	close := func(name string, got, want float64) {
		if math.Abs(got-want) > 0.01 {
			t.Fatalf("%s = %.2f, want %.2f", name, got, want)
		}
	}
	// HeroShell.workspaceChip is content-sized inline-flex under a 360 cap, so
	// a short workspace name keeps the chevron snug instead of pinning 150.
	if workspace.W <= 0 || workspace.W >= 200 {
		t.Fatalf("workspace chip width = %.2f, want a content size under 200", workspace.W)
	}
	close("workspace left inset", float64(workspace.X-row.X), 20)
	close("sourcechip-to-preset gap", float64(preset.X-(workspace.X+workspace.W)), 2)
	if preset.X+preset.W > row.X+row.W {
		t.Fatalf("preset chip overflows the row: %v row=%v", preset, row)
	}
	t.Logf("chip row: row=%v workspace=%v preset=%v", row, workspace, preset)
}
