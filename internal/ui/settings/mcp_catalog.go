package settings

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/ui/i18n"
)

type MCPCatalogAccess struct {
	Snapshot                     mcpconfig.Snapshot
	OnQuery, OnFilter, OnInspect func(string)
}

func mcpCatalogText(zh, en string) string {
	if i18n.Locale() == "zh-CN" {
		return zh
	}
	return en
}

func mcpCatalogControls(c *ui.Context, a *MCPCatalogAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	query := a.Snapshot.Query
	if ui.TextInput(c, &query).FillWidth().Placeholder(mcpCatalogText("搜索名称、ID、命令或 URL", "Search name, ID, command or URL")).Changed() && a.OnQuery != nil {
		a.OnQuery(query)
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, choice := range []struct{ id, zh, en string }{{"all", "全部", "All"}, {"enabled", "已启用", "Enabled"}, {"disabled", "已禁用", "Disabled"}} {
			label := mcpCatalogText(choice.zh, choice.en)
			active := a.Snapshot.EnabledFilter == choice.id || (choice.id == "all" && a.Snapshot.EnabledFilter == "")
			if active {
				label = "• " + label
			}
			if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() && a.OnFilter != nil {
				a.OnFilter(choice.id)
			}
		}
		if (a.Snapshot.Query != "" || (a.Snapshot.EnabledFilter != "" && a.Snapshot.EnabledFilter != "all")) && ui.Button(c, mcpCatalogText("清除筛选", "Clear filters")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			if a.OnQuery != nil {
				a.OnQuery("")
			}
			if a.OnFilter != nil {
				a.OnFilter("all")
			}
		}
	})
	ui.Text(c, fmt.Sprintf(mcpCatalogText("显示 %d 个服务器，共 %d 个", "Showing %d of %d servers"), len(a.Snapshot.CatalogRows()), len(a.Snapshot.Rows))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if a.Snapshot.Err != "" && a.Snapshot.Ready {
		ui.Text(c, mcpCatalogText("操作失败时保留上次成功读取的列表，请查看错误后刷新。", "On failure, the last successful list is retained. Check the error before refreshing.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
}

func mcpCatalogDetails(c *ui.Context, a *MCPCatalogAccess, id string, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	diagnostic, tested := s.Tests[id]
	if tested {
		if diagnostic.Err != "" {
			ui.Text(c, mcpCatalogText("上次测试失败：", "Last test failed: ")+diagnostic.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		} else {
			ui.Text(c, fmt.Sprintf(mcpCatalogText("上次测试成功：%d 个工具 · %.0f 毫秒", "Last test succeeded: %d tools · %.0f ms"), int(diagnostic.Result.ToolCount), diagnostic.Result.ElapsedMs)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	}
	open := s.ExpandedID == id
	label := mcpCatalogText("查看工具明细", "Inspect tools")
	if open {
		label = mcpCatalogText("收起工具明细", "Hide tools")
	}
	if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() && a.OnInspect != nil {
		a.OnInspect(id)
	}
	if !open {
		return
	}
	ui.Text(c, mcpCatalogText("已注册工具（上次刷新）", "Registered tools (last refresh)")).FontSize(fonttheme.InterfaceSize(11)).Bold().TextColor(fg)
	var tools []string
	for _, row := range s.Rows {
		if row.Id == id && row.Connection != nil {
			tools = row.Connection.Tools
			break
		}
	}
	if len(tools) == 0 {
		ui.Text(c, mcpCatalogText("当前快照没有已注册工具；查看明细不会尝试连接。", "No registered tools in this snapshot; inspection does not attempt a connection.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	for _, name := range tools {
		ui.Text(c, name).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
	if tested && diagnostic.Err == "" {
		ui.Text(c, mcpCatalogText("上次显式测试返回的工具", "Tools returned by the last explicit test")).FontSize(fonttheme.InterfaceSize(11)).Bold().TextColor(fg)
		if len(diagnostic.Result.ToolNames) == 0 {
			ui.Text(c, mcpCatalogText("测试返回空工具列表", "Test returned an empty tool list")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for _, name := range diagnostic.Result.ToolNames {
			ui.Text(c, name).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
	}
}
