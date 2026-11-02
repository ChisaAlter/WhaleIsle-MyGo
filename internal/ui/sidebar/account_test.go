package sidebar

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/paths"
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
	if !ok || row.H != 44 || row.W != 252 || row.X != 2 {
		t.Fatalf("account row=%+v, want 252x44 inset 2 in the 280-DIP sidebar", row)
	}
	if tt.HasText("a***@example.invalid") {
		t.Fatalf("unexpected secondary detail line when name is present; texts=%q", tt.Texts())
	}
	if got := accountLabel(Account{Detail: "a***@example.invalid", SignedIn: true}); got != "a***@example.invalid" {
		t.Fatalf("fallback accountLabel=%q, want contact", got)
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
			// Two fixture seats need independent parents; production renders one
			// stable account trigger in its footer, changing only its compact style.
			ui.Column(c.Key("wide-fixture")).FillWidth().Children(func() { accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), false) })
			ui.Column(c.Key("rail-fixture")).FillWidth().Children(func() { accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), true) })
		})
	}, 280, 160)

	// Without a name the row shows the merged seat label and no detail.
	if !tt.HasText(i18n.T(i18n.KeySettingsAccountSeat)) {
		t.Fatalf("missing merged settings label; texts=%q", tt.Texts())
	}
	row, ok := tt.Find(i18n.T(i18n.KeySettingsAccountMenu))
	if !ok || row.H != 36 || row.W != 252 || row.X != 2 {
		t.Fatalf("merged settings seat=%+v, want 252x36 inset 2", row)
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

// The installed trigger is one retained button across the 1024 breakpoint.
// A style-only collapse must keep focus without opening or choosing a menu.
func TestAccountRowFocusAndMenuAcrossResponsiveWidths(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	for _, signedIn := range []bool{false, true} {
		name := "signed-out"
		if signedIn {
			name = "signed-in"
		}
		t.Run(name, func(t *testing.T) {
			opened, signedOut := 0, 0
			m := Model{OnOpenSettings: func() { opened++ }}
			label := i18n.T(i18n.KeySettingsAccountMenu)
			if signedIn {
				m.Account = Account{Name: "Ayase", SignedIn: true, OnOpen: func() { opened++ }, OnSignOut: func() { signedOut++ }}
				label = "Ayase"
			}
			tt := ui.NewTester(func(c *ui.Context) {
				width, _ := c.Size()
				compact := width < 1024
				seatWidth := float32(280)
				if compact {
					seatWidth = 36
				}
				ui.Column(c.Key("footer")).Width(seatWidth).Children(func() {
					accountRow(c, m, theme.Resolve("deepseek.light").Light, ui.RGB(90, 90, 90), compact)
				})
			}, 1200, 300)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			tt.CloseMenu()
			if !tt.Focused(label) {
				t.Fatal("account trigger did not take focus")
			}
			for _, width := range []int{1023, 900, 768, 1024, 1200, 1023} {
				tt.SetSize(width, 300)
				if !tt.Focused(label) {
					t.Fatalf("width%d replaced the focused trigger", width)
				}
				if len(tt.Menu()) != 0 || opened != 0 || signedOut != 0 {
					t.Fatalf("width%d triggered a menu/action during resize", width)
				}
				seat, ok := tt.Find(label)
				wantWidth := float32(252)
				if width < 1024 {
					wantWidth = 36
				}
				if !ok || seat.W != wantWidth {
					t.Fatalf("width%d seat=%+v want width%v", width, seat, wantWidth)
				}
			}
			// An explicit narrow trigger click still opens the menu. Expansion
			// keeps that menu and choosing settings dispatches exactly once.
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			want := []string{i18n.T(i18n.KeySettingsAccountTitle)}
			if signedIn {
				want = append(want, i18n.T(i18n.KeySettingsAccountSignOut))
			}
			if got := tt.Menu(); !slices.Equal(got, want) {
				t.Fatalf("compact menu=%q want%q", got, want)
			}
			tt.SetSize(1024, 300)
			if got := tt.Menu(); !slices.Equal(got, want) || opened != 0 || signedOut != 0 {
				t.Fatalf("expansion changed an open menu or invoked an action: %q", got)
			}
			if err := tt.ChooseMenuItem(i18n.T(i18n.KeySettingsAccountTitle)); err != nil {
				t.Fatal(err)
			}
			if opened != 1 || signedOut != 0 {
				t.Fatalf("settings=%d signout=%d", opened, signedOut)
			}
		})
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

func TestWhaleAssistantNavLabelAndUpdateFooterWidth(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	tmp := t.TempDir()
	t.Setenv(paths.EnvUserData, tmp)
	settingsPath := filepath.Join(paths.DshHome(), "data", "whale", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"name":"Claude Duts"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := whaleAssistantNavLabel(Model{ConnectionReady: false}); got != i18n.T(i18n.KeySidebarAssistant) {
		t.Fatalf("disconnected whaleAssistantNavLabel=%q, want fallback", got)
	}
	if got := whaleAssistantNavLabel(Model{ConnectionReady: true}); got != "Claude Duts" {
		t.Fatalf("connected whaleAssistantNavLabel=%q, want Claude Duts", got)
	}

	const onePixelPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	updateOpened := 0
	m := Model{
		ConnectionReady:    true,
		OnOpenAssistant:    func() {},
		DesktopUpdatePhase: "available",
		OnOpenDesktopUpdate: func() {
			updateOpened++
		},
		Account: Account{
			Name:      "Ayase",
			AvatarURL: onePixelPNG,
			SignedIn:  true,
			OnOpen:    func() {},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)
	if !tt.HasText("Claude Duts") {
		t.Fatalf("missing configured assistant name in sidebar: %q", tt.Texts())
	}
	row, ok := tt.Find("Ayase")
	if !ok || row.W != 212 {
		t.Fatalf("account row with update button=%+v, want width 212", row)
	}
}
