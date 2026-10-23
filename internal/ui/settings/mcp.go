package settings

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

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
	Summary    string
}

// MCPAccess is the dedicated MCP settings page (mcpServers/list when present).
type MCPAccess struct {
	HealthRefreshFailed                           bool
	Import                                        *MCPImportAccess
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
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, i18n.T(i18n.KeySettingsMCPTitle)).FontSize(fonttheme.InterfaceSize(18)).Bold().TextColor(fg)
			ui.Text(c, i18n.T(KeyPass4AMCPHintManaged)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, i18n.T(KeyPass4AMCPHintHTTP)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, i18n.T(KeyPass4AMCPHintComposition)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
			addLabel := mcpCatalogText("添加 MCP 服务器", "Add MCP server")
			if bodyIconAction32(c, addLabel, "plus", a.Busy || a.OnEdit == nil) && a.OnEdit != nil {
				a.OnEdit("")
			}
			if bodyIconAction32(c, i18n.T(i18n.KeySettingsRefresh), "refresh", a.Busy || a.OnRefresh == nil) && a.OnRefresh != nil {
				a.OnRefresh()
			}
		})
	})
	if !a.HasListRPC {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPStubHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.Text(c, i18n.T(i18n.KeySettingsMCPSchemaHint)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, settingsActionText(a.LastAction)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.HealthRefreshFailed {
		ui.Text(c, mcpCatalogText("自动刷新失败，保留上次状态；可点击刷新重试读取。", "Automatic refresh failed; showing the previous state. Refresh reconnects failed managed servers and reloads the list.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	mcpCatalogControls(c, a.Catalog, fg, muted)
	if len(a.Servers) == 0 && a.Busy {
		ui.Text(c, i18n.T(i18n.KeySettingsLoading)).TextColor(muted)
	} else if len(a.Servers) == 0 && a.Err == "" && a.HasListRPC {
		message := i18n.T(i18n.KeySettingsMCPEmpty)
		if a.Catalog != nil && len(a.Catalog.Snapshot.Rows) > 0 {
			message = mcpCatalogText("没有匹配的服务器", "No matching servers")
		}
		mcpEmptyCard(c, message, muted)
	} else {
		for _, row := range a.Servers {
			mcpCompactRow(c, a, row, fg, muted)
		}
	}
	if a.DeletingID != "" {
		open := true
		bodyDialog(c, &open, mcpCatalogText("移除服务器", "Remove server"), func() {
			ui.Text(c, a.DeletingID).TextColor(fg)
			ui.Text(c, mcpCatalogText("此操作会移除服务器配置。", "This removes the server configuration.")).TextColor(muted)
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if a.OnCancelDelete != nil && ui.Button(c, i18n.T(i18n.KeyQuitConfirmCancel)).Disabled(a.Busy).Clicked() {
					a.OnCancelDelete()
				}
				if a.OnConfirmDelete != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPDelete)).Disabled(a.Busy).Clicked() {
					a.OnConfirmDelete()
				}
			})
		})
		if !open && !a.Busy && a.OnCancelDelete != nil {
			a.OnCancelDelete()
		}
	}
	mcpImportPane(c, a.Import, a.Busy, fg, muted)
	formOpen := mcpFormEditor(c, a.Form, a.Busy, a.EditingID, fg, muted)
	if a.DraftSpec != nil && !formOpen {
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsert)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeySettingsMCPUpsertHint)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if a.EditingID != "" {
			ui.Text(c, i18n.T(i18n.KeyFilesEdit)+": "+a.EditingID).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		area := ui.TextArea(c, a.DraftSpec).Fill().MinHeight(72)
		if area.Changed() && a.OnDraftChanged != nil {
			a.OnDraftChanged(*a.DraftSpec)
		}
		ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
			if a.OnCancelEdit != nil && (*a.DraftSpec != "" || a.EditingID != "") && ui.Button(c, i18n.T(i18n.KeyQuitConfirmCancel)).Height(32).Padding(0, 14).Disabled(a.Busy).Clicked() {
				a.OnCancelEdit()
			}
			if a.OnUpsert != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPUpsertSave)).Height(32).Padding(0, 14).Disabled(a.Busy || *a.DraftSpec == "").Clicked() {
				a.OnUpsert(*a.DraftSpec)
			}
		})
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

// FormatMCPSummary mirrors the original server row's transport and endpoint.
func FormatMCPSummary(raw json.RawMessage) string {
	var spec struct {
		Transport string   `json:"transport"`
		Command   string   `json:"command"`
		Args      []string `json:"args"`
		Cwd       string   `json:"cwd"`
		URL       string   `json:"url"`
	}
	if json.Unmarshal(raw, &spec) != nil {
		return ""
	}
	if spec.Transport == "streamable-http" {
		return "HTTP · " + spec.URL
	}
	command := spec.Command
	if len(spec.Args) > 0 {
		command += " " + strings.Join(spec.Args, " ")
	}
	line := "stdio · " + command
	if spec.Cwd != "" {
		line += " · " + spec.Cwd
	}
	return line
}

func mcpTestToolSummary(names []string) string {
	if len(names) == 0 {
		return providerCopy("没有工具", "no tools")
	}
	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf(providerCopy("%s，另外 %d 个", "%s, %d more"), strings.Join(names[:3], ", "), len(names)-3)
}

var mcpAuthFailure = regexp.MustCompile(`(?i)\b(?:HTTP\s+)?(?:401|403)\b|\b(?:unauthorized|forbidden|invalid_token|invalid_grant|insufficient_scope)\b|missing bearer token`)

func mcpNeedsSignIn(a *MCPAccess, r MCPServerRow) bool {
	if !r.Writable || !r.Enabled || a.Catalog == nil {
		return false
	}
	for _, entry := range a.Catalog.Snapshot.Rows {
		if entry.Id == r.ID {
			var spec struct {
				Transport string `json:"transport"`
			}
			_ = json.Unmarshal(entry.Spec, &spec)
			return spec.Transport == "streamable-http" && entry.Connection != nil && entry.Connection.LastError != nil && mcpAuthFailure.MatchString(*entry.Connection.LastError)
		}
	}
	return false
}
func mcpCompactRow(c *ui.Context, a *MCPAccess, r MCPServerRow, fg, muted ui.Color) {
	title := r.ServerName
	if title == "" {
		title = r.ID
	}
	row := ui.Column(c.Key("mcp-server/"+r.ID)).FillWidth().MinWidth(0).Gap(6).Padding(10, 0)
	details := ui.Local(row, "details", func() bool { return false })
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).Wrap().AlignItems(ui.Center).Children(func() {
			// The row's own line is the disclosure target when a catalog can
			// describe the server: a click, Enter or Space opens what it reports.
			heading := func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
						ui.Text(c, title).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
					})
					detail := mcpRowDetail(r)
					ui.Text(c, detail).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Tooltip(detail)
					if r.Summary != "" {
						ui.Text(c, r.Summary).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Tooltip(r.Summary)
					}
				})
			}
			if a.Catalog != nil {
				head := ui.ButtonBase(c).Grow(1).MinWidth(180).Shrink(1).AlignItems(ui.Start).Cursor(ui.CursorPointer).
					Role(ui.RoleDisclosure).Expanded(*details).Label(title + " · " + mcpRowDetail(r))
				head.Children(heading)
				if head.Clicked() {
					*details = !*details
				}
			} else {
				ui.Column(c).Grow(1).MinWidth(180).Children(heading)
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if a.OnTest != nil && ui.Button(c, i18n.T(i18n.KeySettingsMCPTest)).Height(30).Disabled(a.Busy).Clicked() {
					a.OnTest(r.ID)
				}
				if r.Writable {
					enabled := r.Enabled
					if ui.Switch(c, &enabled).Label("Enable MCP server · "+r.ID).Disabled(a.Busy || a.OnSetEnabled == nil).Changed() && a.OnSetEnabled != nil {
						a.OnSetEnabled(r.ID, enabled)
					}
					if mcpNeedsSignIn(a, r) && a.OnAuthorize != nil && ui.Button(c, providerCopy("登录", "Sign in")).Height(30).Disabled(a.Busy).Clicked() {
						a.OnAuthorize(r.ID)
					}
					if a.OnEdit != nil && bodyIconAction(c, "Edit MCP server · "+r.ID, "edit", a.Busy) {
						a.OnEdit(r.ID)
					}
					if a.OnDelete != nil && bodyIconAction(c, "Delete MCP server · "+r.ID, "trash", a.Busy) {
						a.OnDelete(r.ID)
					}
				} else {
					ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			})
		})
		if a.Catalog != nil {
			snap := a.Catalog.Snapshot
			if !*details {
				for _, entry := range snap.Rows {
					if entry.Id != r.ID || entry.Connection == nil {
						continue
					}
					// A connected server with no tools says so, instead of showing the
					// same empty row a not-yet-read server shows.
					if len(entry.Connection.Tools) == 0 {
						ui.Text(c, providerCopy("没有工具", "no tools")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						continue
					}
					if ui.Button(c, fmt.Sprintf(providerCopy("%d 个工具", "%d tools"), len(entry.Connection.Tools))).Height(24).FontSize(fonttheme.InterfaceSize(11)).Clicked() && a.Catalog.OnInspect != nil {
						a.Catalog.OnInspect(r.ID)
					}
				}
			}
			if snap.ExpandedID == r.ID {
				open := true
				bodyDialog(c, &open, title+providerCopy(" 可用工具", " tools"), func() {
					for _, entry := range snap.Rows {
						if entry.Id == r.ID && entry.Connection != nil {
							for _, name := range entry.Connection.Tools {
								ui.Text(c, name).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
							}
						}
					}
				})
				if !open && a.Catalog.OnInspect != nil {
					a.Catalog.OnInspect(r.ID)
				}
			}
			if diagnostic, ok := snap.Tests[r.ID]; ok {
				message := diagnostic.Err
				if message == "" {
					message = fmt.Sprintf(providerCopy("测试成功：%d 个工具（%s）", "Test succeeded: %d tools (%s)"), int(diagnostic.Result.ToolCount), mcpTestToolSummary(diagnostic.Result.ToolNames))
				}
				mcpEmptyCard(c, message, muted)
			}
			if *details {
				// The dead helpers' home: what the last test said, the inspect
				// toggle and the registered tools, opened from the row itself.
				ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(0, 0, 4, 0).Children(func() {
					mcpCatalogDetails(c, a.Catalog, r.ID, fg, muted)
				})
			}
		}
		if strings.Contains(strings.ToLower(r.Status), "failed") && a.OnRetry != nil && r.Writable {
			if bodyIconAction(c, "Retry MCP server · "+r.ID, "refresh", a.Busy) {
				a.OnRetry(r.ID)
			}
		}
	})
}

// mcpEmptyCard is the dashed placeholder the installed build shows before the
// first server: a 1px dashed muted outline, radius 8, two centered lines.
func mcpEmptyCard(c *ui.Context, message string, muted ui.Color) {
	ui.Column(c.Key("mcp-empty")).FillWidth().MinWidth(0).MinHeight(92).Gap(4).Padding(20, 16).
		Radius(8).Border(1, ui.RGBA(muted.R, muted.G, muted.B, .3)).BorderStyle(ui.BorderDashed).
		AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, message).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
		ui.Text(c, i18n.T(KeyPass4AMCPEmptySecondary)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	})
}