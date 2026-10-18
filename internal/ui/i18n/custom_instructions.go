package i18n

const (
	KeyCustomInstructionsTitle = "settings.customInstructions.title"
	KeyCustomInstructionsScope = "settings.customInstructions.scope"
	KeyCustomInstructionsLimit = "settings.customInstructions.limit"
)

func init() {
	zhCN[KeyCustomInstructionsTitle] = "自定义指令"
	en[KeyCustomInstructionsTitle] = "Custom instructions"
	zhCN[KeyCustomInstructionsScope] = "保存后从下一次模型请求起生效，作用于当前 Host 的所有会话及继承全局提示词的子智能体。空白文本表示不启用；{{变量}} 保留原文。此项不修改工作区 AGENTS.md。"
	en[KeyCustomInstructionsScope] = "Save to apply from the next model request to all sessions on this Host and subagents inheriting its global prompt. Blank text disables it; {{variables}} remain literal. Workspace AGENTS.md is unchanged."
	zhCN[KeyCustomInstructionsLimit] = "超过 1500 字符上限，请缩短后保存（部分表情占两个字符）。草稿已保留。"
	en[KeyCustomInstructionsLimit] = "Over the 1500-unit limit. Shorten before saving; some emoji count as two. Your draft is retained."
}
