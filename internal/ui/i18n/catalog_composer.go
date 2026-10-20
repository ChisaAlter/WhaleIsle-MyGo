package i18n

// Composer catalog: the input bar (chips, context gauge, suggestion popups) and
// the statistics line below it. Keys are registered from init so parallel
// feature work never edits the shared map literals in i18n.go.
const (
	KeyComposerPermissionReadOnly      = "composer.permission.read_only"
	KeyComposerPermissionWorkspace     = "composer.permission.workspace_write"
	KeyComposerPermissionFullAccess    = "composer.permission.full_access"
	KeyComposerPermissionAutoReview    = "composer.permission.auto_review"
	KeyComposerModelLoading            = "composer.model.loading"
	KeyComposerModelChoose             = "composer.model.choose"
	KeyComposerModelUnavailable        = "composer.model.unavailable"
	KeyComposerModelReload             = "composer.model.reload"
	KeyComposerContextUsed             = "composer.context.used"
	KeyComposerContextSystem           = "composer.context.system"
	KeyComposerContextTools            = "composer.context.tools"
	KeyComposerContextMessages         = "composer.context.messages"
	KeyComposerContextBreakdown        = "composer.context.breakdown"
	KeyComposerDockTurns               = "composer.dock.turns"
	KeyComposerDockSteps               = "composer.dock.steps"
	KeyComposerDockTokenUnit           = "composer.dock.token_unit"
	KeyComposerDockSpeedUnit           = "composer.dock.speed_unit"
	KeyComposerDockCacheHit            = "composer.dock.cache_hit"
	KeyComposerDockPeak                = "composer.dock.peak"
	KeyComposerDockOffPeak             = "composer.dock.off_peak"
	KeyComposerDockPeakSwitch          = "composer.dock.peak_switch"
	KeyComposerCostBilled              = "composer.cost.billed"
	KeyComposerCostMoneyUnavailable    = "composer.cost.money_unavailable"
	KeyComposerCostUnavailable         = "composer.cost.unavailable"
	KeyComposerCostNoPrices            = "composer.cost.no_prices"
	KeyComposerCostPartlyUnpriced      = "composer.cost.partly_unpriced"
	KeyComposerCostEstimate            = "composer.cost.estimate"
	KeyComposerCostPricesLoading       = "composer.cost.prices_loading"
	KeyComposerCostOffline             = "composer.cost.offline"
	KeyComposerCostPartial             = "composer.cost.partial"
	KeyComposerCostNotABill            = "composer.cost.not_a_bill"
	KeyComposerCostUnknownRoute        = "composer.cost.unknown_route"
	KeyComposerCostSubagents           = "composer.cost.subagents"
	KeyComposerCostMore                = "composer.cost.more"
	KeyComposerCostGlobalBilling       = "composer.cost.global_billing"
	KeyComposerCostUnpriced            = "composer.cost.unpriced"
	KeyComposerCostPriceLine           = "composer.cost.price_line"
	KeyComposerCostPeakPriceLine       = "composer.cost.peak_price_line"
	KeyComposerCostOffPeakPriceLine    = "composer.cost.off_peak_price_line"
	KeyComposerCostHitMissOutput       = "composer.cost.hit_miss_output"
	KeyComposerCostTokensDetail        = "composer.cost.tokens_detail"
	KeyComposerSuggestBrowse           = "composer.suggest.browse"
	KeyComposerSuggestAddReference     = "composer.suggest.add_reference"
	KeyComposerSuggestDropReference    = "composer.suggest.drop_reference"
	KeyComposerSuggestSkillsEmpty      = "composer.suggest.skills_empty"
	KeyComposerModelTooltipUnavailable = "composer.model.tooltip_unavailable"
	KeyComposerModelMenuModel          = "composer.model.menu_model"
	KeyComposerModelMenuEffort         = "composer.model.menu_effort"
	KeyComposerModelBack               = "composer.model.back"
	KeyComposerModelSearch             = "composer.model.search"
	KeyComposerModelNoMatch            = "composer.model.no_match"
	KeyComposerModelEmpty              = "composer.model.empty"
	KeyComposerCommands                = "composer.commands"
	KeyComposerVoiceInput              = "composer.voice_input"
	KeyComposerSlashCommandMenuTitle   = "composer.slash_menu.title"
	KeyComposerMentionMenuTitle        = "composer.mention_menu.title"
	KeyComposerEditNotice              = "composer.edit_notice"
	KeyComposerUnsentDraft             = "composer.unsent_draft"
	KeyComposerUnsentRestore           = "composer.unsent_restore"
	KeyComposerReplyThread             = "composer.reply_thread"
	KeyComposerReplyCancel             = "composer.reply_cancel"
	KeyComposerRetryUpload             = "composer.retry_upload"
	KeyComposerRemoveAttachment        = "composer.remove_attachment"
	KeyComposerRemoveReference         = "composer.remove_reference"
	KeyComposerDropTitle               = "composer.drop.title"
	KeyComposerDropDetail              = "composer.drop.detail"
	KeyComposerShortcutEnter           = "composer.shortcut.enter"
	KeyComposerShortcutEsc             = "composer.shortcut.esc"
	KeyComposerMenuFooter              = "composer.menu.footer"
	KeyComposerDropLimit               = "composer.drop.limit"
	KeyComposerDropFull                = "composer.drop.full"
	KeyComposerDropTooLarge            = "composer.drop.too_large"
	KeyComposerVoiceRecording          = "composer.voice.recording"
	KeyComposerVoiceSettings           = "composer.voice.settings"
	KeyComposerAttachmentPending       = "composer.attachment.pending"
	KeyComposerAttachmentUploading     = "composer.attachment.uploading"
	KeyComposerAttachmentFailed        = "composer.attachment.failed"
	KeyComposerAttachmentUnavailable   = "composer.attachment.unavailable"
	KeyComposerAttachmentImage         = "composer.attachment.image"

	KeyComposerVoiceStop            = "composer.voice.stop"
	KeyComposerVoiceInsert          = "composer.voice.insert"
	KeyComposerVoiceRetry           = "composer.voice.retry"
	KeyComposerVoiceCancel          = "composer.voice.cancel"
	KeyComposerVoiceDetails         = "composer.voice.details"
	KeyComposerVoiceLater           = "composer.voice.later"
	KeyComposerVoiceUnavailable     = "composer.voice.unavailable_title"
	KeyComposerVoiceUnavailableBody = "composer.voice.unavailable_body"
	KeyComposerVoicePrepare         = "composer.voice.prepare_title"
	KeyComposerVoicePrepareBody     = "composer.voice.prepare_body"
)

func init() {
	Register(KeyComposerPermissionReadOnly, "仅可查看", "Read Only")
	Register(KeyComposerPermissionWorkspace, "工作区内修改", "Workspace Write")
	Register(KeyComposerPermissionFullAccess, "完全权限", "Full access")
	Register(KeyComposerPermissionAutoReview, "自动审查", "Auto review")
	Register(KeyComposerModelLoading, "加载中…", "Loading…")
	Register(KeyComposerModelChoose, "选择模型", "Choose model")
	Register(KeyComposerModelUnavailable, "暂无可用模型", "No models available")
	Register(KeyComposerModelReload, "重新加载", "Reload")
	Register(KeyComposerContextUsed, "上下文已用 %s", "%s of context used")
	Register(KeyComposerContextSystem, "系统提示词", "System prompt")
	Register(KeyComposerContextTools, "工具定义", "Tool definitions")
	Register(KeyComposerContextMessages, "对话消息", "Messages")
	Register(KeyComposerContextBreakdown, "估算组成", "Estimated composition")
	Register(KeyComposerDockTurns, "轮", "turns")
	Register(KeyComposerDockSteps, "步", "steps")
	Register(KeyComposerDockTokenUnit, "tok", "tok")
	Register(KeyComposerDockSpeedUnit, "tok/s", "tok/s")
	Register(KeyComposerDockCacheHit, "缓存命中 %s", "Cache hit %s")
	Register(KeyComposerDockPeak, "高峰", "Peak")
	Register(KeyComposerDockOffPeak, "空闲", "Off-peak")
	Register(KeyComposerDockPeakSwitch, "北京时间 %s · 距切换 %s", "Beijing time %s · switch in %s")
	Register(KeyComposerCostBilled, "含子代理计费 Token · %s", "Billed tokens incl. subagents · %s")
	Register(KeyComposerCostMoneyUnavailable, "Host 未提供计费用量", "Billing usage unavailable")
	Register(KeyComposerCostUnavailable, "Host 未提供计费 Token", "Billed tokens unavailable")
	Register(KeyComposerCostNoPrices, "没有设置价格", "No priced routes")
	Register(KeyComposerCostPartlyUnpriced, "（部分模型未定价）", " (partly unpriced)")
	Register(KeyComposerCostEstimate, "会话累计费用", "Session estimate")
	Register(KeyComposerCostPricesLoading, "价格未加载", "Prices not loaded")
	Register(KeyComposerCostOffline, "离线缓存用量", "Offline cached usage")
	Register(KeyComposerCostPartial, "仅包含已上报用量的会话及子代理；缺失部分未计入", "Reported sessions and subagents only; missing usage excluded")
	Register(KeyComposerCostNotABill, "基于全局配置估算，非账单", "Estimate from global prices, not a bill")
	Register(KeyComposerCostUnknownRoute, "未知路由", "Unknown route")
	Register(KeyComposerCostSubagents, "子代理 %d", "Subagents %d")
	Register(KeyComposerCostMore, "还有 %d", "More: %d")
	Register(KeyComposerCostGlobalBilling, "消耗统计中的计费设置", "Global billing settings")
	Register(KeyComposerCostUnpriced, "没有设置价格", "Unpriced")
	Register(KeyComposerCostPriceLine, "CNY / 1M · 命中 %.6g · 未命中 %.6g · 输出 %.6g", "CNY / 1M · hit %.6g · miss %.6g · output %.6g")
	Register(KeyComposerCostPeakPriceLine, "高峰 %s", "Peak %s")
	Register(KeyComposerCostOffPeakPriceLine, "空闲 %s", "Off-peak %s")
	Register(KeyComposerCostHitMissOutput, "命中 %s · 未命中 %s · 输出 %s", "hit %s · miss %s · output %s")
	Register(KeyComposerCostTokensDetail, "命中 %s · 未命中 %s · 输出 %s", "hit %s · miss %s · output %s")
	Register(KeyComposerSuggestBrowse, "浏览目录", "Browse directory")
	Register(KeyComposerSuggestAddReference, "添加文件引用", "Add file reference")
	Register(KeyComposerSuggestDropReference, "拖入以引用", "Drop to reference")
	Register(KeyComposerSuggestSkillsEmpty, "没有可用的技能", "No skills available")
	Register(KeyComposerModelTooltipUnavailable, "模型目录不可用，点击重新加载", "Model catalog unavailable, click to reload")
	Register(KeyComposerModelMenuModel, "模型", "Model")
	Register(KeyComposerModelMenuEffort, "思考强度", "Effort")
	Register(KeyComposerModelBack, "‹ 返回", "‹ Back")
	Register(KeyComposerModelSearch, "搜索模型", "Search models")
	Register(KeyComposerModelNoMatch, "没有匹配模型", "No matching models")
	Register(KeyComposerModelEmpty, "模型", "Model")
	Register(KeyComposerCommands, "命令列表", "Commands")
	Register(KeyComposerVoiceInput, "语音输入", "Voice input")
	Register(KeyComposerSlashCommandMenuTitle, "指令", "Commands")
	Register(KeyComposerMentionMenuTitle, "引用", "References")
	Register(KeyComposerEditNotice, "正在编辑已发送的消息", "Editing a sent message")
	Register(KeyComposerUnsentDraft, "未发送草稿", "Unsent drafts")
	Register(KeyComposerUnsentRestore, "恢复", "Restore")
	Register(KeyComposerReplyThread, "回复线程：%s", "Reply to thread: %s")
	Register(KeyComposerReplyCancel, "取消回复", "Cancel reply")
	Register(KeyComposerRetryUpload, "重试上传", "Retry upload")
	Register(KeyComposerRemoveAttachment, "移除附件", "Remove attachment")
	Register(KeyComposerRemoveReference, "移除引用", "Remove reference")
	Register(KeyComposerDropTitle, "文件或图片拖动到此处即可添加", "Drop files or images here")
	Register(KeyComposerDropDetail, "松手即加入输入区，随消息一起发送", "Release to add them to the message")
	Register(KeyComposerShortcutEnter, "Enter", "Enter")
	Register(KeyComposerShortcutEsc, "Esc", "Esc")
	Register(KeyComposerMenuFooter, "↑↓ 选择 · Tab 钻入 · Enter 确认 · Esc 关闭", "↑↓ select · Tab drill in · Enter confirm · Esc close")
	Register(KeyComposerDropLimit, "最多 %d 张，每张 ≤ %s", "Up to %d images, %s each")
	Register(KeyComposerDropFull, "已达 %d 张上限，超出的图片会被忽略", "The %d image limit is reached; extra images are ignored")
	Register(KeyComposerDropTooLarge, "附件总大小超过 %s，请先移除部分附件", "Attachments exceed %s; remove some first")
	Register(KeyComposerVoiceRecording, "录制中", "Recording")
	Register(KeyComposerVoiceSettings, "语音设置", "Voice settings")
	Register(KeyComposerAttachmentPending, "待上传", "Pending upload")
	Register(KeyComposerAttachmentUploading, "上传中", "Uploading")
	Register(KeyComposerAttachmentFailed, "上传失败，点击重试", "Upload failed, retry")
	Register(KeyComposerAttachmentUnavailable, "文件不可用", "File unavailable")
	Register(KeyComposerAttachmentImage, "图片", "Image")
	Register(KeyComposerVoiceStop, "停止并转写", "Stop recording")
	Register(KeyComposerVoiceInsert, "插入转写", "Insert transcript")
	Register(KeyComposerVoiceRetry, "重试录音", "Retry recording")
	Register(KeyComposerVoiceCancel, "取消录音", "Discard recording")
	Register(KeyComposerVoiceDetails, "打开插件详情", "Open plugin details")
	Register(KeyComposerVoiceLater, "稍后", "Later")
	Register(KeyComposerVoiceUnavailable, "语音识别暂不可用", "Speech recognition unavailable")
	Register(KeyComposerVoiceUnavailableBody, "请在语音输入插件详情中查看识别服务状态和设置。", "Check recognition status and settings in the Voice Input plugin details.")
	Register(KeyComposerVoicePrepare, "准备语音输入", "Set up voice input")
	Register(KeyComposerVoicePrepareBody, "所选本地识别服务首次使用前需要准备模型。请打开插件详情开始准备。", "The selected local recognizer needs its models prepared before first use. Open plugin details to prepare it.")

}
