package interaction

import "whaleisle/internal/ui/i18n"

type interactionLocaleEntry struct {
	module string
	key    string
	zh     string
	en     string
}

var interactionLocaleCatalog = []interactionLocaleEntry{
	{module: `dsh-client-ui-user-questions`, key: `review.skipped`, zh: `这道问题当时被跳过。`, en: `This question was skipped.`},
	{module: `dsh-client-ui-user-questions`, key: `reply.label`, zh: `回答先前等待中的问题`, en: `Reply to earlier pending questions`},
	{module: `dsh-client-ui-user-questions`, key: `reply.open`, zh: `展开问题详情`, en: `Open question details`},
	{module: `dsh-client-ui-user-questions`, key: `reply.close`, zh: `收起问题详情`, en: `Close question details`},
	{module: `dsh-client-ui-user-questions`, key: `nav.minimize`, zh: `收起问题卡片`, en: `Collapse the question card`},
	{module: `dsh-client-ui-user-questions`, key: `nav.maximize`, zh: `展开问题卡片`, en: `Expand the question card`},
	{module: `dsh-client-ui-user-questions`, key: `nav.cancel`, zh: `放弃整组问题`, en: `Dismiss all questions`},
	{module: `dsh-client-ui-user-questions`, key: `nav.close`, zh: `收起问题面板，可从工具调用重新打开`, en: `Close the panel — reopen it from the tool call`},
	{module: `dsh-client-ui-user-questions`, key: `custom.placeholder`, zh: `输入你的答案`, en: `Type your answer`},
	{module: `dsh-client-ui-plan`, key: `preview.document`, zh: `计划 · Markdown`, en: `Plan · Markdown`},
	{module: `dsh-client-ui-plan`, key: `preview.open`, zh: `在侧边栏打开计划`, en: `Open plan in sidebar`},
	{module: `dsh-client-ui-plan`, key: `preview.full`, zh: `查看全文`, en: `View full plan`},
	{module: `dsh-client-ui-plan`, key: `preview.openNamed`, zh: `打开计划：{title}`, en: `Open plan: {title}`},
	{module: `dsh-client-ui-plan`, key: `preview.loading`, zh: `正在读取计划…`, en: `Loading plan…`},
	{module: `dsh-client-ui-plan`, key: `preview.failed`, zh: `无法读取计划`, en: `Could not load plan`},
	{module: `dsh-client-ui-plan`, key: `preview.invalidAddress`, zh: `计划地址无效`, en: `Invalid plan address`},
	{module: `dsh-client-ui-plan`, key: `preview.historyUnavailable`, zh: `无法读取会话历史`, en: `Session history is unavailable`},
	{module: `dsh-client-ui-plan`, key: `preview.unavailable`, zh: `计划预览不可用`, en: `Plan preview is unavailable`},
	{module: `dsh-client-ui-plan`, key: `chip.exitFailed`, zh: `退出计划模式失败`, en: `Failed to exit plan mode`},
	{module: `dsh-client-ui-approval`, key: `escalation`, zh: `工具 {toolName} 请求越权执行`, en: `Tool {toolName} requests privileged execution`},
	{module: `dsh-client-ui-approval`, key: `reject`, zh: `拒绝`, en: `Reject`},
	{module: `dsh-client-ui-user-questions`, key: `error.incomplete`, zh: `请先完成这道问题。`, en: `Please complete this question first.`},
	{module: `dsh-client-ui-user-questions`, key: `error.unanswered`, zh: `请选择一个选项或填写自定义答案。`, en: `Please select an option or enter a custom answer.`},
	{module: `dsh-client-ui-user-questions`, key: `wait.takeTime`, zh: `慢慢回答`, en: `Take time`},
	{module: `dsh-client-ui-user-questions`, key: `wait.held`, zh: `会一直等你回答`, en: `Waiting until you answer`},
	{module: `dsh-client-ui-user-questions`, key: `wait.countdown`, zh: `{seconds} 秒后继续工作`, en: `Continuing in {seconds}s`},
	{module: `dsh-client-ui-user-questions`, key: `wait.paused`, zh: `已暂停 · 剩余 {seconds} 秒`, en: `Paused · {seconds}s remaining`},
	{module: `dsh-client-ui-user-questions`, key: `review.status`, zh: `已回答`, en: `Answered`},
	{module: `dsh-client-ui-user-questions`, key: `plan.approve`, zh: `同意执行`, en: `Approve`},
	{module: `dsh-client-ui-user-questions`, key: `plan.decline`, zh: `拒绝`, en: `Refuse`},
	{module: `dsh-client-ui-user-questions`, key: `action.next`, zh: `下一题`, en: `Next`},
}

func init() {
	for _, entry := range interactionLocaleCatalog {
		i18n.Register("interaction"+"."+entry.module+"."+entry.key, entry.zh, entry.en)
	}
}

func interactionLocaleText(key string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	for _, entry := range interactionLocaleCatalog {
		if entry.key == key || entry.module+"."+entry.key == key {
			if zh {
				return entry.zh
			}
			return entry.en
		}
	}
	return i18n.T(key)
}
