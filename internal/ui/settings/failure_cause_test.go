package settings

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// A write that failed keeps the cause: the page shows the sentence, and 详情
// opens what the store said. Both halves are asserted, because a page that keeps
// the sentence and drops the error looks fine until somebody has to report it.
func TestInterfaceWriteKeepsTheCauseBehindDetail(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	raw := "settings/mutate ui-conversation.viewTabs: error code 40002: invalid arguments"
	want := errcopy.Sentence(errors.New(raw))
	refuse := true
	tt := ui.NewTester(func(c *ui.Context) {
		failure := ui.Local(c.Root(), "probe-write", func() settingsRefusal { return settingsRefusal{} })
		ui.Column(c).Fill().Padding(8).Children(func() {
			if ui.Button(c, "probe-write").Label("write").Clicked() {
				accepted := false
				setInterfaceAccepted(c, failure, &accepted, true, func(bool) (bool, error) {
					if refuse {
						return false, errors.New(raw)
					}
					return true, nil
				})
			}
			settingsAlert(c, "probe/interface-write", failure.failed(), failure.sentence, failure.detail)
		})
	}, 520, 300)
	if err := tt.Click("write"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(want) {
		t.Fatalf("the page does not show the sentence: %v", tt.Texts())
	}
	if tt.HasText(raw) {
		t.Fatalf("the page stands the store's words in the sentence: %v", tt.Texts())
	}
	if !detailHolds(t, tt, raw) {
		t.Fatalf("the refused write dropped the cause instead of filing it behind 详情: %v", tt.Texts())
	}
	refuse = false
	if err := tt.Click("write"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText(raw) || !tt.HasText("write") {
		t.Fatalf("a successful retry kept the old cause: %v", tt.Texts())
	}
}

// The theme editor's failure used to reach the page as a sentence with nothing
// to open; the cause now travels beside it.
func TestThemeFeedbackKeepsTheCauseBehindDetail(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	raw := "write C:/Users/fixture/AppData/Roaming/WhaleIsle/theme.json: Access is denied."
	e := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return errors.New(raw) })
	e.Save()
	if e.StatusDetail != raw {
		t.Fatalf("the editor did not keep the cause: %q", e.StatusDetail)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(8).Children(func() { themeEditorFeedback(c, e) })
	}, 620, 300)
	if !tt.HasText(e.Status) {
		t.Fatalf("the page does not show the sentence: %v", tt.Texts())
	}
	if !detailHolds(t, tt, raw) {
		t.Fatalf("the theme failure dropped the cause instead of filing it behind 详情: %v", tt.Texts())
	}
}

// A validator's refusal is already written for a person, so it is the message,
// and it speaks the interface language rather than the language of its call site.
func TestThemeValidationRefusalSpeaksTheLocale(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			broken := theme.Defaults()
			broken.Chat.Size = 4
			e := theme.NewEditor(broken, func(theme.Preferences) error { return nil })
			e.Save()
			if e.Status != i18n.T(i18n.KeyThemeFontSizeRange) {
				t.Fatalf("status = %q, want the validator's own sentence", e.Status)
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(8).Children(func() { themeEditorFeedback(c, e) })
			}, 620, 300)
			if !tt.HasText(i18n.T(i18n.KeyThemeFontSizeRange)) {
				t.Fatalf("the refusal is not the page's message: %v", tt.Texts())
			}
			if locale == i18n.LocaleZH && tt.HasText("Font size") {
				t.Fatalf("an English literal escaped into the Chinese interface: %v", tt.Texts())
			}
			if locale == i18n.LocaleEN && tt.HasText("字号") {
				t.Fatalf("a Chinese literal escaped into the English interface: %v", tt.Texts())
			}
		})
	}
}
