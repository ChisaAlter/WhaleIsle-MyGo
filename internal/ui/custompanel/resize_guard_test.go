package custompanel

import (
	"testing"

	"github.com/egoist/mygo/ui"

	panel "whaleisle/internal/store/custompanel"
	"whaleisle/internal/ui/i18n"
)

func TestResizeButtonsDoNotPanicWhenOnUpdateNilAndFollowLocale(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)

	for _, tc := range []struct {
		locale string
		labels []string
	}{
		{locale: i18n.LocaleZH, labels: []string{"宽 −", "宽 +", "高 −", "高 +"}},
		{locale: i18n.LocaleEN, labels: []string{"Width −", "Width +", "Height −", "Height +"}},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			m := Model{
				ThemeID:   "deepseek",
				Available: true,
				Editor:    &Editor{},
				State: panel.Snapshot{
					Instances: []panel.Instance{{
						Definition: panel.Definition{
							ID:      "p1",
							Title:   "Metrics",
							Width:   400,
							Height:  240,
							Columns: 1,
						},
					}},
				},
				OnUpdate: nil,
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 900)
			for _, label := range tc.labels {
				if !tt.HasText(label) {
					t.Fatalf("missing localized resize button %q in %v", label, tt.Texts())
				}
				// Must not panic when OnUpdate is nil.
				_ = tt.Click(label)
			}
		})
	}
}
