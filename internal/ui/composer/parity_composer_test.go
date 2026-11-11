package composer

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParityPermissionPickerConfirmDialogAndSwitchFailed(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		m := permissionModelFixture()
		m.ConfirmPermissionInPicker = true
		m.PermissionSwitchFailed = true
		picked := ""
		m.OnPermissionChange = func(v string) { picked = v }

		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 480)

		// Verify switchFailed notice is rendered in both locales.
		wantSwitchFailed := i18n.T(i18n.KeyParityPermSwitchFailed)
		if !tt.HasText(wantSwitchFailed) {
			t.Fatalf("[%s] missing switchFailed %q in %v", locale, wantSwitchFailed, tt.Texts())
		}

		for _, b := range tt.Interactives() {
			if slices.Contains(b.Kinds, "click") && !slices.Contains(b.Kinds, "text") && b.Cursor != ui.CursorPointer {
				t.Fatalf("[%s] clickable composer element missing CursorPointer: %s", locale, b.String())
			}
		}

		// Open permission picker and select Full access.
		if err := tt.Click(i18n.T(i18n.KeyComposerPermissionWorkspace)); err != nil {
			t.Fatalf("[%s] open permission picker: %v", locale, err)
		}
		if err := tt.Click(i18n.T(i18n.KeyComposerPermissionFullAccess)); err != nil {
			t.Fatalf("[%s] click full access: %v", locale, err)
		}
		if picked != "" {
			t.Fatalf("[%s] unconfirmed full access dispatched early: %q", locale, picked)
		}

		// Verify confirmation descriptions and acknowledge checkbox copy.
		if !tt.HasText(i18n.T(i18n.KeyParityPermConfirmDescription)) || !tt.HasText(i18n.T(i18n.KeyParityPermConfirmDescriptionAlt)) {
			t.Fatalf("[%s] missing confirm.description in %v", locale, tt.Texts())
		}
		if !tt.HasText(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledge)) {
			t.Fatalf("[%s] missing auto.confirm.acknowledge in %v", locale, tt.Texts())
		}

		// Acknowledge and confirm.
		if err := tt.Click(i18n.T(i18n.KeyParityPermAutoConfirmAcknowledge)); err != nil {
			t.Fatalf("[%s] click acknowledge checkbox: %v", locale, err)
		}
		if err := tt.Click(composerText("启用完全权限", "Enable Full access")); err != nil {
			t.Fatalf("[%s] click confirm button: %v", locale, err)
		}
		if picked != session.PermissionFullAccess {
			t.Fatalf("[%s] confirmed permission = %q, want %q", locale, picked, session.PermissionFullAccess)
		}
	}
}
