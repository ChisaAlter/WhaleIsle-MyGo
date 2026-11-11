package i18n

import "strings"

const (
	// ui-approval
	KeyParityApprovalEscalation = "chat.parity.approval.escalation"
	KeyParityApprovalReject     = "chat.parity.approval.reject"

	// ui-chat
	KeyParityChatOpenUIActionFailed          = "chat.parity.openui.actionFailed"
	KeyParityChatOpenUITitle                 = "chat.parity.openui.title"
	KeyParityChatStepThinking                = "chat.parity.message.stepProcess.thinking"
	KeyParityChatStepWebFetch                = "chat.parity.message.stepProcess.webFetch"
	KeyParityChatStepSubagents               = "chat.parity.message.stepProcess.subagents"
	KeyParityChatStepPlan                    = "chat.parity.message.stepProcess.plan"
	KeyParityChatStepQuestions               = "chat.parity.message.stepProcess.questions"
	KeyParityChatStepTools                   = "chat.parity.message.stepProcess.tools"
	KeyParityChatStepDoneThinking            = "chat.parity.message.stepProcess.done.thinking"
	KeyParityChatStepDoneWebFetch            = "chat.parity.message.stepProcess.done.webFetch"
	KeyParityChatStepDoneSubagents           = "chat.parity.message.stepProcess.done.subagents"
	KeyParityChatStepDonePlan                = "chat.parity.message.stepProcess.done.plan"
	KeyParityChatStepDoneQuestions           = "chat.parity.message.stepProcess.done.questions"
	KeyParityChatStepDoneTools               = "chat.parity.message.stepProcess.done.tools"
	KeyParityChatImageFailed                 = "chat.parity.image.failed"
	KeyParityChatImageDialog                 = "chat.parity.image.dialog"
	KeyParityChatImageLoading                = "chat.parity.image.loading"
	KeyParityChatSettingsPerformanceTitle    = "chat.parity.settings.performance.title"
	KeyParityChatSettingsPerformanceDesc     = "chat.parity.settings.performance.description"
	KeyParityChatSettingsPerformanceCompact  = "chat.parity.settings.performance.compact"
	KeyParityChatSettingsLinksTitle          = "chat.parity.settings.links.title"
	KeyParityChatSettingsLinksDesc           = "chat.parity.settings.links.description"
	KeyParityChatSettingsLinksSidebar        = "chat.parity.settings.links.sidebar"
	KeyParityChatSettingsTranscriptDesc      = "chat.parity.settings.transcript.description"
	KeyParityChatSettingsCollapseTitle       = "chat.parity.settings.collapse.title"
	KeyParityChatSettingsCollapseDesc        = "chat.parity.settings.collapse.description"
	KeyParityChatSettingsCollapseCompletion  = "chat.parity.settings.collapse.completion"
	KeyParityChatMessageBranch               = "chat.parity.message.branch"
	KeyParityChatMessageRetryScheduled       = "chat.parity.message.retry.scheduled"
	KeyParityChatMessageRetryStatus          = "chat.parity.message.retry.status"
	KeyParityChatTurnUsageSummaryWithCache   = "chat.parity.message.turnUsage.summaryWithCache"
	KeyParityChatTurnUsageSummaryWithCacheV2 = "chat.parity.message.turnUsage.summaryWithCache.v2"
	KeyParityChatTurnUsageConsumed           = "chat.parity.message.turnUsage.consumed"
	KeyParityChatTurnUsageCacheRead          = "chat.parity.message.turnUsage.cacheRead"
	KeyParityChatTurnUsageCacheWrite         = "chat.parity.message.turnUsage.cacheWrite"
	KeyParityChatTurnUsageReasoning          = "chat.parity.message.turnUsage.reasoning"
	KeyParityChatTurnUsageCount              = "chat.parity.message.turnUsage.count"
	KeyParityChatTokensPerSecond             = "chat.parity.message.tokensPerSecond"
	KeyParityChatTurnProcessTook             = "chat.parity.message.turnProcess.took"
	KeyParityChatTurnProcessMessagesOne      = "chat.parity.message.turnProcess.messages.one"
	KeyParityChatTurnProcessMessagesOther    = "chat.parity.message.turnProcess.messages.other"
	KeyParityChatTurnProcessFailed           = "chat.parity.message.turnProcess.failed"
	KeyParityChatStatsCacheHit               = "chat.parity.stats.cacheHit"
	KeyParityChatMessageReferenceSummary     = "chat.parity.message.referenceSummary"
	KeyParityChatDurationMilliseconds        = "chat.parity.duration.milliseconds"
	KeyParityChatDurationSecondUnit          = "chat.parity.duration.secondUnit"
	KeyParityChatDurationMinuteUnit          = "chat.parity.duration.minuteUnit"
	KeyParityChatDurationHourUnit            = "chat.parity.duration.hourUnit"
	KeyParityChatThinking                    = "chat.parity.chat.thinking"
	KeyParityChatCommandTitle                = "chat.parity.command.title"

	// ui-conversation
	KeyParityConvHintPlan                      = "chat.parity.conv.hint.plan"
	KeyParityConvPlaceholderPlan               = "chat.parity.conv.placeholder.plan"
	KeyParityConvPlaceholderManaged            = "chat.parity.conv.placeholder.managed"
	KeyParityConvInputSendQueue                = "chat.parity.conv.input.send.queue"
	KeyParityConvInputSendSteer                = "chat.parity.conv.input.send.steer"
	KeyParityConvAttachmentPending             = "chat.parity.conv.attachment.pending"
	KeyParityConvAttachmentDropDesc            = "chat.parity.conv.attachment.dropDesc"
	KeyParityConvImageOriginal                 = "chat.parity.conv.image.original"
	KeyParityConvImageTooMany                  = "chat.parity.conv.image.tooMany"
	KeyParityConvImageFileTooLarge             = "chat.parity.conv.image.fileTooLarge"
	KeyParityConvImageTotalTooLarge            = "chat.parity.conv.image.totalTooLarge"
	KeyParityConvFileUploading                 = "chat.parity.conv.file.uploading"
	KeyParityConvFileUploadFailed              = "chat.parity.conv.file.uploadFailed"
	KeyParityConvFileRetry                     = "chat.parity.conv.file.retry"
	KeyParityConvContextUsed                   = "chat.parity.conv.context.used"
	KeyParityConvPeakValleyPeak                = "chat.parity.conv.peakValley.peak"
	KeyParityConvPeakValleyOffPeak             = "chat.parity.conv.peakValley.offPeak"
	KeyParityConvPeakValleyCountdown           = "chat.parity.conv.peakValley.countdown"
	KeyParityConvSessionCostNoPrice            = "chat.parity.conv.sessionCost.noPrice"
	KeyParityConvSessionCostPartialNote        = "chat.parity.conv.sessionCost.partialNote"
	KeyParityConvSessionCostCostLine           = "chat.parity.conv.sessionCost.costLine"
	KeyParityConvSessionCostUsagePeak          = "chat.parity.conv.sessionCost.usage.peak"
	KeyParityConvSessionCostUsageIdle          = "chat.parity.conv.sessionCost.usage.idle"
	KeyParityConvSessionCostSubagentTokens     = "chat.parity.conv.sessionCost.subagent.tokens"
	KeyParityConvSessionCostRouteUnknown       = "chat.parity.conv.sessionCost.routeUnknown"
	KeyParityConvSessionCostSubagentSummary    = "chat.parity.conv.sessionCost.subagent.summary"
	KeyParityConvSessionCostSubagentMore       = "chat.parity.conv.sessionCost.subagent.more"
	KeyParityConvSessionCostPriceTitleNone     = "chat.parity.conv.sessionCost.priceTitle.none"
	KeyParityConvSessionCostNoPriceHint        = "chat.parity.conv.sessionCost.noPriceHint"
	KeyParityConvHeroClaudeHeadline            = "chat.parity.conv.hero.claudeHeadline"
	KeyParityConvHeroClaudeHeadlineAlt         = "chat.parity.conv.hero.claudeHeadline.alt"
	KeyParityConvSessionMoreViews              = "chat.parity.conv.session.moreViews"
	KeyParityConvSessionMoreViewsAlt           = "chat.parity.conv.session.moreViews.alt"
	KeyParityConvTodoTitle                     = "chat.parity.conv.todo.title"
	KeyParityConvTodoProgressDone              = "chat.parity.conv.todo.progress.done"
	KeyParityConvTodoProgressActive            = "chat.parity.conv.todo.progress.active"
	KeyParityConvTodoProgressPending           = "chat.parity.conv.todo.progress.pending"
	KeyParityConvTodoCompleted                 = "chat.parity.conv.todo.completed"
	KeyParityConvToolTitleFindReferences       = "chat.parity.conv.tool.title.findReferences"
	KeyParityConvAskReview                     = "chat.parity.conv.ask.review"
	KeyParityConvAskWaiting                    = "chat.parity.conv.ask.waiting"
	KeyParityConvAskCancelled                  = "chat.parity.conv.ask.cancelled"
	KeyParityConvPlanApproved                  = "chat.parity.conv.plan.approved"
	KeyParityConvPlanRejected                  = "chat.parity.conv.plan.rejected"
	KeyParityConvPlanCancelled                 = "chat.parity.conv.plan.cancelled"
	KeyParityConvPlanReviewing                 = "chat.parity.conv.plan.reviewing"
	KeyParityConvPlanView                      = "chat.parity.conv.plan.view"
	KeyParityConvQueueCount                    = "chat.parity.conv.queue.count"
	KeyParityConvQueueReorder                  = "chat.parity.conv.queue.reorder"
	KeyParityConvQueueReorderHint              = "chat.parity.conv.queue.reorder.hint"
	KeyParityConvQueueMoveFailed               = "chat.parity.conv.queue.moveFailed"
	KeyParityConvQueueEdit                     = "chat.parity.conv.queue.edit"
	KeyParityConvQueueRemove                   = "chat.parity.conv.queue.remove"
	KeyParityConvQueueSteer                    = "chat.parity.conv.queue.steer"
	KeyParityConvCommandAttachmentsUnsupported = "chat.parity.conv.command.attachmentsUnsupported"
	KeyParityConvCustomInstructionsPlaceholder = "chat.parity.conv.settings.customInstructions.placeholder"
	KeyParityConvSettingsBeamPanelReset        = "chat.parity.conv.settings.beam.panel.reset"
	KeyParityConvSettingsTypingFxPanelReset    = "chat.parity.conv.settings.typingFx.panel.reset"

	// ui-commands
	KeyParityCommandsOverlayAria                  = "chat.parity.commands.overlay.aria"
	KeyParityCommandsNoticeAttachmentsUnsupported = "chat.parity.commands.notice.attachmentsUnsupported"
	KeyParityCommandsLabelPermission              = "chat.parity.commands.label.permission"
	KeyParityCommandsTokenGoal                    = "chat.parity.commands.token.goal"
	KeyParityCommandsTokenPlan                    = "chat.parity.commands.token.plan"
	KeyParityCommandsTokenPermission              = "chat.parity.commands.token.permission"
	KeyParityCommandsTokenExport                  = "chat.parity.commands.token.export"

	// ui-model-selection
	KeyParityModelSelectionCommandDescription = "chat.parity.modelSelection.command.description"
	KeyParityModelSelectionTriggerFallback    = "chat.parity.modelSelection.trigger.fallback"
	KeyParityModelSelectionWarningGroupLoad   = "chat.parity.modelSelection.warning.groupLoad"
	KeyParityModelSelectionEmptyModels        = "chat.parity.modelSelection.empty.models"

	// ui-permission-presets
	KeyParityPermTitle                     = "chat.parity.perm.title"
	KeyParityPermMode                      = "chat.parity.perm.mode"
	KeyParityPermSwitchFailed              = "chat.parity.perm.switchFailed"
	KeyParityPermSwitchFailedAlt           = "chat.parity.perm.switchFailed.alt"
	KeyParityPermConfirmDescription        = "chat.parity.perm.confirm.description"
	KeyParityPermConfirmNewSessionDesc     = "chat.parity.perm.confirm.newSessionDescription"
	KeyParityPermConfirmDescriptionAlt     = "chat.parity.perm.confirm.description.alt"
	KeyParityPermAutoConfirmAcknowledge    = "chat.parity.perm.auto.confirm.acknowledge"
	KeyParityPermAutoConfirmAcknowledgeAlt = "chat.parity.perm.auto.confirm.acknowledge.alt"

	// ui-plan
	KeyParityPlanChipOnAria  = "chat.parity.plan.chip.on.aria"
	KeyParityPlanChipOnTitle = "chat.parity.plan.chip.on.title"

	// ui-skill
	KeyParitySkillRowInstructions = "chat.parity.skill.row.instructions"
	KeyParitySkillRowTitle        = "chat.parity.skill.row.title"
	KeyParitySkillMenuUserOnly    = "chat.parity.skill.menu.userOnly"

	// ui-user-questions
	KeyParityQuestionsErrorIncomplete = "chat.parity.questions.error.incomplete"
	KeyParityQuestionsErrorUnanswered = "chat.parity.questions.error.unanswered"
	KeyParityQuestionsWaitTakeTime    = "chat.parity.questions.wait.takeTime"
	KeyParityQuestionsWaitHeld        = "chat.parity.questions.wait.held"
	KeyParityQuestionsWaitCountdown   = "chat.parity.questions.wait.countdown"
	KeyParityQuestionsWaitPaused      = "chat.parity.questions.wait.paused"
	KeyParityQuestionsReviewStatus    = "chat.parity.questions.review.status"
	KeyParityQuestionsPlanApprove     = "chat.parity.questions.plan.approve"
	KeyParityQuestionsPlanDecline     = "chat.parity.questions.plan.decline"
	KeyParityQuestionsActionNext      = "chat.parity.questions.action.next"

	// ui-visual-replies
	KeyParityVisualRepliesSettingsDesc = "chat.parity.visualReplies.settings.description"
	KeyParityVisualRepliesCardSave     = "chat.parity.visualReplies.card.save"
	KeyParityVisualRepliesCardTitle    = "chat.parity.visualReplies.card.title"
	KeyParityVisualRepliesCardSource   = "chat.parity.visualReplies.card.source"

	// ui-workflow-run
	KeyParityWorkflowRunMembersOne           = "chat.parity.workflowRun.run.members.one"
	KeyParityWorkflowRunStatusCountCompleted = "chat.parity.workflowRun.statusCount.completed"
	KeyParityWorkflowRunStatusCountCancelled = "chat.parity.workflowRun.statusCount.cancelled"
	KeyParityWorkflowRunStatusCountInterrupt = "chat.parity.workflowRun.statusCount.interrupted"
	KeyParityWorkflowRunStatusCountRunning   = "chat.parity.workflowRun.statusCount.running"
	KeyParityWorkflowRunStatusCountFailed    = "chat.parity.workflowRun.statusCount.failed"
	KeyParityWorkflowRunPhaseEmpty           = "chat.parity.workflowRun.phase.empty"

	// misc chat-adjacent packages in chat.json
	KeyParityGitBranchLoading             = "chat.parity.git.branch.loading"
	KeyParityGitCommitNone                = "chat.parity.git.commit.none"
	KeyParityGitConfirmAbort              = "chat.parity.git.confirm.abort"
	KeyParityDeliverablesReviewWrapAria   = "chat.parity.deliverables.review.wrapAria"
	KeyParityDiffScopeWorktree            = "chat.parity.diff.scope.worktree"
	KeyParityDiffViewUnified              = "chat.parity.diff.view.unified"
	KeyParityDiffViewSplit                = "chat.parity.diff.view.split"
	KeyParityMessageEditEditorEditing     = "chat.parity.messageEdit.editor.editing"
	KeyParityMessageFeedbackLikeActive    = "chat.parity.messageFeedback.action.likeActive"
	KeyParityMessageFeedbackDislikeActive = "chat.parity.messageFeedback.action.dislikeActive"
	KeyParityReferenceSectionSessions     = "chat.parity.reference.section.sessions"
	KeyParitySecretMaskSecretReveal       = "chat.parity.secretMask.secret.reveal"
)

func init() {
	// ui-approval
	Register(KeyParityApprovalEscalation, "工具 {toolName} 请求越权执行", "Tool {toolName} requests privileged execution")
	Register(KeyParityApprovalReject, "拒绝", "Reject")

	// ui-chat
	Register(KeyParityChatOpenUIActionFailed, "操作未完成。", "The action did not complete.")
	Register(KeyParityChatOpenUITitle, "Whale UI", "Whale UI")
	Register(KeyParityChatStepThinking, "正在分析请求", "Analyzing the request")
	Register(KeyParityChatStepWebFetch, "正在访问网页", "Visiting web pages")
	Register(KeyParityChatStepSubagents, "正在协调子智能体", "Coordinating subagents")
	Register(KeyParityChatStepPlan, "正在更新计划", "Updating the plan")
	Register(KeyParityChatStepQuestions, "等待你的操作", "Waiting for your action")
	Register(KeyParityChatStepTools, "正在调用工具", "Calling tools")
	Register(KeyParityChatStepDoneThinking, "已完成分析", "Analysis completed")
	Register(KeyParityChatStepDoneWebFetch, "已访问网页", "Visited web pages")
	Register(KeyParityChatStepDoneSubagents, "已协调子智能体", "Coordinated subagents")
	Register(KeyParityChatStepDonePlan, "更新了计划", "Updated the plan")
	Register(KeyParityChatStepDoneQuestions, "向用户提出了问题", "Asked questions")
	Register(KeyParityChatStepDoneTools, "已调用工具", "Called tools")
	Register(KeyParityChatImageFailed, "图片无法预览", "Image preview unavailable")
	Register(KeyParityChatImageDialog, "图片预览", "Image preview")
	Register(KeyParityChatImageLoading, "加载图片…", "Loading image…")
	Register(KeyParityChatSettingsPerformanceTitle, "性能与用量", "Performance & usage")
	Register(KeyParityChatSettingsPerformanceDesc, "选择性能与用量信息展示的详细程度", "Choose how much performance and usage information to show")
	Register(KeyParityChatSettingsPerformanceCompact, "简洁", "Compact")
	Register(KeyParityChatSettingsLinksTitle, "网页链接默认打开方式", "Open chat links in")
	Register(KeyParityChatSettingsLinksDesc, "对话中网页链接的打开位置", "Choose where to open web links")
	Register(KeyParityChatSettingsLinksSidebar, "应用内侧边栏", "In-App Sidebar")
	Register(KeyParityChatSettingsTranscriptDesc, "选择希望看到多少工具调用细节", "Choose how much detail to show for tool calls")
	Register(KeyParityChatSettingsCollapseTitle, "工作步骤收起时机", "When to Collapse Work Details")
	Register(KeyParityChatSettingsCollapseDesc, "选择何时自动收起工作步骤", "Choose when to automatically collapse work details")
	Register(KeyParityChatSettingsCollapseCompletion, "回答结束后", "On completion")
	Register(KeyParityChatMessageBranch, "在新对话中分支", "Branch into a new conversation")
	Register(KeyParityChatMessageRetryScheduled, "等待重试模型请求", "Waiting to retry model request")
	Register(KeyParityChatMessageRetryStatus, "{label}（{retry}/{maximum}） · {seconds}s", "{label} ({retry}/{maximum}) · {seconds}s")
	Register(KeyParityChatTurnUsageSummaryWithCache, "用量 {total} · 缓存命中 {percent}%", "Usage {total} · {percent}% cache hit")
	Register(KeyParityChatTurnUsageSummaryWithCacheV2, "用量 {total} · 缓存命中 {percent}%", "Usage {total} · Cache hit {percent}%")
	Register(KeyParityChatTurnUsageConsumed, "用量 {total}", "Usage {total}")
	Register(KeyParityChatTurnUsageCacheRead, "缓存读取", "Cached input")
	Register(KeyParityChatTurnUsageCacheWrite, "缓存写入", "Cache write")
	Register(KeyParityChatTurnUsageReasoning, "（其中推理 {tokens}）", " ({tokens} reasoning)")
	Register(KeyParityChatTurnUsageCount, "{count} tok", "{count} tok")
	Register(KeyParityChatTokensPerSecond, "{tps} tok/s", "{tps} tok/s")
	Register(KeyParityChatTurnProcessTook, "已完成，用时 ", "Completed in ")
	Register(KeyParityChatTurnProcessMessagesOne, "{count} 条消息", "{count} message")
	Register(KeyParityChatTurnProcessMessagesOther, "{count} 条消息", "{count} messages")
	Register(KeyParityChatTurnProcessFailed, "处理失败", "Failed")
	Register(KeyParityChatStatsCacheHit, "缓存命中 {percent}%", "Cache hit {percent}%")
	Register(KeyParityChatMessageReferenceSummary, "引用会话 · {labels}", "Referenced session · {labels}")
	Register(KeyParityChatDurationMilliseconds, "{milliseconds}毫秒", "{milliseconds}ms")
	Register(KeyParityChatDurationSecondUnit, "秒", "s")
	Register(KeyParityChatDurationMinuteUnit, "分", "m ")
	Register(KeyParityChatDurationHourUnit, "小时", "h ")
	Register(KeyParityChatThinking, "正在思考", "Thinking")
	Register(KeyParityChatCommandTitle, "指令", "Command")

	// ui-conversation
	Register(KeyParityConvHintPlan, "描述你的任务以生成计划", "describe your task to generate plan")
	Register(KeyParityConvPlaceholderPlan, "描述你的任务以生成计划", "describe your task to generate plan")
	Register(KeyParityConvPlaceholderManaged, "发送消息", "Send a message")
	Register(KeyParityConvInputSendQueue, "排队发送", "Queue message")
	Register(KeyParityConvInputSendSteer, "插话发送", "Steer message")
	Register(KeyParityConvAttachmentPending, "待发送附件", "Pending attachments")
	Register(KeyParityConvAttachmentDropDesc, "图片限制：最多 {count} 张，每张 {size}", "Image limit: up to {count} images, {size} each")
	Register(KeyParityConvImageOriginal, "原图", "Original image")
	Register(KeyParityConvImageTooMany, "一条消息最多添加 {count} 张图片", "A message can include up to {count} images")
	Register(KeyParityConvImageFileTooLarge, "单张图片不能超过 {size}", "Each image must be smaller than {size}")
	Register(KeyParityConvImageTotalTooLarge, "图片总大小超过 {size}，请移除部分图片", "Images exceed {size} in total; remove some and try again")
	Register(KeyParityConvFileUploading, "上传中…", "Uploading…")
	Register(KeyParityConvFileUploadFailed, "上传失败，点击重试", "Upload failed; click to retry")
	Register(KeyParityConvFileRetry, "重试上传 {name}", "Retry uploading {name}")
	Register(KeyParityConvContextUsed, "上下文已用", "of context used")
	Register(KeyParityConvPeakValleyPeak, "高峰时段", "Peak hours")
	Register(KeyParityConvPeakValleyOffPeak, "空闲时段", "Off-peak hours")
	Register(KeyParityConvPeakValleyCountdown, "距离切换剩余时间：{time}", "Switches in {time}")
	Register(KeyParityConvSessionCostNoPrice, "没有设置当前模型价格", "No price set for the current model")
	Register(KeyParityConvSessionCostPartialNote, "（部分模型未定价）", " (some models unpriced)")
	Register(KeyParityConvSessionCostCostLine, "费用 {cost}", "Cost {cost}")
	Register(KeyParityConvSessionCostUsagePeak, "高峰 命中 {hit}{rate} · 未命中 {miss} · 输出 {output}", "Peak hit {hit}{rate} · miss {miss} · output {output}")
	Register(KeyParityConvSessionCostUsageIdle, "空闲 命中 {hit}{rate} · 未命中 {miss} · 输出 {output}", "Off-peak hit {hit}{rate} · miss {miss} · output {output}")
	Register(KeyParityConvSessionCostSubagentTokens, "命中 {hit}{rate} · 未命中 {miss} · 输出 {output}", "Hit {hit}{rate} · miss {miss} · output {output}")
	Register(KeyParityConvSessionCostRouteUnknown, "未知模型", "Unknown model")
	Register(KeyParityConvSessionCostSubagentSummary, "子代理 {count} 个", "Subagents ({count})")
	Register(KeyParityConvSessionCostSubagentMore, "还有 {count} 个", "{count} more")
	Register(KeyParityConvSessionCostPriceTitleNone, "没有设置价格", "No price set")
	Register(KeyParityConvSessionCostNoPriceHint, "在 设置 → 消耗统计 → 计费设置 中设置模型价格", "Set model prices in Settings → Usage stats → Billing settings")
	Register(KeyParityConvHeroClaudeHeadline, "今天想一起探索什么？", "What shall we explore today?")
	Register(KeyParityConvHeroClaudeHeadlineAlt, "今天想一起探索什么？", "What would you like to explore today?")
	Register(KeyParityConvSessionMoreViews, "更多会话视图", "More conversation views")
	Register(KeyParityConvSessionMoreViewsAlt, "更多会话视图", "More session views")
	Register(KeyParityConvTodoTitle, "任务", "To-dos")
	Register(KeyParityConvTodoProgressDone, "{done} 已完成", "{done} completed")
	Register(KeyParityConvTodoProgressActive, "{active} 进行中", "{active} in progress")
	Register(KeyParityConvTodoProgressPending, "{pending} 待处理", "{pending} pending")
	Register(KeyParityConvTodoCompleted, "{done}/{total} 已完成", "{done}/{total} completed")
	Register(KeyParityConvToolTitleFindReferences, "查找引用", "Find references")
	Register(KeyParityConvAskReview, "查看回答", "View answers")
	Register(KeyParityConvAskWaiting, "等待回答", "waiting")
	Register(KeyParityConvAskCancelled, "已取消", "cancelled")
	Register(KeyParityConvPlanApproved, "已批准执行", "Approved")
	Register(KeyParityConvPlanRejected, "已拒绝", "Rejected")
	Register(KeyParityConvPlanCancelled, "已取消", "Cancelled")
	Register(KeyParityConvPlanReviewing, "等待确认", "Waiting for review")
	Register(KeyParityConvPlanView, "查看计划", "View plan")
	Register(KeyParityConvQueueCount, "{n} 条排队消息", "{n} queued messages")
	Register(KeyParityConvQueueReorder, "调整顺序：第 {n} 条，共 {total} 条", "Reorder queued message {n} of {total}")
	Register(KeyParityConvQueueReorderHint, "拖动调整顺序，或按 ↑ ↓", "Drag to reorder, or press ↑ ↓")
	Register(KeyParityConvQueueMoveFailed, "调整顺序失败：这条消息可能已经开始发送。", "Reorder failed: this message may have already started sending.")
	Register(KeyParityConvQueueEdit, "编辑排队消息", "Edit queued message")
	Register(KeyParityConvQueueRemove, "删除排队消息", "Remove queued message")
	Register(KeyParityConvQueueSteer, "插话发送", "Steer queued message")
	Register(KeyParityConvCommandAttachmentsUnsupported, "/{command} 不接受附件，请先移除附件", "/{command} does not accept attachments; remove them first")
	Register(KeyParityConvCustomInstructionsPlaceholder, "输入规则，例如：回复始终使用中文", "Write your rules, e.g. always reply in Chinese")
	Register(KeyParityConvSettingsBeamPanelReset, "恢复默认", "Reset")
	Register(KeyParityConvSettingsTypingFxPanelReset, "恢复默认", "Reset")

	// ui-commands
	Register(KeyParityCommandsOverlayAria, "/{command} 选项", "/{command} options")
	Register(KeyParityCommandsNoticeAttachmentsUnsupported, "/{command} 不接受附件，请先移除附件", "/{command} does not accept attachments; remove them first")
	Register(KeyParityCommandsLabelPermission, "权限", "Permission")
	Register(KeyParityCommandsTokenGoal, "目标", "goal")
	Register(KeyParityCommandsTokenPlan, "计划", "plan")
	Register(KeyParityCommandsTokenPermission, "权限", "permission")
	Register(KeyParityCommandsTokenExport, "导出", "export")

	// ui-model-selection
	Register(KeyParityModelSelectionCommandDescription, "选择本会话使用的模型", "Select the model for this conversation")
	Register(KeyParityModelSelectionTriggerFallback, "请选择模型", "Select model")
	Register(KeyParityModelSelectionWarningGroupLoad, "{name} 加载失败：{message}", "{name} failed to load: {message}")
	Register(KeyParityModelSelectionEmptyModels, "暂无可用模型", "No models available.")

	// ui-permission-presets
	Register(KeyParityPermTitle, "权限", "Permission")
	Register(KeyParityPermMode, "访问模式，当前：{name}", "Access mode, current: {name}")
	Register(KeyParityPermSwitchFailed, "未能切换权限模式，请重试。", "Permission mode could not be changed. Please try again.")
	Register(KeyParityPermSwitchFailedAlt, "未能切换权限模式，请重试。", "Could not switch permission mode. Please try again.")
	Register(KeyParityPermConfirmDescription, "启用完全权限后，智能体将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任当前任务时使用。", "Full access reduces confirmation steps and lets the agent perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust the current task.")
	Register(KeyParityPermConfirmNewSessionDesc, "启用完全权限后，新会话将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任后续任务时使用。", "Full access lets new sessions reduce confirmation steps and perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust subsequent tasks.")
	Register(KeyParityPermConfirmDescriptionAlt, "完整访问权限允许智能体在不询问的情况下读写工作区外文件、执行系统命令并访问网络。仅在受信任的任务中启用。", "Full access allows the agent to read and write files outside the workspace, execute system commands, and access the network without asking. Enable only for trusted tasks.")
	Register(KeyParityPermAutoConfirmAcknowledge, "我已了解这些风险，并愿意继续", "I understand these risks and want to continue")
	Register(KeyParityPermAutoConfirmAcknowledgeAlt, "我已了解这些风险，并愿意继续", "I understand the risks and want to continue")

	// ui-plan
	Register(KeyParityPlanChipOnAria, "计划模式已开启，按下关闭", "Plan mode on, press to turn off")
	Register(KeyParityPlanChipOnTitle, "计划模式已开启 — 点击关闭（/plan off）", "Plan mode on — click to turn off (/plan off)")

	// ui-skill
	Register(KeyParitySkillRowInstructions, "说明", "Instructions")
	Register(KeyParitySkillRowTitle, "加载技能", "Skill")
	Register(KeyParitySkillMenuUserOnly, "仅用户", "user-only")

	// ui-user-questions
	Register(KeyParityQuestionsErrorIncomplete, "请先完成这道问题。", "Please complete this question first.")
	Register(KeyParityQuestionsErrorUnanswered, "请选择一个选项或填写自定义答案。", "Please select an option or enter a custom answer.")
	Register(KeyParityQuestionsWaitTakeTime, "慢慢回答", "Take time")
	Register(KeyParityQuestionsWaitHeld, "会一直等你回答", "Waiting until you answer")
	Register(KeyParityQuestionsWaitCountdown, "{seconds} 秒后继续工作", "Continuing in {seconds}s")
	Register(KeyParityQuestionsWaitPaused, "已暂停 · 剩余 {seconds} 秒", "Paused · {seconds}s remaining")
	Register(KeyParityQuestionsReviewStatus, "已回答", "Answered")
	Register(KeyParityQuestionsPlanApprove, "同意执行", "Approve")
	Register(KeyParityQuestionsPlanDecline, "拒绝", "Refuse")
	Register(KeyParityQuestionsActionNext, "下一题", "Next")

	// ui-visual-replies
	Register(KeyParityVisualRepliesSettingsDesc, "允许智能体生成交互式页面并在会话内展示。截图检查需要支持图片的模型，或已配置的识图模型；关闭后已生成页面仍可查看。", "Allow the agent to create interactive pages and show them in conversations. Screenshot inspection requires an image-capable model or a configured vision model. Existing pages remain available when this is off.")
	Register(KeyParityVisualRepliesCardSave, "保存 HTML", "Save HTML")
	Register(KeyParityVisualRepliesCardTitle, "视觉回复", "Visual reply")
	Register(KeyParityVisualRepliesCardSource, "查看源码", "View source")

	// ui-workflow-run
	Register(KeyParityWorkflowRunMembersOne, "{count} 个成员", "{count} member")
	Register(KeyParityWorkflowRunStatusCountCompleted, "已完成 {count}", "Completed {count}")
	Register(KeyParityWorkflowRunStatusCountCancelled, "已取消 {count}", "Cancelled {count}")
	Register(KeyParityWorkflowRunStatusCountInterrupt, "已中断 {count}", "Interrupted {count}")
	Register(KeyParityWorkflowRunStatusCountRunning, "运行中 {count}", "Running {count}")
	Register(KeyParityWorkflowRunStatusCountFailed, "失败 {count}", "Failed {count}")
	Register(KeyParityWorkflowRunPhaseEmpty, "空阶段名", "Empty phase name")

	// misc chat-adjacent packages in chat.json
	Register(KeyParityGitBranchLoading, "正在读取分支…", "Reading branches…")
	Register(KeyParityGitCommitNone, "无", "none")
	Register(KeyParityGitConfirmAbort, "取消", "Abort")
	Register(KeyParityDeliverablesReviewWrapAria, "自动换行", "Line wrap")
	Register(KeyParityDiffScopeWorktree, "工作区", "Working tree")
	Register(KeyParityDiffViewUnified, "统一视图", "Unified view")
	Register(KeyParityDiffViewSplit, "分栏视图", "Split view")
	Register(KeyParityMessageEditEditorEditing, "正在下方输入框中重新编辑", "Re-editing in the input box below")
	Register(KeyParityMessageFeedbackLikeActive, "取消标记", "Remove rating")
	Register(KeyParityMessageFeedbackDislikeActive, "取消标记", "Remove rating")
	Register(KeyParityReferenceSectionSessions, "对话", "Sessions")
	Register(KeyParitySecretMaskSecretReveal, "显示密钥", "Show secret")
}

// ChatParityf formats a chat parity locale string by substituting {placeholder} pairs.
func ChatParityf(key string, kv ...string) string {
	s := T(key)
	for i := 0; i+1 < len(kv); i += 2 {
		s = strings.ReplaceAll(s, "{"+kv[i]+"}", kv[i+1])
	}
	return s
}
