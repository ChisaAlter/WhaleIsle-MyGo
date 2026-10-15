package remote

import (
	"strconv"
	"strings"
	"time"

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
		struct{ label, value string }{i18n.T(i18n.KeyRemoteMobileAppBase), emptyDash(snap.ConnectURL)},
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

	viewPhoneDevices(c, m, fg, muted, field, border, accent)

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneConnectURL)).FontSize(12).Bold().TextColor(fg)
	if snap.ConnectURL != "" {
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			ui.Text(c, snap.ConnectURL).FontSize(12).TextColor(fg)
		})
	} else {
		ui.Text(c, i18n.T(i18n.KeyRemotePhoneURLUnavailable)).FontSize(11).TextColor(muted)
	}

	runtime := m.PhoneRuntime
	status := runtime.Status
	if !m.RemoteEnabled {
		status = "disabled"
	}
	if status == "" {
		status = "offline"
	}
	ui.Text(c, "offer v2 · "+status).FontSize(12).Bold().TextColor(fg)
	if runtime.Message != "" {
		ui.Text(c, runtime.Message).FontSize(11).TextColor(accent)
	}
	if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) {
		ui.Text(c, "配对链接有效 / Pairing link valid · "+runtime.ExpiresAt.Local().Format("15:04:05")).FontSize(11).TextColor(fg)
		if runtime.PairingImage != nil {
			ui.Image(c, ui.NewBitmap(runtime.PairingImage)).Width(320).Height(320).Fit(ui.Contain)
			ui.Text(c, "用手机扫码配对，请勿分享二维码。 / Scan privately to pair your phone.").FontSize(11).TextColor(muted)
		} else {
			ui.Text(c, "二维码不可用，可复制临时配对链接。 / QR unavailable; copy the temporary link privately.").FontSize(11).TextColor(muted)
		}
	} else {
		ui.Text(c, "没有可用配对链接。启用会自动启动已安装的原版运行时；失败时请检查运行时后刷新重试。 / No usable offer. Enabling starts the installed runtime automatically; check the runtime and refresh to retry if needed.").FontSize(11).TextColor(muted)
	}
	ui.Row(c).Gap(8).Children(func() {
		if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) && m.OnCopyConnectURL != nil && ui.Button(c, "复制配对链接 / Copy pairing link").Clicked() {
			m.OnCopyConnectURL()
		}
		if runtime.PairingURL != "" && status == "ready" && runtime.Listening && runtime.RelayConnected && time.Now().Before(runtime.ExpiresAt) && m.OnOpenConnectURL != nil && ui.Button(c, "打开配对链接 / Open pairing link").Clicked() {
			m.OnOpenConnectURL()
		}
		if m.OnRefreshPhone != nil && ui.Button(c, i18n.T(i18n.KeyRemotePhoneRefresh)).Disabled(runtime.DeviceAction.Busy).Clicked() {
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
