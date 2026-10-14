package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// MCPServerRow is one host MCP server for the dedicated settings page.
type MCPServerRow struct {
	ID      string
	Origin  string
	Enabled bool
	Status  string
}

// MCPAccess is the dedicated MCP settings page (mcpServers/list when present).
type MCPAccess struct {
	Servers      []MCPServerRow
	HasListRPC   bool
	Err          string
	LastAction   string
	OnRefresh    func()
	OnSetEnabled func(id string, enabled bool)
	OnRetry      func(id string)
	OnAuthorize  func(id string)
	OnTest       func(id string)
	OnDelete     func(id string)
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
	if len(a.Servers) == 0 {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPEmpty)).FontSize(12).TextColor(muted)
	} else {
		for _, row := range a.Servers {
			r := row
			desktopRow(c, emptyDash(r.ID), func() {
				ui.Text(c, mcpRowDetail(r)).FontSize(11).TextColor(muted)
				ui.Row(c).Gap(6).Children(func() {
					toggle := i18n.T(i18n.KeySettingsMCPEnable)
					if r.Enabled {
						toggle = i18n.T(i18n.KeySettingsMCPDisable)
					}
					if a.OnSetEnabled != nil && ui.Button(c, toggle).FontSize(11).Clicked() {
						a.OnSetEnabled(r.ID, !r.Enabled)
					}
					if a.OnRetry != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPRetry)).FontSize(11).Clicked() {
						a.OnRetry(r.ID)
					}
					if a.OnAuthorize != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPAuthorize)).FontSize(11).Clicked() {
						a.OnAuthorize(r.ID)
					}
					if a.OnTest != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPTest)).FontSize(11).Clicked() {
						a.OnTest(r.ID)
					}
					if a.OnDelete != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPDelete)).FontSize(11).Clicked() {
						a.OnDelete(r.ID)
					}
				})
			}, fg, muted)
		}
	}
	if a.DraftSpec != nil {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsert)).FontSize(12).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsertHint)).FontSize(11).TextColor(muted)
		ui.TextArea(c, a.DraftSpec).Fill().MinHeight(72)
		if a.OnUpsert != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPUpsertSave)).Clicked() {
			a.OnUpsert(*a.DraftSpec)
		}
	}
	if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
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
