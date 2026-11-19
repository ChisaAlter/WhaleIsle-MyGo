package theme

import (
	"errors"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// A preference the editor refuses is a sentence a person reads, so it speaks the
// interface language and never carries the English the code was written in.
func TestValidationSentencesSpeakTheInterfaceLanguage(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			cases := map[string]struct {
				prefs Preferences
				want  string
			}{
				"font size range": {
					prefs: func() Preferences {
						p := Defaults()
						p.Chat.Size = 4
						return p
					}(),
					want: i18n.T(i18n.KeyThemeFontSizeRange),
				},
				"custom theme name": {
					prefs: func() Preferences {
						p := Defaults()
						p.Custom = []Family{{ID: "", Name: "fixture"}}
						return p
					}(),
					want: i18n.T(i18n.KeyThemeNameRequired),
				},
				"color format": {
					prefs: func() Preferences {
						p := Defaults()
						p.Custom = []Family{{ID: "fixture", Name: "Fixture", Light: Seed{Background: "red", Foreground: "#101010", Accent: "#202020"}}}
						return p
					}(),
					want: i18n.T(i18n.KeyThemeColorFormatSyntax),
				},
			}
			for name, tc := range cases {
				err := tc.prefs.Validate()
				if err == nil {
					t.Fatalf("%s: refused nothing", name)
				}
				if err.Error() != tc.want {
					t.Fatalf("%s: message = %q, want %q", name, err.Error(), tc.want)
				}
				var refused ValidationError
				if !errors.As(err, &refused) {
					t.Fatalf("%s: refusal is not a ValidationError, so the page would classify it as an unknown failure", name)
				}
			}
		})
	}
}

// The stepper's refusal used to be an English literal assembled with Sprintf,
// which the theme feedback drew as the message in both locales.
func TestFontSizeRefusalNamesTheRange(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	i18n.SetLocale(i18n.LocaleEN)
	e := NewEditor(Defaults(), func(Preferences) error { return nil })
	if e.SaveFontSize("code", 12.5) {
		t.Fatal("a fractional size was saved")
	}
	want := "Font size must be an integer from 10 to 20"
	if e.Status != want {
		t.Fatalf("status = %q, want %q", e.Status, want)
	}
	if e.StatusDetail != "" {
		t.Fatalf("a refusal has no cause to disclose, status detail = %q", e.StatusDetail)
	}
	if e.SaveFontSize("text", 40) {
		t.Fatal("an out-of-range size was saved")
	}
	if want := "Font size must be an integer from 10 to 22"; e.Status != want {
		t.Fatalf("text role status = %q, want %q", e.Status, want)
	}
}

// A save that failed keeps the author's sentence and files the cause behind 详情.
func TestFailedSaveKeepsTheCauseForTheDetail(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	i18n.SetLocale(i18n.LocaleEN)
	raw := "write C:/Users/fixture/AppData/Roaming/WhaleIsle/theme.json: Access is denied."
	e := NewEditor(Defaults(), func(Preferences) error { return errors.New(raw) })
	e.Save()
	if e.Status == "" || e.Status == raw {
		t.Fatalf("status = %q, want the classified sentence", e.Status)
	}
	if e.StatusDetail != raw {
		t.Fatalf("status detail = %q, want the store's own words", e.StatusDetail)
	}
}

func TestSavedStatusUsesRegisteredCopy(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			e := NewEditor(Defaults(), func(Preferences) error { return nil })
			if !e.SaveFontSize("code", 13) {
				t.Fatalf("valid size refused: %s", e.Status)
			}
			if want := i18n.T(i18n.KeyThemeEditorSaved); e.Status != want {
				t.Fatalf("status = %q, want %q", e.Status, want)
			}
			if e.StatusDetail != "" {
				t.Fatalf("a successful save has no detail, got %q", e.StatusDetail)
			}
		})
	}
}
