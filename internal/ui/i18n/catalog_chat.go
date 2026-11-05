package i18n

// Conversation-surface copy: chat rows, tool cards, process groups, questions
// and read-only history. Keys are declared here and registered from init so the
// conversation domain never edits the shared catalog literals in i18n.go.
const (
	// Tool card details.
	KeyChatToolInput       = "chat.tool.input"
	KeyChatToolIOOutput    = "chat.tool.io_output"
	KeyChatToolArguments   = "chat.tool.arguments"
	KeyChatToolResult      = "chat.tool.result"
	KeyChatToolEvent       = "chat.tool.event"
	KeyChatToolInspect     = "chat.tool.inspect"
	KeyChatToolCopy        = "chat.tool.copy"
	KeyChatToolCallRelated = "chat.tool.call_relation"
	KeyChatToolFailed      = "chat.tool.state.failed"
	KeyChatToolRunning     = "chat.tool.state.running"
	KeyChatToolPreparing   = "chat.tool.state.preparing"
	KeyChatToolDone        = "chat.tool.state.done"
	KeyChatToolStopped     = "chat.tool.state.stopped"

	// Tool titles and structured body labels.
	KeyChatToolRun      = "chat.tool.run"
	KeyChatToolRead     = "chat.tool.read"
	KeyChatToolWrite    = "chat.tool.write"
	KeyChatToolSearch   = "chat.tool.search"
	KeyChatToolWeb      = "chat.tool.web"
	KeyChatToolTodo     = "chat.tool.todo"
	KeyChatToolAsk      = "chat.tool.ask"
	KeyChatToolAgent    = "chat.tool.agent"
	KeyChatToolGeneric  = "chat.tool.generic"
	KeyChatToolCommand  = "chat.tool.command"
	KeyChatToolOutput   = "chat.tool.output"
	KeyChatToolPath     = "chat.tool.path"
	KeyChatToolQuery    = "chat.tool.query"
	KeyChatToolMatches  = "chat.tool.matches"
	KeyChatToolShowAll  = "chat.tool.show_all"
	KeyChatToolShowLess = "chat.tool.show_less"

	// Process group headers and category summaries.
	KeyChatProcessAnalyzing  = "chat.process.analyzing"
	KeyChatProcessComplete   = "chat.process.complete"
	KeyChatProcessGroup      = "chat.process.group"
	KeyChatProcessMore       = "chat.process.more"
	KeyChatProcessReading    = "chat.process.reading"
	KeyChatProcessRead       = "chat.process.read"
	KeyChatProcessEditing    = "chat.process.editing"
	KeyChatProcessEdited     = "chat.process.edited"
	KeyChatProcessSearching  = "chat.process.searching"
	KeyChatProcessSearched   = "chat.process.searched"
	KeyChatProcessBrowsing   = "chat.process.browsing"
	KeyChatProcessBrowsed    = "chat.process.browsed"
	KeyChatProcessRunning    = "chat.process.running"
	KeyChatProcessRan        = "chat.process.ran"
	KeyChatProcessPlanning   = "chat.process.planning"
	KeyChatProcessPlanned    = "chat.process.planned"
	KeyChatProcessWaiting    = "chat.process.waiting"
	KeyChatProcessAsked      = "chat.process.asked"
	KeyChatProcessDelegating = "chat.process.delegating"
	KeyChatProcessDelegated  = "chat.process.delegated"
	KeyChatProcessTools      = "chat.process.tools"
	KeyChatProcessUsedTools  = "chat.process.used_tools"

	// Turn terminal status. KeyChatTurnDone ("chat.turn_done") already exists in
	// i18n.go and is reused as the completed label.
	KeyChatTurnFailed     = "chat.turn.failed"
	KeyChatTurnStopped    = "chat.turn.stopped"
	KeyChatTurnTruncated  = "chat.turn.truncated"
	KeyChatUnknownContent = "chat.unknown_content"
	KeyChatTurnMaxTokens  = "chat.turn.max_tokens"
	KeyChatTurnSignedOut  = "chat.turn.signed_out"

	// Turn usage disclosure.
	KeyChatUsageTitle     = "chat.usage.title"
	KeyChatUsageTokens    = "chat.usage.tokens"
	KeyChatUsageCache     = "chat.usage.cache"
	KeyChatUsageUncached  = "chat.usage.uncached"
	KeyChatUsageOutput    = "chat.usage.output"
	KeyChatUsageTotal     = "chat.usage.total"
	KeyChatUsageCacheRead = "chat.usage.cache_read"
	KeyChatUsageCacheWrit = "chat.usage.cache_write"
	KeyChatUsageReasoning = "chat.usage.reasoning"
	KeyChatUsageCachePct  = "chat.usage.cache_pct"
	KeyChatStatusLoaded   = "chat.status.loaded"
	KeyChatStatusSpeed    = "chat.status.speed"
	KeyChatStatusTimed    = "chat.status.timed"
	KeyChatStatusInput    = "chat.status.input"

	// Local submission echo.
	KeyChatPendingSending  = "chat.pending.sending"
	KeyChatPendingAccepted = "chat.pending.accepted"
	KeyChatPendingSteering = "chat.pending.steering"

	// Thinking row.
	KeyChatThinkingFor = "chat.thinking_for"
	KeyChatDurationSec = "chat.duration.seconds"
	KeyChatDurationMin = "chat.duration.minutes"
	KeyChatDurationMS  = "chat.duration.min_sec"

	// Question card.
	KeyChatQuestionMinimize   = "chat.question.minimize"
	KeyChatQuestionExpand     = "chat.question.expand"
	KeyChatQuestionReview     = "chat.question.review"
	KeyChatQuestionPrev       = "chat.question.prev"
	KeyChatQuestionNext       = "chat.question.next"
	KeyChatQuestionSkip       = "chat.question.skip"
	KeyChatQuestionUndoSkip   = "chat.question.undo_skip"
	KeyChatQuestionSkipped    = "chat.question.skipped"
	KeyChatQuestionUnanswer   = "chat.question.unanswered"
	KeyChatQuestionStep       = "chat.question.step"
	KeyChatQuestionSubmit     = "chat.question.submit"
	KeyChatQuestionReopen     = "chat.question.reopen"
	KeyChatQuestionHide       = "chat.question.hide"
	KeyChatQuestionReviewN    = "chat.question.review_n"
	KeyChatQuestionAnswer     = "chat.question.answer"
	KeyChatQuestionCopyReply  = "chat.question.copy_reply"
	KeyChatQuestionEarlier    = "chat.question.earlier"
	KeyChatQuestionIncomplete = "chat.question.incomplete"
	KeyChatQuestionPickFirst  = "chat.question.pick_first"
	KeyChatQuestionHistory    = "chat.question.history"
	KeyChatQuestionRecorded   = "chat.question.history_recorded"
	KeyChatQuestionQueued     = "chat.question.history_queued"
	KeyChatQuestionCloseHist  = "chat.question.history_close"

	// Read-only (child) transcript.
	KeyChatReadOnlyBadge   = "chat.readonly.badge"
	KeyChatReadOnlyCopy    = "chat.readonly.copy"
	KeyChatReadOnlyClipped = "chat.readonly.clipped"
	KeyChatReadOnlyNote    = "chat.readonly.clipped_note"
	KeyChatReadOnlyLive    = "chat.readonly.live"

	// Inline images.
	KeyChatImageFailed  = "chat.image.failed"
	KeyChatImageInvalid = "chat.image.invalid"
	KeyChatImageRetry   = "chat.image.retry"
	KeyChatImageEnlarge = "chat.image.enlarge"
	KeyChatImagePrev    = "chat.image.prev"
	KeyChatImageNext    = "chat.image.next"
	KeyChatImageFit     = "chat.image.fit"
	KeyChatImageCount   = "chat.image.count"

	// Isolated OpenUI fence chrome.
	KeyChatIsolated        = "chat.isolated.title"
	KeyChatIsolatedSource  = "chat.isolated.source"
	KeyChatIsolatedOpen    = "chat.isolated.open"
	KeyChatIsolatedStart   = "chat.isolated.start"
	KeyChatIsolatedGesture = "chat.isolated.gesture"
	KeyChatIsolatedSend    = "chat.isolated.send"
	KeyChatIsolatedIgnore  = "chat.isolated.ignore"
	KeyChatIsolatedVisible = "chat.isolated.visible"

	// Context injection rows.
	KeyChatContextTitle       = "chat.context.title"
	KeyChatContextRecall      = "chat.context.recall"
	KeyChatContextContent     = "chat.context.model_content"
	KeyChatContextCopy        = "chat.context.copy"
	KeyChatContextClipped     = "chat.context.clipped"
	KeyChatToolAdded          = "chat.context.tool_added"
	KeyChatToolRemoved        = "chat.context.tool_removed"
	KeyChatToolsAdded         = "chat.context.tools_added"
	KeyChatToolsRemoved       = "chat.context.tools_removed"
	KeyChatToolsAddedCount    = "chat.context.tools_added_count"
	KeyChatToolsRemovedCount  = "chat.context.tools_removed_count"
	KeyChatToolsChanged       = "chat.context.tools_changed"
	KeyChatToolsUpdated       = "chat.context.tools_updated"
	KeyChatTriggerRequest     = "chat.trigger.request"
	KeyChatTriggerGoal        = "chat.trigger.goal"
	KeyChatTriggerAgent       = "chat.trigger.agent"
	KeyChatTriggerTeam        = "chat.trigger.team"
	KeyChatTriggerSubagent    = "chat.trigger.subagent"
	KeyChatTriggerGithub      = "chat.trigger.github"
	KeyChatTriggerWebhook     = "chat.trigger.webhook"
	KeyChatTriggerSchedule    = "chat.trigger.schedule"
	KeyChatTriggerJob         = "chat.trigger.job"
	KeyChatTriggerPlugin      = "chat.trigger.plugin"
	KeyChatTriggerExplanation = "chat.trigger.explanation"

	// Group room transcript.
	KeyChatGroupTitle   = "chat.group.title"
	KeyChatGroupEarlier = "chat.group.earlier"
	KeyChatGroupEmpty   = "chat.group.empty"
	KeyChatGroupReply   = "chat.group.reply"
	KeyChatGroupCount   = "chat.group.count"
	KeyChatGroupLate    = "chat.group.late"

	// Workflow run card.
	KeyChatWorkflowMembers    = "chat.workflow.members"
	KeyChatWorkflowNoMember   = "chat.workflow.no_member"
	KeyChatWorkflowUngrouped  = "chat.workflow.ungrouped"
	KeyChatWorkflowEmptyPhase = "chat.workflow.empty_phase"
	KeyChatWorkflowUnnamed    = "chat.workflow.unnamed"
	KeyChatWorkflowCopy       = "chat.workflow.copy"

	// Tool file/diff bodies.
	KeyChatToolAppliedDiff  = "chat.tool.applied_diff"
	KeyChatToolProposedDiff = "chat.tool.proposed_diff"
	KeyChatToolDiffLarge    = "chat.tool.diff_large"
	KeyChatToolNoDiff       = "chat.tool.no_diff"
	KeyChatToolExpandFile   = "chat.tool.expand_file"
	KeyChatToolCollapseFile = "chat.tool.collapse_file"

	// Read-only resources.
	KeyChatResourcePreview = "chat.resource.preview"
	KeyChatResourceReading = "chat.resource.reading"
	KeyChatResourceMore    = "chat.resource.more"

	// Inline image actions.
	KeyChatImageCloseOne = "chat.image.close_one"
	KeyChatImageGallery  = "chat.image.gallery"

	// Markdown code-card toolbar.
	// Retry and schedule rows.
	KeyChatRetryWaiting       = "chat.retry.waiting"
	KeyChatRetryStarted       = "chat.retry.started"
	KeyChatRetryCancelled     = "chat.retry.cancelled"
	KeyChatSystemPrompt       = "chat.system_prompt"
	KeyChatSystemPromptUpdate = "chat.system_prompt_update"
	KeyChatScheduleTask       = "chat.schedule.task"
	KeyChatScheduleReceipt    = "chat.schedule.receipt"
	KeyChatScheduleDeleted    = "chat.schedule.deleted"
	KeyChatScheduleDetails    = "chat.schedule.details"
	KeyChatImageLoading       = "chat.image.loading"
	KeyChatImageMissing       = "chat.image.missing"
	KeyChatFeedbackEdit       = "chat.feedback.edit"
	KeyChatTrajDisplayLimited = "chat.traj.display_limited"
	KeyChatTrajDepthLimit     = "chat.traj.depth_limit"

	// Turn deliverables card.
	KeyChatDeliveryTitle       = "chat.delivery.title"
	KeyChatDeliveryCheckHost   = "chat.delivery.check_host"
	KeyChatDeliveryRetryHost   = "chat.delivery.retry_host"
	KeyChatDeliveryNoDesktop   = "chat.delivery.no_desktop"
	KeyChatDeliveryReview      = "chat.delivery.review"
	KeyChatDeliveryRetryReview = "chat.delivery.retry_review"
	KeyChatDeliveryReadSummary = "chat.delivery.reading_summary"
	KeyChatDeliveryFiles       = "chat.delivery.files"
	KeyChatDeliveryRetryDiff   = "chat.delivery.retry_diff"
	KeyChatDeliveryReadDiff    = "chat.delivery.reading_diff"
	KeyChatDeliveryBinary      = "chat.delivery.binary"
	KeyChatDeliveryDiffClip    = "chat.delivery.diff_clip"
	KeyChatDeliveryTooLarge    = "chat.delivery.too_large"
	KeyChatDeliveryShowAll     = "chat.delivery.show_all"
	KeyChatDeliveryCollapse    = "chat.delivery.collapse"
	KeyChatDeliveryOpenHost    = "chat.delivery.open_host"
	KeyChatDeliveryRevealHost  = "chat.delivery.reveal_host"
	KeyChatDeliveryRetryAction = "chat.delivery.retry_action"
	KeyChatDeliveryAccepted    = "chat.delivery.accepted"

	// Published-page (visual reply) card.
	KeyChatVisualInteract    = "chat.visual.interact"
	KeyChatVisualStop        = "chat.visual.stop"
	KeyChatVisualSafePreview = "chat.visual.safe_preview"
	KeyChatVisualSource      = "chat.visual.source"
	KeyChatVisualExpand      = "chat.visual.expand"
	KeyChatVisualCollapse    = "chat.visual.collapse"
	KeyChatVisualNoPage      = "chat.visual.no_page"
	KeyChatVisualNoPreview   = "chat.visual.no_preview"
	KeyChatVisualStatic      = "chat.visual.static"
	KeyChatVisualCapped      = "chat.visual.capped"

	// Read-only workbook viewer.
	KeyChatWorkbookReadOnly  = "chat.workbook.read_only"
	KeyChatWorkbookPrevRows  = "chat.workbook.prev_rows"
	KeyChatWorkbookNextRows  = "chat.workbook.next_rows"
	KeyChatWorkbookPrevCols  = "chat.workbook.prev_columns"
	KeyChatWorkbookNextCols  = "chat.workbook.next_columns"
	KeyChatWorkbookCopyShown = "chat.workbook.copy_shown"
	KeyChatWorkbookCopyRaw   = "chat.workbook.copy_raw"

	// Second localization sweep: workbook TSV, inline OpenUI, retry copy, visual
	// reply and workflow notes.
	KeyChatWorkbookCopyRawTSV    = "chat.workbook.copy_raw_tsv"
	KeyChatWorkbookCopyShownTSV  = "chat.workbook.copy_shown_tsv"
	KeyChatOpenUIRetry           = "chat.openui.retry"
	KeyChatOpenUIClose           = "chat.openui.close"
	KeyChatRetryAuthFailed       = "chat.retry.auth_failed"
	KeyChatRetrySignedOut        = "chat.retry.signed_out"
	KeyChatRetrySignIn           = "chat.retry.sign_in"
	KeyChatRetryQuota            = "chat.retry.quota"
	KeyChatRetryAwaitingHost     = "chat.retry.awaiting_host"
	KeyChatVisualPublished       = "chat.visual.published"
	KeyChatVisualIsolatedNote    = "chat.visual.isolated_note"
	KeyChatVisualSaveAs          = "chat.visual.save_as"
	KeyChatVisualReading         = "chat.visual.reading"
	KeyChatVisualPreparing       = "chat.visual.preparing"
	KeyChatVisualChecking        = "chat.visual.checking"
	KeyChatVisualLoadingPage     = "chat.visual.loading_page"
	KeyChatVisualIsolatedTooltip = "chat.visual.isolated_tooltip"
	KeyChatVisualSourceNote      = "chat.visual.source_note"
	KeyChatWorkflowEndedNote     = "chat.workflow.ended_note"

	// Turn rail.
	KeyChatTurnRailLabel    = "chat.turn_rail.label"
	KeyChatTurnRailJump     = "chat.turn_rail.jump"
	KeyChatTurnRailJumpLoad = "chat.turn_rail.jump_load"
	KeyChatTurnRailTurn  = "chat.turn_rail.turn"
	KeyChatTurnRailEmpty = "chat.turn_rail.empty"

	// Queue dock.
	KeyChatQueueTitle        = "chat.queue.title"
	KeyChatQueueEmptyPreview = "chat.queue.empty_preview"

	// Interaction cards.
	KeyChatApprovalWaiting           = "chat.approval.waiting"
	KeyChatQuestionWaiting           = "chat.question.waiting"
	KeyChatQuestionCustomPlaceholder = "chat.question.custom_placeholder"
	KeyChatQuestionTakeTime          = "chat.question.take_time"

	KeyChatAccountStopped = "chat.failure.account_stopped"

	// Search / web / todo evidence blocks.
	KeyChatToolNoMatches = "chat.tool.no_matches"
	KeyChatToolMoreHits  = "chat.tool.more_hits"
	KeyChatWebTruncated  = "chat.web.truncated"
	KeyChatTodoDone      = "chat.todo.done"
	KeyChatTodoRunning   = "chat.todo.running"
	KeyChatTodoPending   = "chat.todo.pending"
	KeyChatTodoEmpty     = "chat.todo.empty"

	// Attachment cards.
	KeyChatAttachmentFile = "chat.attachment.file"
	// Shared row actions.
	KeyChatRowExpand   = "chat.row.expand"
	KeyChatRowCollapse = "chat.row.collapse"

	KeyMarkdownCode      = "markdown.code"
	KeyMarkdownCopyCode  = "markdown.copy_code"
	KeyMarkdownCopied    = "markdown.copied"
	KeyMarkdownWrap      = "markdown.wrap"
	KeyMarkdownNoWrap    = "markdown.nowrap"
	KeyMarkdownTableWide = "markdown.table_wide"
)

func init() {
	// Tool card.
	Register(KeyChatToolInput, "输入", "IN")
	Register(KeyChatToolIOOutput, "输出", "OUT")
	Register(KeyChatToolArguments, "参数", "Arguments")
	Register(KeyChatToolResult, "结果", "Result")
	Register(KeyChatToolEvent, "原始记录", "Raw event")
	Register(KeyChatToolInspect, "查看", "Inspect")
	Register(KeyChatToolCopy, "复制", "Copy")
	Register(KeyChatToolCallRelated, "调用关系", "Call relation")
	Register(KeyChatToolFailed, "失败", "Failed")
	Register(KeyChatToolRunning, "运行中", "Running")
	Register(KeyChatToolPreparing, "准备中", "Preparing")
	Register(KeyChatToolDone, "完成", "Done")
	Register(KeyChatToolStopped, "已停止", "Stopped")
	Register(KeyChatToolRun, "运行命令", "Run command")
	Register(KeyChatToolRead, "读取文件", "Read file")
	Register(KeyChatToolWrite, "修改文件", "Edit file")
	Register(KeyChatToolSearch, "搜索", "Search")
	Register(KeyChatToolWeb, "访问网页", "Browse")
	Register(KeyChatToolTodo, "更新计划", "Update plan")
	Register(KeyChatToolAsk, "提问", "Ask question")
	Register(KeyChatToolAgent, "子代理", "Subagent")
	Register(KeyChatToolGeneric, "调用工具", "Use tool")
	Register(KeyChatToolCommand, "命令", "Command")
	Register(KeyChatToolOutput, "输出", "Output")
	Register(KeyChatToolPath, "路径", "Path")
	Register(KeyChatToolQuery, "关键词", "Query")
	Register(KeyChatToolMatches, "匹配", "Matches")
	Register(KeyChatToolShowAll, "展开全部", "Show all")
	Register(KeyChatToolShowLess, "收起", "Show less")

	// Process groups.
	Register(KeyChatProcessAnalyzing, "正在分析请求", "Analyzing request")
	Register(KeyChatProcessComplete, "已完成分析", "Analysis complete")
	Register(KeyChatProcessGroup, "工作步骤", "Work details")
	Register(KeyChatProcessMore, "等", "more")
	Register(KeyChatProcessReading, "正在读取文件", "Reading files")
	Register(KeyChatProcessRead, "已读取文件", "Read files")
	Register(KeyChatProcessEditing, "正在编辑文件", "Editing files")
	Register(KeyChatProcessEdited, "修改了文件", "Edited files")
	Register(KeyChatProcessSearching, "正在搜索代码", "Searching code")
	Register(KeyChatProcessSearched, "已搜索代码", "Searched code")
	Register(KeyChatProcessBrowsing, "正在访问网页", "Browsing")
	Register(KeyChatProcessBrowsed, "已访问网页", "Browsed")
	Register(KeyChatProcessRunning, "正在运行命令", "Running commands")
	Register(KeyChatProcessRan, "执行了命令", "Ran commands")
	Register(KeyChatProcessPlanning, "正在更新计划", "Updating plan")
	Register(KeyChatProcessPlanned, "更新了计划", "Updated plan")
	Register(KeyChatProcessWaiting, "等待你的操作", "Waiting for input")
	Register(KeyChatProcessAsked, "向用户提出了问题", "Asked for input")
	Register(KeyChatProcessDelegating, "正在协调子智能体", "Coordinating agents")
	Register(KeyChatProcessDelegated, "已协调子智能体", "Coordinated agents")
	Register(KeyChatProcessTools, "正在调用工具", "Using tools")
	Register(KeyChatProcessUsedTools, "已调用工具", "Used tools")

	// Turn status.
	Register(KeyChatTurnFailed, "运行失败", "Failed")
	Register(KeyChatTurnStopped, "已停止", "Stopped")
	Register(KeyChatTurnTruncated, "输出已截断", "Output limit reached")
	Register(KeyChatUnknownContent, "未知内容", "Unknown content")
	Register(KeyChatTurnMaxTokens, "已达到输出 token 上限", "Output token limit reached")
	Register(KeyChatTurnSignedOut, "已停止，因为你已退出 DeepSeek。 [ACCOUNT_SIGNED_OUT]", "Stopped because you signed out of DeepSeek. [ACCOUNT_SIGNED_OUT]")

	// Usage.
	Register(KeyChatUsageTitle, "本轮用量", "Turn usage")
	Register(KeyChatUsageTokens, "Token", "Tokens")
	Register(KeyChatUsageCache, "缓存命中", "Cache")
	Register(KeyChatUsageUncached, "未缓存输入", "Uncached input")
	Register(KeyChatUsageOutput, "输出", "Output")
	Register(KeyChatUsageTotal, "总计", "Total")
	Register(KeyChatUsageCacheRead, "缓存读取", "Cached")
	Register(KeyChatUsageCacheWrit, "缓存写入", "Cache created")
	Register(KeyChatUsageReasoning, "（其中推理 %s）", "(reasoning %s)")
	Register(KeyChatUsageCachePct, "缓存命中 %.1f%%", "Cache %.1f%%")
	Register(KeyChatStatusLoaded, "已加载 %d 轮 · %d 步", "%d turns · %d steps")
	Register(KeyChatStatusSpeed, "输出 %.1f tok/s", "Output %.1f tok/s")
	Register(KeyChatStatusTimed, "有解码时间的输出 %.0f tokens · %.0f ms", "Timed output %.0f tokens · %.0f ms")
	Register(KeyChatStatusInput, "会话输入 %.0f · 缓存读取 %.0f", "Session input %.0f · cache read %.0f")

	// Submission echo.
	Register(KeyChatPendingSending, "发送中…", "Sending…")
	Register(KeyChatPendingAccepted, "已接受，等待会话更新", "Accepted, waiting for conversation update")
	Register(KeyChatPendingSteering, "插话", "Steering")

	// Thinking.
	Register(KeyChatThinkingFor, "正在思考，用时 %s ···", "Thinking for %s ···")
	Register(KeyChatDurationSec, "%d 秒", "%ds")
	Register(KeyChatDurationMin, "%d 分", "%dm")
	Register(KeyChatDurationMS, "%d 分 %d 秒", "%dm %ds")

	// Question card.
	Register(KeyChatQuestionMinimize, "收起问题", "Minimize question")
	Register(KeyChatQuestionExpand, "展开问题", "Expand question")
	Register(KeyChatQuestionReview, "查看答案摘要", "Review answers")
	Register(KeyChatQuestionPrev, "上一题", "Previous question")
	Register(KeyChatQuestionNext, "下一题", "Next question")
	Register(KeyChatQuestionSkip, "跳过此题", "Skip question")
	Register(KeyChatQuestionUndoSkip, "撤销跳过", "Undo skip")
	Register(KeyChatQuestionSkipped, "已跳过", "Skipped")
	Register(KeyChatQuestionUnanswer, "未回答", "Unanswered")
	Register(KeyChatQuestionStep, "下一步", "Next")
	Register(KeyChatQuestionSubmit, "提交", "Submit")
	Register(KeyChatQuestionReopen, "重新打开问题", "Reopen questions")
	Register(KeyChatQuestionHide, "暂时隐藏", "Hide for now")
	Register(KeyChatQuestionReviewN, "回看第 %d 题", "Review question %d")
	Register(KeyChatQuestionAnswer, "回答", "Answer")
	Register(KeyChatQuestionCopyReply, "复制回复", "Copy reply")
	Register(KeyChatQuestionEarlier, "之前问题的回复", "Reply to earlier questions")
	Register(KeyChatQuestionIncomplete, "请回答或跳过其余问题后提交。", "Complete or skip the remaining questions before submitting.")
	Register(KeyChatQuestionPickFirst, "请先回答此题再继续。", "Answer this question before continuing.")
	Register(KeyChatQuestionHistory, "查看回答历史", "Review answer history")
	Register(KeyChatQuestionRecorded, "已记录回答（只读）", "Recorded answers (read-only)")
	Register(KeyChatQuestionQueued, "已排队回答（只读）", "Queued answers (read-only)")
	Register(KeyChatQuestionCloseHist, "关闭历史", "Close history")

	// Read-only transcript.
	Register(KeyChatReadOnlyBadge, "子会话只读阅读", "Child read-only history")
	Register(KeyChatReadOnlyCopy, "复制消息", "Copy message")
	Register(KeyChatReadOnlyClipped, "[展示截断]", "[display clipped]")
	Register(KeyChatReadOnlyNote, "展示已截断：最多 400 行、每行 64 KiB、合计 512 KiB；原始历史与游标未变。", "Display clipped: at most 400 rows, 64 KiB per row, 512 KiB total; history and cursors are unchanged.")
	Register(KeyChatReadOnlyLive, "实时临时输出", "live preview")

	// Images.
	Register(KeyChatImageFailed, "图片无法预览", "Image unavailable")
	Register(KeyChatImageInvalid, "图片无法解码", "Invalid image")
	Register(KeyChatImageRetry, "重试", "Retry")
	Register(KeyChatImageEnlarge, "放大", "Enlarge")
	Register(KeyChatImagePrev, "上一张", "Previous image")
	Register(KeyChatImageNext, "下一张", "Next image")
	Register(KeyChatImageFit, "适应", "Fit")
	Register(KeyChatImageCount, "%d 张图片", "%d images")

	// Isolated OpenUI fence.
	Register(KeyChatIsolated, "OpenUI · 隔离交互", "OpenUI · Isolated interaction")
	Register(KeyChatIsolatedSource, "源码", "Source")
	Register(KeyChatIsolatedOpen, "打开独立窗口", "Open separate window")
	Register(KeyChatIsolatedStart, "正在启动隔离渲染器", "Starting isolated renderer")
	Register(KeyChatIsolatedGesture, "交互请求尚未发送：渲染消息不含可信用户手势证明", "Confirm this action: renderer messages do not attest a user gesture.")
	Register(KeyChatIsolatedSend, "发送交互结果", "Send interaction result")
	Register(KeyChatIsolatedIgnore, "忽略交互请求", "Dismiss interaction request")
	Register(KeyChatIsolatedVisible, "当前可见时加载真实运行资源", "Loads the real runtime when visible")

	// Context injection.
	Register(KeyChatContextTitle, "上下文注入", "Context injection")
	Register(KeyChatContextRecall, "跨会话召回", "Session recall")
	Register(KeyChatToolAdded, "已添加工具：%s", "Tool added: %s")
	Register(KeyChatToolRemoved, "已移除工具：%s", "Tool removed: %s")
	Register(KeyChatToolsAdded, "新增：%s", "Added: %s")
	Register(KeyChatToolsRemoved, "移除：%s", "Removed: %s")
	Register(KeyChatToolsAddedCount, "新增 %d 个", "%d added")
	Register(KeyChatToolsRemovedCount, "移除 %d 个", "%d removed")
	Register(KeyChatToolsChanged, "新增 %d 个，移除 %d 个", "%d added, %d removed")
	Register(KeyChatToolsUpdated, "工具已更新", "Tools updated")
	Register(KeyChatTriggerRequest, "收到执行请求", "Execution requested")
	Register(KeyChatTriggerGoal, "继续执行目标", "Continuing goal")
	Register(KeyChatTriggerAgent, "收到任务消息", "Task message received")
	Register(KeyChatTriggerTeam, "收到团队消息", "Team message received")
	Register(KeyChatTriggerSubagent, "子任务状态更新", "Subtask status updated")
	Register(KeyChatTriggerGithub, "收到 GitHub 事件", "GitHub event received")
	Register(KeyChatTriggerWebhook, "收到外部事件", "External event received")
	Register(KeyChatTriggerSchedule, "自动化任务", "Automation task")
	Register(KeyChatTriggerJob, "后台任务状态更新", "Background task updated")
	Register(KeyChatTriggerPlugin, "插件状态更新", "Plugin status updated")
	Register(KeyChatTriggerExplanation, "这条通知触发了本轮回复。", "This notification triggered this response.")
	Register(KeyChatContextContent, "模型可见内容", "Model-facing content")
	Register(KeyChatContextCopy, "复制上下文", "Copy context")
	Register(KeyChatContextClipped, "展示已截断；复制可获取完整内容。", "Display clipped; copy includes full content.")

	// Group room.
	Register(KeyChatGroupTitle, "群聊讨论", "Group discussion")
	Register(KeyChatGroupEarlier, "加载更早讨论", "Earlier discussions")
	Register(KeyChatGroupEmpty, "尚无讨论", "No discussion yet")
	Register(KeyChatGroupReply, "回复此讨论", "Reply to thread")
	Register(KeyChatGroupCount, "%s · %d 条消息", "%s · %d messages")
	Register(KeyChatGroupLate, "延迟回复", "Late reply")

	// Workflow run.
	Register(KeyChatWorkflowMembers, "%d 成员", "%d members")
	Register(KeyChatWorkflowNoMember, "尚无已启动成员", "No member has started")
	Register(KeyChatWorkflowUngrouped, "未分组", "Unassigned")
	Register(KeyChatWorkflowEmptyPhase, "空阶段名", "Empty phase")
	Register(KeyChatWorkflowUnnamed, "未命名成员", "Unnamed member")
	Register(KeyChatWorkflowCopy, "复制运行记录", "Copy run records")

	// Tool file/diff bodies.
	Register(KeyChatToolAppliedDiff, "已写入差异", "Applied diff")
	Register(KeyChatToolProposedDiff, "拟议差异", "Proposed diff")
	Register(KeyChatToolDiffLarge, "差异过大，请展开原始记录", "Large diff: expand the raw payload")
	Register(KeyChatToolNoDiff, "无文本差异", "No text changes")
	Register(KeyChatToolExpandFile, "展开文件内容", "Expand file content")
	Register(KeyChatToolCollapseFile, "收起文件内容", "Collapse file content")

	// Read-only resources.
	Register(KeyChatResourcePreview, "预览", "Preview")
	Register(KeyChatResourceReading, "读取中…", "Reading…")
	Register(KeyChatResourceMore, "继续读取", "Read more")

	// Image actions.
	Register(KeyChatImageCloseOne, "关闭图片", "Close image")
	Register(KeyChatImageGallery, "%d 张图片", "%d images")

	// Markdown code-card toolbar.
	Register(KeyMarkdownCode, "代码", "Code")
	Register(KeyMarkdownCopyCode, "复制代码", "Copy code")
	Register(KeyMarkdownCopied, "已复制", "Copied")
	Register(KeyMarkdownWrap, "自动换行", "Wrap lines")
	Register(KeyMarkdownNoWrap, "保留列宽", "Preserve columns")
	Register(KeyMarkdownTableWide, "宽表格，可横向滚动", "Wide table, scrolls horizontally")
	Register(KeyChatRetryWaiting, "等待重试模型请求", "Waiting to retry")
	Register(KeyChatRetryStarted, "已重试模型请求", "Retried model request")
	Register(KeyChatRetryCancelled, "模型请求重试已取消", "Model request retry cancelled")
	Register(KeyChatSystemPrompt, "系统提示词", "System prompt")
	Register(KeyChatSystemPromptUpdate, "系统提示词更新", "System prompt update")
	Register(KeyChatScheduleTask, "定时任务", "Scheduled task")
	Register(KeyChatScheduleReceipt, "创建时记录", "Creation receipt")
	Register(KeyChatScheduleDeleted, "任务已删除", "Task deleted")
	Register(KeyChatScheduleDetails, "任务详情", "Task details")
	Register(KeyChatImageLoading, "加载图片", "Loading image")
	Register(KeyChatImageMissing, "图片不存在或无法读取", "Image not found or unreadable")
	Register(KeyChatFeedbackEdit, "编辑反馈", "Edit feedback")
	Register(KeyChatTrajDisplayLimited, "展示已限量；复制保留完整内容", "Display limited; copy retains full content")
	Register(KeyChatTrajDepthLimit, "深度限制；复制包含子树", "Depth limit; copy includes subtree")

	// Turn deliverables.
	Register(KeyChatDeliveryTitle, "本轮交付与变更 · 第 %d 轮", "Turn %d deliveries and review")
	Register(KeyChatDeliveryCheckHost, "检查 Host 文件打开能力", "Check Host desktop")
	Register(KeyChatDeliveryRetryHost, "重试 Host 信息", "Retry Host info")
	Register(KeyChatDeliveryNoDesktop, "桌面不可用", "desktop unavailable")
	Register(KeyChatDeliveryReview, "查看本轮变更 · 第 %d 轮 · 事件 %d", "Review turn %d · event %d")
	Register(KeyChatDeliveryRetryReview, "重试读取本轮变更", "Retry reading this turn's changes")
	Register(KeyChatDeliveryReadSummary, "正在读取该轮摘要…", "Reading this turn's summary…")
	Register(KeyChatDeliveryFiles, "%d 个文件 · +%d −%d", "%d files · +%d −%d")
	Register(KeyChatDeliveryRetryDiff, "重试读取对比", "Retry comparison")
	Register(KeyChatDeliveryReadDiff, "读取对比中…", "Reading comparison…")
	Register(KeyChatDeliveryBinary, "二进制文件，无文本对比", "Binary file; no text comparison")
	Register(KeyChatDeliveryDiffClip, "对比展示截断为 64 KiB", "Comparison display limited to 64 KiB")
	Register(KeyChatDeliveryTooLarge, "文件过大，Host 未提供文本对比", "Too large for a text comparison")
	Register(KeyChatDeliveryShowAll, "展开全部文件", "Show all files")
	Register(KeyChatDeliveryCollapse, "收起", "Collapse")
	Register(KeyChatDeliveryOpenHost, "在 Host 打开", "Open on Host")
	Register(KeyChatDeliveryRevealHost, "在 Host 定位", "Reveal on Host")
	Register(KeyChatDeliveryRetryAction, "重试文件动作", "Retry file action")
	Register(KeyChatDeliveryAccepted, "Host 已接受文件动作", "Host accepted file action")

	// Published-page card.
	Register(KeyChatVisualInteract, "在卡片中交互", "Interact inline")
	Register(KeyChatVisualStop, "停止交互", "Stop interaction")
	Register(KeyChatVisualSafePreview, "安全图片预览", "Safe image preview")
	Register(KeyChatVisualSource, "查看完整源码", "View source")
	Register(KeyChatVisualExpand, "展开", "Expand")
	Register(KeyChatVisualCollapse, "收起", "Collapse")
	Register(KeyChatVisualNoPage, "交互页未启动或已停止，源码仍可查看或另存", "Interactive page unavailable")
	Register(KeyChatVisualNoPreview, "安全预览不可用，仍可查看源码或另存", "Safe preview unavailable")
	Register(KeyChatVisualStatic, "此图为静态 PNG；请在卡片交互表面操作", "This image is static; use the interactive surface for controls")
	Register(KeyChatVisualCapped, "页面超过 4000 像素，预览未覆盖底部", "Preview is capped at 4000 pixels; the bottom is not shown")

	// Read-only workbook.
	Register(KeyChatWorkbookReadOnly, "只读工作簿；公式显示已存储值", "Read-only workbook; formulas show stored values")
	Register(KeyChatWorkbookPrevRows, "上一行页", "Previous rows")
	Register(KeyChatWorkbookNextRows, "下一行页", "Next rows")
	Register(KeyChatWorkbookPrevCols, "上一列页", "Previous columns")
	Register(KeyChatWorkbookNextCols, "下一列页", "Next columns")
	Register(KeyChatWorkbookCopyShown, "复制单元格显示值", "Copy displayed cell value")
	Register(KeyChatWorkbookCopyRaw, "复制单元格原文", "Copy cell value")

	// Attachment cards.
	Register(KeyChatTurnRailLabel, "轮次导航", "Turn navigation")
	Register(KeyChatTurnRailJump, "跳转到第 %d 轮", "Jump to turn %d")
	Register(KeyChatTurnRailJumpLoad, "加载并跳转到第 %d 轮", "Load and jump to turn %d")
	Register(KeyChatTurnRailTurn, "轮次", "Turn")
	Register(KeyChatTurnRailEmpty, "（空白回合）", "(empty turn)")
	Register(KeyChatWorkbookCopyRawTSV, "复制本页原始 TSV", "Copy raw page TSV")
	Register(KeyChatWorkbookCopyShownTSV, "复制本页显示 TSV", "Copy displayed page TSV")
	Register(KeyChatOpenUIRetry, "重试内联 UI", "Retry inline UI")
	Register(KeyChatOpenUIClose, "关闭交互 UI", "Close interactive UI")
	Register(KeyChatRetryAuthFailed, "API 密钥无效", "API key is invalid")
	Register(KeyChatRetrySignedOut, "任务已因退出 DeepSeek 登录而停止。", "Stopped because you signed out of DeepSeek.")
	Register(KeyChatRetrySignIn, "请先登录 DeepSeek，并确认请求地址支持账号认证。", "Sign in to DeepSeek and ensure the request destination supports account authentication.")
	Register(KeyChatRetryQuota, "当前请求的额度已用尽", "Request quota exhausted.")
	Register(KeyChatRetryAwaitingHost, "等待 Host", "Awaiting Host")
	Register(KeyChatVisualPublished, "已发布 HTML", "Published HTML")
	Register(KeyChatVisualIsolatedNote, "在隔离进程中的卡片表面运行控件和脚本；不开放网络、本机文件、应用权限或持久登录。", "Runs this page in an isolated process without network, local files or persistent sign-in.")
	Register(KeyChatVisualSaveAs, "另存 HTML…", "Save as…")
	Register(KeyChatVisualReading, "正在授权读取…", "Reading published page…")
	Register(KeyChatVisualPreparing, "正在准备隔离交互表面…", "Preparing isolated interactive surface…")
	Register(KeyChatVisualChecking, "正在检查隔离环境并生成静态预览…", "Checking isolation and rendering a static preview…")
	Register(KeyChatVisualLoadingPage, "正在加载交互页面…", "Loading interactive page…")
	Register(KeyChatVisualIsolatedTooltip, "在隔离进程中的卡片表面运行控件和脚本；不开放网络、本机文件、应用权限或持久登录。", "Runs this page in an isolated process without network, local files or application access.")
	Register(KeyChatVisualSourceNote, "源码视图为纯文本；安全图片预览为 PNG，交互页在隔离进程运行。另存后在外部浏览器打开可能运行脚本或访问网络。", "Source is plain text; image preview is PNG, and interaction uses a separate isolated process. Opening a saved page externally may run scripts or access the network.")
	Register(KeyChatWorkflowEndedNote, "已结束或不可用成员保留轨迹，不会重新启动。", "Unavailable members retain their recorded status.")
	Register(KeyChatQueueTitle, "待发送队列", "Queued prompts")
	Register(KeyChatQueueEmptyPreview, "（无预览）", "(no preview)")
	Register(KeyChatApprovalWaiting, "等待审批", "Waiting for approval")
	Register(KeyChatQuestionWaiting, "需要你的回答", "Waiting for your answer")
	Register(KeyChatQuestionCustomPlaceholder, "或输入自定义回答…", "Or write your own answer…")
	Register(KeyChatQuestionTakeTime, "让我慢慢回答", "Take my time")
	Register(KeyChatAccountStopped, "任务已停止", "Task stopped")

	Register(KeyChatToolNoMatches, "没有命中", "No matches")
	Register(KeyChatToolMoreHits, "还有 %d 条", "%d more")
	Register(KeyChatWebTruncated, "结果已截断", "Results truncated")
	Register(KeyChatTodoDone, "已完成", "Done")
	Register(KeyChatTodoRunning, "进行中", "In progress")
	Register(KeyChatTodoPending, "待办", "Pending")
	Register(KeyChatTodoEmpty, "没有待办项", "No todo items")
	Register(KeyChatAttachmentFile, "文件", "File")

	// Shared row actions.
	Register(KeyChatRowExpand, "展开", "Expand")
	Register(KeyChatRowCollapse, "收起", "Collapse")
}

// Question option recommendation badge.
const KeyChatQuestionRecommended = "chat.question.recommended"

func init() {
	Register(KeyChatQuestionRecommended, "推荐", "Recommended")
}
