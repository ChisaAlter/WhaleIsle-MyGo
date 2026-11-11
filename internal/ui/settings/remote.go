package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// RemoteAccess configures the dedicated Remote control & pairing settings section.
type RemoteAccess struct {
	Enabled               bool
	Mode                  string
	RelayHost             string
	Port                  int
	BindScope             string
	RelayConnecting       bool
	PairingURL            string
	PairingRefreshSeconds int
	PersistentLink        bool
	ConfirmRevoke         bool
	LinkRevoked           bool
	Status                string
	Err                   string
	OnToggleEnabled       func(bool)
	OnModeChange          func(string)
	OnCopyLink            func()
	OnRefreshPairing      func()
	OnRevokeLink          func()
}

func remotePane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Remote == nil {
		return
	}
	r := m.Remote
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	root := ui.Column(c.Key("settings-remote-pane")).FillWidth().MaxWidth(640).MinWidth(0).Gap(16)
	confirming := ui.Local(root, "confirm-revoke", func() bool { return r.ConfirmRevoke })
	revoked := ui.Local(root, "link-revoked", func() bool { return r.LinkRevoked })
	if r.ConfirmRevoke {
		*confirming = true
	}
	if r.LinkRevoked {
		*revoked = true
	}

	root.Children(func() {
		ui.Text(c, settingsVendorLocale("dsh-client-ui-settings-remote", "heading")).FillWidth().FontSize(theme.InterfaceSize(18)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
		ui.Text(c, settingsVendorLocale("dsh-client-ui-settings-remote", "intro")).FillWidth().FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(19.5)).TextColor(muted)
		settingsNotice(c, "remote-pane-err", "error", r.Err != "", func() string { return r.Err })

		mode := strings.TrimSpace(r.Mode)
		if mode == "" {
			mode = "relay"
		}
		bodyPreferenceRow(c, "remote-mode", settingsVendorLocale("dsh-client-ui-settings-remote", "mode"), settingsVendorLocale("dsh-client-ui-settings-remote", "modeHint"), fg, muted, func() {
			ui.Row(c).Gap(8).Children(func() {
				for _, opt := range []struct{ id, label string }{
					{"lan", settingsVendorLocale("dsh-client-ui-settings-remote", "modeLan")},
					{"relay", settingsVendorLocale("dsh-client-ui-settings-remote", "modeRelay")},
				} {
					selected := mode == opt.id
					variant := widgets.ButtonOutline
					if selected {
						variant = widgets.ButtonPrimary
					}
					if widgets.Button(c, colors, opt.label, widgets.ButtonSpec{Variant: variant, Size: widgets.ButtonSmall}).Clicked() {
						r.Mode = opt.id
						if r.OnModeChange != nil {
							r.OnModeChange(opt.id)
						}
					}
				}
			})
		})

		if r.RelayConnecting {
			ui.Text(c, i18n.T(i18n.KeySettingsParityRemoteRelayConnectingHint)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		} else if mode == "relay" && strings.TrimSpace(r.RelayHost) != "" {
			ui.Text(c, fmt.Sprintf("%s: %s", settingsVendorLocale("dsh-client-ui-settings-remote", "relayHost"), strings.TrimSpace(r.RelayHost))).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		}

		seconds := r.PairingRefreshSeconds
		if seconds <= 0 {
			seconds = 60
		}
		if r.PairingURL != "" || r.PairingRefreshSeconds > 0 || r.Enabled {
			ui.Column(c.Key("remote-pairing-section")).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, settingsVendorLocale("dsh-client-ui-settings-remote", "pairingUrl")).FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
				if r.PairingURL != "" {
					ui.Text(c, r.PairingURL).FillWidth().SingleLine().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				ui.Text(c, i18n.T(i18n.KeySettingsParityRemotePairingLifetimeHint)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				ui.Text(c, i18n.SettingsParityFormat(i18n.KeySettingsParityRemotePairingCountdownHint, "seconds", fmt.Sprintf("%d", seconds))).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				ui.Row(c).Wrap().Gap(8).Children(func() {
					if widgets.Button(c, colors, settingsVendorLocale("dsh-client-ui-settings-remote", "copyLink"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: r.OnCopyLink == nil && r.PairingURL == ""}).Clicked() {
						if r.OnCopyLink != nil {
							r.OnCopyLink()
						}
					}
					if widgets.Button(c, colors, settingsVendorLocale("dsh-client-ui-settings-remote", "rotateToken"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: r.OnRefreshPairing == nil && !r.Enabled}).Clicked() {
						if r.OnRefreshPairing != nil {
							r.OnRefreshPairing()
						}
					}
				})
			})
		}

		if r.PersistentLink || r.PairingURL != "" || r.OnRevokeLink != nil || *confirming || *revoked {
			ui.Column(c.Key("remote-persistent-link-section")).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, i18n.T(i18n.KeySettingsParityRemotePersistentLinkHint)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				ui.Text(c, i18n.T(i18n.KeySettingsParityRemotePersistentLinkWindowHint)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				if !*confirming {
					if widgets.Button(c, colors, i18n.T(i18n.KeySettingsParityRemoteRevokeLink), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
						*confirming = true
						r.ConfirmRevoke = true
						*revoked = false
						r.LinkRevoked = false
					}
				} else {
					ui.Text(c, i18n.T(i18n.KeySettingsParityRemoteRevokeLinkConfirm)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg)
					ui.Text(c, i18n.T(i18n.KeySettingsParityRemoteRevokeLinkDisconnect)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg)
					ui.Row(c).Wrap().Gap(8).Children(func() {
						if widgets.Button(c, colors, i18n.T(i18n.KeySettingsParityRemoteRevokeLinkAction), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
							*confirming = false
							r.ConfirmRevoke = false
							*revoked = true
							r.LinkRevoked = true
							if r.OnRevokeLink != nil {
								r.OnRevokeLink()
							}
						}
						if widgets.Button(c, colors, settingsVendorLocale("dsh-client-ui-settings-remote", "renameCancel"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
							*confirming = false
							r.ConfirmRevoke = false
						}
					})
				}
				if *revoked {
					ui.Text(c, i18n.T(i18n.KeySettingsParityRemoteLinkRevoked)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
					ui.Text(c, i18n.T(i18n.KeySettingsParityRemoteLinkRevokedShort)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				}
			})
		}
	})
}
