package settings

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// Like the voice entry, the rail only offers the page when the app wired it.
func TestDeepSeekAPIsSectionAppearsInFeaturesOnlyWhenWired(t *testing.T) {
	wired := Model{Features: &FeaturesAccess{}, OnSelectDeepSeekAPIs: func() {}}
	if !railHas(wired, SectionDeepSeekAPIs) {
		t.Fatal("DeepSeek entry missing from the rail while OnSelectDeepSeekAPIs is wired")
	}
	if railHas(Model{Features: &FeaturesAccess{}}, SectionDeepSeekAPIs) {
		t.Fatal("DeepSeek entry rendered without a handler")
	}
}

func TestEmbeddedDeepSeekAPIsSectionKeepsTheRailOnIt(t *testing.T) {
	m := Model{
		Features:             &FeaturesAccess{},
		OnSelectDeepSeekAPIs: func() {},
		ActiveSection:        SectionDeepSeekAPIs,
		SectionContent:       func(*ui.Context) {},
	}
	if got := activeSection(m, store.ViewState{}); got != SectionDeepSeekAPIs {
		t.Fatalf("activeSection = %q, want %q", got, SectionDeepSeekAPIs)
	}
	if got, want := sectionLabel(SectionDeepSeekAPIs), i18n.T(i18n.KeySettingsSectionDeepSeek); got != want {
		t.Fatalf("sectionLabel = %q, want %q", got, want)
	}
}

func deepSeekAPIsFixture() *DeepSeekToolsAccess {
	return &DeepSeekToolsAccess{
		Snapshot: store.DeepSeekToolsView{
			Connected: true, Present: true, Writable: true,
			BaseURL: "https://api.deepseek.com/v1/tools", APIKeyEnv: "DEEPSEEK_API_KEY", Voice: "stella",
			FilterMarkdown: true,
			Drafts:         map[string]string{}, DirtyFields: map[string]bool{},
		},
		Key: store.DeepSeekKeyView{Ref: "DEEPSEEK_API_KEY", Writable: true},
	}
}

// Billing is on the page before any switch, and every row is written copy.
func TestDeepSeekAPIsPaneStatesBillingAndTools(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	a := deepSeekAPIsFixture()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			DeepSeekToolsPane(c, a, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 720, 1200)
	for _, want := range []string{"10,000 计费字符", "语音合成（TTS）", "音频转文字（ASR）", "文档识别（OCR）", "默认音色", "朗读前移除 Markdown 格式", "API 基础地址", "凭据引用", "API Key"} {
		if !tt.HasText(want) {
			t.Fatalf("page is missing %q", want)
		}
	}
}

func TestDeepSeekAPIsToggleWritesImmediately(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	a := deepSeekAPIsFixture()
	type staged struct {
		field, value string
		b            bool
	}
	var toggles []staged
	a.OnToggle = func(field string, value bool) { toggles = append(toggles, staged{field: field, b: value}) }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			DeepSeekToolsPane(c, a, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 720, 1200)
	row, ok := tt.Find("语音合成（TTS）")
	if !ok {
		t.Fatal("the TTS row did not paint")
	}
	tt.ClickAt(switchRight(720), row.Y+row.H/2)
	if len(toggles) != 1 || toggles[0].field != "tts" || !toggles[0].b {
		t.Fatalf("toggle did not write the namespace: %+v", toggles)
	}
}

// A Host that did not mount the plugin has no namespace: the page says so and
// offers the next step instead of showing an empty form.
func TestDeepSeekAPIsPaneReportsAnUnmountedPlugin(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	reloads := 0
	a := deepSeekAPIsFixture()
	a.Snapshot.Present = false
	a.OnRefresh = func() { reloads++ }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			DeepSeekToolsPane(c, a, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 720, 1200)
	if tt.HasText("语音合成（TTS）") {
		t.Fatal("an unmounted plugin must not paint an editable form")
	}
	if !tt.HasText("没有加载 DeepSeek 应用接口插件") {
		t.Fatal("the page did not explain the missing plugin")
	}
	if err := tt.Click("重新加载"); err != nil {
		t.Fatal(err)
	}
	if reloads != 1 {
		t.Fatalf("reload = %d, want 1", reloads)
	}
}

// The transport's own words never reach the page.
func TestDeepSeekAPIsPaneKeepsRawErrorsOffThePage(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	raw := "dial tcp 127.0.0.1:5199: connectex: No connection could be made"
	a := deepSeekAPIsFixture()
	a.Snapshot.Err = raw
	a.Key.Err = raw
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			DeepSeekToolsPane(c, a, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 720, 1200)
	rendered := strings.Join(tt.Texts(), " | ")
	if strings.Contains(rendered, "dial tcp") || strings.Contains(rendered, "connectex") {
		t.Fatalf("raw transport text reached the page: %s", rendered)
	}
	if !tt.HasText("刚才的更改没有生效") {
		t.Fatal("the page did not state that the change failed")
	}
}

// switchRight is where the row switch sits: 16 DIP page padding plus half the
// 44 DIP track.
func switchRight(width float32) float32 { return width - 16 - 22 }

// A render of the page for visual review: the same view tree the shell paints,
// captured only when a directory is named.
func TestDeepSeekAPIsPaneRendersASnapshot(t *testing.T) {
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("SECONDARY_SNAPSHOT_DIR not set")
	}
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*DeepSeekToolsAccess){
		"deepseek-apis.png": func(*DeepSeekToolsAccess) {},
		"deepseek-apis-off.png": func(a *DeepSeekToolsAccess) {
			a.Snapshot.Present = false
		},
	} {
		a := deepSeekAPIsFixture()
		mutate(a)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
				DeepSeekToolsPane(c, a, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 720, 1000)
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
