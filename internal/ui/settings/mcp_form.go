package settings

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/desktop/mcpconfig"
)

type MCPFormAccess struct {
	Draft                    *mcpconfig.FormDraft
	Errors                   map[string]string
	OnOpen, OnJSON, OnCancel func()
	OnChange, OnSave         func(mcpconfig.FormDraft)
}

func mcpFormEditor(c *ui.Context, a *MCPFormAccess, busy bool, editingID string, fg, muted ui.Color) bool {
	if a == nil {
		return false
	}
	if a.Draft == nil {
		if ui.Button(c, mcpCatalogText("使用字段表单", "Use field editor")).Disabled(busy).Clicked() && a.OnOpen != nil {
			a.OnOpen()
		}
		return false
	}
	form := *a.Draft
	changed := false
	ui.Text(c, mcpCatalogText("MCP 配置表单", "MCP configuration form")).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
	if editingID != "" {
		ui.Text(c, mcpCatalogText("编辑服务器：", "Editing server: ")+editingID).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	field := func(key, label string, value *string, locked bool, multiline bool) {
		desktopRow(c, label, func() {
			if multiline {
				if ui.TextArea(c, value).FillWidth().MinHeight(72).Disabled(busy || locked).Changed() {
					changed = true
				}
			} else {
				if ui.TextInput(c, value).FillWidth().Disabled(busy || locked).Changed() {
					changed = true
				}
			}
			if issue := a.Errors[key]; issue != "" {
				ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
			}
		}, fg, muted)
	}
	oldID := form.ID
	field("id", mcpCatalogText("服务器 ID", "Server ID"), &form.ID, editingID != "", false)
	if form.ID != oldID && form.ServerName == oldID {
		form.ServerName = form.ID
	}
	field("serverName", mcpCatalogText("工具命名空间", "Tool namespace"), &form.ServerName, false, false)
	ui.Text(c, mcpCatalogText("ID 和命名空间：1–32 个英文字母、数字、下划线或连字符。", "ID and namespace: 1–32 letters, digits, underscores or hyphens.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	desktopRow(c, mcpCatalogText("传输方式", "Transport"), func() {
		if ui.Select(c, &form.Transport, []string{"stdio", "streamable-http"}).FillWidth().Disabled(busy).Changed() {
			changed = true
		}
		if issue := a.Errors["transport"]; issue != "" {
			ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
	}, fg, muted)
	if ui.Checkbox(c, &form.Enabled, mcpCatalogText("保存后启用服务器", "Enable server after saving")).Disabled(busy).Changed() {
		changed = true
	}
	field("timeout", mcpCatalogText("工具超时（毫秒；留空使用默认值）", "Tool timeout (ms; blank uses default)"), &form.Timeout, false, false)
	if form.Transport == "stdio" {
		ui.Text(c, mcpCatalogText("启用并保存会在 Host 上启动或重启下面的命令；请核对命令、参数和环境变量。", "Saving an enabled stdio server starts or restarts this command on the Host. Review the command, arguments and environment first.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		field("command", mcpCatalogText("命令", "Command"), &form.Command, false, false)
		field("args", mcpCatalogText("参数（JSON 字符串数组，保留参数内空格）", "Arguments (JSON string array; spaces within arguments are preserved)"), &form.ArgsJSON, false, false)
		field("cwd", mcpCatalogText("工作目录（可选）", "Working directory (optional)"), &form.Cwd, false, false)
		field("env", mcpCatalogText("环境变量（每行 KEY=value）", "Environment (KEY=value per line)"), &form.Env, false, true)
	} else {
		ui.Text(c, mcpCatalogText("启用并保存会连接下面的 HTTP 地址，并使用填写的请求头。", "Saving an enabled HTTP server connects to this URL using the entered headers.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		field("url", mcpCatalogText("HTTP URL", "HTTP URL"), &form.URL, false, false)
		field("headers", mcpCatalogText("请求头（每行 KEY=value）", "Headers (KEY=value per line)"), &form.Headers, false, true)
	}
	ui.Text(c, mcpCatalogText("已有隐藏值 ******** 会按原 Host 协议保留，不作为新凭据写入。已有键留空也保留原值；清空整个映射仍会保留隐藏键。", "Existing ******** markers preserve the value through the Host protocol and are never saved as new credentials. A blank value for an existing key also preserves it; clearing the entire map still preserves hidden keys.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	ui.Text(c, mcpCatalogText("重连策略、启动错误选项及其他高级字段保留在原 JSON 中。保存失败保留表单；取消放弃草稿。", "Reconnect, startup-error and other advanced fields remain in the original JSON. Save failures keep the form; Cancel discards the draft.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if issue := a.Errors["form"]; issue != "" {
		ui.Text(c, issue).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	if changed && a.OnChange != nil {
		a.OnChange(form)
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, mcpCatalogText("保存配置（可能启动/重启）", "Save configuration (may start/restart)")).Disabled(busy).Clicked() && a.OnSave != nil {
			a.OnSave(form)
		}
		if ui.Button(c, mcpCatalogText("切换到原 JSON", "Switch to raw JSON")).Disabled(busy).Clicked() && a.OnJSON != nil {
			a.OnJSON()
		}
		if ui.Button(c, mcpCatalogText("取消", "Cancel")).Disabled(busy).Clicked() && a.OnCancel != nil {
			a.OnCancel()
		}
	})
	return true
}
