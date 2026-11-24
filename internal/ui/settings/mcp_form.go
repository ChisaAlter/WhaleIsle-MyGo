package settings

import (
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

type MCPFormAccess struct {
	Open                     bool
	JSONText                 *string
	Error                    string
	OnText, OnSaveJSON       func(string)
	Draft                    *mcpconfig.FormDraft
	Errors                   map[string]string
	OnOpen, OnJSON, OnCancel func()
	OnChange, OnSave         func(mcpconfig.FormDraft)
}

func mcpFormEditorContent(c *ui.Context, a *MCPFormAccess, busy bool, editingID string, fg, muted ui.Color) bool {
	if a == nil {
		return false
	}
	if a.Draft == nil {
		if kButtonDisabled(c, mcpCatalogText("使用字段表单", "Use field editor"), busy).AlignSelf(ui.Start).Clicked() && a.OnOpen != nil {
			a.OnOpen()
		}
		return false
	}
	form := *a.Draft
	changed := false
	kHeading(c, mcpCatalogText("MCP 配置表单", "MCP configuration form"))
	if editingID != "" {
		kHint(c, mcpCatalogText("编辑服务器：", "Editing server: ")+editingID)
	}
	field := func(key, label string, value *string, locked bool, multiline bool) {
		bodyRosterRow(c, label, func() {
			issue := a.Errors[key]
			if multiline {
				if kTextarea(c, value, coss.TextareaOptions{
					Label:    label,
					Disabled: busy || locked,
					Invalid:  issue != "",
					Error:    issue,
				}).Changed() {
					changed = true
				}
			} else {
				if kInput(c, value, coss.InputOptions{
					Label:    label,
					Disabled: busy || locked,
					Invalid:  issue != "",
					Error:    issue,
				}).Changed() {
					changed = true
				}
			}
			if issue != "" {
				kError(c, issue)
			}
		}, fg, muted)
	}
	oldID := form.ID
	field("id", mcpCatalogText("服务器 ID", "Server ID"), &form.ID, editingID != "", false)
	if form.ID != oldID && form.ServerName == oldID {
		form.ServerName = form.ID
	}
	field("serverName", mcpCatalogText("工具命名空间", "Tool namespace"), &form.ServerName, false, false)
	kHint(c, mcpCatalogText("ID 和命名空间：1–32 个英文字母、数字、下划线或连字符。", "ID and namespace: 1–32 letters, digits, underscores or hyphens."))
	bodyRosterRow(c, mcpCatalogText("传输方式", "Transport"), func() {
		issue := a.Errors["transport"]
		if coss.Select(c, &form.Transport, kOptions([]string{"stdio", "streamable-http"}), coss.SelectOptions{
			Label:    mcpCatalogText("传输方式", "Transport"),
			Disabled: busy,
			Invalid:  issue != "",
		}).FillWidth().Changed() {
			changed = true
		}
		if issue != "" {
			kError(c, issue)
		}
	}, fg, muted)
	if kCheckbox(c, &form.Enabled, mcpCatalogText("保存后启用服务器", "Enable server after saving"), busy).Changed() {
		changed = true
	}
	field("timeout", mcpCatalogText("工具超时（毫秒；留空使用默认值）", "Tool timeout (ms; blank uses default)"), &form.Timeout, false, false)
	if form.Transport == "stdio" {
		kHint(c, mcpCatalogText("启用并保存会在 Host 上启动或重启下面的命令；请核对命令、参数和环境变量。", "Saving an enabled stdio server starts or restarts this command on the Host. Review the command, arguments and environment first."))
		field("command", mcpCatalogText("命令", "Command"), &form.Command, false, false)
		field("args", mcpCatalogText("参数（JSON 字符串数组，保留参数内空格）", "Arguments (JSON string array; spaces within arguments are preserved)"), &form.ArgsJSON, false, false)
		field("cwd", mcpCatalogText("工作目录（可选）", "Working directory (optional)"), &form.Cwd, false, false)
		field("env", mcpCatalogText("环境变量（每行 KEY=value）", "Environment (KEY=value per line)"), &form.Env, false, true)
	} else {
		kHint(c, mcpCatalogText("启用并保存会连接下面的 HTTP 地址，并使用填写的请求头。", "Saving an enabled HTTP server connects to this URL using the entered headers."))
		field("url", mcpCatalogText("HTTP URL", "HTTP URL"), &form.URL, false, false)
		field("headers", mcpCatalogText("请求头（每行 KEY=value）", "Headers (KEY=value per line)"), &form.Headers, false, true)
	}
	kHint(c, mcpCatalogText("已有隐藏值 ******** 会按原 Host 协议保留，不作为新凭据写入。已有键留空也保留原值；清空整个映射仍会保留隐藏键。", "Existing ******** markers preserve the value through the Host protocol and are never saved as new credentials. A blank value for an existing key also preserves it; clearing the entire map still preserves hidden keys."))
	kHint(c, mcpCatalogText("重连策略、启动错误选项及其他高级字段保留在原 JSON 中。保存失败保留表单；取消放弃草稿。", "Reconnect, startup-error and other advanced fields remain in the original JSON. Save failures keep the form; Cancel discards the draft."))
	if issue := a.Errors["form"]; issue != "" {
		kError(c, issue)
	}
	if changed {
		*a.Draft = form
		if a.OnChange != nil {
			a.OnChange(form)
		}
	}
	return true
}

func mcpFormEditor(c *ui.Context, a *MCPFormAccess, busy bool, editingID string, fg, muted ui.Color) bool {
	if a == nil {
		return false
	}
	if !a.Open && a.Draft == nil {
		return false
	}
	open := true
	bodyDialogFooterWidth(c, &open, mcpCatalogText("MCP 服务器", "MCP server"), 380, func() {
		useForm, useJSON := false, false
		ui.Row(c).Gap(design.Sp(2)).Children(func() {
			if kChoiceButton(c, mcpCatalogText("字段表单", "Form"), a.Draft != nil, busy || a.Draft != nil || a.OnOpen == nil).Clicked() && a.OnOpen != nil {
				useForm = true
			}
			if kChoiceButton(c, mcpCatalogText("原 JSON", "JSON"), a.Draft == nil, busy || a.Draft == nil || a.OnJSON == nil).Clicked() && a.OnJSON != nil {
				useJSON = true
			}
		})
		if a.Error != "" {
			settingsFailureLine(c, "mcp:form", providerCopy("没能保存这个 MCP 服务，请重试。", "The MCP server was not saved. Try again."), a.Error)
		}
		if a.Draft != nil {
			mcpFormEditorContent(c, a, busy, editingID, fg, muted)
		} else if a.JSONText != nil {
			if kTextarea(c, a.JSONText, coss.TextareaOptions{
				Label:    mcpCatalogText("服务器 JSON", "Server JSON"),
				MaxLines: 18,
				ReadOnly: busy,
			}).Lines(8, 18).Changed() && a.OnText != nil {
				a.OnText(*a.JSONText)
			}
		}
		if useForm {
			a.OnOpen()
		} else if useJSON {
			a.OnJSON()
		}
	}, func() {
		if kButtonDisabled(c, mcpCatalogText("取消", "Cancel"), busy || a.OnCancel == nil).Clicked() && a.OnCancel != nil {
			a.OnCancel()
		}
		if kPrimary(c, mcpCatalogText("保存", "Save"), busy || a.Draft == nil && (a.JSONText == nil || a.OnSaveJSON == nil) || a.Draft != nil && a.OnSave == nil).Clicked() {
			if a.Draft != nil {
				a.OnSave(*a.Draft)
			} else {
				a.OnSaveJSON(*a.JSONText)
			}
		}
	}, ui.DialogOptions{DismissDisabled: busy})
	if !open && !busy && a.OnCancel != nil {
		a.OnCancel()
	}
	return true
}
