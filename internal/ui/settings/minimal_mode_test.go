package settings

import (
	"encoding/json"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// minimalModeEvidenceEnv names the directory the render evidence is written to
// when it is set. A normal test run writes nothing.
const minimalModeEvidenceEnv = "WHALEISLE_MINIMAL_MODE_EVIDENCE"

type minimalModeFixture struct {
	access    *AppearanceAccess
	enabled   bool
	suggest   bool
	toggled   []bool
	suggested []bool
}

func newMinimalModeFixture(t *testing.T) *minimalModeFixture {
	t.Helper()
	f := &minimalModeFixture{}
	scheme := "deepseek.light"
	f.access = &AppearanceAccess{
		Host:        bodyFixtureHost(),
		Preferences: theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil }),
		Background:  NewWallpaperEditor(wallpapers.Defaults(), "", func(wallpapers.Preferences) error { return nil }, nil, nil),
		Theme:       &scheme,
		Minimal: &MinimalModeAccess{
			Enabled:     &f.enabled,
			Suggestions: &f.suggest,
			OnToggle: func(on bool) error {
				f.toggled = append(f.toggled, on)
				renderpolicy.SetMinimal(on)
				return nil
			},
			OnToggleSuggestions: func(on bool) error {
				f.suggested = append(f.suggested, on)
				return nil
			},
		},
	}
	return f
}

func (f *minimalModeFixture) frame() *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: f.access, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1400)
}

// evidenceFrame renders the whole page at once: the effect rows sit below the
// fold of a 1400 DIP window, and the evidence has to show them in place.
func (f *minimalModeFixture) evidenceFrame() *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: f.access, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 2600)
}

// TestMinimalModeSwitchPausesAndRestores is the behavioural proof: the switch
// reports the opposite state, and the page it rebuilt says the effects below
// are overridden until it reports them back.
func TestMinimalModeSwitchPausesAndRestores(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	renderpolicy.SetMinimal(false)
	defer renderpolicy.SetMinimal(false)

	f := newMinimalModeFixture(t)
	off := f.frame()
	if !off.HasText("Minimal mode") || !off.HasText("Pauses wallpapers") {
		t.Fatalf("the appearance page lost the minimal-mode block: %v", off.Texts())
	}
	if off.HasText("Overridden by minimal mode") {
		t.Fatal("the page claims an override while minimal mode is off")
	}
	if err := off.Click("Turn on minimal mode"); err != nil {
		t.Fatalf("the switch is not reachable: %v", err)
	}
	if len(f.toggled) != 1 || !f.toggled[0] {
		t.Fatalf("the switch did not report turning on: %v", f.toggled)
	}
	f.enabled = true

	on := f.frame()
	for _, paused := range []string{
		"Wallpaper",
		"Background effect",
		"Cursor effects",
		"Glass opacity",
		"Transparent theme",
		"Thinking glow when sending",
		"Typing effects",
	} {
		if !on.HasText(paused) {
			t.Fatalf("paused row %q is missing while minimal mode is on: %v", paused, on.Texts())
		}
	}
	if !on.HasText("Overridden by minimal mode") {
		t.Fatal("no paused row says it is overridden")
	}
	if err := on.Click("Turn off minimal mode"); err != nil {
		t.Fatalf("the on switch is not reachable: %v", err)
	}
	if len(f.toggled) != 2 || f.toggled[1] {
		t.Fatalf("the switch did not report turning off: %v", f.toggled)
	}
}

// TestMinimalSuggestionCanBeTurnedBackOn pins the row that a user who chose to
// stop the startup offer uses to restore it.
func TestMinimalSuggestionCanBeTurnedBackOn(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	renderpolicy.SetMinimal(false)

	f := newMinimalModeFixture(t)
	f.suggest = false
	tt := f.frame()
	if !tt.HasText("Suggest on slow startup") || !tt.HasText("at most once per launch") {
		t.Fatalf("the startup-offer row is missing: %v", tt.Texts())
	}
	if err := tt.Click("Turn on startup suggestions"); err != nil {
		t.Fatalf("the startup-offer switch is not reachable: %v", err)
	}
	if len(f.suggested) != 1 || !f.suggested[0] {
		t.Fatalf("the startup-offer switch did not report itself restored: %v", f.suggested)
	}
}

// TestMinimalModeRendersOffOnAndOverridden writes the render evidence the lane
// reports: the page with the switch off, with it on, and the startup offer,
// with the measurements the images cannot carry.
func TestMinimalModeRendersOffOnAndOverridden(t *testing.T) {
	dir := os.Getenv(minimalModeEvidenceEnv)
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	defer renderpolicy.SetMinimal(false)

	f := newMinimalModeFixture(t)
	f.suggest = true
	record := map[string]any{"schemaVersion": 1, "page": "settings → appearance", "locale": "en", "screens": []any{}}
	shoot := func(t *testing.T, name string, tt *ui.Tester) {
		t.Helper()
		bounds := tt.Image().Bounds()
		measure := map[string]any{"name": name, "widthDx": bounds.Dx(), "heightDy": bounds.Dy()}
		for _, label := range []string{"Minimal mode", "Turn on minimal mode", "Turn off minimal mode", "Suggest on slow startup", "Wallpaper", "Background effect", "Glass opacity", "Thinking glow when sending"} {
			if r, ok := tt.Find(label); ok {
				measure[label] = map[string]any{"x": r.X, "y": r.Y, "w": r.W, "h": r.H}
			}
		}
		notes := []string{}
		for _, text := range tt.Texts() {
			if containsOverride(text) {
				notes = append(notes, text)
			}
		}
		measure["overriddenRows"], measure["textCount"], measure["overrideNotes"] = len(notes), len(tt.Texts()), notes
		record["screens"] = append(record["screens"].([]any), measure)
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("evidence directory: %v", err)
		}
		path := filepath.Join(dir, name+".png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("evidence image: %v", err)
		}
		defer file.Close()
		if err := png.Encode(file, tt.Image()); err != nil {
			t.Fatalf("encode %s: %v", name, err)
		}
	}

	renderpolicy.SetMinimal(false)
	f.enabled = false
	shoot(t, "appearance-minimal-off", f.evidenceFrame())

	renderpolicy.SetMinimal(true)
	f.enabled = true
	shoot(t, "appearance-minimal-on", f.evidenceFrame())

	raw, _ := json.MarshalIndent(record, "", "  ")
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "render.json"), append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("evidence record: %v", err)
	}
}

func containsOverride(text string) bool {
	const note = " · Overridden by minimal mode"
	for i := 0; i+len(note) <= len(text); i++ {
		if text[i:i+len(note)] == note {
			return true
		}
	}
	return false
}

// TestMinimalModeFailureIsASentenceNotRawText pins the package's copy rule on
// the row this lane added: a refused write shows the shared classifier's
// sentence, and the transport's own words never reach the page.
func TestMinimalModeFailureIsASentenceNotRawText(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)
	renderpolicy.SetMinimal(false)
	defer renderpolicy.SetMinimal(false)

	f := newMinimalModeFixture(t)
	f.access.Minimal.OnToggle = func(bool) error {
		return errors.New("rpc error: code = Unavailable desc = dial tcp 10.0.0.1:443: connect: connection refused")
	}
	tt := f.frame()
	if err := tt.Click("开启极简模式"); err != nil {
		t.Fatalf("the switch is not reachable: %v", err)
	}
	texts := tt.Texts()
	if !tt.HasText("网络连接不可用") {
		t.Fatalf("the refusal is not the classifier's sentence: %v", texts)
	}
	for _, raw := range []string{"dial tcp", "rpc error", "connection refused"} {
		if tt.HasText(raw) {
			t.Fatalf("the page paints the raw failure %q: %v", raw, texts)
		}
	}
	if f.enabled {
		t.Fatal("a refused write still moved the switch")
	}
}
