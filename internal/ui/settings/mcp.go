package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// MCPServerRow is one host MCP server for the dedicated settings page.
type MCPServerRow struct {
	ServerName string
	ID         string
	Origin     string
	Enabled    bool
	Writable   bool
	Status     string
}

// MCPAccess is the dedicated MCP settings page (mcpServers/list when present).
type MCPAccess struct {
	Form                                          *MCPFormAccess
	Catalog                                       *MCPCatalogAccess
	Servers                                       []MCPServerRow
	Busy                                          bool
	EditingID, DeletingID                         string
	OnEdit                                        func(string)
	OnDraftChanged                                func(string)
	OnCancelEdit, OnConfirmDelete, OnCancelDelete func()
	HasListRPC                                    bool
	Err                                           string
	LastAction                                    string
	OnRefresh                                     func()
	OnSetEnabled                                  func(id string, enabled bool)
	OnRetry                                       func(id string)
	OnAuthorize                                   func(id string)
	OnTest                                        func(id string)
	OnDelete                                      func(id string)
	// DraftSpec is optional JSON for mcpServers/upsert (Electron add-server form).
	DraftSpec *string
	OnUpsert  func(specJSON string)
}

func mcpPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.MCP == nil {
		return
	}
	a := m.MCP
	ui.Text(c, i18n.T(i18n.KeySettingsMCPTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsMCPHint)).FontSize(12).TextColor(muted)
	if !a.HasListRPC {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPStubHint)).FontSize(12).TextColor(muted)
		ui.Text(c, i18n.T(i18n.KeySettingsMCPSchemaHint)).FontSize(11).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	mcpCatalogControls(c, a.Catalog, fg, muted)
	if len(a.Servers) == 0 {
		message := i18n.T(i18n.KeySettingsMCPEmpty)
		if a.Catalog != nil && len(a.Catalog.Snapshot.Rows) > 0 {
			message = mcpCatalogText("没有匹配的服务器", "No matching servers")
		}
		ui.Text(c, message).FontSize(12).TextColor(muted)
	} else {
		for _, row := range a.Servers {
			r := row
			title := r.ID
			if r.ServerName != "" && r.ServerName != r.ID {
				title = r.ServerName + " · " + r.ID
			}
			desktopRow(c, emptyDash(title), func() {
				ui.Text(c, mcpRowDetail(r)).FontSize(11).TextColor(muted)
				mcpCatalogDetails(c, a.Catalog, r.ID, fg, muted)
				if !r.Writable {
					ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).FontSize(11).TextColor(muted)
				}
				ui.Row(c).Gap(6).Children(func() {
					if r.Writable && a.OnEdit != nil && ui.Button(c, i18n.T(i18n.KeyFilesEdit)).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnEdit(r.ID)
					}
					toggle := i18n.T(i18n.KeySettingsMCPEnable)
					if r.Enabled {
						toggle = i18n.T(i18n.KeySettingsMCPDisable)
					}
					if r.Writable && a.OnSetEnabled != nil && ui.Button(c, toggle).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnSetEnabled(r.ID, !r.Enabled)
					}
					if r.Writable && a.OnRetry != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPRetry)).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnRetry(r.ID)
					}
					if r.Writable && a.OnAuthorize != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPAuthorize)).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnAuthorize(r.ID)
					}
					if a.OnTest != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPTest)).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnTest(r.ID)
					}
					if r.Writable && a.OnDelete != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPDelete)).FontSize(11).Disabled(a.Busy).Clicked() {
						a.OnDelete(r.ID)
					}
				})
			}, fg, muted)
		}
	}
	if a.DeletingID != "" {
		ui.Text(c, "确认移除 / Confirm removal: "+a.DeletingID).FontSize(12).Bold().TextColor(fg)
		ui.Row(c).Gap(8).Children(func() {
			if a.OnConfirmDelete != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPDelete)).Disabled(a.Busy).Clicked() {
				a.OnConfirmDelete()
			}
			if a.OnCancelDelete != nil && ui.Button(c, i18n.T(i18n.KeyQuitConfirmCancel)).Disabled(a.Busy).Clicked() {
				a.OnCancelDelete()
			}
		})
	}
	formOpen := mcpFormEditor(c, a.Form, a.Busy, a.EditingID, fg, muted)
	if a.DraftSpec != nil && !formOpen {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsert)).FontSize(12).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsertHint)).FontSize(11).TextColor(muted)
		if a.EditingID != "" {
			ui.Text(c, i18n.T(i18n.KeyFilesEdit)+": "+a.EditingID).FontSize(12).TextColor(fg)
		}
		area := ui.TextArea(c, a.DraftSpec).Fill().MinHeight(72)
		if area.Changed() && a.OnDraftChanged != nil {
			a.OnDraftChanged(*a.DraftSpec)
		}
		if a.OnUpsert != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPUpsertSave)).Disabled(a.Busy || *a.DraftSpec == "").Clicked() {
			a.OnUpsert(*a.DraftSpec)
		}
		if a.OnCancelEdit != nil && (*a.DraftSpec != "" || a.EditingID != "") && ui.Button(c, i18n.T(i18n.KeyQuitConfirmCancel)).Disabled(a.Busy).Clicked() {
			a.OnCancelEdit()
		}
	}
	if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Disabled(a.Busy).Clicked() {
		a.OnRefresh()
	}
}

func mcpRowDetail(row MCPServerRow) string {
	enabled := i18n.T(i18n.KeySettingsMCPDisabled)
	if row.Enabled {
		enabled = i18n.T(i18n.KeySettingsMCPEnabled)
	}
	parts := []string{enabled}
	if row.Origin != "" {
		parts = append(parts, row.Origin)
	}
	if row.Status != "" {
		parts = append(parts, row.Status)
	}
	return strings.Join(parts, " · ")
}

// FormatMCPStatus builds a short status line from fiber phase and connection health.
func FormatMCPStatus(fiberPhase, health string, lastError *string) string {
	parts := make([]string, 0, 3)
	if fiberPhase != "" {
		parts = append(parts, fiberPhase)
	}
	if health != "" {
		parts = append(parts, health)
	}
	if lastError != nil && *lastError != "" {
		parts = append(parts, *lastError)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " / ")
}

// FormatMCPServerCount is a small helper kept for tests / status lines.
func FormatMCPServerCount(n int) string {
	return fmt.Sprintf("%d", n)
}
