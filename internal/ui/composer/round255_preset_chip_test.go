package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRound255PresetChipComposition is the round255 diagnostic for the preset
// chip width: it prints the chip, its label and the workspace chip so the
// 100 DIP live width can be decomposed against the native 106.67.
func TestRound255PresetChipComposition(t *testing.T) {
	d := NewDraftState("")
	preset := "标准模式"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{State: d, Draft: &d.Text, Hero: true, WorkspaceRoot: `C:\fixture`,
			AgentPresetPick: &preset, OnPickHeroWorkspace: func() {}})
	}, 1440, 900)
	chip, ok := tt.Find("Composer hero preset")
	if !ok {
		t.Fatal("preset chip missing")
	}
	workspace, ok := tt.Find("Composer workspace")
	if !ok {
		t.Fatal("workspace chip missing")
	}
	label, ok := tt.Find("标准模式")
	if !ok {
		t.Fatal("preset label missing")
	}
	t.Logf("workspace chip  x=%.2f w=%.2f h=%.2f", workspace.X, workspace.W, workspace.H)
	t.Logf("preset chip     x=%.2f w=%.2f h=%.2f (native 655.33 / 106.67)", chip.X, chip.W, chip.H)
	t.Logf("preset label    x=%.2f w=%.2f h=%.2f", label.X, label.W, label.H)
	t.Logf("chip - label:   left=%.2f right=%.2f (padding+gap+chevron)", label.X-chip.X, chip.X+chip.W-(label.X+label.W))
	t.Logf("chip-to-chip gap=%.2f", chip.X-(workspace.X+workspace.W))
}
