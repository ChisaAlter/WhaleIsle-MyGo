package frame

import (
	"math"
	"testing"
)

// TestHeroChipRowKeepsTheNativeInsets checks the round217 rule in the full
// 1440-wide shell: the workspace chip starts 32.67 DIP inside the hero row and
// the preset chip follows 3.33 DIP after its 150-DIP box. The absolute chip x
// (502 native) also depends on the composer card's horizontal centring, which
// is a separate open item, so this test pins the row-relative offsets.
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
	close("workspace chip width", float64(workspace.W), 150)
	close("workspace left inset", float64(workspace.X-row.X), 32.67)
	close("chip-to-preset gap", float64(preset.X-(workspace.X+workspace.W)), 3.33)
	if preset.X+preset.W > row.X+row.W {
		t.Fatalf("preset chip overflows the row: %v row=%v", preset, row)
	}
	t.Logf("chip row: row=%v workspace=%v preset=%v", row, workspace, preset)
}
