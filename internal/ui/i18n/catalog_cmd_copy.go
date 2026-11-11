package i18n

// Copy migrated in the round 2026-10-26 cmd/ inline-text audit. Every key here
// replaces a Chinese literal (or an ad-hoc zh/en helper pair) in cmd/whaleisle
// with a catalog entry, so the desktop entrypoint no longer owns a third copy
// of the zh/en strings. The paired test is cmd_inline_copy_audit_test.go.
//
// Scope: 45 user-visible messages only — confirmation dialogs, plugin/office/
// media preview status, empty states and onboarding-style errors. Internal
// logs, protocol constants and developer diagnostics intentionally stay inline
// and are listed with reasons in docs/parity/cmd-inline-text-audit-20261026.md.
const (
	// Git push review dialog.
	KeyCmdConfirmPushDestination  = "cmd.confirm_push_destination"
	KeyCmdConfirmPush             = "cmd.confirm_push"
	KeyCmdPushReviewCommitted     = "cmd.push_review_committed"
	KeyCmdPushReviewDefaultBranch = "cmd.push_review_default_branch"

	// Session plugin dialog.
	KeyCmdSessionPluginTitle        = "cmd.session_plugin_title"
	KeyCmdSessionPluginHint         = "cmd.session_plugin_hint"
	KeyCmdSessionContentUnavailable = "cmd.session_content_unavailable"
	KeyCmdConnectingSessionContent  = "cmd.connecting_session_content"
	KeyCmdSessionOwnerUnavailable   = "cmd.session_owner_unavailable"
	KeyCmdNoSessionPluginContent    = "cmd.no_session_plugin_content"
	KeyCmdSessionPluginContent      = "cmd.session_plugin_content"

	// Plugin Creator panel.
	KeyCmdCreatorNeedsHost         = "cmd.creator_needs_host"
	KeyCmdCreatorUnavailable       = "cmd.creator_unavailable"
	KeyCmdCreatorChecking          = "cmd.creator_checking"
	KeyCmdCreatorPresetsUnreadable = "cmd.creator_presets_unreadable"
	KeyCmdCreatorResult            = "cmd.creator_result"
	KeyCmdCreatorOpenExisting      = "cmd.creator_open_existing"

	// Office document preview.
	KeyCmdOfficePreviewReady         = "cmd.office_preview_ready"
	KeyCmdOfficePreviewMissingFonts  = "cmd.office_preview_missing_fonts"
	KeyCmdOfficePreviewUnavailable   = "cmd.office_preview_unavailable"
	KeyCmdOfficePreviewCancelled     = "cmd.office_preview_cancelled"
	KeyCmdOfficePreviewTooLarge      = "cmd.office_preview_too_large"
	KeyCmdOfficePreviewInvalid       = "cmd.office_preview_invalid"
	KeyCmdOfficePreviewTimeout       = "cmd.office_preview_timeout"
	KeyCmdOfficePreviewBusy          = "cmd.office_preview_busy"
	KeyCmdOfficePreviewSourceChanged = "cmd.office_preview_source_changed"
	KeyCmdOfficePreviewFailed        = "cmd.office_preview_failed"

	// Workspace document preview errors.
	KeyCmdPreviewSwitchSessionWorkspace = "cmd.preview_switch_session_workspace"
	KeyCmdPreviewSourceClosed           = "cmd.preview_source_closed"
	KeyCmdPreviewSessionChanged         = "cmd.preview_session_changed"
	KeyCmdPreviewChildUnsupported       = "cmd.preview_child_unsupported"
	KeyCmdPreviewRestoreUnsupported     = "cmd.preview_restore_unsupported"

	// Audio/video preview status.
	KeyCmdMediaPathRequired        = "cmd.media_path_required"
	KeyCmdMediaHostServiceRequired = "cmd.media_host_service_required"
	KeyCmdMediaReadingSnapshot     = "cmd.media_reading_snapshot"
	KeyCmdMediaPreviewUnavailable  = "cmd.media_preview_unavailable"
	KeyCmdMediaSnapshotOpened      = "cmd.media_snapshot_opened"

	// Image preview.
	KeyCmdImagePreviewUnavailable = "cmd.image_preview_unavailable"

	// Import flow.
	KeyCmdImportBusy            = "cmd.import_busy"
	KeyCmdImportScanFirst       = "cmd.import_scan_first"
	KeyCmdImportNothingSelected = "cmd.import_nothing_selected"
	KeyCmdImportRestartBlocked  = "cmd.import_restart_blocked"

	// Native pickers and plugin contribution surfaces.
	KeyCmdChooseProjectDirectory = "cmd.choose_project_directory"
	KeyCmdPluginContentOpen      = "cmd.plugin_content_open"
	KeyCmdPluginContentCollapse  = "cmd.plugin_content_collapse"
)

func init() {
	Register(KeyCmdConfirmPushDestination, "确认推送目标", "Confirm push destination")
	Register(KeyCmdConfirmPush, "确认推送", "Confirm push")
	Register(KeyCmdPushReviewCommitted, "将普通推送已提交内容，不会暂存或提交编辑器草稿。", "This pushes committed history without staging files or committing editor drafts.")
	Register(KeyCmdPushReviewDefaultBranch, "目标是默认分支。仅在确认要更新它时继续。", "The destination is the default branch. Continue only if you intend to update it.")

	Register(KeyCmdSessionPluginTitle, "会话插件", "Session plugins")
	Register(KeyCmdSessionPluginHint, "局部内容使用原会话上下文；输入区由原页面完整接管，原生草稿保持不变。", "Local content uses the original Session owner. Its complete composer keeps a separate draft.")
	Register(KeyCmdSessionContentUnavailable, "会话内容不可用。", "Session content unavailable.")
	Register(KeyCmdConnectingSessionContent, "正在连接会话内容…", "Connecting Session content…")
	Register(KeyCmdSessionOwnerUnavailable, "真实会话 owner 尚不可用。", "The original Session owner is unavailable.")
	Register(KeyCmdNoSessionPluginContent, "当前会话没有可显示的插件内容。", "No Session plugin content is available.")
	Register(KeyCmdSessionPluginContent, "会话插件内容", "Session plugin content")

	Register(KeyCmdCreatorNeedsHost, "连接 Host 后可创建插件", "Connect to the Host to create plugins")
	Register(KeyCmdCreatorUnavailable, "当前 Host 的 Creator 不可用", "Creator is unavailable on this Host")
	Register(KeyCmdCreatorChecking, "正在检查 Creator…", "Checking Creator…")
	Register(KeyCmdCreatorPresetsUnreadable, "无法读取 Creator 预设；请刷新预设列表", "Could not read Creator presets; refresh the preset list")
	Register(KeyCmdCreatorResult, "Creator 创建结果", "Creator result")
	Register(KeyCmdCreatorOpenExisting, "打开已有任务", "Open existing task")

	Register(KeyCmdOfficePreviewReady, "Office 转换完成，PDF 只读预览已打开。", "Office conversion finished; the read-only PDF preview is open.")
	Register(KeyCmdOfficePreviewMissingFonts, "缺失字体：%s。本次预览无法使用这些字体，文字和排版可能与原文档不同。", "Missing fonts: %s. This preview cannot use them, so text and layout may differ from the original document.")
	Register(KeyCmdOfficePreviewUnavailable, "Office 预览不可用。请在运行 鲸屿 MyGo Host 的主机上启用文档预览服务，并确认 LibreOffice Kit 运行依赖可用；然后点击“转为 PDF 预览”重试。", "Office preview is unavailable. Enable the document preview service on the host running Whale Isle MyGo, confirm the LibreOffice Kit runtime is available, then retry “Preview as PDF”.")
	Register(KeyCmdOfficePreviewCancelled, "Office 预览已取消；点击“转为 PDF 预览”可重试。", "Office preview was cancelled; retry with “Preview as PDF”.")
	Register(KeyCmdOfficePreviewTooLarge, "Office 文件或转换后的 PDF 超过预览大小上限；请缩小文件后重试。", "The Office file or the converted PDF exceeds the preview size limit; shrink the file and retry.")
	Register(KeyCmdOfficePreviewInvalid, "无法预览此 Office 文件。文件可能已损坏、受密码保护，或与扩展名不符。", "This Office file cannot be previewed. It may be damaged, password-protected, or not match its extension.")
	Register(KeyCmdOfficePreviewTimeout, "Office 转换超时；点击“转为 PDF 预览”可重试。", "Office conversion timed out; retry with “Preview as PDF”.")
	Register(KeyCmdOfficePreviewBusy, "Office 预览任务较多；请稍后点击“转为 PDF 预览”重试。", "Too many Office previews are running; retry with “Preview as PDF” later.")
	Register(KeyCmdOfficePreviewSourceChanged, "Office 文件在读取时已更改；请重新打开预览。", "The Office file changed while it was read; reopen the preview.")
	Register(KeyCmdOfficePreviewFailed, "Office 转换失败，未生成可用的 PDF；请检查文件后点击“转为 PDF 预览”重试。", "Office conversion failed and produced no usable PDF; check the file and retry with “Preview as PDF”.")

	Register(KeyCmdPreviewSwitchSessionWorkspace, "请切回文档所属会话和工作区", "Switch back to the session and workspace that own this document")
	Register(KeyCmdPreviewSourceClosed, "来源代理面板已关闭，请重新打开来源", "The source agent panel is closed; reopen the source")
	Register(KeyCmdPreviewSessionChanged, "文档所属会话已改变", "The document's session changed")
	Register(KeyCmdPreviewChildUnsupported, "子会话文档入口当前只提供 PDF 和 HTML", "The child-session document entry currently supports only PDF and HTML")
	Register(KeyCmdPreviewRestoreUnsupported, "此预览类型尚不支持从源文件恢复", "This preview type cannot be restored from the source file yet")

	Register(KeyCmdMediaPathRequired, "音视频预览需要当前会话工作区和明确支持的单文件路径；不接受 URL 或路径遍历。", "Audio/video preview needs the current session workspace and one supported file path; URLs and path traversal are rejected.")
	Register(KeyCmdMediaHostServiceRequired, "媒体预览需要当前会话的 Host 文件读取服务；不会把 Host 路径当成本机路径打开。", "Media preview needs the current session's Host file-read service; Host paths are never opened as local paths.")
	Register(KeyCmdMediaReadingSnapshot, "正在读取当前 Host/会话的单个音视频快照（最大 32 MiB）…", "Reading one audio/video snapshot from the current Host/session (up to 32 MiB)…")
	Register(KeyCmdMediaPreviewUnavailable, "媒体预览不可用，可在 Files 显式外部打开：%s", "Media preview is unavailable; open it explicitly outside from Files: %s")
	Register(KeyCmdMediaSnapshotOpened, "只读音视频快照已打开。请点击播放器播放；编解码支持取决于 WebView2，30 分钟后过期。", "The read-only audio/video snapshot is open. Use the player to start playback; codec support depends on WebView2 and the snapshot expires after 30 minutes.")

	Register(KeyCmdImagePreviewUnavailable, "当前系统的隔离图片 WebView 不可用；图片未在兼容预览中显示", "The isolated image WebView is unavailable on this system; the image was not shown in the compatible preview")

	Register(KeyCmdImportBusy, "插件操作或运行时切换尚未结束，请稍后再导入。", "Plugin work or a runtime switch is still running; import later.")
	Register(KeyCmdImportScanFirst, "请先扫描导入来源。", "Scan the import source first.")
	Register(KeyCmdImportNothingSelected, "未选择任何项，未写入。", "Nothing was selected; nothing was written.")
	Register(KeyCmdImportRestartBlocked, "导入尚未结束，暂不能重启内核。", "The import has not finished; the core cannot restart yet.")

	Register(KeyCmdChooseProjectDirectory, "选择 Project 工作目录", "Choose Project workspace folder")
	Register(KeyCmdPluginContentOpen, "打开插件内容", "Open plugin content")
	Register(KeyCmdPluginContentCollapse, "收起插件内容", "Collapse plugin content")
}
