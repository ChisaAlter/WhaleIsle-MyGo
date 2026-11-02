package i18n

// User-visible entrypoint copy audited against its actual status/dialog sink.
// Protocol values, external error details and locale-aware helper pairs stay at
// their owners; these entries only replace the desktop's own fixed copy.
const (
	KeyCmdComponentWindowsOnly              = "cmd.inline.component_windows_only"
	KeyCmdComponentRuntimeChanged           = "cmd.inline.component_runtime_changed"
	KeyCmdComponentConfirmationChanged      = "cmd.inline.component_confirmation_changed"
	KeyCmdComponentLifecycleBusy            = "cmd.inline.component_lifecycle_busy"
	KeyCmdRuntimeChangedRetry               = "cmd.inline.runtime_changed_retry"
	KeyCmdComponentTaskProtectionIncomplete = "cmd.inline.component_task_protection_incomplete"
	KeyCmdComponentConfirmationUnavailable  = "cmd.inline.component_confirmation_unavailable"
	KeyCmdHostSelectionBusy                 = "cmd.inline.host_selection_busy"
	KeyCmdHostExternalOverride              = "cmd.inline.host_external_override"
	KeyCmdHostSelectionChanged              = "cmd.inline.host_selection_changed"
	KeyCmdConfigStoreNotReady               = "cmd.inline.config_store_not_ready"
	KeyCmdImageSnapshotPreparing            = "cmd.inline.image_snapshot_preparing"
	KeyCmdImageSnapshotFailed               = "cmd.inline.image_snapshot_failed"
	KeyCmdImageSnapshotOpened               = "cmd.inline.image_snapshot_opened"
	KeyCmdImportPluginProgress              = "cmd.inline.import_plugin_progress"
	KeyCmdImportPluginStateChanged          = "cmd.inline.import_plugin_state_changed"
	KeyCmdImportBuildTitle                  = "cmd.inline.import_build_title"
	KeyCmdImportBuildMessage                = "cmd.inline.import_build_message"
	KeyCmdImportBuildSkip                   = "cmd.inline.import_build_skip"
	KeyCmdImportBuildApprove                = "cmd.inline.import_build_approve"
	KeyCmdImportBuildListInvalid            = "cmd.inline.import_build_list_invalid"
	KeyCmdImportBuildRepeated               = "cmd.inline.import_build_repeated"
	KeyCmdImportBuildDeclined               = "cmd.inline.import_build_declined"
	KeyCmdBridgeStarting                    = "cmd.inline.bridge_starting"
	KeyCmdBridgeDownloading                 = "cmd.inline.bridge_downloading"
	KeyCmdBridgeInstalled                   = "cmd.inline.bridge_installed"
	KeyCmdImportStartBlocked                = "cmd.inline.import_start_blocked"
	KeyCmdMediaBoundedReadFailed            = "cmd.inline.media_bounded_read_failed"
	KeyCmdOfficeTransportTooLarge           = "cmd.inline.office_transport_too_large"
	KeyCmdOfficeRetryFailure                = "cmd.inline.office_retry_failure"
	KeyCmdOfficePreparing                   = "cmd.inline.office_preparing"
	KeyCmdPDFPreparing                      = "cmd.inline.pdf_preparing"
	KeyCmdHTMLPreparing                     = "cmd.inline.html_preparing"
	KeyCmdOfficeReaderFailed                = "cmd.inline.office_reader_failed"
	KeyCmdDocumentPreviewFailed             = "cmd.inline.document_preview_failed"
	KeyCmdHTMLSnapshotOpened                = "cmd.inline.html_snapshot_opened"
	KeyCmdPDFSnapshotOpened                 = "cmd.inline.pdf_snapshot_opened"
	KeyCmdContactsOwnerChanged              = "cmd.inline.contacts_owner_changed"
	KeyCmdNotificationTarget                = "cmd.inline.notification_target"
	KeyCmdPlatformOpenFailed                = "cmd.inline.platform_open_failed"
	KeyCmdPlatformAccountChanged            = "cmd.inline.platform_account_changed"
	KeyCmdRuntimeDisconnected               = "cmd.inline.runtime_disconnected"
	KeyCmdSkillCreated                      = "cmd.inline.skill_created"
	KeyCmdSkillLoading                      = "cmd.inline.skill_loading"
	KeyCmdSkillSaved                        = "cmd.inline.skill_saved"
	KeyCmdImportThemeJSON                   = "cmd.inline.import_theme_json"
	KeyCmdThemeJSONFilter                   = "cmd.inline.theme_json_filter"
	KeyCmdHostAppearanceUnavailable         = "cmd.inline.host_appearance_unavailable"
)

func init() {
	Register(KeyCmdComponentWindowsOnly, "鲸桥组件管理仅支持 Windows x64", "WhaleBridge component management is only supported on Windows x64")
	Register(KeyCmdComponentRuntimeChanged, "运行环境已改变，请重新检查组件并确认", "The runtime changed; inspect the component and confirm again")
	Register(KeyCmdComponentConfirmationChanged, "确认期间运行环境已改变，请重新检查组件并确认", "The runtime changed during confirmation; inspect the component and confirm again")
	Register(KeyCmdComponentLifecycleBusy, "其他生命周期操作正在进行，已保留现有实例和已校验版本", "Another lifecycle action is running; the existing instance and verified version were retained")
	Register(KeyCmdRuntimeChangedRetry, "运行环境已改变，请重新操作", "The runtime changed; retry the action")
	Register(KeyCmdComponentTaskProtectionIncomplete, "操作未执行或任务保护未完成；请查看启动器提示后重试", "The action did not run or task protection did not finish; check the launcher notice and retry")
	Register(KeyCmdComponentConfirmationUnavailable, "无法显示组件操作确认；未更改组件", "Component confirmation is unavailable; the component was not changed")
	Register(KeyCmdHostSelectionBusy, "其他生命周期操作正在进行，请稍后选择 Host", "Another lifecycle action is running; select the Host later")
	Register(KeyCmdHostExternalOverride, "请先在启动设置中清空显式 dshBin/nodeBin，并移除 WHALEISLE_VENDOR 覆盖；现有外部运行时未更改", "Clear explicit dshBin/nodeBin in startup settings and remove the WHALEISLE_VENDOR override first; the external runtime was not changed")
	Register(KeyCmdHostSelectionChanged, "运行环境已改变，请重新选择 Host", "The runtime changed; select the Host again")
	Register(KeyCmdConfigStoreNotReady, "配置存储尚未就绪", "config store is not ready")
	Register(KeyCmdImageSnapshotPreparing, "正在校验并打开隔离图片快照，编辑草稿保持不变…", "Verifying and opening an isolated image snapshot; editor drafts are retained…")
	Register(KeyCmdImageSnapshotFailed, "隔离图片预览未打开：%s", "The isolated image preview did not open: %s")
	Register(KeyCmdImageSnapshotOpened, "已打开隔离图片查看器（不代表编码解码成功）；%s；无脚本/网络/应用桥接。AVIF 显示取决于当前 WebView 编解码器，源文件与草稿不变", "The isolated image viewer opened (decoding is not yet confirmed); %s; no scripts, network or application bridge. AVIF display depends on the current WebView codec; source files and drafts are retained")
	Register(KeyCmdImportPluginProgress, "正在重新安装插件 %s；来源 %s", "Reinstalling plugin %s; source %s")
	Register(KeyCmdImportPluginStateChanged, "导入状态或运行时状态已改变，插件未安装。", "The import or runtime state changed; the plugin was not installed.")
	Register(KeyCmdImportBuildTitle, "批准插件构建脚本", "Approve plugin build scripts")
	Register(KeyCmdImportBuildMessage, "插件：%s\n来源：%s\n目标：%s\n\n以下包要求在本机运行构建脚本，可能访问本机文件和网络：\n%s\n\n仅批准本次列出的脚本并重试该插件安装？", "Plugin: %s\nSource: %s\nTarget: %s\n\nThese packages request local build scripts that may access local files and the network:\n%s\n\nApprove only the scripts listed here and retry this plugin installation?")
	Register(KeyCmdImportBuildSkip, "跳过此插件", "Skip this plugin")
	Register(KeyCmdImportBuildApprove, "批准并继续", "Approve and continue")
	Register(KeyCmdImportBuildListInvalid, "安装器未返回有效的构建脚本清单；未批准。", "The installer returned no valid build-script list; approval was not granted.")
	Register(KeyCmdImportBuildRepeated, "已批准的构建脚本仍被拒绝；请检查安装日志后重试。", "The approved build scripts were rejected again; check the installation log before retrying.")
	Register(KeyCmdImportBuildDeclined, "未批准构建脚本，跳过此插件。", "Build scripts were not approved; this plugin was skipped.")
	Register(KeyCmdBridgeStarting, "正在启动鲸桥并验证就绪状态…", "Starting WhaleBridge and verifying readiness…")
	Register(KeyCmdBridgeDownloading, "正在下载并校验鲸桥组件…", "Downloading and verifying the WhaleBridge component…")
	Register(KeyCmdBridgeInstalled, "鲸桥已安装并校验，可点击启动；未自动运行", "WhaleBridge is installed and verified; click Start to run it. It was not started automatically")
	Register(KeyCmdImportStartBlocked, "导入尚未结束，暂不能启动内核。", "The import has not finished; the core cannot start yet.")
	Register(KeyCmdMediaBoundedReadFailed, "媒体预览失败；当前 Host/backend 必须支持有界字节读取。可在 Files 显式外部打开。", "Media preview failed; the current Host/backend must support bounded byte reads. Use Files to open it externally.")
	Register(KeyCmdOfficeTransportTooLarge, "转换后的 PDF 超过预览传输大小上限；请缩小文件后重试。", "The converted PDF exceeds the preview transfer limit; shrink the file and retry.")
	Register(KeyCmdOfficeRetryFailure, "Office 预览失败：%s。点击“转为 PDF 预览”可重试。", "Office preview failed: %s. Click Convert to PDF preview to retry.")
	Register(KeyCmdOfficePreparing, "正在转换 Office 文档并准备 PDF 预览…", "Converting the Office document and preparing its PDF preview…")
	Register(KeyCmdPDFPreparing, "正在准备 PDF 只读预览…", "Preparing a read-only PDF preview…")
	Register(KeyCmdHTMLPreparing, "正在打开已保存的 HTML 快照，编辑草稿保持不变…", "Opening the saved HTML snapshot; editor drafts are retained…")
	Register(KeyCmdOfficeReaderFailed, "Office 已转换，但 PDF 阅读器无法打开：%s", "Office conversion finished, but the PDF reader could not open: %s")
	Register(KeyCmdDocumentPreviewFailed, "文档预览失败，可在 Files 使用系统打开：%s", "Document preview failed; use Files to open it with the system: %s")
	Register(KeyCmdHTMLSnapshotOpened, "静态 HTML 快照已打开；%s；源文件与编辑草稿保持不变", "The static HTML snapshot opened; %s; source files and editor drafts are retained")
	Register(KeyCmdPDFSnapshotOpened, "PDF 只读预览标签已打开；原文件和编辑草稿保持不变", "The read-only PDF preview tab opened; source files and editor drafts are retained")
	Register(KeyCmdContactsOwnerChanged, "原 Host/账户已切换，操作未发送", "The original Host/account changed; the action was not sent")
	Register(KeyCmdNotificationTarget, " · 通知目标：%s", " · Notification target: %s")
	Register(KeyCmdPlatformOpenFailed, "无法打开私有 Platform 页面", "Could not open private Platform page")
	Register(KeyCmdPlatformAccountChanged, "账户已改变，请重新打开 Platform 页面", "Account changed; reopen the Platform page")
	Register(KeyCmdRuntimeDisconnected, "运行时已断开", "Runtime disconnected")
	Register(KeyCmdSkillCreated, "技能已创建：%s", "Skill created: %s")
	Register(KeyCmdSkillLoading, "正在加载技能…", "Loading skill…")
	Register(KeyCmdSkillSaved, "技能已保存：%s", "Skill saved: %s")
	Register(KeyCmdImportThemeJSON, "导入主题 JSON", "Import theme JSON")
	Register(KeyCmdThemeJSONFilter, "主题 JSON", "Theme JSON")
	Register(KeyCmdHostAppearanceUnavailable, "Host 外观设置不可用", "Host appearance is unavailable")
}

const (
	KeyCmdBotAvatarReviewChanged           = "cmd.inline.bot_avatar_review_changed"
	KeyCmdBotCapabilityReviewChanged       = "cmd.inline.bot_capability_review_changed"
	KeyCmdBotCatalogReviewChanged          = "cmd.inline.bot_catalog_review_changed"
	KeyCmdBotDeleteReviewChanged           = "cmd.inline.bot_delete_review_changed"
	KeyCmdBotDuplicateIdentityChanged      = "cmd.inline.bot_duplicate_identity_changed"
	KeyCmdCreatedBotIdentityChanged        = "cmd.inline.created_bot_identity_changed"
	KeyCmdCreatedBotOpenFailed             = "cmd.inline.created_bot_open_failed"
	KeyCmdDuplicatedBotOpenFailed          = "cmd.inline.duplicated_bot_open_failed"
	KeyCmdBotOpenBindingExpired            = "cmd.inline.bot_open_binding_expired"
	KeyCmdBotTextReviewChanged             = "cmd.inline.bot_text_review_changed"
	KeyCmdBotRouteReviewChanged            = "cmd.inline.bot_route_review_changed"
	KeyCmdBotSenderReviewChanged           = "cmd.inline.bot_sender_review_changed"
	KeyCmdBotVisibilityReviewChanged       = "cmd.inline.bot_visibility_review_changed"
	KeyCmdRoomProfileReviewChanged         = "cmd.inline.room_profile_review_changed"
	KeyCmdCreatedRoomIdentityChanged       = "cmd.inline.created_room_identity_changed"
	KeyCmdCreatedRoomOpenFailed            = "cmd.inline.created_room_open_failed"
	KeyCmdRoomIdentityChanged              = "cmd.inline.room_identity_changed"
	KeyCmdSelectBotOrRoom                  = "cmd.inline.select_bot_or_room"
	KeyCmdCanonicalSessionMissing          = "cmd.inline.canonical_session_missing"
	KeyCmdDraftRetainedNothingSent         = "cmd.inline.draft_retained_nothing_sent"
	KeyCmdBotHostUnavailable               = "cmd.inline.bot_host_unavailable"
	KeyCmdSelectRosterBotOrRoom            = "cmd.inline.select_roster_bot_or_room"
	KeyCmdBotProfileUnavailable            = "cmd.inline.bot_profile_unavailable"
	KeyCmdRoomMemberIdentityChanged        = "cmd.inline.room_member_identity_changed"
	KeyCmdRoomMemberNotRoot                = "cmd.inline.room_member_not_root"
	KeyCmdRoomMemberUnavailable            = "cmd.inline.room_member_unavailable"
	KeyCmdRoomMemberBindingExpired         = "cmd.inline.room_member_binding_expired"
	KeyCmdSharedShortcutsUnavailable       = "cmd.inline.shared_shortcuts_unavailable"
	KeyCmdReloadShortcutsRetry             = "cmd.inline.reload_shortcuts_retry"
	KeyCmdShortcutConflicts                = "cmd.inline.shortcut_conflicts"
	KeyCmdShortcutPersistenceUnavailable   = "cmd.inline.shortcut_persistence_unavailable"
	KeyCmdInvalidHTTPURL                   = "cmd.inline.invalid_h_t_t_p_u_r_l"
	KeyCmdTerminalPathUnavailable          = "cmd.inline.terminal_path_unavailable"
	KeyCmdPanelLibraryPrefix               = "cmd.inline.panel_library_prefix"
	KeyCmdPanelInstanceMissing             = "cmd.inline.panel_instance_missing"
	KeyCmdVisualRepliesHostStopped         = "cmd.inline.visual_replies_host_stopped"
	KeyCmdVisualRepliesHostMismatch        = "cmd.inline.visual_replies_host_mismatch"
	KeyCmdVisualRepliesHostChanged         = "cmd.inline.visual_replies_host_changed"
	KeyCmdProviderHostChanged              = "cmd.inline.provider_host_changed"
	KeyCmdMicrophoneSettingsUnavailable    = "cmd.inline.microphone_settings_unavailable"
	KeyCmdViewPreferencesBaselineMissing   = "cmd.inline.view_preferences_baseline_missing"
	KeyCmdWorkspaceSettingsUnavailable     = "cmd.inline.workspace_settings_unavailable"
	KeyCmdProjectDraftDiscarded            = "cmd.inline.project_draft_discarded"
	KeyCmdProjectDraftBeforeReload         = "cmd.inline.project_draft_before_reload"
	KeyCmdProjectMaterialSaved             = "cmd.inline.project_material_saved"
	KeyCmdProjectCurrentFileNotice         = "cmd.inline.project_current_file_notice"
	KeyCmdProjectFileChangedNotice         = "cmd.inline.project_file_changed_notice"
	KeyCmdProjectSelectReport              = "cmd.inline.project_select_report"
	KeyCmdProjectListChanged               = "cmd.inline.project_list_changed"
	KeyCmdProjectRefreshDelivery           = "cmd.inline.project_refresh_delivery"
	KeyCmdProjectDeliveryReferenceMismatch = "cmd.inline.project_delivery_reference_mismatch"
	KeyCmdProjectRefreshWritableMember     = "cmd.inline.project_refresh_writable_member"
	KeyCmdProjectOpenMemberHistory         = "cmd.inline.project_open_member_history"
	KeyCmdProjectDirectoryUnavailable      = "cmd.inline.project_directory_unavailable"
	KeyCmdProjectLocalDirectoryRemote      = "cmd.inline.project_local_directory_remote"
	KeyCmdWorkspaceHostUnavailable         = "cmd.inline.workspace_host_unavailable"
	KeyCmdWorkspaceHostSelectionChanged    = "cmd.inline.workspace_host_selection_changed"
	KeyCmdWorkspacePreferenceSaveFailed    = "cmd.inline.workspace_preference_save_failed"
	KeyCmdProjectDeliveryAmbiguousRuns     = "cmd.inline.project_delivery_ambiguous_runs"
	KeyCmdProjectEventAmbiguousRuns        = "cmd.inline.project_event_ambiguous_runs"
	KeyCmdProjectLifecycleChanged          = "cmd.inline.project_lifecycle_changed"
	KeyCmdProjectRecoveryUnavailable       = "cmd.inline.project_recovery_unavailable"
	KeyCmdProjectLeadDraftPending          = "cmd.inline.project_lead_draft_pending"
	KeyCmdProjectRequirementChanged        = "cmd.inline.project_requirement_changed"
	KeyCmdProjectWorkRequirementChanged    = "cmd.inline.project_work_requirement_changed"
	KeyCmdProjectWorkRequirementMismatch   = "cmd.inline.project_work_requirement_mismatch"
	KeyCmdProjectQueueUnavailable          = "cmd.inline.project_queue_unavailable"
	KeyCmdProjectRequestNotQueued          = "cmd.inline.project_request_not_queued"
	KeyCmdProjectRequestFirst              = "cmd.inline.project_request_first"
	KeyCmdProjectRequestLast               = "cmd.inline.project_request_last"
	KeyCmdProjectRequestRevision           = "cmd.inline.project_request_revision"
	KeyCmdProjectTeamRefreshPaging         = "cmd.inline.project_team_refresh_paging"
	KeyCmdProjectWorkNoPreviousPage        = "cmd.inline.project_work_no_previous_page"
	KeyCmdProjectTeamRefreshControl        = "cmd.inline.project_team_refresh_control"
	KeyCmdProjectPublishedPageInvalid      = "cmd.inline.project_published_page_invalid"
	KeyCmdProjectHTMLReaderUnavailable     = "cmd.inline.project_h_t_m_l_reader_unavailable"
	KeyCmdProjectPublishedPageClosed       = "cmd.inline.project_published_page_closed"
	KeyCmdComposerHostChanged              = "cmd.inline.composer_host_changed"
	KeyCmdComposerReplyTargetChanged       = "cmd.inline.composer_reply_target_changed"
	KeyCmdComposerFileNotRegular           = "cmd.inline.composer_file_not_regular"
	KeyCmdComposerImageCountLimit          = "cmd.inline.composer_image_count_limit"
	KeyCmdComposerImageSizeLimit           = "cmd.inline.composer_image_size_limit"
	KeyCmdComposerImageTotalLimit          = "cmd.inline.composer_image_total_limit"
	KeyCmdAttachFilesImages                = "cmd.inline.attach_files_images"
	KeyCmdComposerWaitSendFiles            = "cmd.inline.composer_wait_send_files"
	KeyCmdComposerAbsoluteFilePath         = "cmd.inline.composer_absolute_file_path"
	KeyCmdComposerRegularFileDirectory     = "cmd.inline.composer_regular_file_directory"
	KeyCmdComposerUnsafeFilePath           = "cmd.inline.composer_unsafe_file_path"
	KeyCmdComposerFinishComposition        = "cmd.inline.composer_finish_composition"
	KeyCmdComposerSelectConversation       = "cmd.inline.composer_select_conversation"
)

func init() {
	Register(KeyCmdBotAvatarReviewChanged, "已审阅的头像、资料、能力发现或权限已改变，请刷新并重新审阅", "reviewed avatar/profile, capability discovery or authority changed; refresh and review")
	Register(KeyCmdBotCapabilityReviewChanged, "资料、能力发现或写入权限在审阅后已改变", "profile, discovery or write authority changed after review")
	Register(KeyCmdBotCatalogReviewChanged, "目录身份、修订版本或写入权限已改变，请刷新并重新审阅", "catalog identities/revision or write authority changed; refresh and review again")
	Register(KeyCmdBotDeleteReviewChanged, "删除目标、目录修订版本、完整预览或权限已改变，请重新读取并确认", "deletion target, catalog revision, complete preview or authority changed; read and confirm again")
	Register(KeyCmdBotDuplicateIdentityChanged, "目录中的 Bot 身份或已查看配置已改变，请刷新并检查后再打开", "catalog Bot identity or inspected configuration changed; refresh and inspect before opening")
	Register(KeyCmdCreatedBotIdentityChanged, "Bot 已存在，但其标准会话或工作区已改变，请检查后再打开", "Bot already exists, but its canonical session/workspace changed; inspect it before opening")
	Register(KeyCmdCreatedBotOpenFailed, "Bot 已存在；打开标准会话失败：%w", "Bot already exists; canonical opening failed: %w")
	Register(KeyCmdDuplicatedBotOpenFailed, "目录记录已保留；打开标准会话失败：%w", "catalog record remains; canonical opening failed: %w")
	Register(KeyCmdBotOpenBindingExpired, "：原 Host 或会话绑定在跳转前已失效", ": original Host/session binding expired before navigation")
	Register(KeyCmdBotTextReviewChanged, "已审阅的文本、资料、能力发现或权限已改变，请刷新并重新审阅", "reviewed text/profile, capability discovery or authority changed; refresh and review")
	Register(KeyCmdBotRouteReviewChanged, "已审阅的资料、模型选项、目录、历史或权限已改变，请刷新并重新审阅", "reviewed profile, model choices, directory/history or authority changed; refresh and review")
	Register(KeyCmdBotSenderReviewChanged, "已审阅的资料、对方身份、共同房间或权限已改变，请刷新并重新审阅", "reviewed profile, peer identities, shared rooms or authority changed; refresh and review")
	Register(KeyCmdBotVisibilityReviewChanged, "完整目录、修订版本或权限已改变，请刷新并重新审阅", "complete catalog/revision or authority changed; refresh and review again")
	Register(KeyCmdRoomProfileReviewChanged, "目录修订版本、完整资料或权限已改变，请刷新并重新审阅", "catalog revision, complete profile or authority changed; refresh and review again")
	Register(KeyCmdCreatedRoomIdentityChanged, "房间已存在，但其标准会话或工作区已改变，请检查后再打开", "room already exists, but its canonical session/workspace changed; inspect it before opening")
	Register(KeyCmdCreatedRoomOpenFailed, "房间已存在；打开标准会话失败：%w", "room already exists; canonical opening failed: %w")
	Register(KeyCmdRoomIdentityChanged, "房间身份已改变，请刷新联系人目录", "Room identity changed; refresh the contact catalog")
	Register(KeyCmdSelectBotOrRoom, "请先选择 Bot 或房间", "Select a Bot or room first")
	Register(KeyCmdCanonicalSessionMissing, "Host 未返回标准会话", "Host returned no canonical session")
	Register(KeyCmdDraftRetainedNothingSent, "草稿已保留，未发送任何内容", "Draft retained; nothing sent")
	Register(KeyCmdBotHostUnavailable, "Bot Host 未连接或插件已禁用", "Bot Host is not connected or the plugin is disabled")
	Register(KeyCmdSelectRosterBotOrRoom, "请先从当前 Host 名册选择 Bot 或房间", "select a Bot or room from the current Host roster first")
	Register(KeyCmdBotProfileUnavailable, "Bot 资料已不可用，请刷新名册", "Bot profile is no longer available; refresh the roster")
	Register(KeyCmdRoomMemberIdentityChanged, "房间或成员身份已改变，请先读取当前运行状态", "room or member identity changed; read the current runtime first")
	Register(KeyCmdRoomMemberNotRoot, "成员会话不是普通根会话地址", "member session is not an ordinary root address")
	Register(KeyCmdRoomMemberUnavailable, "成员会话无法唯一匹配现有 Host 会话，请使用 Bot 网页检查", "member session is not available as a unique existing Host session; use Bot Web to inspect")
	Register(KeyCmdRoomMemberBindingExpired, "原 Host 或会话绑定在跳转前已失效", "original Host/session binding expired before navigation")
	Register(KeyCmdSharedShortcutsUnavailable, "共享快捷键设置不可用", "shared shortcut preferences unavailable")
	Register(KeyCmdReloadShortcutsRetry, "%s；请重新加载快捷键后再试", "%s; reload shortcuts before retrying")
	Register(KeyCmdShortcutConflicts, "与 %v 冲突", "Conflicts with %v")
	Register(KeyCmdShortcutPersistenceUnavailable, "快捷键持久化存储不可用", "shortcut persistence unavailable")
	Register(KeyCmdInvalidHTTPURL, "HTTP(S) 地址无效", "Invalid HTTP(S) URL")
	Register(KeyCmdTerminalPathUnavailable, "路径不可用或位于此终端的本地工作区之外", "Path is unavailable or outside this terminal's local workspace")
	Register(KeyCmdPanelLibraryPrefix, "面板库：", "Panel library: ")
	Register(KeyCmdPanelInstanceMissing, "面板实例已不存在", "Panel instance no longer exists")
	Register(KeyCmdVisualRepliesHostStopped, "HTML 可视化回复不可用：本机桌面 Host 未运行", "HTML visual replies unavailable: the local desktop Host is not running")
	Register(KeyCmdVisualRepliesHostMismatch, "HTML 可视化回复不可用：此设置连接不是当前运行的本机桌面 Host", "HTML visual replies unavailable: this settings connection is not the running local desktop Host")
	Register(KeyCmdVisualRepliesHostChanged, "HTML 可视化回复的 Host 在运行时验证期间已改变", "HTML visual replies Host changed during runtime verification")
	Register(KeyCmdProviderHostChanged, "Host 已改变，请刷新提供商配置后继续", "Host changed; refresh provider profiles before continuing")
	Register(KeyCmdMicrophoneSettingsUnavailable, "本机麦克风设置不可用", "local microphone settings are unavailable")
	Register(KeyCmdViewPreferencesBaselineMissing, "无法读取此 Host 的视图设置基线", "view preferences have no readable baseline for this Host")
	Register(KeyCmdWorkspaceSettingsUnavailable, "工作区设置不可用", "Workspace settings are unavailable")
	Register(KeyCmdProjectDraftDiscarded, "草稿已放弃；请重新加载最新保存的文本。", "Draft discarded; reload to fetch the latest saved text.")
	Register(KeyCmdProjectDraftBeforeReload, "请先放弃或保存此草稿，再重新加载。", "Discard or save this draft before reloading.")
	Register(KeyCmdProjectMaterialSaved, "已保存到项目。", "Saved to Project.")
	Register(KeyCmdProjectCurrentFileNotice, "正在打开所引用文件的当前内容，而非历史快照。", "Opening the referenced file's current content, not a historical snapshot.")
	Register(KeyCmdProjectFileChangedNotice, "文件在此报告后已改变，正在打开当前内容。", "File changed after this report; opening current content.")
	Register(KeyCmdProjectSelectReport, "请选择已记录的报告", "select a recorded report")
	Register(KeyCmdProjectListChanged, "项目已不在当前列表中，请先刷新", "Project is no longer in the current list; refresh first")
	Register(KeyCmdProjectRefreshDelivery, "请先刷新交付详情", "Refresh the delivery details first")
	Register(KeyCmdProjectDeliveryReferenceMismatch, "所选文件或工作不属于当前显示的交付引用", "Selected file or work does not belong to the displayed delivery references")
	Register(KeyCmdProjectRefreshWritableMember, "请先刷新有写入权限的原项目成员，再查看其差异", "refresh a writable original Project member before viewing its diff")
	Register(KeyCmdProjectOpenMemberHistory, "请先打开成员历史，再翻页", "open member history before paging")
	Register(KeyCmdProjectDirectoryUnavailable, "桌面工作区选择不可用", "desktop workspace selection unavailable")
	Register(KeyCmdProjectLocalDirectoryRemote, "远程模式下无法选择本地项目目录", "local Project directory selection unavailable in remote mode")
	Register(KeyCmdWorkspaceHostUnavailable, "工作区 Host 不可用", "workspace Host unavailable")
	Register(KeyCmdWorkspaceHostSelectionChanged, "工作区 Host 在选择目录期间已改变", "workspace Host changed during directory selection")
	Register(KeyCmdWorkspacePreferenceSaveFailed, "无法保存工作区设置", "workspace preference could not be saved")
	Register(KeyCmdProjectDeliveryAmbiguousRuns, "所选交付包含无法唯一确定的运行引用。", "The selected delivery contains ambiguous run references.")
	Register(KeyCmdProjectEventAmbiguousRuns, "所选事件包含无法唯一确定的运行引用。", "The selected event contains ambiguous run references.")
	Register(KeyCmdProjectLifecycleChanged, "项目生命周期已改变，请刷新后再归档", "Project lifecycle changed; refresh before archiving")
	Register(KeyCmdProjectRecoveryUnavailable, "恢复请求不可用", "recovery request is unavailable")
	Register(KeyCmdProjectLeadDraftPending, "Lead 存在草稿或待发送内容，请先完成该草稿", "Lead has a draft or pending send; finish that draft first")
	Register(KeyCmdProjectRequirementChanged, "需求已改变，请先刷新任务详情", "Requirement changed; refresh task details first")
	Register(KeyCmdProjectWorkRequirementChanged, "工作已不属于当前显示的需求", "Work no longer belongs to the displayed requirement")
	Register(KeyCmdProjectWorkRequirementMismatch, "工作对应的需求身份不匹配", "Work requirement identity does not match")
	Register(KeyCmdProjectQueueUnavailable, "此 Host 的项目请求队列不可用", "Project request queue is unavailable on this Host")
	Register(KeyCmdProjectRequestNotQueued, "需求已不在队列中，请先刷新", "Requirement is no longer queued; refresh first")
	Register(KeyCmdProjectRequestFirst, "需求已位于队首", "Requirement is already first")
	Register(KeyCmdProjectRequestLast, "需求已位于队尾", "Requirement is already last")
	Register(KeyCmdProjectRequestRevision, "请先刷新当前需求的修订版本", "Refresh the current requirement revision first")
	Register(KeyCmdProjectTeamRefreshPaging, "请先刷新团队，再翻页", "refresh Team before paging")
	Register(KeyCmdProjectWorkNoPreviousPage, "当前工作已无上一页", "no previous current-work page")
	Register(KeyCmdProjectTeamRefreshControl, "请先刷新已就绪的项目团队，再控制工作", "refresh a ready Project Team before controlling work")
	Register(KeyCmdProjectPublishedPageInvalid, "已发布项目页面的身份或高度无效", "published Project page has an invalid identity or height")
	Register(KeyCmdProjectHTMLReaderUnavailable, "已发布 HTML 阅读器不可用：%s", "published HTML reader is unavailable: %s")
	Register(KeyCmdProjectPublishedPageClosed, "隔离的已发布页面已关闭", "isolated published page closed")
	Register(KeyCmdComposerHostChanged, "Host 已改变，草稿已保留", "Host changed; draft retained")
	Register(KeyCmdComposerReplyTargetChanged, "回复目标已改变，请重新打开房间", "Reply target changed; reopen the room")
	Register(KeyCmdComposerFileNotRegular, "不是普通文件：%s", "not a regular file: %s")
	Register(KeyCmdComposerImageCountLimit, "图片数量超过 Host 上限 %d", "image count exceeds Host limit %d")
	Register(KeyCmdComposerImageSizeLimit, "图片超过 Host 单张大小上限 %d 字节", "image exceeds Host per-image limit %d bytes")
	Register(KeyCmdComposerImageTotalLimit, "图片总大小超过 Host 上限 %d 字节", "images exceed Host total limit %d bytes")
	Register(KeyCmdAttachFilesImages, "添加文件或图片", "Attach files or images")
	Register(KeyCmdComposerWaitSendFiles, "请等待当前发送完成，再添加文件", "wait for the current send to finish before adding files")
	Register(KeyCmdComposerAbsoluteFilePath, "本机文件路径不是绝对路径", "native file path is not absolute")
	Register(KeyCmdComposerRegularFileDirectory, "只能引用普通文件和目录", "only regular files and directories can be referenced")
	Register(KeyCmdComposerUnsafeFilePath, "无法安全插入此文件路径", "this file path cannot be inserted safely")
	Register(KeyCmdComposerFinishComposition, "请完成文本输入，再添加文件", "finish text composition before adding files")
	Register(KeyCmdComposerSelectConversation, "请先选择会话，再向其草稿添加文本", "select a conversation before adding text to its draft")
}

const (
	KeyCmdDesktopInvalidPort              = "cmd.inline.desktop_invalid_port"
	KeyCmdRestartPolicyNotSaved           = "cmd.inline.restart_policy_not_saved"
	KeyCmdSelectCurrentRosterBot          = "cmd.inline.select_current_roster_bot"
	KeyCmdURLUnavailable                  = "cmd.inline.u_r_l_unavailable"
	KeyCmdMarketExportErrorTitle          = "cmd.inline.market_export_error_title"
	KeyCmdUsageExportErrorTitle           = "cmd.inline.usage_export_error_title"
	KeyCmdUsageExportStatistics           = "cmd.inline.usage_export_statistics"
	KeyCmdActivityEntryCount              = "cmd.inline.activity_entry_count"
	KeyCmdWorkspaceRenameIDMissing        = "cmd.inline.workspace_rename_i_d_missing"
	KeyCmdWorkspaceDeleteIDMissing        = "cmd.inline.workspace_delete_i_d_missing"
	KeyCmdRemoteConfigUnavailable         = "cmd.inline.remote_config_unavailable"
	KeyCmdTaskConfirmationWindowMissing   = "cmd.inline.task_confirmation_window_missing"
	KeyCmdTaskCoverageUnknown             = "cmd.inline.task_coverage_unknown"
	KeyCmdImportBeforeQuit                = "cmd.inline.import_before_quit"
	KeyCmdRuntimeChangedRetryQuit         = "cmd.inline.runtime_changed_retry_quit"
	KeyCmdRuntimeChangedRetryStop         = "cmd.inline.runtime_changed_retry_stop"
	KeyCmdRuntimeChangedRetryUpdate       = "cmd.inline.runtime_changed_retry_update"
	KeyCmdRuntimeChangedRetryUninstall    = "cmd.inline.runtime_changed_retry_uninstall"
	KeyCmdRuntimeChangedRetryRestart      = "cmd.inline.runtime_changed_retry_restart"
	KeyCmdPluginPreferencesUnavailable    = "cmd.inline.plugin_preferences_unavailable"
	KeyCmdWorkbookReaderFormats           = "cmd.inline.workbook_reader_formats"
	KeyCmdDocumentViewerOwnerChanged      = "cmd.inline.document_viewer_owner_changed"
	KeyCmdPhoneRuntimeBeforeDevices       = "cmd.inline.phone_runtime_before_devices"
	KeyCmdLegacyRelayOnlyClear            = "cmd.inline.legacy_relay_only_clear"
	KeyCmdPhonePRDraft                    = "cmd.inline.phone_p_r_draft"
	KeyCmdStartupConfigUnavailable        = "cmd.inline.startup_config_unavailable"
	KeyCmdStartupPluginInventoryChanged   = "cmd.inline.startup_plugin_inventory_changed"
	KeyCmdStartupRuntimeChanged           = "cmd.inline.startup_runtime_changed"
	KeyCmdLocalPreviewExactFile           = "cmd.inline.local_preview_exact_file"
	KeyCmdLocalPreviewOutsideWorkspace    = "cmd.inline.local_preview_outside_workspace"
	KeyCmdLocalPreviewRegularFile         = "cmd.inline.local_preview_regular_file"
	KeyCmdLocalPreviewSourceChanged       = "cmd.inline.local_preview_source_changed"
	KeyCmdLocalPreviewPDFSource           = "cmd.inline.local_preview_p_d_f_source"
	KeyCmdLocalPreviewHTMLSource          = "cmd.inline.local_preview_h_t_m_l_source"
	KeyCmdLocalPreviewMediaSource         = "cmd.inline.local_preview_media_source"
	KeyCmdLocalPreviewUnsupported         = "cmd.inline.local_preview_unsupported"
	KeyCmdLocalPreviewRegularMedia        = "cmd.inline.local_preview_regular_media"
	KeyCmdLocalPreviewMediaLimit          = "cmd.inline.local_preview_media_limit"
	KeyCmdLocalPreviewMediaBeforeRead     = "cmd.inline.local_preview_media_before_read"
	KeyCmdLocalPreviewMediaDuringRead     = "cmd.inline.local_preview_media_during_read"
	KeyCmdLocalPreviewWorkspaceDuringRead = "cmd.inline.local_preview_workspace_during_read"
	KeyCmdLocalPreviewMediaReplaced       = "cmd.inline.local_preview_media_replaced"
)

func init() {
	Register(KeyCmdDesktopInvalidPort, "端口无效（1024–65535）", "invalid port (1024-65535)")
	Register(KeyCmdRestartPolicyNotSaved, "Harness 重启策略未保存，草稿已保留", "Harness restart policy was not saved; draft retained")
	Register(KeyCmdSelectCurrentRosterBot, "请先从当前 Host 名册选择 Bot", "Select a Bot from the current Host roster first")
	Register(KeyCmdURLUnavailable, "地址不可用", "URL unavailable")
	Register(KeyCmdMarketExportErrorTitle, "市场导出", "Market export")
	Register(KeyCmdUsageExportErrorTitle, "用量导出", "Usage export")
	Register(KeyCmdUsageExportStatistics, "导出用量统计", "Export usage statistics")
	Register(KeyCmdActivityEntryCount, "%d 条记录", "%d entries")
	Register(KeyCmdWorkspaceRenameIDMissing, "无法重命名工作区：缺少工作区 ID", "workspace rename: no workspace id")
	Register(KeyCmdWorkspaceDeleteIDMissing, "无法删除工作区：缺少工作区 ID", "workspace delete: no workspace id")
	Register(KeyCmdRemoteConfigUnavailable, "远程配置不可用", "remote configuration unavailable")
	Register(KeyCmdTaskConfirmationWindowMissing, "没有可用于任务确认的桌面窗口", "no desktop window for the task confirmation")
	Register(KeyCmdTaskCoverageUnknown, "：未知", ": unknown")
	Register(KeyCmdImportBeforeQuit, "导入仍在进行，请完成或取消后再退出", "import is still running; finish or cancel it before quitting")
	Register(KeyCmdRuntimeChangedRetryQuit, "运行时或导入状态已改变，请重新退出", "runtime or import state changed; retry quit")
	Register(KeyCmdRuntimeChangedRetryStop, "运行时或导入状态已改变，请重新停止", "runtime or import state changed; retry stop")
	Register(KeyCmdRuntimeChangedRetryUpdate, "运行时或导入状态已改变，请重新更新", "runtime or import state changed; retry update")
	Register(KeyCmdRuntimeChangedRetryUninstall, "运行时或导入状态已改变，请重新卸载", "runtime or import state changed; retry uninstall")
	Register(KeyCmdRuntimeChangedRetryRestart, "运行时或导入状态已改变，请重新启动", "runtime or import state changed; retry restart")
	Register(KeyCmdPluginPreferencesUnavailable, "本机插件设置不可用", "local plugin preferences are unavailable")
	Register(KeyCmdWorkbookReaderFormats, "本机工作簿阅读器支持 XLSX、XLS、CSV 和 TSV 文件", "Native workbook reader supports XLSX, XLS, CSV and TSV files")
	Register(KeyCmdDocumentViewerOwnerChanged, "文档查看器的 Host 或会话已改变", "document viewer Host or Session changed")
	Register(KeyCmdPhoneRuntimeBeforeDevices, "请先启用并刷新手机运行时，再管理设备", "enable and refresh the phone runtime before managing devices")
	Register(KeyCmdLegacyRelayOnlyClear, "旧版中继凭据仅支持清除", "legacy relay credentials can only be cleared")
	Register(KeyCmdPhonePRDraft, "草稿：%t", "Draft: %t")
	Register(KeyCmdStartupConfigUnavailable, "启动配置不可用。", "Startup configuration is unavailable.")
	Register(KeyCmdStartupPluginInventoryChanged, "疑似问题插件或插件库存已改变，请刷新诊断", "suspects or plugin inventory changed; refresh diagnostics")
	Register(KeyCmdStartupRuntimeChanged, "运行时或导入状态已改变，请刷新诊断后重试", "runtime or import changed; retry from fresh diagnostics")
	Register(KeyCmdLocalPreviewExactFile, "预览需要明确的本地工作区文件", "preview requires an exact local workspace file")
	Register(KeyCmdLocalPreviewOutsideWorkspace, "预览文件位于会话工作区之外", "preview is outside the session workspace")
	Register(KeyCmdLocalPreviewRegularFile, "预览源不是普通工作区文件", "preview source is not a regular workspace file")
	Register(KeyCmdLocalPreviewSourceChanged, "预览源在打开前已改变", "preview source changed before opening")
	Register(KeyCmdLocalPreviewPDFSource, "PDF 预览需要 PDF 源文件", "PDF preview requires a PDF source")
	Register(KeyCmdLocalPreviewHTMLSource, "HTML 预览需要 HTML 源文件", "HTML preview requires an HTML source")
	Register(KeyCmdLocalPreviewMediaSource, "不支持此媒体源", "unsupported media source")
	Register(KeyCmdLocalPreviewUnsupported, "不支持此本地交付文档预览", "unsupported local delivery document preview")
	Register(KeyCmdLocalPreviewRegularMedia, "媒体不是普通工作区文件", "media is not a regular workspace file")
	Register(KeyCmdLocalPreviewMediaLimit, "媒体预览上限为 32 MiB", "media preview limit is 32 MiB")
	Register(KeyCmdLocalPreviewMediaBeforeRead, "媒体源在读取前已改变", "media source changed before reading")
	Register(KeyCmdLocalPreviewMediaDuringRead, "媒体源在读取期间已改变", "media source changed while reading")
	Register(KeyCmdLocalPreviewWorkspaceDuringRead, "媒体工作区在读取期间已改变", "media workspace changed while reading")
	Register(KeyCmdLocalPreviewMediaReplaced, "媒体源在读取期间被替换", "media source was replaced while reading")
}

const (
	KeyCmdSSHTunnelSaved                = "cmd.inline.s_s_h_tunnel_saved"
	KeyCmdWorkspaceOpenUnavailable      = "cmd.inline.workspace_open_unavailable"
	KeyCmdStartupRecoveryUnavailable    = "cmd.inline.startup_recovery_unavailable"
	KeyCmdDesktopPreferencesUnavailable = "cmd.inline.desktop_preferences_unavailable"
	KeyCmdResumeStorageUnavailable      = "cmd.inline.resume_storage_unavailable"
	KeyCmdResumeSnapshotLimit           = "cmd.inline.resume_snapshot_limit"
	KeyCmdPublishedHTMLSessionChanged   = "cmd.inline.published_h_t_m_l_session_changed"
	KeyCmdIsolatedPageClosed            = "cmd.inline.isolated_page_closed"
	KeyCmdLocalPreviewPermissionEnded   = "cmd.inline.local_preview_permission_ended"
	KeyCmdPetLookApprovalDetails        = "cmd.inline.pet_look_approval_details"
)

func init() {
	Register(KeyCmdSSHTunnelSaved, "已保存 ssh-tunnel.json", "saved ssh-tunnel.json")
	Register(KeyCmdWorkspaceOpenUnavailable, "无法打开工作区路径", "workspace path open unavailable")
	Register(KeyCmdStartupRecoveryUnavailable, "启动恢复存储不可用", "startup recovery storage is unavailable")
	Register(KeyCmdDesktopPreferencesUnavailable, "桌面设置不可用", "desktop preferences are unavailable")
	Register(KeyCmdResumeStorageUnavailable, "恢复存储不可用", "resume storage is not available")
	Register(KeyCmdResumeSnapshotLimit, "恢复快照超过 %d MiB，未丢弃任何内容", "resume snapshot exceeds %d MiB; nothing was discarded")
	Register(KeyCmdPublishedHTMLSessionChanged, "已发布 HTML 的会话已改变", "published HTML session changed")
	Register(KeyCmdIsolatedPageClosed, "隔离页面已关闭", "isolated page closed")
	Register(KeyCmdLocalPreviewPermissionEnded, "本地交付预览权限已结束，请重新打开原交付卡片", "local delivery preview permission ended; reopen the original delivery card")
	Register(KeyCmdPetLookApprovalDetails, "提供商：%s\n模型：%s\nHost：%s", "Provider: %s\nModel: %s\nHost: %s")
}

const (
	KeyCmdWhaleVisionModelMissing = "cmd.inline.whale_vision_model_missing"
	KeyCmdWhaleLookImageLimit     = "cmd.inline.whale_look_image_limit"
	KeyCmdCreatorResultMetadata   = "cmd.inline.creator_result_metadata"
)

func init() {
	Register(KeyCmdWhaleVisionModelMissing, "尚未选择鲸鱼模型（请打开鲸鱼 WebView 并选择视觉模型）", "no whale model selected (open whale WebView and pick a vision model)")
	Register(KeyCmdWhaleLookImageLimit, "图片为空或超过 4 MiB", "image empty or larger than 4MiB")
	Register(KeyCmdCreatorResultMetadata, "Host：%s\n工作区：%s\n会话：%s\n预设：cordis · %s", "Host: %s\nWorkspace: %s\nSession: %s\nPreset: cordis · %s")
}
