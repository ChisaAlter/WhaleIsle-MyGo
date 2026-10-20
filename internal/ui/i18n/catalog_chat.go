package i18n

// Conversation-surface copy: chat rows, tool cards, process groups, questions
// and read-only history. Keys are declared here and registered from init so the
// conversation domain never edits the shared catalog literals in i18n.go.
const (
	// Tool card details.
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
	KeyChatTurnFailed    = "chat.turn.failed"
	KeyChatTurnStopped   = "chat.turn.stopped"
	KeyChatTurnTruncated = "chat.turn.truncated"
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
	KeyChatQuestionMinimize  = "chat.question.minimize"
	KeyChatQuestionExpand    = "chat.question.expand"
	KeyChatQuestionReview    = "chat.question.review"
	KeyChatQuestionPrev      = "chat.question.prev"
	KeyChatQuestionNext      = "chat.question.next"
	KeyChatQuestionSkip      = "chat.question.skip"
	KeyChatQuestionUndoSkip  = "chat.question.undo_skip"
	KeyChatQuestionSkipped   = "chat.question.skipped"
	KeyChatQuestionUnanswer  = "chat.question.unanswered"
	KeyChatQuestionStep      = "chat.question.step"
	KeyChatQuestionSubmit    = "chat.question.submit"
	KeyChatQuestionReopen    = "chat.question.reopen"
	KeyChatQuestionHide      = "chat.question.hide"
	KeyChatQuestionReviewN   = "chat.question.review_n"
	KeyChatQuestionAnswer    = "chat.question.answer"
	KeyChatQuestionCopyReply = "chat.question.copy_reply"
	KeyChatQuestionEarlier   = "chat.question.earlier"
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
	KeyChatContextTitle   = "chat.context.title"
	KeyChatContextRecall  = "chat.context.recall"
	KeyChatContextContent = "chat.context.model_content"
	KeyChatContextCopy    = "chat.context.copy"
	KeyChatContextClipped = "chat.context.clipped"

	// Group room transcript.
	KeyChatGroupTitle   = "chat.group.title"
	KeyChatGroupEarlier = "chat.group.earlier"
	KeyChatGroupEmpty   = "chat.group.empty"
	KeyChatGroupReply   = "chat.group.reply"
	KeyChatGroupCount   = "chat.group.count"
	KeyChatGroupLate    = "chat.group.late"

	// Workflow run card.
	KeyChatWorkflowMembers  = "chat.workflow.members"
	KeyChatWorkflowNoMember = "chat.workflow.no_member"
	KeyChatWorkflowUngrouped = "chat.workflow.ungrouped"
	KeyChatWorkflowEmptyPhase = "chat.workflow.empty_phase"
	KeyChatWorkflowUnnamed  = "chat.workflow.unnamed"
	KeyChatWorkflowCopy     = "chat.workflow.copy"

	// Tool file/diff bodies.
	KeyChatToolAppliedDiff = "chat.tool.applied_diff"
	KeyChatToolProposedDiff = "chat.tool.proposed_diff"
	KeyChatToolDiffLarge   = "chat.tool.diff_large"
	KeyChatToolNoDiff      = "chat.tool.no_diff"
	KeyChatToolExpandFile  = "chat.tool.expand_file"
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
	KeyChatRetryWaiting    = "chat.retry.waiting"
	KeyChatRetryStarted    = "chat.retry.started"
	KeyChatRetryCancelled  = "chat.retry.cancelled"
	KeyChatSystemPrompt    = "chat.system_prompt"
	KeyChatSystemPromptUpdate = "chat.system_prompt_update"
	KeyChatScheduleTask    = "chat.schedule.task"
	KeyChatScheduleReceipt = "chat.schedule.receipt"
	KeyChatScheduleDeleted = "chat.schedule.deleted"
	KeyChatScheduleDetails = "chat.schedule.details"
	KeyChatImageLoading    = "chat.image.loading"
	KeyChatFeedbackEdit    = "chat.feedback.edit"
	KeyChatTrajDisplayLimited = "chat.traj.display_limited"
	KeyChatTrajDepthLimit  = "chat.traj.depth_limit"

	KeyMarkdownCode      = "markdown.code"
	KeyMarkdownCopyCode  = "markdown.copy_code"
	KeyMarkdownCopied    = "markdown.copied"
	KeyMarkdownWrap      = "markdown.wrap"
	KeyMarkdownNoWrap    = "markdown.nowrap"
	KeyMarkdownTableWide = "markdown.table_wide"
)

func init() {
	// Tool card.
	Register(KeyChatToolArguments, "参数", "Arguments")
	Register(KeyChatToolResult, "结果", "Result")
	Register(KeyChatToolEvent, "原始记录", "Raw event")
	Register(KeyChatToolInspect, "检查此调用", "Inspect this call")
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
	Register(KeyChatToolAsk, "提问", "Ask")
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
	Register(KeyChatProcessEditing, "正在修改文件", "Editing files")
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
	Register(KeyChatProcessDelegating, "正在协调子代理", "Coordinating agents")
	Register(KeyChatProcessDelegated, "已协调子代理", "Coordinated agents")
	Register(KeyChatProcessTools, "正在调用工具", "Using tools")
	Register(KeyChatProcessUsedTools, "已调用工具", "Used tools")

	// Turn status.
	Register(KeyChatTurnFailed, "运行失败", "Failed")
	Register(KeyChatTurnStopped, "已停止", "Stopped")
	Register(KeyChatTurnTruncated, "输出已截断", "Output limit reached")
	Register(KeyChatUnknownContent, "未知内容", "Unknown content")
	Register(KeyChatTurnMaxTokens, "输出达到本次请求的 token 上限，回答可能不完整。", "Output reached the request token limit; the answer may be incomplete.")
	Register(KeyChatTurnSignedOut, "已停止，因为你已退出 DeepSeek。 [ACCOUNT_SIGNED_OUT]", "Stopped because you signed out of DeepSeek. [ACCOUNT_SIGNED_OUT]")

	// Usage.
	Register(KeyChatUsageTitle, "本轮用量", "Turn usage")
	Register(KeyChatUsageTokens, "tokens", "tokens")
	Register(KeyChatUsageCache, "缓存命中", "Cache")
	Register(KeyChatUsageUncached, "未缓存输入", "Uncached input")
	Register(KeyChatUsageOutput, "输出", "Output")
	Register(KeyChatUsageTotal, "总计", "Total")
	Register(KeyChatUsageCacheRead, "缓存读取", "Cache read")
	Register(KeyChatUsageCacheWrit, "缓存写入", "Cache write")
	Register(KeyChatUsageReasoning, "其中思考", "Reasoning subset")
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
	Register(KeyChatThinkingFor, "思考了 %s", "Thought for %s")
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
	Register(KeyChatImageFailed, "图片加载失败", "Image failed")
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
	Register(KeyChatContextTitle, "上下文注入", "Injected context")
	Register(KeyChatContextRecall, "会话回忆", "Recalled context")
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
	Register(KeyChatRetryWaiting, "等待重试", "Waiting to retry")
	Register(KeyChatRetryStarted, "重试已开始", "Retry started")
	Register(KeyChatRetryCancelled, "重试已取消", "Retry cancelled")
	Register(KeyChatSystemPrompt, "系统提示词", "System prompt")
	Register(KeyChatSystemPromptUpdate, "系统提示词更新", "System prompt update")
	Register(KeyChatScheduleTask, "定时任务", "Scheduled task")
	Register(KeyChatScheduleReceipt, "创建时记录", "Creation receipt")
	Register(KeyChatScheduleDeleted, "任务已删除", "Task deleted")
	Register(KeyChatScheduleDetails, "任务详情", "Task details")
	Register(KeyChatImageLoading, "加载图片", "Loading image")
	Register(KeyChatFeedbackEdit, "编辑反馈", "Edit feedback")
	Register(KeyChatTrajDisplayLimited, "展示已限量；复制保留完整内容", "Display limited; copy retains full content")
	Register(KeyChatTrajDepthLimit, "深度限制；复制包含子树", "Depth limit; copy includes subtree")
}