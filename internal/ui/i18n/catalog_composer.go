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
	KeyComposerModelMenuModel           = "composer.model.menu_model"
	KeyComposerModelMenuEffort          = "composer.model.menu_effort"
	KeyComposerModelBack                = "composer.model.back"
	KeyComposerModelSearch              = "composer.model.search"
	KeyComposerModelNoMatch             = "composer.model.no_match"
	KeyComposerModelEmpty               = "composer.model.empty"
	KeyComposerCommands                 = "composer.commands"
	KeyComposerVoiceInput               = "composer.voice_input"
	KeyComposerSlashCommandMenuTitle    = "composer.slash_menu.title"
	KeyComposerMentionMenuTitle         = "composer.mention_menu.title"
	KeyComposerEditNotice               = "composer.edit_notice"
	KeyComposerUnsentDraft              = "composer.unsent_draft"
	KeyComposerUnsentRestore            = "composer.unsent_restore"
	KeyComposerReplyThread              = "composer.reply_thread"
	KeyComposerReplyCancel              = "composer.reply_cancel"
	KeyComposerRetryUpload              = "composer.retry_upload"
	KeyComposerRemoveAttachment         = "composer.remove_attachment"
	KeyComposerRemoveReference          = "composer.remove_reference"
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
	Register(KeyComposerRemoveAttachment, "移除附件 %s", "Remove attachment %s")
	Register(KeyComposerRemoveReference, "移除引用", "Remove reference")
}
