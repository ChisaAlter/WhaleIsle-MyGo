package settings

import (
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type agentPresetCopy struct{ Name, Description string }

func agentPresetDisplay(row session.ManagedAgentPreset) (string, string) {
	if row.BuiltIn {
		copies := agentPresetCopiesZH
		if i18n.Locale() == i18n.LocaleEN {
			copies = agentPresetCopiesEN
		}
		if copy, ok := copies[row.ID]; ok {
			return copy.Name, copy.Description
		}
	}
	name := row.Name
	if name == "" {
		name = row.ID
	}
	return name, row.Description
}

var agentPresetCopiesEN = map[string]agentPresetCopy{
	"standard": {Name: "Standard mode", Description: "Work with code, files, and information. Suitable for most tasks, with search, editing, terminal commands, and other tools available as needed."},
	"ptc":      {Name: "PTC mode", Description: "Includes all Standard mode capabilities. Better suited to tasks that call tools in batches and then filter, organize, deduplicate, count, or summarize the results."},
	"minimal":  {Name: "Minimal mode", Description: "The agent uses a terminal and image reading, with context compaction. Useful for testing and comparing its basic performance."},
	"cordis":   {Name: "Creator mode", Description: "Customize DSH through conversation. Let the agent write plugins that add features or UI, or combine tools and prompts to create your own mode."},
}

var agentPresetCopiesZH = map[string]agentPresetCopy{
	"standard": {Name: "标准模式", Description: "处理代码、文件和资料，适合大多数任务。Agent 会按需使用检索、编辑和终端等工具。"},
	"ptc":      {Name: "PTC 模式", Description: "包含标准模式的所有能力，更适合批量调用工具，并对结果进行筛选、整理、去重、统计或汇总的任务。"},
	"minimal":  {Name: "极简模式", Description: "Agent 使用终端和图片读取工具，并支持上下文压缩，适合测试和对比其基础表现。"},
	"cordis":   {Name: "创造模式", Description: "用对话定制 DSH：让 Agent 编写插件，添加新功能或界面；也能组合工具和提示词，创建自己的模式。"},
}
