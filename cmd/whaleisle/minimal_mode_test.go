package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

// minimalModeEvidenceEnv names the directory the offer's render evidence is
// written to when it is set. A normal test run writes nothing.
const minimalModeEvidenceEnv = "WHALEISLE_MINIMAL_MODE_EVIDENCE"

func minimalModeStore(t *testing.T) *config.Store {
	t.Helper()
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatalf("config store: %v", err)
	}
	return store
}

// TestMinimalSuggestionGate pins when the startup offer may be made at all: a
// profile that already runs minimal mode, turned the offer off, or snoozed it is
// never offered again until the stored answer says so.
func TestMinimalSuggestionGate(t *testing.T) {
	store := minimalModeStore(t)
	s := &mainState{cfgStore: store}
	if !s.minimalSuggestionWanted() {
		t.Fatal("a fresh profile is not offered minimal mode")
	}
	if err := store.Update(func(c *config.Config) { c.MinimalMode = true }); err != nil {
		t.Fatal(err)
	}
	if s.minimalSuggestionWanted() {
		t.Fatal("a profile already on minimal mode was offered it")
	}
	if err := store.Update(func(c *config.Config) { c.MinimalMode = false; c.MinimalModeSuggestionsOff = true }); err != nil {
		t.Fatal(err)
	}
	if s.minimalSuggestionWanted() {
		t.Fatal("a profile that turned the offer off was offered it")
	}
	if err := store.Update(func(c *config.Config) { c.MinimalModeSuggestionsOff = false }); err != nil {
		t.Fatal(err)
	}
	s.snoozeMinimalModeOffer()
	cfg := store.Get()
	until, err := time.Parse(time.RFC3339, cfg.MinimalModeSnoozedUntil)
	if err != nil {
		t.Fatalf("the snooze is not a stored instant: %q", cfg.MinimalModeSnoozedUntil)
	}
	if left := time.Until(until); left < 6*24*time.Hour || left > 8*24*time.Hour {
		t.Fatalf("the snooze is %v, not about a week", left)
	}
	if s.minimalSuggestionWanted() {
		t.Fatal("a snoozed offer was made anyway")
	}
	if err := store.Update(func(c *config.Config) {
		c.MinimalModeSnoozedUntil = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	}); err != nil {
		t.Fatal(err)
	}
	if !s.minimalSuggestionWanted() {
		t.Fatal("an expired snooze still blocks the offer")
	}
	// A snooze that cannot be read is no snooze; the config normalises it away.
	dir := t.TempDir()
	raw := `{"goMinimalModeSnoozedUntil":"whenever"}`
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get().MinimalModeSnoozedUntil; got != "" {
		t.Fatalf("an unreadable snooze survived as %q", got)
	}
}

// TestMinimalModeOfferPresentsThreeAnswers renders the offer and writes the
// evidence: the copy, the three answers and where they sit, none of which the
// image alone carries.
func TestMinimalModeOfferPresentsThreeAnswers(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	dir := os.Getenv(minimalModeEvidenceEnv)
	state := &mainState{}
	state.minimalOfferOpen = true
	tt := ui.NewTester(func(c *ui.Context) { state.syncMinimalModeOffer(c) }, 900, 700)
	for _, text := range []string{"Startup felt slow", "Turn on minimal mode", "Not now", "Don't ask again", "Your appearance preferences are not changed"} {
		if !tt.HasText(text) {
			t.Fatalf("the offer is missing %q: %v", text, tt.Texts())
		}
	}
	record := map[string]any{"schemaVersion": 1, "screen": "startup-offer", "locale": "en"}
	bounds := tt.Image().Bounds()
	record["widthDx"], record["heightDy"] = bounds.Dx(), bounds.Dy()
	for _, label := range []string{"Startup felt slow", "Turn on minimal mode", "Not now", "Don't ask again"} {
		if r, ok := tt.Find(label); ok {
			record[label] = map[string]any{"x": r.X, "y": r.Y, "w": r.W, "h": r.H}
		}
	}
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("evidence directory: %v", err)
	}
	file, err := os.Create(filepath.Join(dir, "startup-offer.png"))
	if err != nil {
		t.Fatalf("evidence image: %v", err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatalf("encode offer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close offer image: %v", err)
	}
	raw, _ := json.MarshalIndent(record, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "offer.json"), append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("evidence record: %v", err)
	}
}

// TestMinimalModePausesBackdropMaterials proves the switch reaches the paint
// path and not only the stored preference: the acrylic blur material and the
// ambient backdrop are gone while it is on, and are back when it is off.
func TestMinimalModePausesBackdropMaterials(t *testing.T) {
	useGlassTheme(t)
	s := &mainState{cfgStore: minimalModeStore(t)}
	// Acrylic is the flat canvas material: no wallpaper and no gradient.
	flat := wallpapers.Defaults()
	paint := func() {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(s.wallpaperPainter("C:/wallpapers/sea.png", true, false))
		}, 200, 160)
		if bound := tt.Image().Bounds(); bound.Dx() != 200 {
			t.Fatalf("painter probe rendered %v", bound)
		}
	}
	defer renderpolicy.SetMinimal(false)

	renderpolicy.SetMinimal(false)
	paint()
	if !chrome.BackdropActive() {
		t.Fatal("the ambient backdrop is not painted with minimal mode off")
	}
	if runtime.GOOS == "windows" {
		if got := desktopGlassMaterial("", flat, true, false); got != mygo.VibrancyAcrylic {
			t.Fatalf("the glass material is %v with minimal mode off", got)
		}
	}

	renderpolicy.SetMinimal(true)
	paint()
	if chrome.BackdropActive() {
		t.Fatal("the ambient backdrop survived minimal mode")
	}
	if got := desktopGlassMaterial("", flat, true, false); got != mygo.VibrancyNone {
		t.Fatalf("the blur material survived minimal mode: %v", got)
	}
}
