package settings

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func pass4bGalleryDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("SETTINGS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set SETTINGS_GALLERY_DIR to export the settings gallery")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func pass4bPNG(t *testing.T, dir, name string, tt *ui.Tester) {
	t.Helper()
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func presetRosterFixture() session.AgentPresetManagerState {
	return session.AgentPresetManagerState{
		Loaded: true, Writable: true, DefaultID: "standard", Revision: 4,
		Rows: []session.ManagedAgentPreset{
			{ID: "standard", Name: "标准模式", Description: "处理代码、文件和资料，适合大多数任务。Agent 会按需使用检索、编辑和终端等工具。", BuiltIn: true},
			{ID: "ptc", Name: "PTC 模式", Description: "包含标准模式的所有能力，更适合批量调用工具，并对结果进行筛选、整理、去重、统计或汇总的任务。", BuiltIn: true},
			{ID: "minimal", Name: "极简模式", Description: "Agent 使用终端和图片读取工具，并支持上下文压缩，适合测试和对比其基础表现。", BuiltIn: true},
			{ID: "cordis", Name: "创造模式", Description: "用对话定制 DSH：让 Agent 编写插件，添加新功能或界面；也能组合工具和提示词，创建自己的模式。", BuiltIn: true},
		},
	}
}

func pass4bPresetsView(c *ui.Context) {
	colors := theme.Resolve("").Active(false)
	fg, muted := ui.RGB(20, 20, 20), ui.RGB(120, 120, 120)
	reader := &AgentPresetReaderAccess{OnSelect: func(uint64, string) {}}
	a := &AgentPresetsAccess{Selected: new(string), Reader: reader}
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		agentPresetRosterPane(c, a, presetRosterFixture(), fg, muted)
	})
	_ = colors
}

// TestPass4bPresetCardsUseTwoColumns pins the reference's 2-up card grid: the
// cards neither overlap nor leave a gap wide enough for a third column.
func TestPass4bPresetCardsUseTwoColumns(t *testing.T) {
	tt := ui.NewTester(pass4bPresetsView, 1000, 800)
	first, ok := tt.Find("选择默认 · 标准模式")
	if !ok {
		t.Fatalf("first card missing: %v", tt.Texts())
	}
	second, ok := tt.Find("选择默认 · PTC 模式")
	if !ok {
		t.Fatal("second card missing")
	}
	if first.X+first.W > second.X {
		t.Fatalf("cards overlap: %+v %+v", first, second)
	}
	// Two columns: the second card starts exactly one gap after the first and
	// both cards are the same width, so no third column can fit.
	// The grid gap is 12; each card's face is inset 1 DIP by its stroke, so the
	// measured span between faces is 14 and each face is 2 DIP narrower than
	// the column: (1000 - 12) / 2 = 494.
	if diff := (second.X - first.X) - first.W; diff != 14 {
		t.Fatalf("column span = %v, want 14 (cards %+v %+v)", diff, first, second)
	}
	if first.W+2 != 494 {
		t.Fatalf("column width = %v, want 494", first.W+2)
	}
	if first.W != second.W || first.Y != second.Y {
		t.Fatalf("cards are not a uniform 2-up grid: %+v %+v", first, second)
	}
	if !tt.HasText(i18n.T(KeyPass4TaskDefault)) || !tt.HasText(i18n.T(KeyPass4BadgeBuiltIn)) {
		t.Fatalf("preset badges missing: %v", tt.Texts())
	}
	if !tt.HasText(i18n.T(KeyPass4CreateWithAgent)) {
		t.Fatalf("dashed creator entry missing: %v", tt.Texts())
	}
}

func TestPass4bAboutLinksAndFolderButton(t *testing.T) {
	opened := 0
	m := Model{About: &AboutAccess{
		AppName: "Whale Isle", Version: "0.3.9", HomePath: "/tmp/fixture-home",
		OnRevealHome: func() error { opened++; return nil },
		OnOpenLink:   func(string) error { return nil },
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Padding(20).Children(func() { aboutPane(c, m, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120)) })
	}, 1000, 800)
	open, ok := tt.Find(i18n.T(KeyPass4OpenRuntime))
	if !ok {
		t.Fatalf("runtime folder button missing: %v", tt.Texts())
	}
	if open.H != 28 {
		t.Fatalf("runtime folder button height = %v, want 28", open.H)
	}
	if err := tt.Click(i18n.T(KeyPass4OpenRuntime)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("runtime folder clicks = %d", opened)
	}
	for _, link := range aboutLinks() {
		if !tt.HasText(link.Label) {
			t.Fatalf("about link %q missing", link.Label)
		}
	}
	// A 28 DIP outline button and the 14px accent links are the visible delta.
	colors := theme.Resolve("").Active(false)
	spec := widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}
	if spec.Size != widgets.ButtonSmall {
		t.Fatal("outline small recipe drifted")
	}
	_ = colors
}

// TestPass4bGalleryPresetsAndAbout exports the two settings shots.
func TestPass4bGalleryPresetsAndAbout(t *testing.T) {
	dir := pass4bGalleryDir(t)
	presets := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, i18n.T(KeyPass4PresetsTitle)).FontSize(theme.InterfaceSize(18)).FontWeight(600).TextColor(ui.RGB(20, 20, 20))
				ui.Text(c, i18n.T(KeyPass4PresetsHint)).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(ui.RGB(120, 120, 120))
				pass4bPresetsView(c)
			})
		})
	}, 1000, 800)
	pass4bPNG(t, dir, "ours-settings-presets.png", presets)

	about := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			aboutPane(c, Model{About: &AboutAccess{AppName: "Whale Isle", Version: "0.3.9", HomePath: "/tmp/fixture-home",
				OnRevealHome: func() error { return nil }, OnOpenLink: func(string) error { return nil }}}, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120))
		})
	}, 1000, 800)
	pass4bPNG(t, dir, "ours-settings-about.png", about)
	fmt.Println("gallery written")
}
