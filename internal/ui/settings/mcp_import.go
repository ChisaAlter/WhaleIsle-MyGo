package settings

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/mcpconfig"
)

type MCPImportAccess struct {
	State                     mcpconfig.ImportState
	OnInput                   func(string, string)
	OnParse, OnClear, OnStage func()
	OnSelect                  func(int)
	OnReplace                 func(bool)
}

func mcpImportPane(c *ui.Context, a *MCPImportAccess, busy bool, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.State
	ui.Text(c, mcpCatalogText("从 JSON 导入服务器", "Import server from JSON")).FontSize(13).Bold().TextColor(fg)
	ui.Text(c, mcpCatalogText("支持单个 spec、命名对象或 mcpServers 对象。每次只选择一项；解析和预览不连接、不保存，也不执行文档里的指令。", "Accepts a single spec, named object or mcpServers object. Select one entry at a time. Parsing and previewing neither connect nor save or execute document instructions.")).FontSize(11).TextColor(muted)
	text, name := s.Text, s.DefaultName
	changed := ui.TextArea(c, &text).FillWidth().MinHeight(80).Disabled(busy).Changed()
	ui.Text(c, mcpCatalogText("单个 spec 的默认名称（可留空沿用当前表单）", "Default name for a single spec (blank uses the current form)")).FontSize(11).TextColor(muted)
	if ui.TextInput(c, &name).FillWidth().Disabled(busy).Changed() {
		changed = true
	}
	if changed && a.OnInput != nil {
		a.OnInput(text, name)
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, mcpCatalogText("解析并预览", "Parse and preview")).Disabled(busy || text == "").Clicked() && a.OnParse != nil {
			a.OnParse()
		}
		if ui.Button(c, mcpCatalogText("清除导入预览", "Clear import preview")).Disabled(busy).Clicked() && a.OnClear != nil {
			a.OnClear()
		}
	})
	if s.Err != "" {
		ui.Text(c, s.Err).FontSize(11).TextColor(fg)
	}
	if s.Notice != "" {
		ui.Text(c, s.Notice).FontSize(11).TextColor(muted)
	}
	if !s.Ready {
		return
	}
	ui.Text(c, fmt.Sprintf(mcpCatalogText("%d 个候选配置；忽略 %d 个非服务器条目。尚未批量应用任何配置。", "%d candidate configurations; %d non-server entries ignored. No batch application is performed."), len(s.Candidates), s.Ignored)).FontSize(11).TextColor(muted)
	for index, candidate := range s.Candidates {
		row := candidate
		choice := index
		label := row.ID + " · " + row.Transport
		if choice == s.Selected {
			label = "• " + label
		}
		if ui.Button(c, label).FontSize(11).Disabled(busy || row.Err != "").Clicked() && a.OnSelect != nil {
			a.OnSelect(choice)
		}
		if row.Err != "" {
			ui.Text(c, row.Err).FontSize(11).TextColor(fg)
		}
	}
	if s.Selected < 0 || s.Selected >= len(s.Candidates) {
		return
	}
	selected := s.Candidates[s.Selected]
	ui.Text(c, selected.ID+" · "+selected.Name+" · "+selected.Transport).FontSize(12).Bold().TextColor(fg)
	enabled := mcpCatalogText("禁用", "disabled")
	if selected.Enabled {
		enabled = mcpCatalogText("启用", "enabled")
	}
	ui.Text(c, mcpCatalogText("保存后状态：", "State after saving: ")+enabled).FontSize(11).TextColor(muted)
	if selected.Transport == "stdio" {
		ui.Text(c, mcpCatalogText("命令：", "Command: ")+selected.Command).FontSize(11).TextColor(fg)
		ui.Text(c, mcpCatalogText("参数：", "Arguments: ")+selected.ArgsJSON).FontSize(11).TextColor(fg)
		if selected.Cwd != "" {
			ui.Text(c, mcpCatalogText("工作目录：", "Working directory: ")+selected.Cwd).FontSize(11).TextColor(muted)
		}
	} else {
		ui.Text(c, "URL: "+selected.URL).FontSize(11).TextColor(fg)
	}
	ui.Text(c, fmt.Sprintf(mcpCatalogText("环境变量 %d 项，请求头 %d 项；值在载入后的编辑器中复核。", "%d environment entries, %d headers; review their values in the editor after loading."), selected.EnvCount, selected.HeaderCount)).FontSize(11).TextColor(muted)
	for _, warning := range selected.Warnings {
		ui.Text(c, warning).FontSize(11).TextColor(muted)
	}
	replacement := s.ReplaceExisting
	if selected.Existing {
		if ui.Checkbox(c, &replacement, mcpCatalogText("明确选择编辑同 ID 的现有服务器：", "Explicitly choose to edit the existing server with this ID: ")+selected.ID).Disabled(busy || !selected.Writable).Changed() && a.OnReplace != nil {
			a.OnReplace(replacement)
		}
	}
	ui.Text(c, mcpCatalogText("载入会替换当前未保存草稿；必须再点击“保存配置”才写入 Host。保存启用配置可能启动命令或连接 HTTP。", "Loading replaces the current unsaved draft. A separate Save configuration action writes to the Host and may start a command or connect over HTTP when enabled.")).FontSize(11).TextColor(muted)
	if ui.Button(c, mcpCatalogText("确认载入所选 1 项到草稿", "Confirm loading this one entry into the draft")).Disabled(busy || selected.Err != "" || (selected.Existing && !replacement)).Clicked() && a.OnStage != nil {
		a.OnStage()
	}
}
