package sidebar

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestSidebarFooterUsesAccountSeatWhenSignedIn keeps the production footer
// from rendering the settings gear and the account seat at the same time.
func TestSidebarFooterUsesAccountSeatWhenSignedIn(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	m := Model{
		ThemeID: "deepseek.dark", SystemDark: true,
		Snapshot:       session.ViewState{WorkspaceReady: true},
		OnOpenSettings: func() {},
		Account:        Account{Name: "Ayase", SignedIn: true, OnOpen: func() {}},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)
	if !tt.HasText("Ayase") {
		t.Fatalf("signed-in footer lost the account seat: %q", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeySettingsAccountTitle)) {
		t.Fatalf("signed-in footer rendered two account/settings entries: %q", tt.Texts())
	}
}

func TestAccountRowShowsIdentityAndMenu(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	opened, signedOut := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			m := Model{Account: Account{
				Name:     "Ayase",
				Detail:   "a***@example.invalid",
				SignedIn: true,
				OnOpen:   func() { opened++ },
				OnSignOut: func() {
					signedOut++
				},
			}}
			accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), false)
		})
	}, 280, 120)

	row, ok := tt.Find("Ayase")
	if !ok || row.H != 44 || row.W != 280 {
		t.Fatalf("account row=%+v", row)
	}
	if !tt.HasText("a***@example.invalid") {
		t.Fatalf("missing account detail; texts=%q", tt.Texts())
	}
	if got := accountInitial("Ayase"); got != "A" {
		t.Fatalf("initial=%q", got)
	}

	if err := tt.Click("Ayase"); err != nil {
		t.Fatal(err)
	}
	want := []string{i18n.T(i18n.KeySettingsAccountTitle), i18n.T(i18n.KeySettingsAccountSignOut)}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Fatalf("menu=%q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySettingsAccountSignOut)); err != nil {
		t.Fatal(err)
	}
	if signedOut != 1 {
		t.Fatalf("sign out calls=%d", signedOut)
	}

	if err := tt.Click("Ayase"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySettingsAccountTitle)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("account open calls=%d", opened)
	}
}

// TestAccountRowFallsBackToSettings keeps the seat usable before the shell
// publishes a profile: the profile-less seat is the merged settings/account
// trigger, labelled launcherLabel (设置与账号) with the accessible name 账号菜单,
// and the settings entry is the destination. The collapsed rail is 36x36.
func TestAccountRowFallsBackToSettings(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	settings := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			m := Model{OnOpenSettings: func() { settings++ }}
			accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), false)
			accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), true)
		})
	}, 280, 160)

	// Without a name the row shows the merged seat label and no detail.
	if !tt.HasText(i18n.T(i18n.KeySettingsAccountSeat)) {
		t.Fatalf("missing merged settings label; texts=%q", tt.Texts())
	}
	row, ok := tt.Find(i18n.T(i18n.KeySettingsAccountMenu))
	if !ok || row.H != 36 || row.W != 280 {
		t.Fatalf("merged settings seat=%+v, want 280x36", row)
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsAccountSeat)); err != nil {
		t.Fatal(err)
	}
	want := []string{i18n.T(i18n.KeySettingsAccountTitle)}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Fatalf("menu=%q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySettingsAccountTitle)); err != nil {
		t.Fatal(err)
	}
	if settings != 1 {
		t.Fatalf("settings calls=%d", settings)
	}
}

// TestAccountRowSkipWithoutAnyDestination keeps the footer stable when the
// shell publishes neither an account entry point nor a sign-out flow.
func TestAccountRowSkipWithoutAnyDestination(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			accountRow(c, Model{}, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), false)
		})
	}, 280, 80)
	if texts := tt.Texts(); len(texts) != 0 {
		t.Fatalf("row rendered without a destination: %q", texts)
	}
}
