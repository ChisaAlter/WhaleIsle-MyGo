package custompanel

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	panel "whaleisle/internal/store/custompanel"
	"whaleisle/internal/ui/i18n"
)

// detailHolds opens the first 详情 disclosure, checks the store's own words
// behind it, and collapses it again.
func detailHolds(t *testing.T, tt *ui.Tester, raw string) bool {
	t.Helper()
	if tt.HasText(raw) {
		return false
	}
	if err := tt.Click(i18n.Pick("详情", "Details")); err != nil {
		return false
	}
	tt.Frame()
	held := tt.HasText(raw)
	if err := tt.Click(i18n.Pick("收起", "Hide")); err == nil {
		tt.Frame()
	}
	return held
}

// Every problem the store can name speaks the interface language, and none of
// them falls back to the generic failure sentence.
func TestEveryProblemSpeaksTheLocale(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	problems := []panel.Problem{
		panel.ProblemJSON, panel.ProblemTooLarge, panel.ProblemTrailingContent,
		panel.ProblemMissingTitle, panel.ProblemSize, panel.ProblemWidgetCount,
		panel.ProblemWidgetID, panel.ProblemWidgetKind, panel.ProblemFontSize,
		panel.ProblemProgressRange, panel.ProblemSelectOptions,
		panel.ProblemLibraryVersion, panel.ProblemScope, panel.ProblemAlreadyAdded,
		panel.ProblemTemplateMissing, panel.ProblemInstanceMissing,
	}
	const raw = "panel title is required"
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			for _, problem := range problems {
				failure := Word(panel.Refusal{Problem: problem, Detail: raw})
				if strings.TrimSpace(failure.Sentence) == "" {
					t.Fatalf("%s has no sentence", problem)
				}
				if failure.Sentence == raw {
					t.Fatalf("%s shows the store's words as the sentence", problem)
				}
				if failure.Detail != raw {
					t.Fatalf("%s dropped the cause: %q", problem, failure.Detail)
				}
			}
		})
	}
}

// A refused panel draws the sentence and keeps the store's own words behind
// 详情, so the person knows what to fix and whoever they report to has the cause.
func TestRefusedPanelKeepsTheCauseBehindDetail(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	_, err := panel.Parse(`{"title":"","width":420,"height":320,"columns":1,"widgets":[]}`)
	if err == nil {
		t.Fatal("the fixture was accepted")
	}
	editor := &Editor{Failure: Word(err)}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Editor: editor, BotLabel: "Fixture Bot", Available: true})
	}, 720, 600)
	if !tt.HasText(i18n.T(i18n.KeyCmdPanelWidgetCount)) && !tt.HasText(i18n.T(i18n.KeyCmdPanelTitleRequired)) {
		t.Fatalf("the panel does not say what is wrong: %v", tt.Texts())
	}
	if tt.HasText(err.Error()) {
		t.Fatalf("the panel stands the store's words in the sentence: %v", tt.Texts())
	}
	if !detailHolds(t, tt, err.Error()) {
		t.Fatalf("the refusal dropped the cause instead of filing it behind 详情: %v", tt.Texts())
	}
}

// The panel words an error the store did not name the same way every other
// surface does: errcopy's sentence, with the error's own text behind 详情.
func TestOtherPanelErrorsKeepTheirCause(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	raw := "write C:/Users/fixture/AppData/Roaming/WhaleIsle/assistant/panels.json: Access is denied."
	failure := Word(&failureProbe{raw})
	if failure.Sentence == "" || failure.Sentence == raw {
		t.Fatalf("sentence = %q, want the classified line", failure.Sentence)
	}
	if failure.Detail != raw {
		t.Fatalf("detail = %q, want the store's own words", failure.Detail)
	}
}

type failureProbe struct{ text string }

func (p *failureProbe) Error() string { return p.text }
