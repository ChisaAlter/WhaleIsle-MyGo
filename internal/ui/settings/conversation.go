package settings

import (
	"github.com/egoist/mygo/ui"
)

// ConversationAccess is the 对话 page: the conversation preferences that the
// installed build moved out of General and Interface. Every row is backed by a
// described Host namespace; a row the Host does not describe is not drawn.
type ConversationAccess struct {
	Host *BodySettingsAccess
}

func conversationPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Conversation == nil {
		return
	}
	host := m.Conversation.Host
	bodyHostState(c, host, muted)
	generalPermissionPreference(c, host, fg, muted)
	bodyHostPreference(c, host, "ui-conversation", "busyEnter", providerCopy("繁忙时的发送行为", "Send behavior while busy"), providerCopy("智能体运行时 Enter 键和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为", "What Enter and Send do while the agent runs; Cmd/Ctrl+Enter uses the other action"), fg, muted)
	bodyHostPreference(c, host, "ui-chat", "transcriptView", providerCopy("工作步骤展示", "Work step display"), providerCopy("选择希望看到多少工具调用细节", "Choose how much tool-call detail to show"), fg, muted)
	bodyHostPreference(c, host, "ui-chat", "collapseTiming", providerCopy("工作步骤收起时机", "Collapse completed work steps"), providerCopy("选择何时自动收起工作步骤", "Choose when completed work steps collapse"), fg, muted)
	bodyHostPreference(c, host, "ui-chat", "linkOpening", providerCopy("网页链接默认打开方式", "Default link opening"), providerCopy("对话中网页链接的打开位置", "Where links in the conversation open"), fg, muted)
	bodyHostPreference(c, host, "ui-conversation", "nextPrompt", providerCopy("下一句提示", "Next prompt suggestions"), providerCopy("回复结束后建议下一句，可按 Tab 采纳，不会自动发送。生成提示会额外调用当前模型；关闭后不再调用。", "Suggest the next message after a reply; Tab accepts it and nothing is sent automatically. Generating a suggestion calls the current model once more; turning this off stops those calls."), fg, muted)
	bodyHostPreference(c, host, "visual-replies", "enabled", providerCopy("视觉回复", "Visual replies"), providerCopy("允许智能体生成交互式页面并在会话内展示。截图检查需要支持图片的模型，或已配置的识图模型；关闭后已生成页面仍可查看。", "Let the agent generate interactive pages and show them in the conversation. Screenshot checks need an image-capable model or a configured vision model; pages already generated stay viewable when this is off."), fg, muted)
	generalCustomInstructions(c, host, fg, muted)
}
