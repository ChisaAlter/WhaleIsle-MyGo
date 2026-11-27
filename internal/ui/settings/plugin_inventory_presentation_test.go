package settings

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func TestPluginInventoryCardsReflowAndExpandTheirOwnEntry(t *testing.T) {
	rows := []plugininventory.DisplayRow{
		{Row: plugininventory.Row{Key: "preset-a/one", EntryID: "loader-one", Module: "module-one", Enabled: "enabled", Phase: "active"}, Title: "First entry", Description: "First description"},
		{Row: plugininventory.Row{Key: "preset-a/two", EntryID: "loader-two", Module: "module-two", Enabled: "conditional", Phase: "failed", Condition: "fixture-condition"}, Title: "Second entry", Description: "Second description"},
	}
	a := &PluginInventoryAccess{}
	a.OnToggleRow = func(key string) {
		if a.Snapshot.Expanded == key {
			a.Snapshot.Expanded = ""
		} else {
			a.Snapshot.Expanded = key
		}
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { pluginInventoryCards(c, a, rows, c.Theme().Text, c.Theme().TextMuted) })
	}, 760, 1000)
	first, ok1 := tt.Find("First entry")
	second, ok2 := tt.Find("Second entry")
	if !ok1 || !ok2 || first.Y != second.Y || second.X <= first.X {
		t.Fatalf("inventory cards were not in two columns: %v %v", first, second)
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-wide.png")
	if err := tt.Click("Second entry"); err != nil {
		t.Fatal(err)
	}
	if a.Snapshot.Expanded != rows[1].Key || !tt.HasText("fixture-condition") || tt.HasText("Module: module-one") {
		t.Fatal("expansion did not preserve the selected Loader identity")
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-expanded.png")
	tt.SetSize(420, 1000)
	first, ok1 = tt.Find("First entry")
	second, ok2 = tt.Find("Second entry")
	if !ok1 || !ok2 || second.Y <= first.Y || first.X != second.X || second.X+second.W > 420 {
		t.Fatalf("narrow inventory did not stack within viewport: %v %v", first, second)
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-narrow.png")
}

// TestPluginInventoryTwoColumnCardsEqualHeightAndToolScheduleLocalized pins
// two-column uniform card heights across uneven 1-line vs 2-line descriptions
// and Chinese localization for the tool-schedule built-in entry.
func TestPluginInventoryTwoColumnCardsEqualHeightAndToolScheduleLocalized(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	rows := []plugininventory.DisplayRow{
		{
			Row:         plugininventory.Row{Key: "p/instructions", EntryID: "include:ctx-instructions", Module: "ctx-instructions", Enabled: "enabled", Phase: "active"},
			Title:       "工作区指令加载",
			Description: "将工作区级和项目级的指令文件（如 AGENTS.md、CLAUDE.md、MEMORY.md）自动注入到每轮对话上下文中。",
		},
		{
			Row:         plugininventory.Row{Key: "p/persona", EntryID: "include:ctx-persona", Module: "ctx-persona", Enabled: "enabled", Phase: "active"},
			Title:       "智能体角色设定",
			Description: "为智能体注入基础角色与行为规范。",
		},
		{
			Row:         plugininventory.Row{Key: "p/schedule", EntryID: "include:tool-schedule", Module: "tool-schedule", Enabled: "enabled", Phase: "active"},
			Title:       "tool-schedule",
			Description: "Model-facing reminder management tools",
		},
		{
			Row:         plugininventory.Row{Key: "p/files", EntryID: "include:tool-files", Module: "tool-files", Enabled: "enabled", Phase: "active"},
			Title:       "文件读写工具",
			Description: "提供工作区内文件读取、编辑与搜索能力。",
		},
	}
	a := &PluginInventoryAccess{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Children(func() {
				pluginInventoryCards(c, a, rows, c.Theme().Text, c.Theme().TextMuted)
			})
		})
	}, 760, 800)
	tt.Frame()

	// tool-schedule must be localized in Chinese and not show the raw English description.
	if !tt.HasText("定时提醒工具") {
		t.Fatalf("tool-schedule missing Chinese title: %v", tt.Texts())
	}
	if tt.HasText("Model-facing reminder management tools") {
		t.Fatalf("tool-schedule leaked raw English description in Chinese locale: %v", tt.Texts())
	}

	// All 4 cards in the 2-column grid must have identical outer height (< 0.5 DIP diff),
	// and cards in the same row must share top and bottom edges.
	cardRect := func(label string) ui.Rect {
		t.Helper()
		var best ui.Rect
		for _, r := range tt.FindAll(label) {
			if r.W*r.H > best.W*best.H {
				best = r
			}
		}
		if best.W == 0 {
			t.Fatalf("card %q not found: %v", label, tt.Texts())
		}
		return best
	}
	c0 := cardRect("工作区指令加载")
	c1 := cardRect("智能体角色设定")
	c2 := cardRect("定时提醒工具")
	c3 := cardRect("文件读写工具")
	cards := []ui.Rect{c0, c1, c2, c3}
	minH, maxH := cards[0].H, cards[0].H
	for _, r := range cards[1:] {
		if r.H < minH {
			minH = r.H
		}
		if r.H > maxH {
			maxH = r.H
		}
	}
	if maxH-minH >= 0.5 {
		t.Fatalf("built-in plugin cards do not share uniform height (diff=%.2f): c0=%+v c1=%+v c2=%+v c3=%+v", maxH-minH, c0, c1, c2, c3)
	}
	if math.Abs(float64(c0.Y-c1.Y)) >= 0.5 || math.Abs(float64((c0.Y+c0.H)-(c1.Y+c1.H))) >= 0.5 {
		t.Fatalf("row 1 top/bottom misaligned: c0=%+v c1=%+v", c0, c1)
	}
	if math.Abs(float64(c2.Y-c3.Y)) >= 0.5 || math.Abs(float64((c2.Y+c2.H)-(c3.Y+c3.H))) >= 0.5 {
		t.Fatalf("row 2 top/bottom misaligned: c2=%+v c3=%+v", c2, c3)
	}
}

func saveInventorySnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
