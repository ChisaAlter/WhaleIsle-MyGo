package composer

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

type localePair struct {
	ZH string
	EN string
}

func resolveLocalePair(pair localePair) string {
	if i18n.Locale() == i18n.LocaleZH {
		if pair.ZH != "" {
			return pair.ZH
		}
		return pair.EN
	}
	if pair.EN != "" {
		return pair.EN
	}
	return pair.ZH
}

func formatLocaleTemplate(tmpl string, vars map[string]string) string {
	out := tmpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// commandLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-commands locales.js.
var commandLocaleCatalog = map[string]localePair{
	"command.attachmentsUnsupported": {
		ZH: "/{name} 不支持附加图片或文件，请移除后再试",
		EN: "/{name} does not support attached images or files; remove them and try again",
	},
	"search.placeholder": {
		ZH: "筛选选项…",
		EN: "Filter options…",
	},
	"search.aria": {
		ZH: "筛选选项",
		EN: "Filter options",
	},
	"status.loading": {
		ZH: "正在加载选项…",
		EN: "Loading options…",
	},
	"status.empty": {
		ZH: "无选项",
		EN: "No options",
	},
	"status.noMatch": {
		ZH: "无匹配项",
		EN: "No matches",
	},
	"listbox.aria": {
		ZH: "/{command} 匹配项",
		EN: "/{command} matches",
	},
}

// CommandLocaleText returns the localized text for a dsh-client-ui-commands locale key.
func CommandLocaleText(key string) string {
	if pair, ok := commandLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatCommandLocale formats a dsh-client-ui-commands locale template with named variables.
func FormatCommandLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(CommandLocaleText(key), vars)
}

// CommandSearchAriaLabel returns the localized aria label for command option search.
func CommandSearchAriaLabel() string {
	return CommandLocaleText("search.aria")
}

// CommandLoadingStatusText returns the localized loading options status string.
func CommandLoadingStatusText() string {
	return CommandLocaleText("status.loading")
}

// CommandEmptyStatusText returns the localized empty options status string.
func CommandEmptyStatusText() string {
	return CommandLocaleText("status.empty")
}

// FormatCommandListboxAria formats the localized aria label for a slash command listbox.
func FormatCommandListboxAria(command string) string {
	cmd := strings.TrimPrefix(strings.TrimSpace(command), "/")
	if cmd == "" {
		cmd = "command"
	}
	return FormatCommandLocale("listbox.aria", map[string]string{"command": cmd})
}

// inputTriggerLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-input-trigger locales.js.
var inputTriggerLocaleCatalog = map[string]localePair{
	"drill.aria": {
		ZH: "进入目录",
		EN: "Browse folder",
	},
	"drill.hint": {
		ZH: "进入目录",
		EN: "Browse folder",
	},
	"crumbs.aria": {
		ZH: "目录导航",
		EN: "Folder navigation",
	},
	"suggestions.aria": {
		ZH: "触发候选建议",
		EN: "Trigger suggestions",
	},
}

// InputTriggerLocaleText returns the localized text for a dsh-client-ui-input-trigger key.
func InputTriggerLocaleText(key string) string {
	if pair, ok := inputTriggerLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// InputTriggerDrillAriaLabel returns the localized aria label for drilling into a directory.
func InputTriggerDrillAriaLabel() string {
	return InputTriggerLocaleText("drill.aria")
}

// InputTriggerDrillHintText returns the localized hint for drilling into a directory.
func InputTriggerDrillHintText() string {
	return InputTriggerLocaleText("drill.hint")
}

// InputTriggerCrumbsAriaLabel returns the localized aria label for directory breadcrumbs.
func InputTriggerCrumbsAriaLabel() string {
	return InputTriggerLocaleText("crumbs.aria")
}

// InputTriggerSuggestionsAriaLabel returns the localized aria label for trigger suggestions.
func InputTriggerSuggestionsAriaLabel() string {
	return InputTriggerLocaleText("suggestions.aria")
}

// modelSelectionLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-model-selection locales.js.
var modelSelectionLocaleCatalog = map[string]localePair{
	"option.loadError": {
		ZH: "目录加载失败：{message}",
		EN: "Catalog failed to load: {message}",
	},
	"trigger.loading": {
		ZH: "正在加载模型…",
		EN: "Loading models…",
	},
	"trigger.aria": {
		ZH: "选择模型，当前 {model}",
		EN: "Select model, current {model}",
	},
	"trigger.ariaEffort": {
		ZH: "选择模型，当前 {model}，推理等级 {effort}",
		EN: "Select model, current {model}, reasoning effort {effort}",
	},
	"menu.aria": {
		ZH: "模型与推理等级",
		EN: "Model and reasoning effort",
	},
	"status.loading": {
		ZH: "正在刷新模型列表…",
		EN: "Refreshing model list…",
	},
	"error.action": {
		ZH: "模型操作失败：{message}",
		EN: "Model operation failed: {message}",
	},
	"search.empty": {
		ZH: "没有匹配的模型。",
		EN: "No matching models.",
	},
	"empty.efforts": {
		ZH: "当前模型未提供推理等级。",
		EN: "This model provides no reasoning effort levels.",
	},
	"command.description": {
		ZH: "选择本会话使用的模型",
		EN: "Select the model for this conversation",
	},
	"trigger.fallback": {
		ZH: "请选择模型",
		EN: "Select model",
	},
	"warning.groupLoad": {
		ZH: "{name} 加载失败：{message}",
		EN: "{name} failed to load: {message}",
	},
	"empty.models": {
		ZH: "暂无可用模型",
		EN: "No models available.",
	},
}

// ModelSelectionLocaleText returns the localized text for a dsh-client-ui-model-selection key.
func ModelSelectionLocaleText(key string) string {
	if pair, ok := modelSelectionLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatModelSelectionLocale formats a dsh-client-ui-model-selection locale template.
func FormatModelSelectionLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(ModelSelectionLocaleText(key), vars)
}

// FormatModelTriggerAria returns the localized trigger aria label for the current model and effort.
func FormatModelTriggerAria(modelName, effortLabel string) string {
	modelName = strings.TrimSpace(modelName)
	effortLabel = strings.TrimSpace(effortLabel)
	if effortLabel != "" {
		return FormatModelSelectionLocale("trigger.ariaEffort", map[string]string{
			"model":  modelName,
			"effort": effortLabel,
		})
	}
	return FormatModelSelectionLocale("trigger.aria", map[string]string{
		"model": modelName,
	})
}

// FormatModelCatalogLoadError formats a catalog load failure message.
func FormatModelCatalogLoadError(message string) string {
	return FormatModelSelectionLocale("option.loadError", map[string]string{
		"message": strings.TrimSpace(message),
	})
}

// FormatModelActionError formats a model operation failure message.
func FormatModelActionError(message string) string {
	return FormatModelSelectionLocale("error.action", map[string]string{
		"message": strings.TrimSpace(message),
	})
}

// referenceLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-reference locales.js.
var referenceLocaleCatalog = map[string]localePair{
	"section.files": {
		ZH: "文件与文件夹",
		EN: "Files & folders",
	},
	"candidate.noCwd": {
		ZH: "（无工作目录）",
		EN: "(no cwd)",
	},
	"fileOpen.failed": {
		ZH: "无法打开“{path}”，请重试或关闭此提示",
		EN: "Couldn’t open “{path}”. Try again or close this message.",
	},
	"time.days": {
		ZH: "{n}天",
		EN: "{n}d",
	},
	"time.years": {
		ZH: "{n}年",
		EN: "{n}y",
	},
}

// ReferenceLocaleText returns the localized text for a dsh-client-ui-reference key.
func ReferenceLocaleText(key string) string {
	if pair, ok := referenceLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatReferenceLocale formats a dsh-client-ui-reference locale template.
func FormatReferenceLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(ReferenceLocaleText(key), vars)
}

// FormatReferenceFileOpenFailed formats the file-open failure toast/banner message.
func FormatReferenceFileOpenFailed(path string) string {
	return FormatReferenceLocale("fileOpen.failed", map[string]string{
		"path": strings.TrimSpace(path),
	})
}

// FormatReferenceRelativeDays formats a day-granularity relative age badge.
func FormatReferenceRelativeDays(days int) string {
	if days < 1 {
		days = 1
	}
	return FormatReferenceLocale("time.days", map[string]string{
		"n": fmt.Sprintf("%d", days),
	})
}

// FormatReferenceRelativeYears formats a year-granularity relative age badge.
func FormatReferenceRelativeYears(years int) string {
	if years < 1 {
		years = 1
	}
	return FormatReferenceLocale("time.years", map[string]string{
		"n": fmt.Sprintf("%d", years),
	})
}

// conversationLocaleCatalog mirrors @deepseek-ai/dsh-client-ui-conversation locales.js.
var conversationLocaleCatalog = map[string]localePair{
	"hint.goal": {
		ZH: "输入目标，智能体将持续执行",
		EN: "describe the objective for a long-running task",
	},
	"hint.goal.active": {
		ZH: "当前目标进行中。可输入 edit 修改 / pause 暂停 / resume 继续 / clear 清除",
		EN: "goal active — edit / pause / resume / clear",
	},
	"input.commands": {
		ZH: "添加文件或调用指令",
		EN: "Add files or run commands",
	},
	"input.editCancel": {
		ZH: "取消重新编辑",
		EN: "Cancel re-edit",
	},
	"attachment.scrollLeft": {
		ZH: "向左滚动附件",
		EN: "Scroll attachments left",
	},
	"attachment.scrollRight": {
		ZH: "向右滚动附件",
		EN: "Scroll attachments right",
	},
	"attachment.dropBlocked": {
		ZH: "当前无法添加文件或图片",
		EN: "Files and images cannot be added right now",
	},
	"attachment.directoryDesktopOnly": {
		ZH: "只有桌面端支持添加文件夹，浏览器里请添加单个文件",
		EN: "Folders can only be added in the desktop app; add individual files in the browser",
	},
	"attachment.pathUnavailable": {
		ZH: "无法获取文件夹路径，请重新拖入",
		EN: "Could not obtain the folder path; drag it in again",
	},
	"attachment.pathUnsupported": {
		ZH: "路径含有无法引用的字符，请改名后再试",
		EN: "The path contains characters a reference cannot carry; rename it and try again",
	},
	"image.pending": {
		ZH: "待发送图片",
		EN: "Pending images",
	},
	"image.remove": {
		ZH: "移除图片 {name}",
		EN: "Remove image {name}",
	},
	"image.unsupportedType": {
		ZH: "仅支持 PNG、JPG、WebP、GIF 格式的图片",
		EN: "Only PNG, JPG, WebP, and GIF images are supported",
	},
	"image.tooManyPixels": {
		ZH: "图片分辨率过大，请压缩后重试",
		EN: "Image resolution is too high; compress it and try again",
	},
	"image.dimensionTooLarge": {
		ZH: "图片宽高不能超过 {size}px，请缩小后重试",
		EN: "Image sides must be at most {size}px; downscale it and try again",
	},
	"image.modelUnsupported": {
		ZH: "当前模型不支持图片，请切换支持图片的模型",
		EN: "The current model does not support images; switch to a model that does",
	},
	"image.sendFailed": {
		ZH: "图片发送失败（{reason}），请重新添加图片后再试",
		EN: "Sending images failed ({reason}); re-add them and try again",
	},
	"file.pending": {
		ZH: "待发送文件",
		EN: "Pending files",
	},
	"file.remove": {
		ZH: "移除文件 {name}",
		EN: "Remove file {name}",
	},
	"file.directoryRejected": {
		ZH: "「{name}」是文件夹，无法作为附件",
		EN: "\"{name}\" is a folder and cannot be attached",
	},
	"file.directoriesRejected": {
		ZH: "{count} 个文件夹无法作为附件",
		EN: "{count} items are folders and cannot be attached",
	},
	"file.stillUploading": {
		ZH: "文件还在上传，请等待上传完成后发送",
		EN: "Files are still uploading; send after they finish",
	},
	"file.sessionUnavailable": {
		ZH: "会话不可用，无法上传文件",
		EN: "Session unavailable; files cannot be uploaded",
	},
	"file.notStaged": {
		ZH: "文件尚未上传成功，请重新添加后再试",
		EN: "The file has not finished uploading; re-add it and try again",
	},
	"settings.customInstructions.counter": {
		ZH: "{count} / {max}",
		EN: "{count} / {max}",
	},
	"settings.beam.configure": {
		ZH: "设置思考炫光",
		EN: "Configure thinking glow",
	},
	"settings.beam.panel.close": {
		ZH: "关闭思考炫光设置",
		EN: "Close thinking glow settings",
	},
	"settings.beam.panel.preview": {
		ZH: "思考炫光预览",
		EN: "Thinking glow preview",
	},
	"settings.beam.panel.motion": {
		ZH: "运动",
		EN: "Motion",
	},
	"settings.beam.panel.night": {
		ZH: "夜间调暗",
		EN: "Night dimming",
	},
	"settings.beam.panel.presets": {
		ZH: "用户预设",
		EN: "User presets",
	},
	"settings.beam.panel.presetSelect": {
		ZH: "选择预设",
		EN: "Select preset",
	},
	"settings.beam.panel.exchange": {
		ZH: "配置交换",
		EN: "Configuration exchange",
	},
	"settings.beam.panel.importJson": {
		ZH: "导入 JSON",
		EN: "Import JSON",
	},
	"settings.beam.panel.jsonLimit": {
		ZH: "最大 {value} KiB；导入后仍需点击保存",
		EN: "Maximum {value} KiB; click Save after importing",
	},
	"settings.beam.panel.presetNameRequired": {
		ZH: "请输入预设名称",
		EN: "Enter a preset name",
	},
	"settings.beam.panel.clipboardWriteFailed": {
		ZH: "无法写入剪贴板",
		EN: "Could not write to the clipboard",
	},
	"settings.beam.panel.clipboardReadFailed": {
		ZH: "剪贴板 JSON 无效或不可读取",
		EN: "Clipboard JSON is invalid or unavailable",
	},
	"settings.beam.panel.saveFailed": {
		ZH: "保存失败，请重试",
		EN: "Save failed; try again",
	},
	"settings.beam.panel.seconds": {
		ZH: "{value} 秒",
		EN: "{value} s",
	},
	"settings.beam.panel.degrees": {
		ZH: "{value}°",
		EN: "{value}°",
	},
	"settings.typingFx.configure": {
		ZH: "设置输入特效",
		EN: "Configure typing effects",
	},
	"settings.typingFx.panel.close": {
		ZH: "关闭输入特效设置",
		EN: "Close typing effect settings",
	},
	"settings.typingFx.panel.preview": {
		ZH: "输入特效预览",
		EN: "Typing effect preview",
	},
	"settings.typingFx.panel.presets": {
		ZH: "用户预设",
		EN: "User presets",
	},
	"settings.typingFx.panel.presetSelect": {
		ZH: "选择预设",
		EN: "Select a preset",
	},
	"settings.typingFx.panel.exchange": {
		ZH: "配置交换",
		EN: "Configuration exchange",
	},
	"settings.typingFx.panel.importJson": {
		ZH: "导入 JSON",
		EN: "Import JSON",
	},
	"settings.typingFx.panel.jsonLimit": {
		ZH: "最大 {value} KiB；导入后仍需点击保存",
		EN: "Maximum {value} KiB; click Save after importing",
	},
	"settings.typingFx.panel.presetNameRequired": {
		ZH: "请输入预设名称",
		EN: "Enter a preset name",
	},
	"settings.typingFx.panel.clipboardWriteFailed": {
		ZH: "无法写入剪贴板",
		EN: "Could not write to the clipboard",
	},
	"settings.typingFx.panel.clipboardReadFailed": {
		ZH: "剪贴板 JSON 无效或不可读取",
		EN: "Clipboard JSON is invalid or unavailable",
	},
	"settings.typingFx.panel.saveFailed": {
		ZH: "保存失败，请重试",
		EN: "Save failed; try again",
	},
	"peakValley.hint.offPeak": {
		ZH: "空闲时段（高峰以外全部时间，含周末）",
		EN: "Off-peak hours (everything outside the peak windows, weekends included)",
	},
	"sessionCost.label": {
		ZH: "当前会话费用：{cost}",
		EN: "Session cost: {cost}",
	},
	"sessionCost.labelPartial": {
		ZH: "当前会话费用：{cost}（部分模型未定价）",
		EN: "Session cost: {cost} (some models unpriced)",
	},
	"sessionCost.hitRateInline": {
		ZH: "（{percent}%）",
		EN: " ({percent}%)",
	},
	"composer.resize": {
		ZH: "拖动调整输入框高度",
		EN: "Resize composer height",
	},
	"composer.resizeWidth": {
		ZH: "拖动调整输入框宽度",
		EN: "Resize composer width",
	},
	"session.hierarchy": {
		ZH: "会话层级",
		EN: "Session hierarchy",
	},
	"header.actions.more": {
		ZH: "更多会话操作",
		EN: "More session actions",
	},
	"detail.days": {
		ZH: "{count} 天",
		EN: "{count} d",
	},
	"detail.seconds": {
		ZH: "{count} 秒",
		EN: "{count} s",
	},
	"detail.schedule.every": {
		ZH: "每 {interval}",
		EN: "Every {interval}",
	},
	"tool.title.findDefinition": {
		ZH: "查找定义",
		EN: "Find definition",
	},
	"tool.title.findImplementation": {
		ZH: "查找实现",
		EN: "Find implementation",
	},
	"tool.title.hoverSymbol": {
		ZH: "查看符号信息",
		EN: "Inspect symbol",
	},
	"detail.receipt.signal": {
		ZH: "信号已发送",
		EN: "Signal delivered",
	},
	"row.commandArguments": {
		ZH: "命令参数",
		EN: "Command arguments",
	},
	"tool.autoReviewRejected": {
		ZH: "Auto review 已拒绝",
		EN: "Rejected by Auto review",
	},
	"tool.autoReviewNotExecuted": {
		ZH: "工具未执行。原因：{reason}",
		EN: "Tool was not executed. Reason: {reason}",
	},
	"tool.autoReviewReasonFallback": {
		ZH: "Auto review 未授权此次操作",
		EN: "Auto review did not authorize this action",
	},
	"details.running": {
		ZH: "运行中…",
		EN: "Running…",
	},
	"queue.image": {
		ZH: "排队消息图片",
		EN: "Queued message image",
	},
	"queue.file": {
		ZH: "排队文件 {name}",
		EN: "Queued file {name}",
	},
	"queue.editFailed": {
		ZH: "编辑失败：这条消息可能已经开始发送。",
		EN: "Edit failed: this message may have already started sending.",
	},
	"queue.removeFailed": {
		ZH: "删除失败：这条消息可能已经开始发送。",
		EN: "Removal failed: this message may have already started sending.",
	},
	"queue.steerFailed": {
		ZH: "插话发送失败，请重试。",
		EN: "Steering failed. Try again.",
	},
	"queue.reorder": {
		ZH: "调整顺序：第 {n} 条，共 {total} 条",
		EN: "Reorder queued message {n} of {total}",
	},
	"queue.reorder.hint": {
		ZH: "拖动调整顺序，或按 ↑ ↓",
		EN: "Drag to reorder, or press ↑ ↓",
	},
	"queue.moveFailed": {
		ZH: "调整顺序失败：这条消息可能已经开始发送。",
		EN: "Reorder failed: this message may have already started sending.",
	},
	"hero.claudeHeadline": {
		ZH: "今天想一起探索什么？",
		EN: "What shall we explore today?",
	},
	"session.moreViews": {
		ZH: "更多会话视图",
		EN: "More conversation views",
	},
	"plan.approved": {
		ZH: "已批准执行",
		EN: "Approved",
	},
	"plan.rejected": {
		ZH: "已拒绝",
		EN: "Rejected",
	},
	"plan.cancelled": {
		ZH: "已取消",
		EN: "Cancelled",
	},
	"plan.reviewing": {
		ZH: "等待确认",
		EN: "Waiting for review",
	},
	"plan.view": {
		ZH: "查看计划",
		EN: "View plan",
	},
}

// ConversationLocaleText returns the localized text for a dsh-client-ui-conversation key.
func ConversationLocaleText(key string) string {
	if pair, ok := conversationLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatConversationLocale formats a dsh-client-ui-conversation locale template.
func FormatConversationLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(ConversationLocaleText(key), vars)
}

// GoalCommandHintText returns the localized hint for /goal depending on whether a goal is active.
func GoalCommandHintText(active bool) string {
	if active {
		return ConversationLocaleText("hint.goal.active")
	}
	return ConversationLocaleText("hint.goal")
}

// FormatSessionCostLabel formats the session cost summary string.
func FormatSessionCostLabel(cost string, partial bool) string {
	key := "sessionCost.label"
	if partial {
		key = "sessionCost.labelPartial"
	}
	return FormatConversationLocale(key, map[string]string{"cost": strings.TrimSpace(cost)})
}

// FormatSessionCostHitRateInline formats the inline cache hit rate suffix.
func FormatSessionCostHitRateInline(percent int) string {
	return FormatConversationLocale("sessionCost.hitRateInline", map[string]string{
		"percent": fmt.Sprintf("%d", percent),
	})
}

// FormatAutoReviewNotExecuted formats the auto-review rejection notice with a fallback reason.
func FormatAutoReviewNotExecuted(reason string) string {
	r := strings.TrimSpace(reason)
	if r == "" {
		r = ConversationLocaleText("tool.autoReviewReasonFallback")
	}
	return FormatConversationLocale("tool.autoReviewNotExecuted", map[string]string{"reason": r})
}

// voiceInputLocaleCatalog mirrors @deepseek-ai/dsh-experimental-client-ui-voice-input locales.js.
var voiceInputLocaleCatalog = map[string]localePair{
	"dictate": {
		ZH: "听写",
		EN: "Dictate",
	},
	"providerFeedback": {
		ZH: "{name}：{message}",
		EN: "{name}: {message}",
	},
	"saving": {
		ZH: "正在保存语音设置…",
		EN: "Saving voice preferences…",
	},
	"setupPrompt.title": {
		ZH: "使用语音输入前需要安装",
		EN: "Set up voice input before recording",
	},
	"setupPrompt.body": {
		ZH: "语音输入插件已开启。首次使用需要在本机下载并准备识别模型，请前往插件详情页查看空间、内存和时间说明，再开始安装。",
		EN: "Voice input is enabled. First use requires downloading and preparing local recognition models. Open the plugin details to review disk space, memory and time estimates before starting setup.",
	},
	"setupPrompt.open": {
		ZH: "前往安装",
		EN: "Go to setup",
	},
	"setupPrompt.trigger": {
		ZH: "打开语音输入引导",
		EN: "Open voice input setup",
	},
	"setupPrompt.details": {
		ZH: "前往语音插件设置",
		EN: "Open voice plugin settings",
	},
	"setupPrompt.unavailableTitle": {
		ZH: "语音识别尚未就绪",
		EN: "Speech recognition is not ready",
	},
	"setupPrompt.unavailableBody": {
		ZH: "请前往语音插件详情页查看识别服务状态、准备进度或错误信息",
		EN: "Open the voice plugin details to check recognition status, preparation progress, or errors.",
	},
	"setup.local": {
		ZH: "将在运行 DSH 的机器上下载本地模型，无需安装 Python 或编译工具。",
		EN: "Local models will be downloaded to the machine running DSH. No Python or compiler is required.",
	},
	"setup.disk": {
		ZH: "硬盘空间",
		EN: "Disk space",
	},
	"setup.diskValue": {
		ZH: "建议预留约 {gb} GB，包含模型、运行时和下载缓存",
		EN: "Allow about {gb} GB for models, runtime, and download caches",
	},
	"setup.memory": {
		ZH: "运行内存",
		EN: "Memory",
	},
	"setup.memoryValue": {
		ZH: "模型加载后约 {gb} GB，识别时可能更高",
		EN: "About {gb} GB with the model loaded; recognition may use more",
	},
	"setup.time": {
		ZH: "首次准备",
		EN: "First setup",
	},
	"setup.timeValue": {
		ZH: "参考 {min}–{max} 分钟，取决于网络和机器性能",
		EN: "Allow roughly {min}–{max} minutes, depending on the network and machine",
	},
	"setup.estimateNote": {
		ZH: "以上为估算；网络较慢时可能更久。已下载的资源会复用，默认在闲置后释放模型内存。",
		EN: "These are estimates; slow networks may take longer. Downloads are reused; idle models release memory by default.",
	},
	"step.model": {
		ZH: "准备语音识别模型",
		EN: "Prepare recognition model",
	},
	"step.vad": {
		ZH: "准备语音检测模型",
		EN: "Prepare voice detection model",
	},
	"step.verify": {
		ZH: "校验模型文件",
		EN: "Verify model files",
	},
	"short.downloading": {
		ZH: "下载中",
		EN: "Downloading",
	},
	"retryRecording": {
		ZH: "重新录音",
		EN: "Record again",
	},
	"preparationSteps": {
		ZH: "准备步骤",
		EN: "Preparation steps",
	},
	"prepare": {
		ZH: "下载并准备",
		EN: "Download and prepare",
	},
	"downloadBytes": {
		ZH: "下载：{completed} / {total} MB，{percent}%",
		EN: "Download: {completed} / {total} MB, {percent}%",
	},
	"preparationFailed": {
		ZH: "准备失败：{message}",
		EN: "Preparation failed: {message}",
	},
	"download.network": {
		ZH: "无法下载 {resource}：连接下载服务失败。",
		EN: "Cannot download {resource}: connection to the download service failed.",
	},
	"download.dns": {
		ZH: "无法下载 {resource}：无法解析下载地址。",
		EN: "Cannot download {resource}: the download address could not be resolved.",
	},
	"download.timeout": {
		ZH: "下载 {resource} 超时。",
		EN: "The download of {resource} timed out.",
	},
	"download.certificate": {
		ZH: "无法下载 {resource}：安全连接的证书校验失败。",
		EN: "Cannot download {resource}: certificate verification failed.",
	},
	"download.http": {
		ZH: "无法下载 {resource}：下载服务返回 HTTP {status}。",
		EN: "Cannot download {resource}: the download service returned HTTP {status}.",
	},
	"download.integrity": {
		ZH: "{resource} 下载不完整或文件校验失败。",
		EN: "The download of {resource} is incomplete or failed verification.",
	},
	"download.storage": {
		ZH: "无法保存 {resource}：磁盘空间不足或没有写入权限。",
		EN: "Cannot save {resource}: insufficient disk space or write permission.",
	},
	"downloadAdvice.network": {
		ZH: "请检查运行 DSH 的机器能否访问下载来源及其模型下载服务；如需代理，请在该机器上配置后重试。",
		EN: "Check that the machine running DSH can reach the download source and its model download services. Configure a proxy on that machine if needed, then retry.",
	},
	"downloadAdvice.dns": {
		ZH: "请检查运行 DSH 的机器的 DNS 和代理设置，确认可以解析下载来源的域名后重试。",
		EN: "Check DNS and proxy settings on the machine running DSH. Confirm that it can resolve the download source domain, then retry.",
	},
	"downloadAdvice.timeout": {
		ZH: "请检查运行 DSH 的机器的网络或代理连接，稍后重试。已下载并通过校验的文件会保留。",
		EN: "Check the network or proxy connection on the machine running DSH, then retry later. Verified downloaded files are retained.",
	},
	"downloadAdvice.certificate": {
		ZH: "请检查运行 DSH 的机器的系统时间、受信任证书和代理设置后重试。",
		EN: "Check the system clock, trusted certificates and proxy settings on the machine running DSH, then retry.",
	},
	"downloadAdvice.http": {
		ZH: "请确认下载地址可用、代理可正常连接下载服务，或稍后重试。",
		EN: "Confirm that the download address is available and the proxy can reach the download service, or retry later.",
	},
	"downloadAdvice.integrity": {
		ZH: "请重试下载。已完成并通过校验的其他文件会保留。",
		EN: "Retry the download. Other completed and verified files are retained.",
	},
	"downloadAdvice.storage": {
		ZH: "请检查运行 DSH 的机器的剩余磁盘空间，以及模型目录的写入权限后重试。",
		EN: "Check available disk space and write permission for the model directory on the machine running DSH, then retry.",
	},
	"downloadAdvice.unknown": {
		ZH: "请检查运行 DSH 的机器的网络、磁盘空间和模型目录权限后重试。",
		EN: "Check the network, disk space and model directory permissions on the machine running DSH, then retry.",
	},
	"sourceAutoHelp": {
		ZH: "优先使用响应较快的源，下载失败时自动尝试其他源。",
		EN: "Prefer the first responding source and try another if a download fails.",
	},
	"reconnecting": {
		ZH: "正在连接语音服务…",
		EN: "Connecting to speech recognition…",
	},
	"preparation.unprepared": {
		ZH: "首次使用需要下载本地识别模型。",
		EN: "First use downloads the local recognition models.",
	},
	"preparation.ready": {
		ZH: "本地语音已就绪。",
		EN: "Local speech recognition is ready.",
	},
	"preparation.standby": {
		ZH: "本地资源已准备，录音时自动唤醒。",
		EN: "Local resources are prepared. Recording wakes the recognizer.",
	},
	"preparation.cancelled": {
		ZH: "准备已取消，已完成的下载会保留。",
		EN: "Preparation cancelled. Completed downloads are retained.",
	},
	"preparation.cancelling": {
		ZH: "正在取消准备…",
		EN: "Cancelling preparation…",
	},
	"cloudReady": {
		ZH: "云端语音识别已就绪。",
		EN: "Cloud speech recognition is ready.",
	},
	"discard": {
		ZH: "丢弃识别文字",
		EN: "Discard transcript",
	},
	"transcribingShort": {
		ZH: "识别中…",
		EN: "Transcribing…",
	},
	"wakingShort": {
		ZH: "唤醒中…",
		EN: "Waking…",
	},
	"inputDevice": {
		ZH: "输入设备",
		EN: "Input device",
	},
	"systemMicrophoneNamed": {
		ZH: "系统默认（{name}）",
		EN: "System default ({name})",
	},
	"deviceMissing": {
		ZH: "麦克风不可用，请选择其他输入设备",
		EN: "Microphone unavailable. Choose another input device.",
	},
	"stop": {
		ZH: "停止并识别",
		EN: "Stop and transcribe",
	},
	"insert": {
		ZH: "插入文字",
		EN: "Insert text",
	},
	"loading": {
		ZH: "正在读取识别服务…",
		EN: "Loading recognition services…",
	},
	"requesting": {
		ZH: "请允许使用麦克风…",
		EN: "Allow microphone access to continue…",
	},
	"recording": {
		ZH: "正在录音…",
		EN: "Recording…",
	},
	"interrupted": {
		ZH: "录音中断，请重试。",
		EN: "Recording was interrupted. Please try again.",
	},
	"local": {
		ZH: "音频在运行 DSH 的机器上识别。需要下载模型时，请确保该机器能访问所选下载源及其文件服务。如需代理，请在该机器上配置。",
		EN: "Audio is recognized on the machine running DSH. If models need downloading, that machine must be able to reach the selected source and its file services. Configure a proxy on that machine if needed.",
	},
	"cloud": {
		ZH: "音频将发送至所选云端服务。",
		EN: "Audio will be sent to the selected cloud service.",
	},
	"empty": {
		ZH: "未识别到语音",
		EN: "No speech recognized",
	},
	"cancelled": {
		ZH: "已取消语音输入。",
		EN: "Voice input cancelled.",
	},
	"conflict": {
		ZH: "草稿已被修改。识别文字已保留，可在当前光标位置插入。",
		EN: "Your draft changed. The transcript is preserved and can be inserted at the current cursor.",
	},
	"failed": {
		ZH: "语音识别失败：{message}",
		EN: "Speech recognition failed: {message}",
	},
	"unavailable": {
		ZH: "当前浏览器不支持录音，请使用支持麦克风的浏览器。",
		EN: "This browser cannot record audio. Use a browser with microphone support.",
	},
	"permission": {
		ZH: "麦克风权限未开启，请在浏览器和系统设置中允许访问。",
		EN: "Microphone access is disabled. Allow it in browser and system settings.",
	},
	"tooLarge": {
		ZH: "录音超过服务限制，请缩短录音后重试。",
		EN: "The recording exceeds the service limit. Try a shorter recording.",
	},
}

// VoiceInputLocaleText returns the localized text for a dsh-experimental-client-ui-voice-input key.
func VoiceInputLocaleText(key string) string {
	if pair, ok := voiceInputLocaleCatalog[strings.TrimSpace(key)]; ok {
		return resolveLocalePair(pair)
	}
	return ""
}

// FormatVoiceInputLocale formats a dsh-experimental-client-ui-voice-input locale template.
func FormatVoiceInputLocale(key string, vars map[string]string) string {
	return formatLocaleTemplate(VoiceInputLocaleText(key), vars)
}

// FormatVoiceDownloadBytes formats the voice model download progress string.
func FormatVoiceDownloadBytes(completedMB, totalMB string, percent int) string {
	return FormatVoiceInputLocale("downloadBytes", map[string]string{
		"completed": strings.TrimSpace(completedMB),
		"total":     strings.TrimSpace(totalMB),
		"percent":   fmt.Sprintf("%d", percent),
	})
}

// FormatVoiceSystemMicrophoneNamed formats the system default microphone label with device name.
func FormatVoiceSystemMicrophoneNamed(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return VoiceInputLocaleText("inputDevice")
	}
	return FormatVoiceInputLocale("systemMicrophoneNamed", map[string]string{"name": n})
}

// VoiceDownloadFailureAdvice returns the localized download failure message and recovery advice for a failure kind.
func VoiceDownloadFailureAdvice(kind, resource, status string) (string, string) {
	k := strings.ToLower(strings.TrimSpace(kind))
	res := strings.TrimSpace(resource)
	if res == "" {
		res = "model"
	}
	vars := map[string]string{
		"resource": res,
		"status":   strings.TrimSpace(status),
	}
	switch k {
	case "network":
		return FormatVoiceInputLocale("download.network", vars), VoiceInputLocaleText("downloadAdvice.network")
	case "dns":
		return FormatVoiceInputLocale("download.dns", vars), VoiceInputLocaleText("downloadAdvice.dns")
	case "timeout":
		return FormatVoiceInputLocale("download.timeout", vars), VoiceInputLocaleText("downloadAdvice.timeout")
	case "certificate":
		return FormatVoiceInputLocale("download.certificate", vars), VoiceInputLocaleText("downloadAdvice.certificate")
	case "http":
		return FormatVoiceInputLocale("download.http", vars), VoiceInputLocaleText("downloadAdvice.http")
	case "integrity":
		return FormatVoiceInputLocale("download.integrity", vars), VoiceInputLocaleText("downloadAdvice.integrity")
	case "storage":
		return FormatVoiceInputLocale("download.storage", vars), VoiceInputLocaleText("downloadAdvice.storage")
	default:
		return FormatVoiceInputLocale("preparationFailed", map[string]string{"message": res}), VoiceInputLocaleText("downloadAdvice.unknown")
	}
}
