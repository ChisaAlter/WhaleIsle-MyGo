package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSidebarCustomPluginEntriesExpandedAndCollapsed(t *testing.T) {
	var opened []string
	m := Model{
		Collapsed: false,
		OnOpenMarket: func() {
			opened = append(opened, "market")
		},
		OnOpenSchedule: func() {
			opened = append(opened, "schedule")
		},
		OnOpenPlugin: func(id string) {
			opened = append(opened, id)
		},
		PluginEntries: []PluginEntry{
			{ID: "qa-panel", Label: "QA Panel", Order: 60},
		},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 320, 720)
	if _, ok := tt.Find("QA Panel"); !ok {
		t.Fatal("expected custom plugin button in expanded sidebar")
	}
	if err := tt.Click("QA Panel"); err != nil {
		t.Fatalf("click expanded custom plugin: %v", err)
	}
	if len(opened) != 1 || opened[0] != "qa-panel" {
		t.Fatalf("expected OnOpenPlugin(\"qa-panel\"), got %v", opened)
	}

	opened = nil
	m.Collapsed = true
	ttCollapsed := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 96, 720)
	if err := ttCollapsed.Click("QA Panel"); err != nil {
		t.Fatalf("click collapsed custom plugin rail button: %v", err)
	}
	if len(opened) != 1 || opened[0] != "qa-panel" {
		t.Fatalf("expected collapsed rail OnOpenPlugin(\"qa-panel\"), got %v", opened)
	}

	if got := sidebarPanelRows(Model{}); len(got) != 2 {
		t.Fatalf("empty PluginEntries must keep 2 default rows, got %d", len(got))
	}
}
