package composer

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// Provider-anchored occupancy and heuristic composition are deliberately separate.
func paintContextMeter(c *ui.Context, snapshot session.ViewState, muted ui.Color, height float32) {
	root := ui.Column(c.Key("context-meter")).Gap(4)
	state := ui.Local(root, "context-panel", func() contextMeterPanel { return contextMeterPanel{} })
	binding := fmt.Sprintf("%d:%s:%s", snapshot.ComposerHostGeneration, snapshot.PlanHistoryBinding, snapshot.SelectedID)
	if state.Binding != binding || snapshot.ContextOccupancy == nil {
		state.Open = false
		state.Binding = binding
	}
	value := snapshot.ContextOccupancy
	if value == nil {
		return
	}
	root.Children(func() {
		trigger := ui.Button(c, fmt.Sprintf("上下文 %d%%", value.Percent)).Label("Context occupancy").Height(height).FontSize(12).TextColor(muted)
		if trigger.Clicked() {
			state.Open = !state.Open
		}
		ui.Popover(c, trigger, &state.Open, func() {
			ui.Column(c).Width(260).Gap(6).Padding(8).Children(func() {
				ui.Text(c, fmt.Sprintf("上下文已用 %d%%", value.Percent)).FontSize(13)
				ui.Text(c, fmt.Sprintf("~%g / %g tokens", value.UsedTokens, value.ContextWindow)).FontSize(12).Selectable()
				// The bar length follows pressure; estimates only divide that length.
				ui.Row(c).Width(240).Height(6).Background(ui.RGB(210, 210, 210)).Children(func() {
					weights := []float64{1}
					if b := value.Breakdown; b != nil {
						total := b.SystemTokens + b.ToolsTokens + b.MessageTokens
						if total > 0 {
							weights = []float64{b.SystemTokens / total, b.ToolsTokens / total, b.MessageTokens / total}
						}
					}
					colors := []ui.Color{ui.RGB(99, 102, 241), ui.RGB(14, 165, 233), ui.RGB(16, 185, 129)}
					for i, w := range weights {
						if w > 0 && value.Percent > 0 {
							ui.Box(c).Width(float32(240 * float64(value.Percent) / 100 * w)).Height(6).Background(colors[i])
						}
					}
				})
				if b := value.Breakdown; b != nil {
					ui.Text(c, "估算组成 / Estimated composition").FontSize(11).TextColor(muted)
					for _, row := range []struct {
						name   string
						tokens float64
					}{{"系统提示 / System", b.SystemTokens}, {"工具 / Tools", b.ToolsTokens}, {"会话 / Messages", b.MessageTokens}} {
						ui.Text(c, fmt.Sprintf("%s: ~%g", row.name, row.tokens)).FontSize(12).Selectable()
					}
				}
			})
		})
	})
}

type contextMeterPanel struct {
	Binding string
	Open    bool
}
