package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestContextMeterDisclosureAndInvalidation(t *testing.T) {
	snap := session.ViewState{SelectedID: "s", ContextOccupancy: &session.ContextOccupancy{UsedTokens: 250, ContextWindow: 1000, Percent: 25, Breakdown: &session.ContextBreakdown{SystemTokens: 20, ToolsTokens: 30, MessageTokens: 50}}}
	tt := ui.NewTester(func(c *ui.Context) { paintContextMeter(c, snap, ui.RGB(80, 80, 80), 24) }, 800, 600)
	if err := tt.Click("Context occupancy"); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"~250 / 1000 tokens", "系统提示 / System: ~20", "工具 / Tools: ~30", "会话 / Messages: ~50"} {
		if !tt.HasText(text) {
			t.Fatalf("missing %s", text)
		}
	}
	saved := snap.ContextOccupancy
	snap.ContextOccupancy = nil
	tt.Frame()
	if _, ok := tt.Find("Context occupancy"); ok {
		t.Fatal("unavailable meter visible")
	}
	snap.ContextOccupancy = saved
	tt.Frame()
	if tt.HasText("~250 / 1000 tokens") {
		t.Fatal("stale panel reopened")
	}
	tt.Click("Context occupancy")
	snap.SelectedID = "other"
	tt.Frame()
	if tt.HasText("~250 / 1000 tokens") {
		t.Fatal("panel crossed session")
	}
}
