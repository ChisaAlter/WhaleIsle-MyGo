package settings

import (
	"fmt"

	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

type MCPImportAccess struct {
	State                     mcpconfig.ImportState
	OnInput                   func(string, string)
	OnParse, OnClear, OnStage func()
	OnSelect                  func(int)
	OnReplace                 func(bool)
}

func mcpImportContent(c *ui.Context, a *MCPImportAccess, busy bool, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.State
	kHeading(c, mcpCatalogText("从 JSON 导入服务器", "Import server from JSON"))
	kHint(c, mcpCatalogText("支持单个 spec、命名对象或 mcpServers 对象。每次只选择一项；解析和预览不连接、不保存，也不执行文档里的指令。", "Accepts a single spec, named object or mcpServers object. Select one entry at a time. Parsing and previewing neither connect nor save or execute document instructions."))
	text, name := s.Text, s.DefaultName
	changed := kTextarea(c, &text, coss.TextareaOptions{
		Label:    mcpCatalogText("从 JSON 导入服务器", "Import server from JSON"),
		Disabled: busy,
	}).MinHeight(80).Changed()
	kHint(c, mcpCatalogText("单个 spec 的默认名称（可留空沿用当前表单）", "Default name for a single spec (blank uses the current form)"))
	if kInput(c, &name, coss.InputOptions{
		Label:    mcpCatalogText("默认名称", "Default name"),
		Disabled: busy,
	}).Changed() {
		changed = true
	}
	if changed && a.OnInput != nil {
		a.OnInput(text, name)
	}
	ui.Row(c).Gap(design.Sp(2)).Children(func() {
		if kPrimary(c, mcpCatalogText("解析并预览", "Parse and preview"), busy || text == "").Clicked() && a.OnParse != nil {
			a.OnParse()
		}
		if kButtonDisabled(c, mcpCatalogText("清除导入预览", "Clear import preview"), busy).Clicked() && a.OnClear != nil {
			a.OnClear()
		}
	})
	if s.Err != "" {
		settingsFailureLine(c, "mcp:import", providerCopy("没能解析这份 MCP 配置，请检查内容后重试。", "The MCP configuration could not be parsed. Check it and try again."), s.Err)
	}
	settingsNotice(c, "mcp_import:48", s.Notice != "", func() string { return s.Notice })
	if !s.Ready {
		return
	}
	kHint(c, fmt.Sprintf(mcpCatalogText("%d 个候选配置；忽略 %d 个非服务器条目。尚未批量应用任何配置。", "%d candidate configurations; %d non-server entries ignored. No batch application is performed."), len(s.Candidates), s.Ignored))
	for index, candidate := range s.Candidates {
		row := candidate
		choice := index
		label := row.ID + " · " + row.Transport
		if choice == s.Selected {
			label = "• " + label
		}
		if kChoiceButton(c, label, choice == s.Selected, busy || row.Err != "").AlignSelf(ui.Start).Clicked() && a.OnSelect != nil {
			a.OnSelect(choice)
		}
		if row.Err != "" {
			settingsFailureLine(c, "mcp:import-row/"+row.ID, providerCopy("这一行没能导入，请检查后重试。", "This row could not be imported. Check it and try again."), row.Err)
		}
	}
	if s.Selected < 0 || s.Selected >= len(s.Candidates) {
		return
	}
	selected := s.Candidates[s.Selected]
	kLabel(c, selected.ID+" · "+selected.Name+" · "+selected.Transport, fg)
	enabled := mcpCatalogText("禁用", "disabled")
	if selected.Enabled {
		enabled = mcpCatalogText("启用", "enabled")
	}
	kHint(c, mcpCatalogText("保存后状态：", "State after saving: ")+enabled)
	if selected.Transport == "stdio" {
		kText(c, mcpCatalogText("命令：", "Command: ")+selected.Command, design.TextXS, design.WeightNormal, fg).FillWidth()
		kText(c, mcpCatalogText("参数：", "Arguments: ")+selected.ArgsJSON, design.TextXS, design.WeightNormal, fg).FillWidth()
		if selected.Cwd != "" {
			kHint(c, mcpCatalogText("工作目录：", "Working directory: ")+selected.Cwd)
		}
	} else {
		kText(c, "URL: "+selected.URL, design.TextXS, design.WeightNormal, fg).FillWidth()
	}
	kHint(c, fmt.Sprintf(mcpCatalogText("环境变量 %d 项，请求头 %d 项；值在载入后的编辑器中复核。", "%d environment entries, %d headers; review their values in the editor after loading."), selected.EnvCount, selected.HeaderCount))
	for _, warning := range selected.Warnings {
		kHint(c, warning)
	}
	replacement := s.ReplaceExisting
	if selected.Existing {
		if kCheckbox(c, &replacement, mcpCatalogText("明确选择编辑同 ID 的现有服务器：", "Explicitly choose to edit the existing server with this ID: ")+selected.ID, busy || !selected.Writable).Changed() && a.OnReplace != nil {
			a.OnReplace(replacement)
		}
	}
	kHint(c, mcpCatalogText("载入会替换当前未保存草稿；必须再点击“保存配置”才写入 Host。保存启用配置可能启动命令或连接 HTTP。", "Loading replaces the current unsaved draft. A separate Save configuration action writes to the Host and may start a command or connect over HTTP when enabled."))
	if kPrimary(c, mcpCatalogText("确认载入所选 1 项到草稿", "Confirm loading this one entry into the draft"), busy || selected.Err != "" || (selected.Existing && !replacement)).AlignSelf(ui.Start).Clicked() && a.OnStage != nil {
		a.OnStage()
	}
}

func mcpImportPane(c *ui.Context, a *MCPImportAccess, busy bool, fg, muted ui.Color) {
	if a == nil {
		return
	}
	box := ui.Column(c.Key("mcp-import-dialog"))
	open := ui.Local(box, "open", func() bool { return false })
	box.Children(func() {
		if kButtonDisabled(c, mcpCatalogText("从 JSON 导入", "Import from JSON"), busy).AlignSelf(ui.Start).Clicked() {
			*open = true
		}
		bodyDialog(c, open, mcpCatalogText("导入 MCP 服务器", "Import MCP server"), func() { mcpImportContent(c, a, busy, fg, muted) })
	})
}
