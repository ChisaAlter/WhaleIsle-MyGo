package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/whale"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// whaleFixture is the catalog the Whale page renders: one identity, no persona
// or model route, two skills, and a memory note.
func whaleFixture() *whale.Catalog {
	return &whale.Catalog{
		Name:               "鲸鱼娘",
		UserTitle:          "主人",
		PersonaText:        "",
		ModelProvider:      "",
		ModelModel:         "",
		IMDefault:          true,
		Personality:        "natural",
		PersonalityOptions: []string{"natural", "genki", "tsundere", "poison"},
		Skills:             []whale.Skill{{Name: "diary", Enabled: true}, {Name: "art", Enabled: false}},
		Memory:             "# 鲸鱼娘的长期记忆\n\n（还没有记住什么。）\n",
		HomeDir:            "/home/fixture/.dsh/data/whale",
	}
}

type whaleRecorder struct {
	patches  []whale.Patch
	memory   []string
	clears   int
	toggles  []string
	loads    int
}

func (r *whaleRecorder) access(catalog *whale.Catalog) *PetSettingsAccess {
	return &PetSettingsAccess{
		AssistantAvailable: true,
		Whale:              catalog,
		OnLoadWhale:        func() { r.loads++ },
		OnUpdateWhale:      func(p whale.Patch) { r.patches = append(r.patches, p) },
		OnReplaceMemory:    func(text string) { r.memory = append(r.memory, text) },
		OnClearMemory:      func() { r.clears++ },
		OnToggleSkill:      func(name string, enabled bool) { r.toggles = append(r.toggles, name) },
	}
}

func whalePetTester(t *testing.T, a *PetSettingsAccess, w, h int) *ui.Tester {
	t.Helper()
	companion := &pet.Companion{Enabled: true, Behavior: config.DefaultPetBehavior()}
	colors := theme.Resolve("").Active(false)
	return ui.NewTester(func(c *ui.Context) {
		petPane(c, colors, companion, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), a)
	}, w, h)
}

func TestWhalePetPageCommitsIdentityOnBlur(t *testing.T) {
	rec := &whaleRecorder{}
	tt := whalePetTester(t, rec.access(whaleFixture()), 900, 900)
	name, ok := tt.Find(i18n.T(i18n.KeySettingsWhaleName))
	if !ok {
		t.Fatal("name field missing")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySettingsWhaleTitle)); !ok {
		t.Fatal("title field missing")
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleName)); err != nil {
		t.Fatal(err)
	}
	tt.Type("小鲸鱼")
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleTitle)); err != nil {
		t.Fatal(err)
	}
	if len(rec.patches) != 1 || rec.patches[0].Name == nil || *rec.patches[0].Name != "鲸鱼娘小鲸鱼" {
		t.Fatalf("blur did not commit the typed name: %+v (name=%v)", rec.patches, name)
	}
	if rec.patches[0].UserTitle != nil {
		t.Fatal("identity commit sent the untouched title")
	}
}

func TestWhalePetPageTogglesIMAndSummarisesTheCatalog(t *testing.T) {
	rec := &whaleRecorder{}
	tt := whalePetTester(t, rec.access(whaleFixture()), 900, 900)
	if !tt.HasText(i18n.T(i18n.KeySettingsWhaleFollowDefault)) {
		t.Fatalf("empty model route did not read as follow-session: %v", tt.Texts())
	}
	if !tt.HasText("1/2") {
		t.Fatalf("skills summary missing: %v", tt.Texts())
	}
	if err := tt.Click("Whale IM toggle"); err != nil {
		t.Fatalf("IM row is not clickable: %v", err)
	}
	if len(rec.patches) != 1 || rec.patches[0].IMDefault == nil || *rec.patches[0].IMDefault {
		t.Fatalf("IM toggle patch = %+v", rec.patches)
	}
}

func TestWhalePetMemoryDialogSavesAndClears(t *testing.T) {
	rec := &whaleRecorder{}
	tt := whalePetTester(t, rec.access(whaleFixture()), 900, 900)
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleMemoryOpen)); err != nil {
		t.Fatalf("memory action missing: %v", err)
	}
	if !tt.HasText("（还没有记住什么。）") {
		t.Fatalf("memory dialog did not load MEMORY.md: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsSave)); err != nil {
		t.Fatalf("memory save missing: %v", err)
	}
	if len(rec.memory) != 1 {
		t.Fatalf("memory save calls = %d", len(rec.memory))
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleMemoryOpen)); err != nil {
		t.Fatalf("memory action missing on reopen: %v", err)
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleMemoryClear)); err != nil {
		t.Fatalf("clear action missing: %v", err)
	}
	if rec.clears != 0 {
		t.Fatal("clear ran without confirmation")
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleMemoryClearConfirmAct)); err != nil {
		t.Fatalf("confirmation action missing: %v", err)
	}
	if rec.clears != 1 {
		t.Fatalf("clear calls = %d", rec.clears)
	}
}

func TestWhalePetSkillsToggleInline(t *testing.T) {
	rec := &whaleRecorder{}
	tt := whalePetTester(t, rec.access(whaleFixture()), 900, 900)
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleManage)); err != nil {
		t.Fatalf("skills action missing: %v", err)
	}
	if !tt.HasText("diary") || !tt.HasText("art") {
		t.Fatalf("skill rows missing: %v", tt.Texts())
	}
	if err := tt.Click("Whale skill diary"); err != nil {
		t.Fatalf("skill switch missing: %v", err)
	}
	if len(rec.toggles) != 1 || rec.toggles[0] != "diary" {
		t.Fatalf("skill toggles = %v", rec.toggles)
	}
}

func TestWhalePetPageShowsPlaceholderWithoutCatalog(t *testing.T) {
	rec := &whaleRecorder{}
	tt := whalePetTester(t, rec.access(nil), 900, 900)
	if !tt.HasText(i18n.T(i18n.KeySettingsWhaleUnavailable)) {
		t.Fatalf("missing placeholder: %v", tt.Texts())
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySettingsWhaleName)); ok {
		t.Fatal("placeholder page rendered identity controls")
	}
}
