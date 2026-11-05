package remote

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The installed build's gateway page: a 56-wide section label, a 260-wide
// control column and 64 DIP rows. Values mirror config.Defaults(), which the
// page shows when a caller passes a zero Model (embedded settings tests).
const (
	remoteGatewayDefaultRelay         = "ayase.cn:443"
	remoteGatewayDefaultPort          = 6767
	remoteGatewayDefaultBind          = "127.0.0.1"
	remoteGatewayControlWidth float32 = 260
	remoteGatewayStackWidth   float32 = 480
)

func settingsView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	content := func() {
		ui.Column(c.Key("remote-settings-body")).FillWidth().MinWidth(0).Gap(0).Children(func() {
			if m.SettingsPage == "workspace" {
				ui.Text(c, label("远程工作区", "Remote workspace")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				description, descriptionColor := label("把 SSH 机器变成工作区：本地镜像目录 + SFTP 同步 + 远程文件与命令。", "Turn SSH machines into workspaces: local mirror directories with SFTP sync, remote files and commands."), muted
				if m.WorkspaceToggleError != "" {
					description, descriptionColor = m.WorkspaceToggleError, chrome.Color(colors.Accent)
				}
				ui.Text(c, description).FontSize(theme.InterfaceSize(14)).TextColor(descriptionColor)
				if !m.HarnessReady {
					ui.Text(c, label("连接 Host 后即可管理远程机器。", "Connect to a Host to manage remote machines.")).TextColor(muted)
					return
				}
				if !m.RemoteWorkspaceEnabled {
					ui.Text(c, label("远程工作区尚未启用。", "Remote workspace is disabled.")).TextColor(muted)
					if m.OnToggleWorkspace != nil && widgets.Button(c, colors, label("启用远程工作区", "Enable remote workspace"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: m.WorkspaceToggleBusy}).Clicked() && !m.WorkspaceToggleBusy {
						m.OnToggleWorkspace(true)
					}
					return
				}
				ensureToolsLoaded(c, m)
				viewActiveCard(c, m, border)
				viewMachines(c, m)
				viewSSHConfigCard(c, m, m.MachineEditor, border)
				viewForwardsCard(c, m, m.MachineEditor, border)
				viewAuditCard(c, m, m.MachineEditor, border)
				return
			}
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
				ui.Text(c, i18n.T(KeyPass6RemoteTitle)).FillWidth().FontSize(theme.InterfaceSize(18)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
				ui.Text(c, i18n.T(KeyPass6RemoteIntro)).FillWidth().FontSize(theme.InterfaceSize(13)).
					FixedLineHeight(theme.InterfaceSize(17.3)).TextColor(muted)
			})
			ui.Box(c).FillWidth().Height(14)
			remoteSettingsTabBlock(c, m, fg, muted, border)
		})
	}
	if m.Embedded {
		content()
		return
	}
	ui.Scroll(c).Grow(1).MinHeight(0).Padding(20).Children(content)
}

// remoteSettingsTabBlock paints the two tabs and decides which body follows.
// It owns the tab state when the caller passes no ActiveTab pointer, and keeps
// the gateway tab selected while no channel contribution exists.
func remoteSettingsTabBlock(c *ui.Context, m Model, fg, muted, border ui.Color) {
	active := m.ActiveTab
	tabRow := ui.Row(c.Key("remote-tab-row")).FillWidth().Gap(22).Role(ui.RoleTabList).AlignItems(ui.End)
	if active == nil {
		active = ui.Local(tabRow, "active", func() string { return "gateway" })
	}
	if *active != "channels" || m.ChannelsContent == nil {
		*active = "gateway"
	}
	entries := []struct{ id, title string }{{"gateway", i18n.T(KeyPass6RemoteTabGateway)}}
	if m.ChannelsContent != nil {
		entries = append(entries, struct{ id, title string }{"channels", i18n.T(KeyPass6RemoteTabChannels)})
	}
	buttons := make([]ui.Element, len(entries))
	choose := func(index int) { *active = entries[index].id; buttons[index].Focus() }
	tabRow.Children(func() {
		for index, tab := range entries {
			i, entry := index, tab
			button := ui.Button(c.Key("remote-tab-"+entry.id), "").Label(entry.title).
				Height(36).Padding(7, 1, 9, 1).BorderWidth(0).Role(ui.RoleTab).
				Selected(*active == entry.id).Background(ui.Transparent).
				AlignItems(ui.Center).Justify(ui.Center)
			color, weight := muted, 400
			if *active == entry.id {
				color, weight = fg, 400
			}
			button.Children(func() {
				ui.Text(c, entry.title).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(weight).TextColor(color)
			})
			// The 2 DIP rule sits on the hairline, as the installed page draws it.
			button.Draw(func(p *ui.Painter, r ui.Rect) {
				if *active == entry.id && r.W > 0 && r.H > 0 {
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, fg, 0)
				}
			})
			button.OnShortcut(0, ui.KeyRight, func() { choose((i + 1) % len(entries)) })
			button.OnShortcut(0, ui.KeyLeft, func() { choose((i + len(entries) - 1) % len(entries)) })
			button.OnShortcut(0, ui.KeyHome, func() { choose(0) })
			button.OnShortcut(0, ui.KeyEnd, func() { choose(len(entries) - 1) })
			buttons[i] = button
			if button.Clicked() {
				*active = entry.id
			}
		}
	})
	ui.Box(c).FillWidth().Height(.667).Background(border)
	if *active == "channels" {
		ui.Box(c).FillWidth().Height(14)
		m.ChannelsContent(c)
		return
	}
	ui.Box(c).FillWidth().Height(14)
	gatewaySettingsBody(c, m, fg, muted)
}

func gatewaySettingsBody(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	e := m.GatewayEditor
	if m.OnSaveGateway == nil || e == nil {
		ui.Text(c, i18n.T(KeyPass6RemoteUnavailable)).FillWidth().FontSize(theme.InterfaceSize(12)).
			FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		return
	}
	ui.Column(c.Key("gateway-settings")).FillWidth().MinWidth(0).Gap(14).Children(func() {
		gatewaySettingsFields(c, m, fg, muted)
		gatewayRefreshRow(c, m, e, fg, muted, colors)
		// Device actions finish asynchronously in the runtime; show their real
		// progress/result for this page's rotate action only.
		if action := m.PhoneRuntime.DeviceAction; action.Operation == "rotate" && (action.Busy || action.Error != "" || action.Message != "") {
			text, color := action.Message, muted
			switch {
			case action.Busy:
				text = label("正在更新配对码…", "Renewing the pairing code…")
			case action.Error != "":
				text, color = action.Error, chrome.Color(colors.Accent)
			}
			ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
				TextColor(color).Padding(10, 0, 10, 0)
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
				TextColor(fg).Padding(10, 0, 10, 0)
			return
		}
		// The application reports a committed save here; runtime status words are
		// already shown live by the pairing surface above, so only a real action
		// message is echoed.
		if msg := strings.TrimSpace(m.PhoneActionMsg); msg != "" && !runtimeStatusWord(msg) {
			ui.Text(c, msg).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
				TextColor(muted).Padding(10, 0, 10, 0)
		}
	})
}

// gatewaySettingsFields renders the gateway rows in the installed order. Each
// row commits only its own field, so an unrelated text draft never rides along
// with a mode or bind change.
func gatewaySettingsFields(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	e := m.GatewayEditor
	if e == nil {
		return
	}
	current := remoteGatewayCurrent(m)
	if !e.Open {
		e.Expected, e.Draft = current, current
		e.Draft.RelayEndpoint = remoteGatewayRelayDisplay(current.RelayEndpoint)
	}
	busy := m.PhoneRuntime.Status == "starting" || m.PhoneRuntime.DeviceAction.Busy
	save := func(next config.RemoteGatewayPreferences) {
		if err := m.OnSaveGateway(current, next); err != nil {
			e.Error = err.Error()
			return
		}
		e.Error = ""
		e.Expected = next
		if next.Mode != current.Mode {
			e.Draft.Mode = next.Mode
		}
		if next.BindAddress != current.BindAddress {
			e.Draft.BindAddress = next.BindAddress
		}
		if next.RelayTLS != current.RelayTLS {
			e.Draft.RelayTLS = next.RelayTLS
		}
		if next.RelayEndpoint != current.RelayEndpoint {
			e.Draft.RelayEndpoint = remoteGatewayRelayDisplay(next.RelayEndpoint)
		}
		if next.Port != current.Port {
			e.Draft.Port = next.Port
		}
		e.Open = strings.TrimSpace(e.Draft.RelayEndpoint) != remoteGatewayRelayDisplay(next.RelayEndpoint) ||
			strings.TrimSpace(e.Draft.Port) != next.Port
	}

	gatewaySettingRow(c, "gateway-relay", i18n.T(KeyPass6RemoteRelay), i18n.T(KeyPass6RemoteRelayHint), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.RelayEndpoint).Label(i18n.T(KeyPass6RemoteRelayLabel)).
			FillWidth().Height(32).Radius(8).Background(theme.Color(colors.SelectorFill)).Border(.667, theme.Color(colors.Border)).Padding(0, 10).FontSize(theme.InterfaceSize(13)).Disabled(busy).Changed() {
			e.Open = true
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			commit := widgets.Button(c, colors, i18n.T(i18n.KeySurfacesSettingsSave),
				widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: busy})
			commit.Label(i18n.T(KeyPass6RemoteSaveRelay))
			if commit.Clicked() {
				next := current
				next.RelayEndpoint = strings.TrimSpace(e.Draft.RelayEndpoint)
				if next.RelayEndpoint == "" || next.RelayEndpoint == remoteGatewayDefaultRelay {
					next.RelayTLS = true
				}
				save(next)
			}
			reset := widgets.Button(c, colors, i18n.T(KeyPass6RemoteRestoreDefault),
				widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: busy})
			if reset.Clicked() {
				next := current
				next.RelayEndpoint = ""
				next.RelayTLS = true
				save(next)
			}
		})
	})

	gatewayLegacySection(c, m, e, fg, muted, colors)

	gatewaySettingRow(c, "gateway-mode", i18n.T(KeyPass6RemoteMode), i18n.T(KeyPass6RemoteModeHint), fg, muted, func() {
		active := -1
		switch current.Mode {
		case "lan":
			active = 0
		case "relay":
			active = 1
		}
		labels := []string{i18n.T(KeyPass6RemoteModeLAN), i18n.T(KeyPass6RemoteModeServer)}
		if picked := remoteGatewaySegmented(c, "gateway-mode", labels, active, busy); picked >= 0 {
			mode := "lan"
			if picked == 1 {
				mode = "relay"
			}
			if mode != current.Mode {
				next := current
				next.Mode = mode
				save(next)
			}
		}
	})

	gatewaySettingRow(c, "gateway-port", i18n.T(KeyPass6RemotePort), i18n.T(KeyPass6RemotePortHint), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.Port).Label(i18n.T(KeyPass6RemotePortLabel)).
			FillWidth().Height(32).Radius(8).Background(theme.Color(colors.SelectorFill)).Border(.667, theme.Color(colors.Border)).Padding(0, 10).FontSize(theme.InterfaceSize(13)).Disabled(busy).Changed() {
			e.Open = true
		}
		ui.Row(c).FillWidth().MinWidth(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			commit := widgets.Button(c, colors, i18n.T(i18n.KeySurfacesSettingsSave),
				widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: busy})
			commit.Label(i18n.T(KeyPass6RemoteSavePort))
			if commit.Clicked() {
				next := current
				next.Port = strings.TrimSpace(e.Draft.Port)
				save(next)
			}
		})
	})

	gatewaySettingRow(c, "gateway-bind", i18n.T(KeyPass6RemoteBind), gatewayBindHint(current.BindAddress, m.PhoneRuntime.BindAddresses), fg, muted, func() {
		bind := gatewayBindAddress(current.BindAddress)
		selected := gatewayBindLabel(bind, m.PhoneRuntime.BindAddresses)
		labels, values := gatewayBindOptions(bind, m.PhoneRuntime.BindAddresses)
		if widgets.PillSelect(c, colors, &selected, labels, 18).Label(i18n.T(KeyPass6RemoteBindLabel)).AlignSelf(ui.End).Disabled(busy).Changed() {
			if address, ok := values[selected]; ok && address != bind {
				next := current
				next.BindAddress = address
				save(next)
			}
		}
	})

	gatewaySettingRow(c, "gateway-encryption", i18n.T(KeyPass6RemoteEncryption), i18n.T(KeyPass6RemoteEncryptionHint), fg, muted, nil)
}

// gatewayRefreshRow is the last row: the QR code is renewed without touching
// paired devices, so it is a text action rather than a destructive control.
func gatewayRefreshRow(c *ui.Context, m Model, e *GatewayEditor, fg, muted ui.Color, colors theme.Colors) {
	if m.OnRotateToken == nil {
		return
	}
	busy := m.PhoneRuntime.Status == "starting" || m.PhoneRuntime.DeviceAction.Busy
	gatewaySettingRow(c, "gateway-refresh", i18n.T(KeyPass6RemoteRefresh), i18n.T(KeyPass6RemoteRefreshHint), fg, muted, func() {
		ui.Row(c).FillWidth().MinWidth(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			refresh := widgets.Button(c, colors, i18n.T(KeyPass6RemoteRefreshAction),
				widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: busy})
			if refresh.Clicked() {
				if err := m.OnRotateToken(); err != nil {
					e.Error = err.Error()
				} else {
					e.Error = ""
				}
			}
		})
	})
}

// gatewayLegacySection keeps the old relay credential out of the main flow: a
// muted section label exists only while such a token is stored, and opening it
// reveals the one destructive action.
func gatewayLegacySection(c *ui.Context, m Model, e *GatewayEditor, fg, muted ui.Color, colors theme.Colors) {
	if !m.LegacyRelayTokenSet {
		return
	}
	anchor := ui.Column(c.Key("gateway-legacy-section")).FillWidth().MinWidth(0).Gap(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	anchor.Children(func() {
		toggle := ui.Button(c.Key("gateway-legacy-toggle"), "").Label(i18n.T(KeyPass6RemoteLegacySection)).
			FillWidth().Height(24).Padding(0).Gap(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center)
		toggle.Children(func() {
			ui.Box(c).Size(16, 16).Shrink(0).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
				var path ui.Path
				if *open {
					path.MoveTo(r.X+4, r.Y+6).LineTo(r.X+8, r.Y+10).LineTo(r.X+12, r.Y+6)
				} else {
					path.MoveTo(r.X+6, r.Y+4).LineTo(r.X+10, r.Y+8).LineTo(r.X+6, r.Y+12)
				}
				p.StrokePath(&path, 1.2, muted)
			})
			ui.Text(c, i18n.T(KeyPass6RemoteLegacySection)).FontSize(theme.InterfaceSize(13)).
				FixedLineHeight(theme.InterfaceSize(24)).TextColor(muted).Grow(1).MinWidth(0)
		})
		if toggle.Hovered() {
			toggle.Background(c.Theme().SurfaceHover)
		}
		if toggle.Clicked() {
			*open = !*open
		}
		if !*open {
			return
		}
		gatewaySettingRow(c, "gateway-legacy", i18n.T(KeyPass6RemoteLegacyTitle), i18n.T(KeyPass6RemoteLegacyHint), fg, muted, func() {
			if m.OnSetRelayToken == nil {
				ui.Text(c, i18n.T(KeyPass6RemoteLegacyNoCallback)).FillWidth().FontSize(theme.InterfaceSize(12)).
					FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				return
			}
			ui.Row(c).FillWidth().MinWidth(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
				clear := widgets.Button(c, colors, i18n.T(KeyPass6RemoteLegacyClear),
					widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall})
				if clear.Clicked() {
					if err := m.OnSetRelayToken(""); err != nil {
						e.Error = err.Error()
					} else {
						e.Error = ""
					}
				}
			})
		})
	})
}

// gatewaySettingRow lays out one gateway setting: the title and its description
// on the left, a 260 DIP control column on the right, 10 DIP above and below.
// Under 480 DIPs the control moves below the text instead of squeezing it.
func gatewaySettingRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, control func()) ui.Element {
	root := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	width := root.Bounds().W
	root.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	textColumn := func() {
		column := ui.Column(c).MinWidth(0).Gap(4)
		if control != nil && width >= remoteGatewayStackWidth {
			column.Grow(1)
		} else {
			column.FillWidth()
		}
		column.Children(func() {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(13)).
				FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
			if hint != "" {
				ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).
					FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			}
		})
	}
	controlColumn := func() {
		if control == nil {
			return
		}
		column := ui.Column(c).MinWidth(0).Gap(8).AlignItems(ui.End)
		if width >= remoteGatewayStackWidth {
			column.Width(gatewayControlWidth(key)).Shrink(0)
		} else {
			column.FillWidth()
		}
		column.Children(control)
	}
	root.Children(func() {
		if control != nil && width >= remoteGatewayStackWidth {
			ui.Row(c).FillWidth().MinWidth(0).Gap(16).AlignItems(ui.Start).Children(func() {
				textColumn()
				controlColumn()
			})
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			textColumn()
			controlColumn()
		})
	})
	return root
}

// remoteGatewaySegmented is the installed page's two-part switch: equal
// segments, transparent except the chosen one, which is filled black. It
// reports the picked index, or -1 when the frame had no click.
func remoteGatewaySegmented(c *ui.Context, key string, labels []string, active int, disabled bool) int {
	const height float32 = 28
	root := ui.Row(c.Key(key)).Width(gatewayControlWidth(key)).Height(height).Shrink(0).Gap(8).AlignItems(ui.Stretch)
	picked := -1
	root.Children(func() {
		for index, caption := range labels {
			i, text := index, caption
			segment := ui.Button(c.Key(key+"-"+strconv.Itoa(i)), "").Label(text).
				Width((gatewayControlWidth(key) - 8) / 2).Height(height).Padding(0).Radius(8).
				BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center)
			if disabled {
				segment = segment.Disabled(true).Opacity(0.5)
			}
			labelColor := c.Theme().Text
			if i == active {
				segment.Background(labelColor)
				labelColor = c.Theme().Background
			}
			segment.Children(func() {
				ui.Text(c, text).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(labelColor)
			})
			if segment.Clicked() && !disabled {
				picked = i
			}
		}
	})
	return picked
}

// remoteGatewayCurrent turns the model into the persisted preferences the
// gateway editor compares against, filling the defaults a zero Model omits.
func remoteGatewayCurrent(m Model) config.RemoteGatewayPreferences {
	port := m.RemotePort
	if port < 1024 || port > 65535 {
		port = remoteGatewayDefaultPort
	}
	return config.RemoteGatewayPreferences{
		Mode:          m.RemoteMode,
		Port:          strconv.Itoa(port),
		BindAddress:   gatewayBindAddress(m.RemoteBindAddress),
		RelayEndpoint: m.RemoteRelayEndpoint,
		RelayTLS:      m.RemoteRelayUseTLS,
	}
}

// remoteGatewayRelayDisplay is the effective endpoint: the built-in relay is
// shown even when the stored value is blank, so the field always reads a host.
func remoteGatewayRelayDisplay(endpoint string) string {
	if endpoint = strings.TrimSpace(endpoint); endpoint == "" {
		return remoteGatewayDefaultRelay
	}
	return endpoint
}

func gatewayBindAddress(address string) string {
	if address = strings.TrimSpace(address); address == "" {
		return remoteGatewayDefaultBind
	}
	return address
}

// gatewayBindHint describes the chosen scope, in the installed build's words.
func gatewayBindHint(address string, detected []string) string {
	address = gatewayBindAddress(address)
	switch address {
	case "127.0.0.1":
		return i18n.T(KeyPass6RemoteBindHintLocal)
	case "0.0.0.0":
		return i18n.T(KeyPass6RemoteBindHintLAN)
	}
	for _, current := range detected {
		if current == address {
			return i18n.T(KeyPass6RemoteBindHintNIC)
		}
	}
	if detected != nil {
		return i18n.T(KeyPass6RemoteBindHintStale)
	}
	return i18n.T(KeyPass6RemoteBindHintNIC)
}

// gatewayBindOptions lists loopback, the whole LAN and every detected NIC,
// keeping a saved address selectable even when the scan no longer reports it.
func gatewayBindOptions(bind string, detected []string) (labels []string, values map[string]string) {
	options := []string{remoteGatewayDefaultBind, "0.0.0.0"}
	for _, address := range append(append([]string(nil), detected...), bind) {
		present := false
		for _, option := range options {
			if option == address {
				present = true
				break
			}
		}
		if !present && address != "" {
			options = append(options, address)
		}
	}
	labels = make([]string, len(options))
	values = make(map[string]string, len(options))
	for i, address := range options {
		labels[i] = gatewayBindLabel(address, detected)
		values[labels[i]] = address
	}
	return labels, values
}

// A detected empty list differs from an unknown scan. Saved per-NIC choices
// remain selectable, with the same stale-address feedback as the original UI.
func gatewayBindLabel(address string, detected []string) string {
	switch address {
	case "0.0.0.0":
		return label("整个局域网", "Whole LAN")
	case "127.0.0.1":
		return label("仅本机", "This machine only")
	}
	if detected != nil {
		for _, current := range detected {
			if current == address {
				return address
			}
		}
		return address + " · " + label("已失效", "unavailable")
	}
	return address
}

// runtimeStatusWord reports whether msg is one of the phone runtime's own state
// words (possibly with a detail suffix). Those are rendered live by
// viewPhoneSection, so echoing the boot-time copy here would go stale.
func runtimeStatusWord(msg string) bool {
	head := strings.TrimSpace(strings.SplitN(msg, ":", 2)[0])
	switch head {
	case "starting", "ready", "offline", "disabled", "missing", "expired", "pairing-unavailable", "relay-offline", "updating", "stopped":
		return true
	}
	return false
}

// gatewayControlWidth is the control column of a gateway row in the 0.3.13
// page: 220 for the text fields, 320 for the two-part switch, 200 otherwise.
func gatewayControlWidth(key string) float32 {
	switch key {
	case "gateway-relay", "gateway-port":
		return 220
	case "gateway-mode":
		return 320
	}
	return 200
}
