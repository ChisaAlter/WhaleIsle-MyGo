package main

import (
	"context"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) phoneRuntimeManager() *phoneremote.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phoneRuntime == nil {
		s.phoneRuntime = phoneremote.NewRuntime()
		s.phoneRuntime.SetGitAuthorizer(s.authorizePhoneGit)
		s.phoneRuntime.SetGitReviewer(s.reviewPhonePullRequest)
		s.phoneRuntime.SetHostActions(phoneremote.HostActions{OpenSettings: s.openPhoneSettingsSection, SaveConfig: s.savePhoneConfigPatch})
	}
	return s.phoneRuntime
}

func (s *mainState) stopPhoneRuntime() {
	s.mu.Lock()
	s.phoneIntent++
	s.mu.Unlock()
	s.revokePhoneGit()
	s.phoneOp.Lock()
	defer s.phoneOp.Unlock()
	s.cancelPhoneStopResumeLocked()
	s.phoneRuntimeManager().Stop()
}

func (s *mainState) refreshPhoneRemote() string {
	s.mu.Lock()
	intent := s.phoneIntent
	s.mu.Unlock()
	s.phoneOp.Lock()
	defer s.phoneOp.Unlock()
	return s.refreshPhoneRemoteLocked(false, intent)
}

// A single owner applies saved topology, revocation, stop and restart. The
// latest committed configuration is read after stopping as well as on entry.
// UI saves remain independent of a slow daemon stop; their queued reconcile
// consumes the latest desired settings instead of a captured obsolete config.
func (s *mainState) refreshPhoneRemoteLocked(restart bool, intent uint64) (msg string) {
	if !s.phoneIntentCurrent(intent) {
		return i18n.T(i18n.KeyCmdGatewaySettingsSavedApplying)
	}
	defer func() {
		s.mu.Lock()
		s.phoneActionMsg = msg
		s.mu.Unlock()
	}()
	manager := s.phoneRuntimeManager()
	if s.cfgStore == nil {
		return i18n.T(i18n.KeyCmdRemoteConfigUnavailable)
	}
	lifetime := s.lifecycleContext()
	cfg := s.cfgStore.Get()
	if restart || lifetime.Err() != nil || !cfg.RemoteEnabled {
		s.cancelPhoneStopResumeLocked()
		if stopped := manager.Stop(); stopped != nil {
			if lifetime.Err() == nil && s.phoneIntentCurrent(intent) {
				ctx, cancel := context.WithCancel(lifetime)
				s.phoneStopResume = cancel
				go func() {
					defer cancel()
					s.resumePhoneRemoteAfterStop(ctx, stopped, intent)
				}()
			}
			return manager.Snapshot().Message
		}
		cfg = s.cfgStore.Get()
	}
	s.mu.Lock()
	ep, quitting := s.bootStatus.Endpoint, s.quitting
	creds := remoteRuntimeCredentials{Local: s.remoteToken, Relay: s.remoteRelayToken}
	currentIntent := s.phoneIntent == intent
	s.mu.Unlock()
	if quitting || lifetime.Err() != nil {
		return "offline"
	}
	if !currentIntent {
		return i18n.T(i18n.KeyCmdGatewaySettingsSavedApplying)
	}
	if !cfg.RemoteEnabled {
		return "disabled"
	}
	gitRoots := s.phoneGitRoots()
	cfgForRemote := cfg
	cfgForRemote.RemotePort = effectiveGoRemotePort(cfg)
	runtimeConfig := buildRemoteRuntimeConfig(cfgForRemote, creds, gitRoots, ep.Origin, ep.Cookie)
	current := manager.Snapshot()
	if current.Status == "starting" || current.Status == "stopping" || current.Status == "stop-failed" {
		return current.Status
	}
	if current.Listening {
		manager.Refresh(runtimeConfig)
	} else {
		manager.Start(paths.UserDataDir(), paths.VendorDir(), runtimeConfig)
	}
	snapshot := manager.Snapshot()
	s.writePhoneDiag(snapshot)
	s.startPhoneDiagSampler(manager)
	msg = snapshot.Status
	if snapshot.Message != "" {
		msg += ": " + snapshot.Message
	}
	return msg
}

// phoneOp owns the one pending continuation. A later save replaces it and an
// explicit Host/application stop cancels it, even if remoteEnabled stays true.
func (s *mainState) cancelPhoneStopResumeLocked() {
	if s.phoneStopResume != nil {
		s.phoneStopResume()
		s.phoneStopResume = nil
	}
}

func (s *mainState) phoneIntentCurrent(intent uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phoneIntent == intent
}

func (s *mainState) resumePhoneRemoteAfterStop(ctx context.Context, stopped <-chan struct{}, intent uint64) {
	select {
	case <-ctx.Done():
		return
	case <-stopped:
	}
	s.phoneOp.Lock()
	if ctx.Err() != nil || !s.phoneIntentCurrent(intent) {
		s.phoneOp.Unlock()
		return
	}
	s.phoneStopResume = nil
	// A newer apply may already have started the gateway. Reconcile the latest
	// saved configuration without stopping that newer child.
	s.refreshPhoneRemoteLocked(false, intent)
	s.phoneOp.Unlock()
	s.invalidatePhoneRuntimeWindow()
}

func (s *mainState) invalidatePhoneRuntimeWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) phoneConnectURL() string {
	if s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return ""
	}
	return s.phoneRuntimeManager().Snapshot().PairingURL
}

func (s *mainState) applySavedPhoneGateway() {
	s.mu.Lock()
	s.phoneIntent++
	intent := s.phoneIntent
	s.phoneActionMsg = i18n.T(i18n.KeyCmdGatewaySettingsSavedApplying)
	s.mu.Unlock()
	s.revokePhoneGit()
	go s.applyPhoneGatewayIntent(intent)
}

func (s *mainState) applyPhoneGatewayIntent(intent uint64) {
	s.phoneOp.Lock()
	if !s.phoneIntentCurrent(intent) {
		s.phoneOp.Unlock()
		return
	}
	s.refreshPhoneRemoteLocked(true, intent)
	s.phoneOp.Unlock()
	s.invalidatePhoneRuntimeWindow()
}
