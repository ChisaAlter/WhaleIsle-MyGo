package terminal

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func paneStripPanel(active uint64) *Panel {
	return &Panel{
		cwd:    "workspace",
		panes:  []*terminalPane{{id: 1}, {id: 2}},
		groups: []*terminalGroup{{id: 1, panes: []uint64{1, 2}}},
		active: active,
	}
}

func renderPaneStrip(t *testing.T, p *Panel) *ui.Tester {
	t.Helper()
	m := Model{Panel: p}
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			terminalTabs(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 140)
}

// TestPaneStripReplacesTheSidebar keeps the panes in one horizontal strip: each
// pane is a chip named by its shell, and the old group sidebar is gone.
func TestPaneStripReplacesTheSidebar(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	p := paneStripPanel(2)
	tt := renderPaneStrip(t, p)
	text := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(text, "Terminal 1") || !strings.Contains(text, "Terminal 2") {
		t.Fatalf("pane chips missing: %s", text)
	}
	if strings.Contains(text, "GROUP") || strings.Contains(text, "组 1") {
		t.Fatalf("group sidebar label survived the strip: %s", text)
	}
}

// TestPaneStripActivatesOnClick keeps the chip a real tab: clicking an inactive
// chip focuses that pane.
func TestPaneStripActivatesOnClick(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	p := paneStripPanel(2)
	tt := renderPaneStrip(t, p)
	if err := tt.Click("Terminal 1"); err != nil {
		t.Fatalf("inactive chip is not reachable: %v", err)
	}
	if p.active != 1 {
		t.Fatalf("chip click did not activate the pane: %d", p.active)
	}
}
