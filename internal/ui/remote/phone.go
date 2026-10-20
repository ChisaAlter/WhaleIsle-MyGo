package remote

import (
	"strconv"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// phoneSnapshot builds AssessPhoneRemote status from the panel model.
func phoneSnapshot(m Model) phoneremote.Snapshot {
	names := append([]string(nil), m.RemoteDeviceNames...)
	count := m.RemoteDeviceCount
	if m.PhoneRuntime.DevicesLoaded {
		names = make([]string, len(m.PhoneRuntime.Devices))
		for i, device := range m.PhoneRuntime.Devices {
			names[i] = device.Name
		}
		count = len(names)
	}
	if count == 0 && len(names) > 0 {
		count = len(names)
	}
	s := phoneremote.Snapshot{
		RemoteEnabled:       m.RemoteEnabled,
		RemoteMode:          m.RemoteMode,
		RemotePort:          m.RemotePort,
		RemoteBindAddress:   m.RemoteBindAddress,
		RemoteLanTLS:        m.RemoteLanTls,
		RemoteAppBaseURL:    m.RemoteAppBaseURL,
		RemoteRelayEndpoint: m.RemoteRelayEndpoint,
		RemoteRelayUseTLS:   m.RemoteRelayUseTLS,
		HasRemoteToken:      m.HasRemoteToken,
		HasRemoteRelayToken: m.HasRemoteRelayToken,
		MuxReady:            m.HarnessReady && m.MuxConnected,
		DeviceCount:         count,
		DeviceNames:         names,
	}
	s.Readiness, s.Missing = phoneremote.Assess(s)
	if m.GatewayProtocol == "chisacode-v2" {
		// V2 uses the daemon's device store and short-lived offers. Legacy
		// desktop tokens and the main-window mux are not pairing prerequisites.
		s.Missing = nil
		s.Readiness = phoneremote.ReadinessIncomplete
		if !s.RemoteEnabled {
			s.Readiness = phoneremote.ReadinessDisabled
		} else if m.PhoneRuntime.Status == "ready" && m.PhoneRuntime.Listening && m.PhoneRuntime.RelayConnected {
			s.Readiness = phoneremote.ReadinessReady
		}
	}
	s.ConnectURL = phoneremote.ConnectURL(s)
	return s
}

func phoneReadinessLabel(r phoneremote.Readiness) string {
	switch r {
	case phoneremote.ReadinessReady:
		return i18n.T(i18n.KeyRemotePhoneReady)
	case phoneremote.ReadinessIncomplete:
		return i18n.T(i18n.KeyRemotePhoneIncomplete)
	case phoneremote.ReadinessDisabled:
		return i18n.T(i18n.KeyRemotePhoneDisabled)
	default:
		return i18n.T(i18n.KeyRemotePhoneIncomplete)
	}
}

// viewPhoneSection paints the M11 phone remote panel (status, URL, devices, actions).
func viewPhoneSection(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	snap := phoneSnapshot(m)

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneTitle)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeyRemotePhoneHowTo)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)

	rows := []struct{ label, value string }{
		{i18n.T(i18n.KeyRemotePhoneStatus), phoneReadinessLabel(snap.Readiness)},
		{i18n.T(i18n.KeyRemotePairingEnabled), onOff(snap.RemoteEnabled)},
		{i18n.T(i18n.KeyRemoteMode), emptyDash(snap.RemoteMode)},
	}
	if m.GatewayProtocol != "chisacode-v2" {
		rows = append(rows,
			struct{ label, value string }{i18n.T(i18n.KeyRemoteCredentials), credLabel(snap.HasRemoteToken)},
			struct{ label, value string }{i18n.T(i18n.KeyRemotePhoneRelayToken), credLabel(snap.HasRemoteRelayToken)},
			struct{ label, value string }{i18n.T(i18n.KeyRemoteMux), muxLabel(m.HarnessReady, m.MuxConnected)},
		)
	} else {
		rows = append(rows,
			struct{ label, value string }{label("网关监听", "Gateway listening"), onOff(m.PhoneRuntime.Listening)},
			struct{ label, value string }{label("中继连接", "Relay connection"), onOff(m.PhoneRuntime.RelayConnected)},
		)
	}
	mode := strings.ToLower(strings.TrimSpace(snap.RemoteMode))
	if mode == "lan" || strings.Contains(mode, "lan") {
		rows = append(rows,
			struct{ label, value string }{i18n.T(i18n.KeyRemotePhoneLanBind), emptyDash(snap.RemoteBindAddress)},
			struct{ label, value string }{i18n.T(i18n.KeyRemotePort), strconv.Itoa(snap.RemotePort)},
			struct{ label, value string }{i18n.T(i18n.KeyRemotePhoneLanTLS), onOff(snap.RemoteLanTLS)},
		)
	}
	rows = append(rows,
		struct{ label, value string }{i18n.T(i18n.KeyRemoteMobileAppBase), emptyDash(snap.ConnectURL)},
		struct{ label, value string }{i18n.T(i18n.KeyRemoteMobileDevices), strconv.Itoa(snap.DeviceCount)},
	)

	ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
		for _, row := range rows {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, row.label).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Width(160)
				ui.Text(c, row.value).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
			})
		}
	})

	viewPhoneDevices(c, m, fg, muted, field, border, accent)

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneConnectURL)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	if snap.ConnectURL != "" {
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			ui.Text(c, snap.ConnectURL).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		})
	} else {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneURLUnavailable)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}

	runtime := m.PhoneRuntime
	status := runtime.Status
	if !m.RemoteEnabled {
		status = "disabled"
	}
	if status == "" {
		status = "offline"
	}
	ui.Text(c, "offer v2 · "+status).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	if runtime.Message != "" {
		ui.Text(c, runtime.Message).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) {
		ui.Text(c, i18n.T(i18n.KeyMiscRemotePairingLinkValid)+runtime.ExpiresAt.Local().Format("15:04:05")).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		if runtime.PairingImage != nil {
			ui.Image(c, ui.NewBitmap(runtime.PairingImage)).Width(320).Height(320).Fit(ui.Contain)
			ui.Text(c, i18n.T(i18n.KeyMiscRemoteScanPrivatelyToPairYourPhone)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else {
			ui.Text(c, i18n.T(i18n.KeyMiscRemoteQrUnavailableCopyTheTemporaryLinkP)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	} else {
		ui.Text(c, i18n.T(i18n.KeyMiscRemoteNoUsableOfferEnablingStartsTheInst)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Row(c).Gap(8).Children(func() {
		if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) && m.OnCopyConnectURL != nil && ui.Button(c, i18n.T(i18n.KeyMiscRemoteCopyPairingLink)).Clicked() {
			m.OnCopyConnectURL()
		}
		if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) && m.OnOpenConnectURL != nil && ui.Button(c, i18n.T(i18n.KeyMiscRemoteOpenPairingLink)).Clicked() {
			m.OnOpenConnectURL()
		}
		if m.OnRefreshPhone != nil && ui.Button(c, i18n.T(i18n.KeyRemotePhoneRefresh)).Disabled(runtime.DeviceAction.Busy).Clicked() {
			m.OnRefreshPhone()
		}
	})
	if m.PhoneActionMsg != "" {
		ui.Text(c, m.PhoneActionMsg).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}

	if len(snap.Missing) > 0 {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneMissing)+": "+strings.Join(snap.Missing, ", ")).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
	}
	if m.GatewayProtocol != "chisacode-v2" && snap.RemoteEnabled && !snap.HasRemoteToken {
		ui.Text(c, i18n.T(i18n.KeyRemoteMobileNeedToken)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
	}
}
