package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestComposerHoverFocusZeroShiftSweep(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	for _, hero := range []bool{true, false} {
		name := "conversation"
		if hero {
			name = "hero"
		}
		t.Run(name, func(t *testing.T) {
			d := NewDraftState("")
			var wsAnchor ui.Handle
			m := Model{
				State:                 d,
				Draft:                 &d.Text,
				Hero:                  hero,
				WorkspaceRoot:         `C:\work\whaleisle`,
				WorkspacePickerAnchor: &wsAnchor,
				Snapshot: session.ViewState{
					ClientReady:         true,
					SelectedID:          "s1",
					SelectedModel:       "deepseek-official/deepseek-flash",
					SelectedModelEffort: "high",
					ModelLabels:         []string{"deepseek-official/deepseek-flash"},
					NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true},
				},
				OnPickHeroWorkspace: func() {},
				OnPickImage:         func() {},
				OnPickDirectory:     func() {},
				OnPickSpeech:        func() {},
				OnPromptModeChange:  func(session.PromptMode) {},
				OnToggleSkills:      func(bool) {},
				OnRetryModels:       func() {},
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Size(800, 400).Justify(ui.End).Padding(16).Children(func() {
					View(c, m)
				})
			}, 800, 400)
			tt.Frame()
			tt.Frame()
			ui.AssertHoverFocusZeroShift(t, tt)
		})
	}
}
