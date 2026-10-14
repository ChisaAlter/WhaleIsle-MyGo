// Package remote provides native remote workspace and SSH controls.
package remote

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/sshremote"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model carries remote plugin and desktop connection state.
type Model struct {
	Machines      *remotestore.Store
	MachineEditor *MachineEditor
	ThemeID       string
	SystemDark    bool

	RemoteWorkspaceEnabled bool
	RemoteEnabled          bool
	RemoteMode             string
	RemotePort             int
	RemoteBindAddress      string
	RemoteLanTls           bool
	RemoteAppBaseURL       string
	RemoteRelayEndpoint    string
	RemoteRelayUseTLS      bool
	RemoteDeviceCount      int
	RemoteDeviceNames      []string
	PhoneActionMsg         string
	HarnessRemoteOverlay   bool
	MuxConnected           bool
	HarnessReady           bool
	HasRemoteToken         bool
	HasRemoteRelayToken    bool
	SSHProbeLine           string

	SSHHost        *string
	SSHUser        *string
	SSHPort        *string
	SSHIdentity    *string
	SSHForward     *string
	SSHTunnelState string
	SSHTunnelMsg   string

	OnClose         func()
	OnProbeSSH      func()
	OnSaveTunnel    func()
	OnConnectSSH    func()
	OnDisconnectSSH func()

	OnCopyConnectURL func()
	OnOpenConnectURL func()
	OnRefreshPhone   func()
	OnCopySSHTarget  func(target string)
	// OnToggleWorkspace / OnTogglePairing persist config remote flags (Electron toggles).
	OnToggleWorkspace func(enabled bool)
	OnTogglePairing   func(enabled bool)
}

// View paints the remote workspace panel.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyRemoteTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyRemoteBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Scroll(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				viewMachines(c, m)

				ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, i18n.T(i18n.KeyRemoteWorkspaceToggle)).FontSize(12).TextColor(muted).Width(160)
						ui.Text(c, onOff(m.RemoteWorkspaceEnabled)).FontSize(12).TextColor(fg).Grow(1)
						if m.OnToggleWorkspace != nil {
							label := i18n.T(i18n.KeyRemoteToggleOn)
							if m.RemoteWorkspaceEnabled {
								label = i18n.T(i18n.KeyRemoteToggleOff)
							}
							if ui.Button(c, label).FontSize(11).Clicked() {
								m.OnToggleWorkspace(!m.RemoteWorkspaceEnabled)
							}
						}
					})
					ui.Row(c).Gap(12).Children(func() {
						ui.Text(c, i18n.T(i18n.KeyRemoteHarnessOverlay)).FontSize(12).TextColor(muted).Width(160)
						ui.Text(c, onOff(m.HarnessRemoteOverlay)).FontSize(12).TextColor(fg).Grow(1)
					})
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, i18n.T(i18n.KeyRemotePairingEnabled)).FontSize(12).TextColor(muted).Width(160)
						ui.Text(c, onOff(m.RemoteEnabled)).FontSize(12).TextColor(fg).Grow(1)
						if m.OnTogglePairing != nil {
							label := i18n.T(i18n.KeyRemoteToggleOn)
							if m.RemoteEnabled {
								label = i18n.T(i18n.KeyRemoteToggleOff)
							}
							if ui.Button(c, label).FontSize(11).Clicked() {
								m.OnTogglePairing(!m.RemoteEnabled)
							}
						}
					})
					for _, row := range []struct{ label, value string }{
						{i18n.T(i18n.KeyRemoteMode), emptyDash(m.RemoteMode)},
						{i18n.T(i18n.KeyRemotePort), strconv.Itoa(m.RemotePort)},
						{i18n.T(i18n.KeyRemoteMux), muxLabel(m.HarnessReady, m.MuxConnected)},
						{i18n.T(i18n.KeyRemoteCredentials), credLabel(m.HasRemoteToken)},
					} {
						ui.Row(c).Gap(12).Children(func() {
							ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(160)
							ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
						})
					}
				})

				if !m.RemoteWorkspaceEnabled {
					ui.Text(c, i18n.T(i18n.KeyRemoteWorkspaceOff)).FontSize(12).TextColor(accent)
				}

				viewPhoneSection(c, m, fg, muted, field, border, accent)

				sshSnap := sshremote.Snapshot{
					Active:                 sshremote.ModeUsesSSH(m.RemoteMode),
					RemoteWorkspaceEnabled: m.RemoteWorkspaceEnabled,
					RemoteEnabled:          m.RemoteEnabled,
					RemotePort:             m.RemotePort,
					RemoteBindAddress:      m.RemoteBindAddress,
					RemoteLanTLS:           m.RemoteLanTls,
					HasRemoteToken:         m.HasRemoteToken,
					HasRemoteRelayToken:    m.HasRemoteRelayToken,
				}
				sshSnap.Readiness, sshSnap.Missing = sshremote.Assess(sshSnap)
				if sshSnap.Active {
					ui.Text(c, i18n.T(i18n.KeyRemoteSSHStubNote)).FontSize(12).Bold().TextColor(fg)
					sshRows := []struct{ label, value string }{
						{i18n.T(i18n.KeyRemoteSSHStatus), sshReadinessLabel(sshSnap.Readiness)},
						{i18n.T(i18n.KeyRemoteSSHBind), emptyDash(sshSnap.RemoteBindAddress)},
						{i18n.T(i18n.KeyRemoteSSHLanTLS), onOff(sshSnap.RemoteLanTLS)},
						{i18n.T(i18n.KeyRemoteSSHRelayToken), credLabel(sshSnap.HasRemoteRelayToken)},
						{i18n.T(i18n.KeyRemoteSSHProbe), emptyDash(m.SSHProbeLine)},
					}
					ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
						for _, row := range sshRows {
							ui.Row(c).Gap(12).Children(func() {
								ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(160)
								ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
							})
						}
					})
					if m.OnProbeSSH != nil && ui.Button(c, i18n.T(i18n.KeyRemoteSSHProbeBtn)).Clicked() {
						m.OnProbeSSH()
					}
					if len(sshSnap.Missing) > 0 {
						ui.Text(c, i18n.T(i18n.KeyRemoteSSHMissing)+": "+strings.Join(sshSnap.Missing, ", ")).FontSize(12).TextColor(accent)
					}
					if !sshSnap.RemoteEnabled {
						ui.Text(c, i18n.T(i18n.KeyRemoteSSHPairingOff)).FontSize(12).TextColor(accent)
					}

					ui.Text(c, i18n.T(i18n.KeyRemoteSSHTunnelTitle)).FontSize(12).Bold().TextColor(fg)
					ui.Text(c, i18n.T(i18n.KeyRemoteSSHTunnelNote)).FontSize(11).TextColor(muted)
					ui.Text(c, i18n.T(i18n.KeyRemoteSSHTunnelState)+": "+emptyDash(m.SSHTunnelState)).FontSize(12).TextColor(fg)
					if m.SSHTunnelMsg != "" {
						ui.Text(c, m.SSHTunnelMsg).FontSize(11).TextColor(accent)
					}
					if m.SSHHost != nil {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyRemoteSSHFieldHost)).FontSize(11).TextColor(muted).Width(80)
							ui.TextInput(c, m.SSHHost).FillWidth()
						})
					}
					if m.SSHUser != nil {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyRemoteSSHFieldUser)).FontSize(11).TextColor(muted).Width(80)
							ui.TextInput(c, m.SSHUser).FillWidth()
						})
					}
					if m.SSHPort != nil {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyRemoteSSHFieldPort)).FontSize(11).TextColor(muted).Width(80)
							ui.TextInput(c, m.SSHPort).FillWidth()
						})
					}
					if m.SSHIdentity != nil {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyRemoteSSHFieldIdentity)).FontSize(11).TextColor(muted).Width(80)
							ui.TextInput(c, m.SSHIdentity).FillWidth()
						})
					}
					if m.SSHForward != nil {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyRemoteSSHFieldForward)).FontSize(11).TextColor(muted).Width(80)
							ui.TextInput(c, m.SSHForward).FillWidth()
						})
					}
					ui.Row(c).Gap(8).Children(func() {
						if m.OnCopySSHTarget != nil && m.SSHHost != nil && strings.TrimSpace(*m.SSHHost) != "" {
							if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
								user := ""
								if m.SSHUser != nil {
									user = strings.TrimSpace(*m.SSHUser)
								}
								port := ""
								if m.SSHPort != nil {
									port = strings.TrimSpace(*m.SSHPort)
								}
								target := strings.TrimSpace(*m.SSHHost)
								if user != "" {
									target = user + "@" + target
								}
								if port != "" && port != "22" {
									target = target + ":" + port
								}
								m.OnCopySSHTarget(target)
							}
						}
						if m.OnSaveTunnel != nil && ui.Button(c, i18n.T(i18n.KeyRemoteSSHTunnelSave)).Clicked() {
							m.OnSaveTunnel()
						}
						if m.OnConnectSSH != nil && ui.Button(c, i18n.T(i18n.KeyRemoteSSHTunnelConnect)).Clicked() {
							m.OnConnectSSH()
						}
						if m.OnDisconnectSSH != nil && ui.Button(c, i18n.T(i18n.KeyRemoteSSHTunnelDisconnect)).Clicked() {
							m.OnDisconnectSSH()
						}
					})
				}
			})
		})
	})
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyRemoteOn)
	}
	return i18n.T(i18n.KeyRemoteOff)
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func muxLabel(ready, connected bool) string {
	if !ready {
		return i18n.T(i18n.KeyRemoteMuxWaiting)
	}
	if connected {
		return i18n.T(i18n.KeyRemoteMuxConnected)
	}
	return i18n.T(i18n.KeyRemoteMuxDisconnected)
}

func credLabel(has bool) string {
	if has {
		return i18n.T(i18n.KeyRemoteCredPresent)
	}
	return i18n.T(i18n.KeyRemoteCredAbsent)
}

func sshReadinessLabel(r sshremote.Readiness) string {
	switch r {
	case sshremote.ReadinessReady:
		return i18n.T(i18n.KeyRemoteSSHReady)
	case sshremote.ReadinessIncomplete:
		return i18n.T(i18n.KeyRemoteSSHIncomplete)
	case sshremote.ReadinessDisabled:
		return i18n.T(i18n.KeyRemoteSSHDisabled)
	default:
		return i18n.T(i18n.KeyRemoteSSHStubPending)
	}
}

// RunSSHProbe is a helper for OnProbeSSH wiring.
func RunSSHProbe(m Model) string {
	snap := sshremote.Snapshot{
		Active:                 sshremote.ModeUsesSSH(m.RemoteMode),
		RemoteWorkspaceEnabled: m.RemoteWorkspaceEnabled,
		RemoteEnabled:          m.RemoteEnabled,
		RemotePort:             m.RemotePort,
		RemoteBindAddress:      m.RemoteBindAddress,
		RemoteLanTLS:           m.RemoteLanTls,
		HasRemoteToken:         m.HasRemoteToken,
		HasRemoteRelayToken:    m.HasRemoteRelayToken,
	}
	return sshremote.FormatProbe(sshremote.ProbeTCP(context.Background(), snap))
}

// SummaryLine is a one-line status for optional chrome hints.
func SummaryLine(m Model) string {
	if !m.RemoteWorkspaceEnabled {
		return i18n.T(i18n.KeyRemoteSummaryDisabled)
	}
	return fmt.Sprintf("%s · %s", i18n.T(i18n.KeyRemoteSummaryEnabled), muxLabel(m.HarnessReady, m.MuxConnected))
}
