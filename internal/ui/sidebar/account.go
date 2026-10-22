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

// accountLabel is the row's accessible name and, when the shell published no
// profile, its visible title too.
func accountLabel(a Account) string {
	if name := strings.TrimSpace(a.Name); name != "" {
		return name
	}
	return i18n.T(i18n.KeySettingsAccountTitle)
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
	height := float32(44) // AccountMenu.module.css .trigger
	key := "sidebar-account"
	if compact {
		height = 36 // AccountMenu.module.css [data-collapsed='true']
		key = "sidebar-account-rail"
	}
	row := ui.Button(c.Key(key), "").FillWidth().Height(height).Radius(12).
		BorderWidth(0).Background(ui.Transparent).Padding(6).
		Label(label).Tooltip(label).Disabled(a.Busy)
	if row.Hovered() {
		row.Background(chrome.Color(colors.Hover))
	}
	row.Menu(accountMenu(a, open))
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.Start).Children(func() {
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
