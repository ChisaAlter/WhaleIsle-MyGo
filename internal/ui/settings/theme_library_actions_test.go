package settings

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func libraryActionsFixture(t *testing.T, width int, writable bool) (*theme.Editor, *ui.Tester, *int) {
	t.Helper()
	old, locale := theme.CurrentPreferences(), i18n.Locale()
	t.Cleanup(func() { theme.SetPreferences(old); i18n.SetLocale(locale) })
	i18n.SetLocale("en")
	prefs := theme.Defaults()
	custom := prefs.DuplicateFamily(prefs.Family("celadon"))
	custom.ID, custom.Name = "custom-c", "Custom C"
	prefs.Custom = []theme.Family{custom}
	writes := new(int)
	var save func(theme.Preferences) error
	if writable {
		save = func(theme.Preferences) error { *writes++; return nil }
	}
	e := theme.NewEditor(prefs, save)
	tt := ui.NewTester(func(c *ui.Context) {
		colors := theme.ResolveWithSystem("system", c.Theme().Dark).Active(c.Theme().Dark)
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			themeLibraryPane(c, e, colors, c.Theme().Text, c.Theme().TextMuted)
		})
	}, width, 3000)
	return e, tt, writes
}

func TestThemeLibraryHalvesKeepIndependentSelectionAndResolvedMode(t *testing.T) {
	e, tt, _ := libraryActionsFixture(t, 760, true)
	if err := tt.Click(e.Draft.Family("midnight").Name + " dark"); err != nil {
		t.Fatal(err)
	}
	if e.Saved.Light != "deepseek" || e.Saved.Dark != "midnight" {
		t.Fatalf("halves coupled: %+v", e.Saved)
	}
	checkCount := 0
	for _, text := range tt.Texts() {
		if text == "✓" {
			checkCount++
		}
	}
	if checkCount != 2 {
		t.Fatalf("want independently selected half checks, got %d", checkCount)
	}
	if _, ok := tt.Find("Theme current mode DeepSeek light"); !ok {
		t.Fatalf("missing separate current-light pill: %v", tt.Texts())
	}
	if _, ok := tt.Find("Theme current mode DeepSeek dark"); ok {
		t.Fatal("light frame tagged dark as current")
	}
	tt.SetDark(true)
	if _, ok := tt.Find("Theme current mode DeepSeek dark"); !ok {
		t.Fatalf("missing separate current-dark pill: %v", tt.Texts())
	}
	if _, ok := tt.Find("Theme current mode DeepSeek light"); ok {
		t.Fatal("dark frame tagged light as current")
	}
	if e.Saved.Light != "deepseek" || e.Saved.Dark != "midnight" {
		t.Fatal("switching resolved mode changed selected families")
	}
	// Selected() must retain the preview seed instead of painting host accent.
	r, ok := tt.Find("DeepSeek light")
	if !ok {
		t.Fatal("missing light half")
	}
	// Read inside the seed, clear of the newly added 2px selected ring.
	pixel := tt.Image().RGBAAt(int(r.X+6), int(r.Y+r.H/2))
	seed := chrome.Color(e.Draft.Family("deepseek").Light.Background)
	if pixel.R != seed.R || pixel.G != seed.G || pixel.B != seed.B {
		t.Fatalf("selected half lost seed background: %v", pixel)
	}
}

func TestThemeLibraryDuplicateUsesAddressedFamilyAndCancelRestores(t *testing.T) {
	e, tt, writes := libraryActionsFixture(t, 760, true)
	before := e.Saved.Clone()
	source := e.Draft.Family("midnight")
	if err := tt.Click("Duplicate theme " + source.Name); err != nil {
		t.Fatal(err)
	}
	copy := e.Draft.Family(e.Selected)
	if copy.ID == source.ID || !reflect.DeepEqual(copy.Light, source.Light) || !reflect.DeepEqual(copy.Dark, source.Dark) {
		t.Fatalf("copied active family instead of addressed family: %+v", copy)
	}
	if *writes != 0 || !reflect.DeepEqual(e.Saved, before) || e.Draft.Light != before.Light || e.Draft.Dark != before.Dark {
		t.Fatal("duplicate footer activated/persisted source")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if e.Selected != "" || !reflect.DeepEqual(e.Draft, before) || !reflect.DeepEqual(theme.CurrentPreferences(), before) {
		t.Fatal("cancel did not restore saved selection and remove duplicate")
	}
}

func TestThemeLibraryCustomActionsAddressFamilyWithoutPickingHalf(t *testing.T) {
	e, tt, writes := libraryActionsFixture(t, 760, true)
	before := e.Saved.Clone()
	family := e.Draft.Family("custom-c")
	if err := tt.Click("Export theme " + family.Name); err != nil {
		t.Fatal(err)
	}
	var exported theme.Family
	if err := json.Unmarshal([]byte(tt.Clipboard()), &exported); err != nil {
		t.Fatal(err)
	}
	if exported.ID != family.ID || exported.Name != family.Name || !reflect.DeepEqual(exported.Light, family.Light) {
		t.Fatalf("wrong export: %+v", exported)
	}
	if *writes != 0 || e.Selected != "" || !reflect.DeepEqual(e.Draft, before) {
		t.Fatal("export selected or wrote theme")
	}
	if err := tt.Click("Edit theme " + family.Name); err != nil {
		t.Fatal(err)
	}
	if e.Selected != family.ID || *writes != 0 || e.Saved.Light != before.Light || e.Saved.Dark != before.Dark {
		t.Fatal("edit changed saved half selection")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Delete theme " + family.Name); err != nil {
		t.Fatal(err)
	}
	if len(e.Saved.Custom) != 0 || *writes != 1 || e.Saved.Light != before.Light || e.Saved.Dark != before.Dark {
		t.Fatalf("delete selected wrong source: %+v writes=%d", e.Saved, *writes)
	}
}

func TestThemeLibraryNarrowActionsAndReadOnlyExport(t *testing.T) {
	e, tt, writes := libraryActionsFixture(t, 390, false)
	for _, f := range e.Draft.Families() {
		labels := []string{f.Name + " light", f.Name + " dark", "Duplicate theme " + f.Name}
		if f.ID == "custom-c" {
			labels = append(labels, "Edit theme "+f.Name, "Export theme "+f.Name, "Delete theme "+f.Name)
		}
		for _, label := range labels {
			r, ok := tt.Find(label)
			if !ok || r.W <= 0 || r.X < 15.5 || r.X+r.W > 374.5 {
				t.Fatalf("narrow action clipped %q: %+v present=%v", label, r, ok)
			}
		}
	}
	if err := tt.Click("Export theme Custom C"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tt.Clipboard(), `"id": "custom-c"`) || *writes != 0 {
		t.Fatal("read-only export failed or persisted")
	}
	if err := tt.Click("Duplicate theme Custom C"); err != nil {
		t.Fatal(err)
	}
	if e.Selected != "" || *writes != 0 {
		t.Fatal("read-only duplicate changed state")
	}
}

func TestThemeLibraryHeaderWrapsWithoutSplittingTitle(t *testing.T) {
	var titleWidth float32
	for _, width := range []int{760, 324} {
		_, tt, _ := libraryActionsFixture(t, width, true)
		title, ok := tt.Find("Themes")
		if !ok || title.H != 24 || title.W <= 0 {
			t.Fatalf("header title split at width %d: %+v", width, title)
		}
		if width == 760 {
			titleWidth = title.W
		} else if title.W != titleWidth {
			t.Fatalf("header title shortened: normal=%g narrow=%g", titleWidth, title.W)
		}
		for _, label := range []string{"Create theme", "Import theme"} {
			r, ok := tt.Find(label)
			if !ok || r.X < 16 || r.X+r.W > float32(width-16)+.5 {
				t.Fatalf("header action clipped at width %d: %s %+v", width, label, r)
			}
			if width == 324 && r.Y < title.Y+title.H {
				t.Fatalf("header actions compressed title rather than wrapping: title=%+v action=%+v", title, r)
			}
		}
	}
}

func TestThemeLibraryActionsRevealOnHoverAndKeyboardFocus(t *testing.T) {
	_, tt, _ := libraryActionsFixture(t, 760, true)
	label := "Duplicate theme DeepSeek"
	area, ok := tt.Find(label)
	if !ok {
		t.Fatal("duplicate action missing")
	}
	pixels := func() []byte {
		img := tt.Image()
		out := make([]byte, 0, int(area.W*area.H*4))
		for y := int(area.Y) + 2; y < int(area.Y+area.H)-2; y++ {
			left := img.PixOffset(int(area.X)+2, y)
			right := img.PixOffset(int(area.X+area.W)-2, y)
			out = append(out, img.Pix[left:right]...)
		}
		return out
	}
	hidden := pixels()
	half, ok := tt.Find("DeepSeek light")
	if !ok {
		t.Fatal("light preview missing")
	}
	tt.Move(half.X+half.W/2, half.Y+half.H/2)
	if bytes.Equal(hidden, pixels()) {
		t.Fatal("hovered card did not reveal footer action")
	}
	tt.Move(0, 0)
	if !bytes.Equal(hidden, pixels()) {
		t.Fatal("unfocused footer action remained visible after pointer left")
	}
	if err := tt.Click("DeepSeek light"); err != nil {
		t.Fatal(err)
	}
	tt.Move(0, 0)
	if !tt.Focused("DeepSeek light") {
		t.Fatal("preview did not keep native keyboard focus")
	}
	if bytes.Equal(hidden, pixels()) {
		t.Fatal("focus-within did not reveal footer action")
	}
}

func TestThemeLibraryKeyboardPicksOnlyFocusedHalf(t *testing.T) {
	e, tt, _ := libraryActionsFixture(t, 760, true)
	for n := 0; n < 40 && !tt.Focused("Codex dark"); n++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Codex dark") {
		t.Fatal("keyboard cannot reach the Codex dark preview")
	}
	tt.Key(0, ui.KeyEnter)
	if e.Saved.Light != "deepseek" || e.Saved.Dark != "codex" {
		t.Fatalf("keyboard coupled theme halves: light=%s dark=%s", e.Saved.Light, e.Saved.Dark)
	}
}
