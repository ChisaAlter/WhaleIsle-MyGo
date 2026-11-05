package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// An async launcher registration can arrive between the platform choice and
// the frame that consumes it. Native account actions retain their own identity.
func TestAccountNativeChoiceSurvivesLauncherInsertion(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, action := range []string{"contact", "signout", "signin"} {
		t.Run(action, func(t *testing.T) {
			published, calls := false, 0
			a := Account{
				Name: "Fixture", SignedIn: action != "signin", OnOpen: func() {},
				OnContact: func() {
					if action == "contact" {
						calls++
					}
				},
				OnSignOut: func() {
					if action == "signout" {
						calls++
					}
				},
				OnSignIn: func() {
					if action == "signin" {
						calls++
					}
				},
				LauncherActions: func(menu *ui.Menu) {
					menu.Live()
					if published {
						menu.Item("Async launcher").Key("fixture:launcher")
					}
				},
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					accountRow(c, Model{Account: a}, theme.Resolve(theme.IDDeepseekLight).Light, ui.RGB(90, 90, 90), false)
				})
			}, 280, 200)
			if err := tt.Click("Fixture"); err != nil {
				t.Fatal(err)
			}
			label := workspaceText("联系我们", "Contact us")
			if action == "signout" {
				label = i18n.T(i18n.KeySettingsAccountSignOut)
			}
			if action == "signin" {
				label = i18n.T(i18n.KeySettingsAccountSignIn)
			}
			published = true
			if err := tt.ChooseMenuItem(label); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("%s calls=%d after launcher insertion", action, calls)
			}
		})
	}
}
