package sidebar

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// Account is the sidebar's bottom account seat, mirroring the original
// AccountMenu trigger (24px avatar, name, menu; 36x36 when the rail is
// collapsed). The zero value still renders the row and its menu so the entry
// points exist before the shell publishes a profile: OnOpen then falls back to
// the settings page, which already owns account details and sign-in.
type Account struct {
	Name                string // display name; empty shows the localized account title
	Detail              string // secondary line, e.g. the masked contact
	AvatarURL           string // optional profile picture URL or data:image/... URI
	SignedIn            bool
	Busy                bool
	OnOpen              func() // account details; nil falls back to the settings entry
	OnSignOut           func() // nil hides the sign-out item (e.g. signed out already)
	OnSignIn            func()
	OnContact           func()
	LauncherActions     func(*ui.Menu)
	RenderLauncherPopup func(*ui.Context, ui.Element)
}

// accountAvatarBitmap reads the account avatar shared with the settings page;
// downloads and decoding run on the media service.
func accountAvatarBitmap(_ *ui.Context, rawURL string) *ui.Bitmap {
	return media.Default().Avatar(rawURL).Bitmap()
}

var accountPersonIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.05" stroke-linecap="round"><circle cx="8" cy="5.4" r="2.6"/><path d="M2.9 13.1c.7-2.4 2.7-3.7 5.1-3.7s4.4 1.3 5.1 3.7"/></svg>`))

func accountSignedOut(a Account) bool {
	return !a.SignedIn && strings.TrimSpace(a.Name) == "" && strings.TrimSpace(a.Detail) == ""
}

// accountLabel is the row's visible title. A profile-less seat is the merged
// settings/account trigger, whose installed label is launcherLabel
// (设置与账号 / Settings & account).
func accountLabel(a Account) string {
	if name := strings.TrimSpace(a.Name); name != "" {
		return name
	}
	if detail := strings.TrimSpace(a.Detail); detail != "" {
		return detail
	}
	return i18n.T(i18n.KeySettingsAccountSeat)
}

// accountInitial is the avatar monogram: the first non-space rune of the name.
func accountInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsSpace(r) {
			continue
		}
		return strings.ToUpper(string(r))
	}
	return ""
}

func accountMenu(a Account, open func()) func(*ui.Menu) {
	return func(menu *ui.Menu) {
		if accountSignedOut(a) {
			menu.Rows(ui.MenuRows{MinHeight: 40, PadV: 8, PadH: 10, Gap: 8, FontSize: fs(14), LineHeight: fs(22), IconSize: 16})
		}
		if open != nil && menu.Item(i18n.T(i18n.KeySettingsOpen)).Key("account:settings").Chosen() {
			open()
		}
		if a.LauncherActions != nil {
			a.LauncherActions(menu)
		}
		if a.OnContact != nil && menu.Item(workspaceText("联系我们", "Contact us")).Key("account:contact").Chosen() {
			a.OnContact()
		}
		if a.SignedIn && a.OnSignOut != nil && menu.Item(i18n.T(i18n.KeySettingsAccountSignOut)).Key("account:signout").Chosen() {
			a.OnSignOut()
		}
		if !a.SignedIn && a.OnSignIn != nil && menu.Item(i18n.T(i18n.KeySettingsAccountSignIn)).Key("account:signin").Chosen() {
			a.OnSignIn()
		}
	}
}

// accountRow paints the account seat. compact is the collapsed rail's 36x36
// control, which keeps the same menu.
func accountRow(c *ui.Context, m Model, compact bool) {
	a := m.Account
	open := m.OnOpenSettings
	if open == nil {
		open = a.OnOpen
	}
	if open == nil && a.OnSignOut == nil && a.OnSignIn == nil {
		return
	}
	t := tok(c)
	label := accountLabel(a)
	signedOut := accountSignedOut(a)
	menuLabel := label
	if signedOut {
		menuLabel = i18n.T(i18n.KeySettingsAccountMenu)
	}
	height := float32(44)
	if compact || signedOut {
		height = 36
	}
	// Installed AccountMenu retains the same trigger/ref when wide changes.
	// Keep its identity so auto-collapse preserves keyboard focus and menu state.
	row := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonMD,
		Disabled: a.Busy,
		Label:    menuLabel,
		Key:      "sidebar-account",
		Content:  func() {},
	}).FillWidth().Height(height).Radius(design.RadiusXL).
		BorderWidth(0).Padding(design.Sp(1.5)).
		Label(menuLabel).Tooltip(menuLabel)
	if !compact {
		maxW := float32(252)
		if sidebarUpdatePhase(m) != "" {
			maxW = 212
		}
		row.Margin(0, design.Sp(0.5), 0, design.Sp(0.5)).MaxWidth(maxW)
	}
	if signedOut && !compact {
		row.Padding(design.Sp(1.75), design.Sp(2), design.Sp(1.75), design.Sp(2))
	}
	row.Menu(accountMenu(a, open))
	if a.RenderLauncherPopup != nil {
		a.RenderLauncherPopup(c, row)
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Justify(ui.Start).Children(func() {
			if signedOut {
				ui.Icon(c, settingsGear).Size(design.IconSM, design.IconSM).TextColor(t.Foreground)
				if compact {
					return
				}
				ui.Text(c, label).Grow(1).FontSize(fs(14)).TextColor(t.Foreground).SingleLine()
				return
			}
			avatar := ui.Box(c).Size(24, 24).Radius(design.RadiusXL).Clip().Center()
			avatar.Background(t.Primary.Alpha(0.12))
			avatar.Children(func() {
				if bmp := accountAvatarBitmap(c, a.AvatarURL); bmp != nil {
					ui.Image(c, bmp).Size(24, 24).Fit(ui.Cover).Radius(design.RadiusXL)
					return
				}
				if initial := accountInitial(a.Name); initial != "" {
					ui.Text(c, initial).FontSize(fs(11)).FontWeight(design.WeightSemibold).TextColor(t.Primary)
					return
				}
				ui.Icon(c, accountPersonIcon).Size(design.IconXS, design.IconXS).TextColor(t.MutedForeground)
			})
			if compact {
				return
			}
			ui.Text(c, label).MinWidth(0).Grow(1).
				FontSize(fs(14)).FixedLineHeight(fs(22)).
				FontWeight(design.WeightNormal).TextColor(t.Foreground).SingleLine()
		})
	})
}
