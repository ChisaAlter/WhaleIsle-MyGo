package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) whaleSettingsOwner(ep harness.Endpoint) (*dsh.Client, string) {
	s.mu.Lock()
	petSettings := s.settingsPetActiveLocked()
	active := (s.assistantOpen && !s.proactivePanel.open || petSettings) && s.bootStatus.State == harness.Ready && s.cfgStore != nil && s.cfgStore.Get().WhaleAssistantEnabled
	readyEndpoint := s.bootStatus.Endpoint
	generation := s.bootStatus.Generation
	asst := s.assistant
	s.mu.Unlock()
	if !active || (!petSettings && (asst == nil || asst.Snapshot().Mode != assiststore.ModeWhale)) || readyEndpoint.Origin != ep.Origin || readyEndpoint.Cookie != ep.Cookie {
		return nil, ""
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if client == nil || client.Endpoint.Origin != ep.Origin || client.Endpoint.Cookie != ep.Cookie {
		return nil, ""
	}
	// Do not expose the authentication cookie or fingerprint in the UI/logs.
	digest := sha256.Sum256([]byte(fmt.Sprintf("%p\x00%s\x00%s\x00%d", client, ep.Origin, ep.Cookie, generation)))
	return client, fmt.Sprintf("%x", digest)
}
func (s *mainState) refreshWhaleSettingsUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
func (s *mainState) assistantWhaleSettingsModel(ep harness.Endpoint) assistantui.WhaleSettingsModel {
	client, host := s.whaleSettingsOwner(ep)
	// A mode switch hides this form but retains its draft for the same Host.
	// Explicit Close/disable handlers revoke in-flight tickets separately.
	if host != "" {
		s.whaleSettings.Bind(host)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && host != "" && key == host
	}
	write := func(action string) {
		if !current() || s.whaleSettings.Preferences.Snapshot().Busy {
			return
		}
		release, allowed := s.whaleSettings.BeginMutation(host)
		if !allowed {
			return
		}
		ticket, command, ok := s.whaleSettings.BeginWrite(host, action)
		if !ok {
			release()
			s.refreshWhaleSettingsUI()
			return
		}
		s.refreshWhaleSettingsUI()
		go func() {
			defer func() { release(); s.refreshWhaleSettingsUI() }()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cli := s.pluginRPCClient(ep)
			var actual *pluginrpc.WhaleSettings
			accepted := false
			execute := func() error {
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host or Whale editor changed before submission")
				}
				before, err := cli.WhaleSettings(ctx)
				if err != nil {
					return err
				}
				if err = command.Check(before); err != nil {
					return err
				}
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host or Whale editor changed before submission")
				}
				// Exactly one user-requested mutation; never ensure/open a session, prompt
				// a model, retry a write, or use dshbot task/catalog RPCs here.
				switch command.Action {
				case "memory":
					err = cli.WhaleReplaceMemory(ctx, command.Value.Memory)
				case "clear":
					err = cli.WhaleClearMemory(ctx)
				case "identity":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{Name: &command.Value.Name, UserTitle: &command.Value.UserTitle})
				case "name":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{Name: &command.Value.Name})
				case "title":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{UserTitle: &command.Value.UserTitle})
				case "persona":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{PersonaText: &command.Value.PersonaText})
				}
				if err != nil {
					return err
				}
				accepted = true
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host changed after confirmation; refresh to verify")
				}
				read, err := cli.WhaleSettings(ctx)
				if err == nil {
					actual = &read
				}
				return err
			}
			err := execute()
			if !current() {
				return
			}
			if s.whaleSettings.FinishWrite(ticket, command, actual, accepted, err) {
				if accepted && err == nil && (command.Action == "identity" || command.Action == "name") {
					s.mu.Lock()
					sessions := s.sessions
					if actual != nil {
						s.whaleNavName = strings.TrimSpace(actual.Name)
						s.whaleNavFetchedAt = time.Now()
					} else {
						// Read-back failed: expire the published name so the
						// observer refetches on the next frame.
						s.whaleNavFetchedAt = time.Time{}
					}
					s.mu.Unlock()
					if sessions != nil {
						sessions.RefreshList(ctx)
					}
				}
			}
		}()
	}
	preferences := s.assistantWhalePreferencesModel(ep)
	state := s.whaleSettings.Snapshot()
	state.Busy = state.Busy || preferences.State.Busy
	if host == "" {
		state = assiststore.WhaleSettingsSnapshot{}
	}
	return assistantui.WhaleSettingsModel{
		Preferences: preferences,
		State:       state,
		OnRefresh: func() {
			if !current() {
				return
			}
			ticket, ok := s.whaleSettings.BeginLoad(host)
			if !ok {
				return
			}
			s.refreshWhaleSettingsUI()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if !current() || !s.whaleSettings.Current(ticket) {
					return
				}
				value, err := s.pluginRPCClient(ep).WhaleSettings(ctx)
				if current() && s.whaleSettings.FinishLoad(ticket, value, err) {
					s.refreshWhaleSettingsUI()
				}
			}()
		},
		OnEdit: func(field, value string) {
			if current() {
				s.whaleSettings.Edit(host, field, value)
				s.refreshWhaleSettingsUI()
			}
		},
		OnRestore: func(field string) {
			if current() {
				s.whaleSettings.Restore(host, field)
				s.refreshWhaleSettingsUI()
			}
		},
		OnSave: write,
		OnReviewClear: func() {
			if current() {
				s.whaleSettings.ReviewClear(host)
				s.refreshWhaleSettingsUI()
			}
		},
		OnCancelClear: func() {
			if current() {
				s.whaleSettings.CancelClear(host)
				s.refreshWhaleSettingsUI()
			}
		},
		OnConfirmClear: func() { write("clear") },
	}
}

// Called for every native frame, including navigation that bypasses this panel.
func (s *mainState) observeWhaleSettings() {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	_, host := s.whaleSettingsOwner(ep)
	if host == "" {
		s.whaleSettings.Suspend()
		s.whaleTasks.Suspend()
	} else {
		s.whaleSettings.Bind(host)
	}
	s.observeWhaleNavLabel(ep)
}

// The sidebar's Whale row shows the assistant's display name. Unlike the
// settings editor it is visible on every frame, so the name is fetched once per
// Host through the plugin RPC and republished as a snapshot; periodic refetch
// keeps external edits converging without any draw-path I/O.
const whaleNavRefetch = 30 * time.Second

func (s *mainState) observeWhaleNavLabel(ep harness.Endpoint) {
	s.mu.Lock()
	enabled := s.bootStatus.State == harness.Ready && s.cfgStore != nil && s.cfgStore.Get().WhaleAssistantEnabled
	endpoint := s.bootStatus.Endpoint
	generation := s.bootStatus.Generation
	s.mu.Unlock()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	key := ""
	if enabled && ep.Origin != "" && client != nil && client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie && endpoint.Origin == ep.Origin && endpoint.Cookie == ep.Cookie {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%p\x00%s\x00%s\x00%d", client, ep.Origin, ep.Cookie, generation)))
		key = fmt.Sprintf("%x", digest)
	}
	s.mu.Lock()
	if s.whaleNavKey != key {
		s.whaleNavKey = key
		s.whaleNavName = ""
		s.whaleNavBusy = false
		s.whaleNavFetchedAt = time.Time{}
		s.whaleNavSerial++
	}
	if key == "" || s.whaleNavBusy || time.Since(s.whaleNavFetchedAt) < whaleNavRefetch {
		s.mu.Unlock()
		return
	}
	s.whaleNavBusy = true
	serial := s.whaleNavSerial
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		name := ""
		if settings, err := s.pluginRPCClient(ep).WhaleSettings(ctx); err == nil {
			name = strings.TrimSpace(settings.Name)
		}
		s.mu.Lock()
		if s.whaleNavKey == key && s.whaleNavSerial == serial {
			s.whaleNavName = name
			s.whaleNavBusy = false
			s.whaleNavFetchedAt = time.Now()
		}
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
	}()
}
