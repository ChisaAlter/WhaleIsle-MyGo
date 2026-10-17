package remote

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// phoneSnapshot builds AssessPhoneRemote status from the panel model.
func phoneSnapshot(m Model) phoneremote.Snapshot {
	names := append([]string(nil), m.RemoteDeviceNames...)
	count := m.RemoteDeviceCount
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

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneTitle)).FontSize(12).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeyRemotePhoneHowTo)).FontSize(11).TextColor(muted)

	rows := []struct{ label, value string }{
		{i18n.T(i18n.KeyRemotePhoneStatus), phoneReadinessLabel(snap.Readiness)},
		{i18n.T(i18n.KeyRemotePairingEnabled), onOff(snap.RemoteEnabled)},
		{i18n.T(i18n.KeyRemoteMode), emptyDash(snap.RemoteMode)},
		{i18n.T(i18n.KeyRemoteCredentials), credLabel(snap.HasRemoteToken)},
		{i18n.T(i18n.KeyRemotePhoneRelayToken), credLabel(snap.HasRemoteRelayToken)},
		{i18n.T(i18n.KeyRemoteMux), muxLabel(m.HarnessReady, m.MuxConnected)},
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
		struct{ label, value string }{i18n.T(i18n.KeyRemoteMobileAppBase), emptyDash(snap.RemoteAppBaseURL)},
		struct{ label, value string }{i18n.T(i18n.KeyRemoteMobileDevices), strconv.Itoa(snap.DeviceCount)},
	)

	ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
		for _, row := range rows {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(160)
				ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
			})
		}
	})

	if len(snap.DeviceNames) > 0 {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneDeviceList)).FontSize(12).Bold().TextColor(fg)
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			for _, name := range snap.DeviceNames {
				ui.Text(c, "· "+name).FontSize(12).TextColor(fg)
			}
		})
	}

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneConnectURL)).FontSize(12).Bold().TextColor(fg)
	if snap.ConnectURL != "" {
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			ui.Text(c, snap.ConnectURL).FontSize(12).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyRemotePhoneQRPlaceholder)).FontSize(11).TextColor(muted)
		})
	} else {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneURLUnavailable)).FontSize(11).TextColor(muted)
	}

	ui.Row(c).Gap(8).Children(func() {
		if m.OnCopyConnectURL != nil && ui.Button(c, i18n.T(i18n.KeyRemotePhoneCopyURL)).Clicked() {
			m.OnCopyConnectURL()
		}
		if m.OnOpenConnectURL != nil && ui.Button(c, i18n.T(i18n.KeyRemotePhoneOpenURL)).Clicked() {
			m.OnOpenConnectURL()
		}
		if m.OnRefreshPhone != nil && ui.Button(c, i18n.T(i18n.KeyRemotePhoneRefresh)).Clicked() {
			m.OnRefreshPhone()
		}
	})
	if m.PhoneActionMsg != "" {
		ui.Text(c, m.PhoneActionMsg).FontSize(11).TextColor(accent)
	}

	if len(snap.Missing) > 0 {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneMissing)+": "+strings.Join(snap.Missing, ", ")).FontSize(12).TextColor(accent)
	}
	if snap.RemoteEnabled && !snap.HasRemoteToken {
		ui.Text(c, i18n.T(i18n.KeyRemoteMobileNeedToken)).FontSize(12).TextColor(accent)
	}
}
