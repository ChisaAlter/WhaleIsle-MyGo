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
func (s *mainState) runMCPSettings(controller *mcpconfig.Controller, action func(context.Context, mcpconfig.Caller)) {
	s.dshMu.Lock()
	if s.settingsMCPController != controller {
		s.dshMu.Unlock()
		return
	}
	client := s.dshClient
	s.dshMu.Unlock()
	// Avoid converting a nil concrete pointer to a non-nil interface.
	var caller mcpconfig.Caller
	if client != nil {
		caller = notifyingMCPCaller{Caller: client, notify: s.updateMCPSettingsUI}
	}
	go func() {
		s.dshMu.Lock()
		current := s.settingsMCPController == controller
		s.dshMu.Unlock()
		if !current {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		action(ctx, caller)
		s.updateMCPSettingsUI()
	}()
	s.updateMCPSettingsUI()
}

// mcpController is scoped to the currently attached Host. A Host replacement
// swaps the pointer under dshMu; old replies cannot populate the new catalog.
func (s *mainState) mcpController() *mcpconfig.Controller {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	if s.settingsMCPController == nil {
		s.settingsMCPController = &mcpconfig.Controller{}
	}
	return s.settingsMCPController
}
func (s *mainState) refreshMCPController() {
	controller := s.mcpController()
	s.runMCPSettings(controller, func(ctx context.Context, caller mcpconfig.Caller) { controller.Refresh(ctx, caller) })
}
func (s *mainState) settingsMCPAccess() *settings.MCPAccess {
	controller := s.mcpController()
	snap := controller.Snapshot()
	rows := make([]settings.MCPServerRow, 0, len(snap.Rows))
	for _, r := range snap.CatalogRows() {
		health := ""
		var lastError *string
		if r.Connection != nil {
			health = r.Connection.Health
			lastError = r.Connection.LastError
		}
		rows = append(rows, settings.MCPServerRow{ServerName: mcpconfig.ServerName(r), ID: r.Id, Origin: r.Origin, Enabled: r.Enabled, Writable: r.Writable, Status: settings.FormatMCPStatus(r.FiberPhase, health, lastError)})
	}
	rowAction := func(id, action string) {
		s.runMCPSettings(controller, func(ctx context.Context, client mcpconfig.Caller) {
			controller.RowAction(ctx, client, id, action)
		})
	}
	return &settings.MCPAccess{
		Form: &settings.MCPFormAccess{
			Draft: snap.Form, Errors: snap.FormErrors,
			OnOpen:   func() { controller.OpenForm(); s.updateMCPSettingsUI() },
			OnChange: func(form mcpconfig.FormDraft) { controller.SetForm(form); s.updateMCPSettingsUI() },
			OnJSON:   func() { controller.ShowJSON(); s.updateMCPSettingsUI() },
			OnCancel: func() { controller.CancelEdit(); s.updateMCPSettingsUI() },
			OnSave: func(form mcpconfig.FormDraft) {
				s.runMCPSettings(controller, func(ctx context.Context, caller mcpconfig.Caller) {
					controller.SaveFormExpected(ctx, caller, form, snap.EditingID)
				})
			},
		},
		Catalog: &settings.MCPCatalogAccess{
			Snapshot:  snap,
			OnQuery:   func(value string) { controller.SetQuery(value); s.updateMCPSettingsUI() },
			OnFilter:  func(value string) { controller.SetEnabledFilter(value); s.updateMCPSettingsUI() },
			OnInspect: func(id string) { controller.Inspect(id); s.updateMCPSettingsUI() },
		},
		Servers: rows, HasListRPC: snap.Ready, Busy: snap.Busy, Err: snap.Err, LastAction: snap.LastAction,
		DraftSpec: &snap.Draft, EditingID: snap.EditingID, DeletingID: snap.DeletingID,
		OnDraftChanged: controller.SetDraft,
		OnRefresh: func() {
			s.runMCPSettings(controller, func(ctx context.Context, caller mcpconfig.Caller) { controller.Refresh(ctx, caller) })
		},
		OnEdit:       func(id string) { controller.Edit(id); s.updateMCPSettingsUI() },
		OnCancelEdit: func() { controller.CancelEdit(); s.updateMCPSettingsUI() },
		OnUpsert: func(text string) {
			controller.SetDraft(text)
			s.runMCPSettings(controller, func(ctx context.Context, client mcpconfig.Caller) {
				controller.SaveExpected(ctx, client, text, snap.EditingID)
			})
		},
		OnSetEnabled: func(id string, enabled bool) {
			s.runMCPSettings(controller, func(ctx context.Context, client mcpconfig.Caller) {
				controller.SetEnabled(ctx, client, id, enabled)
			})
		},
		OnRetry:        func(id string) { rowAction(id, "retry") },
		OnAuthorize:    func(id string) { rowAction(id, "authorize") },
		OnTest:         func(id string) { rowAction(id, "test") },
		OnDelete:       func(id string) { controller.RequestDelete(id); s.updateMCPSettingsUI() },
		OnCancelDelete: func() { controller.CancelDelete(); s.updateMCPSettingsUI() },
		OnConfirmDelete: func() {
			s.runMCPSettings(controller, func(ctx context.Context, client mcpconfig.Caller) {
				controller.DeleteExpected(ctx, client, snap.DeletingID)
			})
		},
	}
}
