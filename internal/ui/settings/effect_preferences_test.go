package settings

import (
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func scrollEffectDialogToBottom(t *testing.T, tt *ui.Tester, target string) {
	t.Helper()
	tt.Scroll(450, 450, 0, 4000)
	r, ok := tt.Find(target)
	if !ok || r.Y < 100 || r.Y+r.H > 760 {
		t.Fatalf("target %q not visible after scroll: ok=%v rect=%+v", target, ok, r)
	}
}

func TestEffectPreferencesModalCancelPresetCapAndTypingControls(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)

	t.Run("beam-cancel-discards-draft-and-caps-5-presets", func(t *testing.T) {
		h := bodyFixtureHost()
		patches := 0
		changes := 0
		h.OnPatch = func(string, map[string]string) { patches++ }
		h.OnChange = func(string, string, string) { changes++ }

		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(24).Children(func() {
				bodyEffectPreference(c, h, false, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 900, 950)

		if err := tt.Click("Thinking glow settings"); err != nil {
			t.Fatal(err)
		}
		// Verify key beam parameter sections exist in modal.
		for _, want := range []string{
			"Mode",
			"Direction",
			"Easing",
			"Rotation period",
			"Overall intensity",
			"Bloom intensity",
			"Track width",
			"Glow blur",
			"Brightness breathing",
			"Hue cycling",
			"Enable night window",
			"Color source",
			"Personal presets",
		} {
			if !tt.HasText(want) {
				t.Fatalf("BeamSettingsModal missing %q", want)
			}
		}

		// Scroll presets section into the dialog viewport before interacting with it.
		scrollEffectDialogToBottom(t, tt, "Save preset")

		// Add 5 presets; 6th must be rejected by the <=5 source limit.
		for i := 1; i <= 5; i++ {
			if err := tt.SetValue("effect-preset-name", fmt.Sprintf("preset-%d", i)); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Save preset"); err != nil {
				t.Fatal(err)
			}
			if detailHolds(t, tt, "Save up to 5 presets.") {
				t.Fatalf("preset %d unexpectedly hit 5-preset cap", i)
			}
		}
		if err := tt.SetValue("effect-preset-name", "preset-6"); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click("Save preset"); err != nil {
			t.Fatal(err)
		}
		if !detailHolds(t, tt, "Save up to 5 presets.") {
			t.Fatalf("6th preset did not trigger 5-preset cap error behind Details; texts=%v", tt.Texts())
		}

		// Copy configuration JSON -> clipboard must carry version 1 envelope with 5 presets.
		scrollEffectDialogToBottom(t, tt, "Copy configuration JSON")
		if err := tt.Click("Copy configuration JSON"); err != nil {
			t.Fatal(err)
		}
		raw := tt.Clipboard()
		var env struct {
			Version int                        `json:"version"`
			Core    string                     `json:"core"`
			Presets map[string]json.RawMessage `json:"presets"`
		}
		if err := json.Unmarshal([]byte(raw), &env); err != nil || env.Version != 1 || env.Core != "dsh-composer-beam" || len(env.Presets) != 5 {
			t.Fatalf("exported v1 envelope invalid: version=%d core=%q presets=%d err=%v", env.Version, env.Core, len(env.Presets), err)
		}

		// Cancel must close the modal without patching/persisting draft changes,
		// and reopening must start from clean Host state (no preset-1 left in draft).
		if err := tt.Click("Cancel"); err != nil {
			t.Fatal(err)
		}
		if tt.HasText("Save and apply") {
			t.Fatal("Cancel did not close BeamSettingsModal")
		}
		if patches != 0 || changes != 0 {
			t.Fatalf("Cancel persisted changes: patches=%d changes=%d", patches, changes)
		}
		if err := tt.Click("Thinking glow settings"); err != nil {
			t.Fatal(err)
		}
		scrollEffectDialogToBottom(t, tt, "Copy configuration JSON")
		if err := tt.Click("Copy configuration JSON"); err != nil {
			t.Fatal(err)
		}
		var reopened struct {
			Presets map[string]json.RawMessage `json:"presets"`
		}
		if err := json.Unmarshal([]byte(tt.Clipboard()), &reopened); err != nil || len(reopened.Presets) != 0 {
			t.Fatalf("Cancel leaked unsaved presets into reopened modal: %+v err=%v", reopened.Presets, err)
		}
	})

	t.Run("typing-fx-controls-and-cancel", func(t *testing.T) {
		h := bodyFixtureHost()
		patches := 0
		h.OnPatch = func(string, map[string]string) { patches++ }

		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(24).Children(func() {
				bodyEffectPreference(c, h, true, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 900, 950)

		if err := tt.Click("Typing effect settings"); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"Effect",
			"Caret",
			"Caret blink",
			"Echo speed",
			"Color source",
			"Personal presets",
		} {
			if !tt.HasText(want) {
				t.Fatalf("TypingFxModal missing %q", want)
			}
		}
		if err := tt.Click("Cancel"); err != nil {
			t.Fatal(err)
		}
		if tt.HasText("Save and apply") || patches != 0 {
			t.Fatalf("TypingFxModal Cancel failed: open=%v patches=%d", tt.HasText("Save and apply"), patches)
		}
	})
}
