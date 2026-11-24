package frame

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// TestPermissionRiskAcknowledgementEnablesConfirm pins the regression where
// the acknowledgement was a fresh local each frame: the checkbox flipped back
// to off and the primary button stayed disabled, so the dialog looked dead.
func TestPermissionRiskAcknowledgementEnablesConfirm(t *testing.T) {
	colors := theme.Resolve("deepseek.light").Active(false)
	acknowledged := false
	frames, ackFrame := 0, 0
	cancelled, confirmed := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		frames++
		permissionRiskDialog(c, colors, 7,
			"确认启用完全权限？", "说明", "session-7", "启用完全权限", acknowledged,
			func(v bool) {
				acknowledged = v
				if v && ackFrame == 0 {
					ackFrame = frames
				}
			},
			func() { cancelled++ }, func() { confirmed++ })
	}, 800, 600)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))
	if _, ok := tt.Find("取消"); !ok {
		t.Fatalf("dialog has no cancel; texts=%q", tt.Texts())
	}
	cb, ok := tt.Find("我已了解风险，并愿意继续")
	if !ok {
		t.Fatalf("dialog has no checkbox; texts=%q", tt.Texts())
	}
	cx, cy := cb.X+cb.W/2, cb.Y+cb.H/2
	tt.Press(cx, cy)
	beforeRelease := frames
	tt.Release(cx, cy)
	if !acknowledged {
		t.Fatal("clicking the checkbox did not acknowledge the risk")
	}
	if ackFrame != beforeRelease+1 {
		t.Fatalf("acknowledgement lagged to frame %d after release (want click frame %d)", ackFrame, beforeRelease+1)
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
