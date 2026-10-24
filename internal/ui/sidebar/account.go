package sidebar

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Account is the sidebar's bottom account seat, mirroring the original
// AccountMenu trigger (24px avatar, name, menu; 36x36 when the rail is
// collapsed). The zero value still renders the row and its menu so the entry
// points exist before the shell publishes a profile: OnOpen then falls back to
// the settings page, which already owns account details and sign-in.
type Account struct {
	Name      string // display name; empty shows the localized account title
	Detail    string // secondary line, e.g. the masked contact
	SignedIn  bool
	Busy      bool
	OnOpen    func() // account details; nil falls back to the settings entry
	OnSignOut func() // nil hides the sign-out item (e.g. signed out already)
}

var accountPersonIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"><circle cx="8" cy="5.4" r="2.6"/><path d="M2.9 13.1c.7-2.4 2.7-3.7 5.1-3.7s4.4 1.3 5.1 3.7"/></svg>`))

// accountLabel is the row's visible title. A profile-less seat is the merged
// settings/account trigger, whose installed label is launcherLabel
// (设置与账号 / Settings & account).
func accountLabel(a Account) string {
	if name := strings.TrimSpace(a.Name); name != "" {
		return name
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
		if open != nil && menu.Item(i18n.T(i18n.KeySettingsAccountTitle)).Chosen() {
			open()
		}
		if a.OnSignOut != nil && menu.Item(i18n.T(i18n.KeySettingsAccountSignOut)).Chosen() {
			a.OnSignOut()
		}
	}
}

// accountRow paints the account seat. compact is the collapsed rail's 36x36
// control, which keeps the same menu.
func accountRow(c *ui.Context, m Model, colors theme.Colors, muted ui.Color, compact bool) {
	a := m.Account
	open := a.OnOpen
	if open == nil {
		open = m.OnOpenSettings
	}
	if open == nil && a.OnSignOut == nil {
		return
	}
	label := accountLabel(a)
	signedOut := strings.TrimSpace(a.Name) == ""
	// The native trigger's accessible name stays 账号菜单 even when the signed-out
	// label reads launcherLabel.
	menuLabel := label
	if signedOut {
		menuLabel = i18n.T(i18n.KeySettingsAccountMenu)
	}
	height := float32(44) // AccountMenu.module.css .trigger
	key := "sidebar-account"
	if compact {
		height = 36 // AccountMenu.module.css [data-collapsed='true']
		key = "sidebar-account-rail"
	} else if signedOut {
		// Installed AccountMenu trigger[data-signed-out='true']: 36 DIP tall with
		// 7/8 padding, replacing the avatar with a 16 DIP settings glyph whose
		// label is launcherLabel (设置与账号 / Settings & account).
		height = 36
	}
	row := ui.Button(c.Key(key), "").FillWidth().Height(height).Radius(12).
		BorderWidth(0).Background(ui.Transparent).Padding(6).
		Label(menuLabel).Tooltip(menuLabel).Disabled(a.Busy)
	if !compact {
		// The footer seat sits on the page's 14-DIP inset with the native 252-DIP
		// width, as the 账号菜单 button does in the installed page.
		row.Margin(0, 2, 0, 2).MaxWidth(252)
	}
	if signedOut && !compact {
		// Installed AccountMenu trigger[data-signed-out='true']: 7/8 padding.
		row.Padding(7, 8, 7, 8)
	}
	if row.Hovered() {
		row.Background(chrome.Color(colors.Hover))
	}
	row.Menu(accountMenu(a, open))
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.Start).Children(func() {
			if signedOut {
				// The trigger swaps the avatar for a 16 DIP settings glyph.
				ui.Icon(c, settingsGear).Size(16, 16).TextColor(chrome.Color(colors.FG))
				if compact {
					return
				}
				ui.Text(c, label).Grow(1).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.FG)).SingleLine()
				return
			}
			avatar := ui.Box(c).Size(24, 24).Radius(12).Clip().Center()
			avatar.Background(chrome.Color(colors.Accent).Alpha(.12))
			avatar.Children(func() {
				if initial := accountInitial(a.Name); initial != "" {
					ui.Text(c, initial).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(chrome.Color(colors.Accent))
					return
				}
				ui.Icon(c, accountPersonIcon).Size(14, 14).TextColor(muted)
			})
			if compact {
				return
			}
			ui.Column(c).MinWidth(0).Grow(1).Gap(0).Children(func() {
				ui.Text(c, label).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(chrome.Color(colors.FG)).SingleLine()
				if detail := strings.TrimSpace(a.Detail); detail != "" {
					ui.Text(c, detail).FontSize(theme.InterfaceSize(11)).TextColor(muted).SingleLine()
				}
			})
		})
	})
}
