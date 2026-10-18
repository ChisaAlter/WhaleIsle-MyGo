package agents

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestAgentMetricUnknown(t *testing.T) {
	row := session.AgentNavigationRow{Address: session.AgentSessionAddress{Mode: "continuable"}}
	text, _ := agentMetricLabel(row, 6000)
	if strings.Contains(text, "0") || strings.Contains(text, "Completed") || strings.Contains(text, "已完成") {
		t.Fatal("unknown invented metric", text)
	}
	row.ActivityKnown = true
	row.Metrics.Completed = true
	tokens := 1001.0
	row.Metrics.Tokens = &tokens
	row.Metrics.Timing = &session.AgentTiming{SettledMs: 65000}
	text, tip := agentMetricLabel(row, 6000)
	if !strings.Contains(text, "1.0K tokens") || !strings.Contains(tip, "1001 tokens") || !strings.Contains(tip, "65 s active") {
		t.Fatal(text, tip)
	}
}
func TestCatalogScopedKeyboard(t *testing.T) {
	selected := -1
	backs := 0
	focus := -1
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			var buttons []ui.Element
			for i, label := range []string{"First", "Second", "Third"} {
				b := ui.Button(c, label)
				if i == 0 {
					b.AutoFocus()
				}
				buttons = append(buttons, b)
				if b.Focused() {
					focus = i
				}
			}
			catalogKeyboard(buttons, func(i int) { selected = i }, func() { backs++ })
		})
	}, 300, 300)
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyRight)
	if selected != 2 {
		t.Fatal("End/Right did not open last", selected, focus)
	}
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyRight)
	if selected != 1 {
		t.Fatal("Home/Down did not open next", selected, focus)
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyLeft)
	if selected != 0 || backs != 1 {
		t.Fatal("Up/Left navigation failed", selected, backs)
	}
}
