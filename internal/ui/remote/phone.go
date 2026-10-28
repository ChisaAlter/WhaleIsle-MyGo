package remote

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// phoneSnapshot builds AssessPhoneRemote status from the panel model.
func phoneSnapshot(m Model) phoneremote.Snapshot {
	names := append([]string(nil), m.RemoteDeviceNames...)
	count := m.RemoteDeviceCount
	if m.RemoteDeviceNames != nil {
		count = len(names)
	}
	if m.PhoneRuntime.DevicesLoaded {
		names = make([]string, len(m.PhoneRuntime.Devices))
		for i, device := range m.PhoneRuntime.Devices {
			names[i] = device.Name
		}
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
		} else if m.PhoneRuntime.Status == "ready" && m.PhoneRuntime.Listening && (m.PhoneRuntime.RelayConnected || m.RemoteMode == "lan") {
			s.Readiness = phoneremote.ReadinessReady
		}
	}
	s.ConnectURL = phoneremote.ConnectURL(s)
	// A listening LAN gateway has its own phone page (the daemon's SPA), so the
	// entry the user opens is that page rather than a relay appBaseUrl. Derive it
	// from the live pairing URL; the relay path keeps using remoteAppBaseUrl.
	if s.ConnectURL == "" && lanMode(s.RemoteMode) && m.PhoneRuntime.PairingURL != "" {
		if u, err := url.Parse(m.PhoneRuntime.PairingURL); err == nil &&
			(u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" {
			s.ConnectURL = u.Scheme + "://" + u.Host
		}
	}
	// The readiness row follows the runtime: a gateway that is ready, listening
	// and reachable over its transport is verified, even when legacy desktop
	// credentials are absent. The unverified wording stays for "no runtime yet".
	if s.Readiness != phoneremote.ReadinessDisabled && m.PhoneRuntime.Status == "ready" &&
		m.PhoneRuntime.Listening && (m.PhoneRuntime.RelayConnected || lanMode(s.RemoteMode)) {
		s.Readiness = phoneremote.ReadinessReady
		s.Missing = nil
	}
	return s
}

// lanMode reports whether the gateway transport is a LAN bind.
func lanMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "lan" || strings.Contains(m, "lan")
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
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	snap := phoneSnapshot(m)

	ui.Text(c, i18n.T(i18n.KeyRemotePhoneTitle)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeyRemotePhoneHowTo)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)

	serviceLive := m.PhoneRuntime.Status == "ready" && m.PhoneRuntime.Listening &&
		(m.PhoneRuntime.RelayConnected || lanMode(snap.RemoteMode))
	readiness := phoneReadinessLabel(snap.Readiness)
	if serviceLive {
		// The daemon is listening over its transport, so pairing really is
		// available; the legacy "config complete, unverified" wording would
		// contradict the live status line below.
		readiness = i18n.T(i18n.KeyRemotePhoneServiceReady)
	}
	rows := []struct{ label, value string }{
		{i18n.T(i18n.KeyRemotePhoneStatus), readiness},
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
	// The gateway transport decides whether a relay session is required.
	lan := m.RemoteMode == "lan"
	status := runtime.Status
	if !m.RemoteEnabled {
		status = "disabled"
	}
	if status == "" {
		status = "offline"
	}
	ui.Text(c, "offer v2 · "+status).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	pairingSteps(c, runtime, status, lan, fg, muted, accent)
	if runtime.Message != "" {
		ui.Text(c, runtime.Message).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	if offerUsable(runtime, status, lan) {
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
		if offerUsable(runtime, status, lan) && m.OnCopyConnectURL != nil && widgets.Button(c, colors, i18n.T(i18n.KeyMiscRemoteCopyPairingLink), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
			m.OnCopyConnectURL()
		}
		if offerUsable(runtime, status, lan) && m.OnOpenConnectURL != nil && widgets.Button(c, colors, i18n.T(i18n.KeyMiscRemoteOpenPairingLink), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			m.OnOpenConnectURL()
		}
		if m.OnRefreshPhone != nil && widgets.Button(c, colors, i18n.T(i18n.KeyRemotePhoneRefresh), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(runtime.DeviceAction.Busy).Clicked() {
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

// pairingStep is one stage of the pairing flow.
type pairingStep struct {
	title  string
	detail string
	state  string // done, active, failed, pending
}

// pairingSteps is the pairing flow as three stages — code, wait, result — with
// the stage the runtime is in marked, so the page says where the reader is
// instead of only printing a status word.
func pairingSteps(c *ui.Context, runtime phoneremote.RuntimeState, status string, lan bool, fg, muted, accent ui.Color) {
	valid := offerUsable(runtime, status, lan)
	expired := runtime.PairingURL != "" && !time.Now().Before(runtime.ExpiresAt)
	steps := []pairingStep{
		{label("生成配对码", "Create a pairing code"), label("打开手机运行时", "Start the phone runtime"), "pending"},
		{label("等待扫码", "Wait for the scan"), label("用手机扫描二维码或打开链接", "Scan the QR code or open the link"), "pending"},
		{label("配对完成", "Paired"), label("设备出现在已配对列表", "The device appears in the paired list"), "pending"},
	}
	switch {
	case runtime.PairingURL != "":
		steps[0].state = "done"
		switch {
		case valid:
			steps[1].state = "active"
		case expired:
			steps[1].state = "failed"
		default:
			steps[1].state = "pending"
		}
	case status == "ready":
		steps[0].state = "active"
	}
	if len(runtime.Devices) > 0 {
		for index := range steps {
			steps[index].state = "done"
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).Children(func() {
		for index, step := range steps {
			dot := muted
			switch step.state {
			case "done":
				dot = accent
			case "active":
				dot = accent
			case "failed":
				dot = c.Theme().Danger
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(6, 10).Radius(8).Border(.5, c.Theme().Border).Children(func() {
				ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(dot)
				ui.Text(c, fmt.Sprintf("%d. %s", index+1, step.title)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
				ui.Text(c, step.detail).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
			})
		}
	})
}

// offerUsable reports whether the daemon currently exposes a usable pairing
// offer. Native builds urls[].pairingUrl for both transports and only flags
// relayDown in relay mode (reference/electron/src/main/remote.js:344-360), so a
// listening LAN gateway needs no relay session; relay mode stays strict.
func offerUsable(runtime phoneremote.RuntimeState, status string, lan bool) bool {
	return runtime.PairingURL != "" && status == "ready" && runtime.Listening &&
		(runtime.RelayConnected || lan) && time.Now().Before(runtime.ExpiresAt)
}
