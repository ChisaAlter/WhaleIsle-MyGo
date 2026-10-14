package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/ui/settings"
)

type notifyingMCPCaller struct {
	mcpconfig.Caller
	notify func()
}

func (c notifyingMCPCaller) Call(ctx context.Context, method string, args, out any) error {
	c.notify()
	return c.Caller.Call(ctx, method, args, out)
}

func (s *mainState) updateMCPSettingsUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) runMCPSettings(action func(context.Context, mcpconfig.Caller)) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	// Avoid converting a nil concrete pointer to a non-nil interface.
	var caller mcpconfig.Caller
	if client != nil {
		caller = notifyingMCPCaller{Caller: client, notify: s.updateMCPSettingsUI}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		action(ctx, caller)
		s.updateMCPSettingsUI()
	}()
	s.updateMCPSettingsUI()
}
func (s *mainState) settingsMCPAccess() *settings.MCPAccess {
	snap := s.settingsMCPController.Snapshot()
	rows := make([]settings.MCPServerRow, 0, len(snap.Rows))
	for _, r := range snap.Rows {
		health := ""
		var lastError *string
		if r.Connection != nil {
			health = r.Connection.Health
			lastError = r.Connection.LastError
		}
		rows = append(rows, settings.MCPServerRow{ID: r.Id, Origin: r.Origin, Enabled: r.Enabled, Writable: r.Writable, Status: settings.FormatMCPStatus(r.FiberPhase, health, lastError)})
	}
	rowAction := func(id, action string) {
		s.runMCPSettings(func(ctx context.Context, client mcpconfig.Caller) {
			s.settingsMCPController.RowAction(ctx, client, id, action)
		})
	}
	return &settings.MCPAccess{
		Servers: rows, HasListRPC: snap.Ready, Busy: snap.Busy, Err: snap.Err, LastAction: snap.LastAction,
		DraftSpec: &snap.Draft, EditingID: snap.EditingID, DeletingID: snap.DeletingID,
		OnDraftChanged: s.settingsMCPController.SetDraft,
		OnRefresh:      s.refreshSettingsMCP,
		OnEdit:         func(id string) { s.settingsMCPController.Edit(id); s.updateMCPSettingsUI() },
		OnCancelEdit:   func() { s.settingsMCPController.CancelEdit(); s.updateMCPSettingsUI() },
		OnUpsert: func(text string) {
			s.settingsMCPController.SetDraft(text)
			s.runMCPSettings(func(ctx context.Context, client mcpconfig.Caller) {
				s.settingsMCPController.SaveExpected(ctx, client, text, snap.EditingID)
			})
		},
		OnSetEnabled: func(id string, enabled bool) {
			s.runMCPSettings(func(ctx context.Context, client mcpconfig.Caller) {
				s.settingsMCPController.SetEnabled(ctx, client, id, enabled)
			})
		},
		OnRetry:        func(id string) { rowAction(id, "retry") },
		OnAuthorize:    func(id string) { rowAction(id, "authorize") },
		OnTest:         func(id string) { rowAction(id, "test") },
		OnDelete:       func(id string) { s.settingsMCPController.RequestDelete(id); s.updateMCPSettingsUI() },
		OnCancelDelete: func() { s.settingsMCPController.CancelDelete(); s.updateMCPSettingsUI() },
		OnConfirmDelete: func() {
			s.runMCPSettings(func(ctx context.Context, client mcpconfig.Caller) {
				s.settingsMCPController.DeleteExpected(ctx, client, snap.DeletingID)
			})
		},
	}
}
