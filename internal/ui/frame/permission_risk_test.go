package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// TestPermissionRiskAcknowledgementEnablesConfirm pins the regression where
// the acknowledgement was a fresh local each frame: the checkbox flipped back
// to off and the primary button stayed disabled, so the dialog looked dead.
func TestPermissionRiskAcknowledgementEnablesConfirm(t *testing.T) {
	colors := theme.Resolve("deepseek.light").Active(false)
	acknowledged := false
	cancelled, confirmed := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		permissionRiskDialog(c, colors, 7,
			"确认启用完全权限？", "说明", "session-7", "启用完全权限", acknowledged,
			func(v bool) { acknowledged = v },
			func() { cancelled++ }, func() { confirmed++ })
	}, 800, 600)
	tt.Frame()
	if _, ok := tt.Find("取消"); !ok {
		t.Fatalf("dialog has no cancel; texts=%q", tt.Texts())
	}
	if err := tt.Click("我已了解这些风险，并愿意继续"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !acknowledged {
		t.Fatal("clicking the checkbox did not acknowledge the risk")
	}
	if err := tt.Click("启用完全权限"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if confirmed != 1 {
		t.Fatalf("confirm clicked %d times, want 1", confirmed)
	}
	if cancelled != 0 {
		t.Fatalf("cancel clicked %d times, want 0", cancelled)
	}
}
