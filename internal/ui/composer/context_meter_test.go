package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestContextMeterDisclosureAndInvalidation(t *testing.T) {
	snap := session.ViewState{SelectedID: "s", ContextOccupancy: &session.ContextOccupancy{UsedTokens: 250, ContextWindow: 1000, Percent: 25, Breakdown: &session.ContextBreakdown{SystemTokens: 20, ToolsTokens: 30, MessageTokens: 50}}}
	tt := ui.NewTester(func(c *ui.Context) { paintContextMeter(c, snap, testColors(), 24) }, 800, 600)
	if _, ok := tt.Find("25%"); !ok {
		t.Fatal("meter does not show its reading")
	}
	if err := tt.Click("Context occupancy"); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"~250 / 1K", "系统提示词", "工具定义", "对话消息"} {
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
	if tt.HasText("—%") {
		t.Fatal("unavailable meter painted a placeholder")
	}
	if _, ok := tt.Find("context-meter-placeholder"); ok {
		t.Fatal("unavailable meter kept its seat")
	}
	snap.ContextOccupancy = saved
	tt.Frame()
	if tt.HasText("~250 / 1K") {
		t.Fatal("stale panel reopened")
	}
	tt.Click("Context occupancy")
	snap.SelectedID = "other"
	tt.Frame()
	if tt.HasText("~250 / 1K") {
		t.Fatal("panel crossed session")
	}
}
