package main

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/sshremote"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
	remoteui "whaleisle/internal/ui/remote"
	"whaleisle/internal/ui/settings"
)

// openPhonePopup is the native consumer of the existing phone launcher action.
// Its launcher row stays mounted as the popup anchor; no second menu is added.
func (s *mainState) openPhonePopup() {
	s.mu.Lock()
	s.phonePopupOpen = true
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) renderPhonePopup(c *ui.Context, anchor ui.Element) {
	s.mu.Lock()
	open := s.phonePopupOpen
	s.mu.Unlock()
	if !open {
		return
	}
	remoteui.PhonePopup(c, anchor, &open, s.remoteModel("", nil))
	s.mu.Lock()
	s.phonePopupOpen = open
	s.mu.Unlock()
}

// renderRemoteView and renderPhonePopup share the complete model, runtime,
// editors and action owners. SSH text drafts still commit after the full view.
func (s *mainState) renderRemoteView(c *ui.Context, embeddedSection settings.SectionID, channelsContent func(*ui.Context)) {
	m := s.remoteModel(embeddedSection, channelsContent)
	remoteui.View(c, m)
	s.mu.Lock()
	s.sshHost = *m.SSHHost
	s.sshUser = *m.SSHUser
	s.sshPort = *m.SSHPort
	s.sshIdentity = *m.SSHIdentity
	s.sshForward = *m.SSHForward
	s.mu.Unlock()
}

// remoteModel snapshots the existing in-memory owners. It does not load a
// profile or credentials, and keeps all previous remote view callbacks.
func (s *mainState) remoteModel(embeddedSection settings.SectionID, channelsContent func(*ui.Context)) remoteui.Model {
	state := s
	state.mu.Lock()
	win, sessions := state.win, state.sessions
	themeID, st := state.themeID, state.bootStatus
	hasRemoteToken, hasRemoteRelayToken := state.hasRemoteToken, state.hasRemoteRelayToken
	remoteDeviceCount := state.remoteDeviceCount
	remoteDeviceNames := append([]string(nil), state.remoteDeviceNames...)
	phoneActionMsg := state.phoneActionMsg
	cfgPhoneGitEnabled := state.phoneGitSession != ""
	cfg := config.Config{}
	if state.cfgStore != nil {
		cfg = state.cfgStore.Get()
	}
	cfgRemoteWorkspace, cfgRemoteEnabled := cfg.RemoteWorkspaceEnabled, cfg.RemoteEnabled
	cfgRemoteMode, cfgRemotePort := cfg.RemoteMode, cfg.RemotePort
	cfgRemoteBind, cfgRemoteLanTls := cfg.RemoteBindAddress, cfg.RemoteLanTls
	cfgRemoteAppBase := cfg.RemoteAppBaseUrl
	cfgRemoteRelayEndpoint, cfgRemoteRelayUseTLS := cfg.RemoteRelayEndpoint, cfg.RemoteRelayUseTls
	muxReady := state.dshMuxReady
	remoteMachines := state.remoteMachines
	sshProbeLine := state.sshProbeLine
	sshHost := state.sshHost
	sshUser := state.sshUser
	sshPort := state.sshPort
	sshIdentity := state.sshIdentity
	sshForward := state.sshForward
	sshTunnelMsg := state.sshTunnelMsg
	tunnelState := string(sshremote.TunnelIdle)
	if state.sshTunnel != nil {
		_, stt, errLine := state.sshTunnel.Snapshot()
		tunnelState = string(stt)
		if errLine != "" && sshTunnelMsg == "" {
			sshTunnelMsg = errLine
		}
	}
	state.mu.Unlock()
	workspacePreference := state.featurePreferenceAccess(remoteWorkspaceFeature)
	workspaceToggleBusy, workspaceToggleError := false, ""
	if workspacePreference != nil {
		workspaceToggleBusy = workspacePreference.State.Busy
		workspaceToggleError = workspacePreference.State.Error
	}
	return remoteui.Model{
		WorkspaceToggleBusy:    workspaceToggleBusy,
		WorkspaceToggleError:   workspaceToggleError,
		Embedded:               embeddedSection != "",
		SettingsPage:           map[settings.SectionID]string{settings.SectionRemote: "remote", settings.SectionRemoteWorkspace: "workspace"}[embeddedSection],
		ActiveTab:              &state.settingsRemoteTab,
		ChannelsContent:        channelsContent,
		GatewayProtocol:        "chisacode-v2",
		LegacyRelayTokenSet:    hasRemoteRelayToken,
		OnSetRelayToken:        state.clearLegacyRelayToken,
		OnRotateToken:          func() error { return state.phoneDeviceAction("rotate", "", "") },
		OnUseWorkspaceChoice:   state.remoteWorkspaceChoiceHandler(win, sessions, remoteMachines),
		Machines:               remoteMachines,
		Files:                  state.remoteSessionFiles(win, sessions),
		FileEditor:             &state.remoteFileEditor,
		MachineEditor:          &state.remoteMachineEditor,
		ThemeID:                themeID,
		SystemDark:             mygo.Theme.IsDark(),
		RemoteWorkspaceEnabled: cfgRemoteWorkspace,
		RemoteEnabled:          cfgRemoteEnabled,
		PhoneGitEnabled:        cfgPhoneGitEnabled,
		PhoneGitWorkspaces:     state.phoneGitWorkspaceRows(),
		OnTogglePhoneGitWorkspace: func(id string, enabled bool) {
			state.setPhoneGitWorkspace(id, enabled)
			go state.refreshPhoneRemote()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnTogglePhoneGit: func(enabled bool) {
			state.setPhoneGit(enabled)
			go state.refreshPhoneRemote()
			if win != nil {
				win.Update(func() {})
			}
		},
		RemoteMode:           cfgRemoteMode,
		RemotePort:           cfgRemotePort,
		RemoteBindAddress:    cfgRemoteBind,
		RemoteLanTls:         cfgRemoteLanTls,
		RemoteAppBaseURL:     cfgRemoteAppBase,
		RemoteRelayEndpoint:  cfgRemoteRelayEndpoint,
		RemoteRelayUseTLS:    cfgRemoteRelayUseTLS,
		RemoteDeviceCount:    remoteDeviceCount,
		RemoteDeviceNames:    remoteDeviceNames,
		PhoneActionMsg:       phoneActionMsg,
		PhoneRuntime:         state.phoneRuntimeManager().Snapshot(),
		PhoneDeviceEditor:    &state.phoneDeviceEditor,
		GatewayEditor:        &state.phoneGatewayEditor,
		OnSaveGateway:        state.savePhoneGateway,
		OnPhoneDeviceAction:  state.phoneDeviceAction,
		HarnessRemoteOverlay: cfgRemoteWorkspace,
		MuxConnected:         muxReady,
		HarnessReady:         st.State == harness.Ready,
		HasRemoteToken:       hasRemoteToken,
		HasRemoteRelayToken:  hasRemoteRelayToken,
		SSHProbeLine:         sshProbeLine,
		SSHHost:              &sshHost,
		SSHUser:              &sshUser,
		SSHPort:              &sshPort,
		SSHIdentity:          &sshIdentity,
		SSHForward:           &sshForward,
		SSHTunnelState:       tunnelState,
		SSHTunnelMsg:         sshTunnelMsg,
		OnClose: func() {
			state.revokePhoneGit()
			state.mu.Lock()
			state.remoteOpen = false
			state.sshHost = sshHost
			state.sshUser = sshUser
			state.sshPort = sshPort
			state.sshIdentity = sshIdentity
			state.sshForward = sshForward
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnToggleWorkspace: state.setRemoteWorkspacePreference,
		OnTogglePairing: func(enabled bool) {
			if state.cfgStore == nil {
				return
			}
			if err := state.cfgStore.Update(func(c *config.Config) {
				c.RemoteEnabled = enabled
			}); err != nil {
				state.mu.Lock()
				state.phoneActionMsg = i18n.T(i18n.KeyCmdPhoneRemoteSettingWasNotSavedRetry)
				state.mu.Unlock()
				if win != nil {
					win.Update(func() {})
				}
				return
			}
			if !enabled {
				// Revoke before the asynchronous stop; no phone Git
				// request may use the previous desktop grant meanwhile.
				state.revokePhoneGit()
			}
			go func() {
				// Read the latest saved setting inside phoneOp. A queued
				// disable must not stop a more recent enable request.
				_ = state.refreshPhoneRemote()
				if win != nil {
					win.Update(func() {})
				}
			}()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnCopyConnectURL: func() {
			url := state.phoneConnectURL()
			state.mu.Lock()
			if url == "" {
				state.phoneActionMsg = i18n.T(i18n.KeyRemotePhoneNoURL)
			} else {
				mygo.Clipboard.WriteText(url)
				state.phoneActionMsg = i18n.T(i18n.KeyRemotePhoneCopied)
			}
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnCopySSHTarget: func(target string) {
			target = strings.TrimSpace(target)
			if target == "" {
				return
			}
			mygo.Clipboard.WriteText(target)
			state.mu.Lock()
			state.sshTunnelMsg = i18n.T(i18n.KeyChatCopy) + ": " + target
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnOpenConnectURL: func() {
			url := state.phoneConnectURL()
			message := i18n.T(i18n.KeyRemotePhoneNoURL)
			if url != "" {
				if err := desktopwebview.OpenExternal(url); err == nil {
					message = i18n.T(i18n.KeyRemotePhoneOpened)
				} else {
					message = i18n.T(i18n.KeyRemotePhoneOpenFailed)
				}
			}
			// Never put an offer into persisted webview navigation/history.
			state.mu.Lock()
			state.phoneActionMsg = message
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnRefreshPhone: func() {
			go func() {
				_ = state.refreshPhoneRemote()
				if win != nil {
					win.Update(func() {})
				}
			}()
		},
		OnProbeSSH: func() {
			line := remoteui.RunSSHProbe(remoteui.Model{
				RemoteWorkspaceEnabled: cfgRemoteWorkspace,
				RemoteEnabled:          cfgRemoteEnabled,
				RemoteMode:             cfgRemoteMode,
				RemotePort:             cfgRemotePort,
				RemoteBindAddress:      cfgRemoteBind,
				RemoteLanTls:           cfgRemoteLanTls,
				HasRemoteToken:         hasRemoteToken,
				HasRemoteRelayToken:    hasRemoteRelayToken,
			})
			state.mu.Lock()
			state.sshProbeLine = line
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnSaveTunnel: func() {
			port, _ := strconv.Atoi(strings.TrimSpace(sshPort))
			cfg := sshremote.TunnelConfig{
				Host:         strings.TrimSpace(sshHost),
				User:         strings.TrimSpace(sshUser),
				Port:         port,
				IdentityFile: strings.TrimSpace(sshIdentity),
				LocalForward: strings.TrimSpace(sshForward),
			}
			state.mu.Lock()
			state.invalidateSSHConnectLocked()
			err := sshremote.SaveTunnelConfig(paths.UserDataDir(), cfg)
			state.sshHost = cfg.Host
			state.sshUser = cfg.User
			state.sshPort = strconv.Itoa(cfg.Port)
			state.sshIdentity = cfg.IdentityFile
			state.sshForward = cfg.LocalForward
			if state.sshTunnel == nil {
				state.sshTunnel = sshremote.NewTunnelSession()
			}
			state.sshTunnel.SetConfig(cfg)
			if err != nil {
				state.sshTunnelMsg = err.Error()
			} else {
				state.sshTunnelMsg = i18n.T(i18n.KeyCmdSSHTunnelSaved)
			}
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnConnectSSH: func() {
			port, _ := strconv.Atoi(strings.TrimSpace(sshPort))
			cfg := sshremote.TunnelConfig{
				Host:         strings.TrimSpace(sshHost),
				User:         strings.TrimSpace(sshUser),
				Port:         port,
				IdentityFile: strings.TrimSpace(sshIdentity),
				LocalForward: strings.TrimSpace(sshForward),
			}
			state.mu.Lock()
			op := state.beginSSHConnectLocked(cfg)
			state.mu.Unlock()
			kh := sshremote.KnownHostsPath(paths.UserDataDir())
			go op.run(sshConnectEffects{
				connect: op.tunnel.Connect,
				prompt: func(unknown *sshremote.UnknownHostKey, beforeShow func() bool) bool {
					return promptAcceptUnknownSSHHost(win, unknown, beforeShow)
				},
				changed: func(changed *sshremote.HostKeyChanged, beforeShow func() bool) {
					showSSHHostKeyChanged(win, changed, beforeShow)
				},
				accept: func(unknown *sshremote.UnknownHostKey) error {
					return sshremote.AcceptHostKey(kh, unknown.Address, unknown.Key)
				},
				redraw: func() {
					if win != nil {
						win.Update(func() {})
					}
				},
			})
		},
		OnDisconnectSSH: func() {
			state.mu.Lock()
			state.invalidateSSHConnectLocked()
			if state.sshTunnel != nil {
				state.sshTunnel.Disconnect()
			}
			state.sshTunnelMsg = "disconnected"
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
	}
}
