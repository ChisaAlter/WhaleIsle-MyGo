package i18n

// Keys for copy that originates below the UI layer: store error and status
// text plus desktop runtime messages. Registered from init so the shared
// catalog stays the single source of truth for the active locale.
const (

	// cmd
	KeyCmdChooseLocalAvatarImage               = "cmd.choose_local_avatar_image"
	KeyCmdBotCreatedSessionConnectionUnavailab = "cmd.bot_created_session_connection_unavailab"
	KeyCmdCreatedBotOpenCanonicalSession       = "cmd.created_bot_open_canonical_session"
	KeyCmdCatalogRecordRetainedSessionConnecti = "cmd.catalog_record_retained_session_connecti"
	KeyCmdOpenInspectedCatalogBot              = "cmd.open_inspected_catalog_bot"
	KeyCmdWatchingThisBotPageSCompleteCatalo   = "cmd.watching_this_bot_page_s_complete_catalo"
	KeyCmdSystemNotificationsUnavailableOrNot  = "cmd.system_notifications_unavailable_or_not"
	KeyCmdCouldNotRestoreTheNotificationTarge  = "cmd.could_not_restore_the_notification_targe"
	KeyCmdNotificationTargetVerifyInVisibility = "cmd.notification_target_verify_in_visibility"
	KeyCmdSystemNotificationDeliveryFailedChec = "cmd.system_notification_delivery_failed_chec"
	KeyCmdConversationOpenedReadMarkerFailed   = "cmd.conversation_opened_read_marker_failed"
	KeyCmdOpeningCanonicalConversation         = "cmd.opening_canonical_conversation"
	KeyCmdResolveTheExistingChatDraftBeforeS   = "cmd.resolve_the_existing_chat_draft_before_s"
	KeyCmdCanonicalConversationOpened          = "cmd.canonical_conversation_opened"
	KeyCmdNeedsAttention                       = "cmd.needs_attention"
	KeyCmdCreatedRoomOpenCanonicalSession      = "cmd.created_room_open_canonical_session"
	KeyCmdUnsupportedTriggerAddressNoCredentia = "cmd.unsupported_trigger_address_no_credentia"
	KeyCmdNoImageOnTheClipboard                = "cmd.no_image_on_the_clipboard"
	KeyCmdEnterAReplyOrAddAnAttachment         = "cmd.enter_a_reply_or_add_an_attachment"
	KeyCmdHostOffline                          = "cmd.host_offline"
	KeyCmdNoCompletedBotJsonReply              = "cmd.no_completed_bot_json_reply"
	KeyCmdResolveTheExistingChatDraftFirst     = "cmd.resolve_the_existing_chat_draft_first"
	KeyCmdDraftPreparedReviewAndSend           = "cmd.draft_prepared_review_and_send"
	KeyCmdImportedForPreviewSaveToApply        = "cmd.imported_for_preview_save_to_apply"
	KeyCmdSavedOtherBotCopiesRemainIndependen  = "cmd.saved_other_bot_copies_remain_independen"
	KeyCmdIndependentCopyAdded                 = "cmd.independent_copy_added"
	KeyCmdTemplateUpdatedForFutureCopies       = "cmd.template_updated_for_future_copies"
	KeyCmdRemovedFromThisBotTemplateRetained   = "cmd.removed_from_this_bot_template_retained"
	KeyCmdExportInstallationHistory            = "cmd.export_installation_history"
	KeyCmdPhoneRemoteSettingWasNotSavedRetry   = "cmd.phone_remote_setting_was_not_saved_retry"
	KeyCmdWhaleAssistantIsDisabled             = "cmd.whale_assistant_is_disabled"
	KeyCmdHostOffline2                         = "cmd.host_offline_2"
	KeyCmdConnectionChanged                    = "cmd.connection_changed"
	KeyCmdConversationChanged                  = "cmd.conversation_changed"
	KeyCmdPetChatUnavailable                   = "cmd.pet_chat_unavailable"
	KeyCmdPetChatOpened                        = "cmd.pet_chat_opened"
	KeyCmdRuntimeChanged                       = "cmd.runtime_changed"
	KeyCmdCancelled                            = "cmd.cancelled"
	KeyCmdScanUnavailableSavedFoodRetained     = "cmd.scan_unavailable_saved_food_retained"
	KeyCmdLocalFoodRefreshed                   = "cmd.local_food_refreshed"
	KeyCmdConnectHostAndEnableWhaleFirst       = "cmd.connect_host_and_enable_whale_first"
	KeyCmdSelectThePetLookModelAndRefreshIt    = "cmd.select_the_pet_look_model_and_refresh_it"
	KeyCmdLookAtScreen                         = "cmd.look_at_screen"
	KeyCmdCancel                               = "cmd.cancel"
	KeyCmdCaptureAndSend                       = "cmd.capture_and_send"
	KeyCmdCancelledNoScreenshotTaken           = "cmd.cancelled_no_screenshot_taken"
	KeyCmdWindowConversationOrVisionRouteChan  = "cmd.window_conversation_or_vision_route_chan"
	KeyCmdContextChangedScreenshotWasNotSent   = "cmd.context_changed_screenshot_was_not_sent"
	KeyCmdContextChanged                       = "cmd.context_changed"
	KeyCmdLookFailedCheckTheVisionModel        = "cmd.look_failed_check_the_vision_model"
	KeyCmdHostOfflineReopenAfterConnecting     = "cmd.host_offline_reopen_after_connecting"
	KeyCmdGatewaySettingsSavedApplying         = "cmd.gateway_settings_saved_applying"
	KeyCmdPhonePrReview                        = "cmd.phone_pr_review"
	KeyCmdWorking                              = "cmd.working"
	KeyCmdPhoneRequestedAPrReviewAndConfirm    = "cmd.phone_requested_a_pr_review_and_confirm"
	KeyCmdCommitPushMayAlreadyBeCompleteCanc   = "cmd.commit_push_may_already_be_complete_canc"
	KeyCmdPrTitle                              = "cmd.pr_title"
	KeyCmdPrBody                               = "cmd.pr_body"
	KeyCmdCreateDraftPr                        = "cmd.create_draft_pr"
	KeyCmdGenerateCandidate                    = "cmd.generate_candidate"
	KeyCmdCandidateNotYetApplied               = "cmd.candidate_not_yet_applied"
	KeyCmdApplyCandidate                       = "cmd.apply_candidate"
	KeyCmdPrepareTargetReview                  = "cmd.prepare_target_review"
	KeyCmdReviewTheExactContentBelow           = "cmd.review_the_exact_content_below"
	KeyCmdConfirmThisPr                        = "cmd.confirm_this_pr"
	KeyCmdOutcomeUncertainCheckExistingPrBefo  = "cmd.outcome_uncertain_check_existing_pr_befo"
	KeyCmdPublishedHtmlUnavailable             = "cmd.published_html_unavailable"
	KeyCmdSavePublishedHtml                    = "cmd.save_published_html"
	KeyCmdIsolatedPreviewHelperIsNotInstalled  = "cmd.isolated_preview_helper_is_not_installed"
)

func init() {

	// cmd
	Register(KeyCmdChooseLocalAvatarImage, "选择本地头像图片", "Choose local avatar image")
	Register(KeyCmdBotCreatedSessionConnectionUnavailab, "Bot 已创建；当前会话连接暂不可用于打开。保留已有身份，连接恢复后单独打开", "Bot created; session connection unavailable, open the existing Bot after reconnecting")
	Register(KeyCmdCreatedBotOpenCanonicalSession, "已创建 Bot：打开专属会话", "Created Bot: open canonical session")
	Register(KeyCmdCatalogRecordRetainedSessionConnecti, "目录记录保留，但当前会话连接不能打开", "Catalog record retained; session connection unavailable")
	Register(KeyCmdOpenInspectedCatalogBot, "打开已核对目录 Bot", "Open inspected catalog Bot")
	Register(KeyCmdWatchingThisBotPageSCompleteCatalo, "Bot 通知观察中：仅当前 Bot 页完整目录", "Watching this Bot page’s complete catalog")
	Register(KeyCmdSystemNotificationsUnavailableOrNot, "系统通知不可用或未获允许；Bot 页状态仍可查看", "System notifications unavailable or not allowed")
	Register(KeyCmdCouldNotRestoreTheNotificationTarge, "无法显示通知目标窗口", "Could not restore the notification target window")
	Register(KeyCmdNotificationTargetVerifyInVisibility, "；请在隐藏/显示管理块核对", "Notification target: verify in visibility management")
	Register(KeyCmdSystemNotificationDeliveryFailedChec, "系统通知未能投递；请检查系统通知权限", "System notification delivery failed; check OS notification permission")
	Register(KeyCmdConversationOpenedReadMarkerFailed, "会话已打开；已读状态保存失败", "Conversation opened; read marker failed")
	Register(KeyCmdOpeningCanonicalConversation, "正在打开主会话", "Opening canonical conversation…")
	Register(KeyCmdResolveTheExistingChatDraftBeforeS, "目标会话已有草稿；请先处理主聊天草稿，再返回发送。", "Resolve the existing chat draft before sending.")
	Register(KeyCmdCanonicalConversationOpened, "已打开主会话", "Canonical conversation opened")
	Register(KeyCmdNeedsAttention, "需要处理", "Needs attention")
	Register(KeyCmdCreatedRoomOpenCanonicalSession, "已创建群聊：打开专属会话", "Created room: open canonical session")
	Register(KeyCmdUnsupportedTriggerAddressNoCredentia, "此 Host/任务无法表示为安全触发地址；未请求凭据", "Unsupported trigger address; no credential requested")
	Register(KeyCmdNoImageOnTheClipboard, "剪贴板没有图片", "No image on the clipboard")
	Register(KeyCmdEnterAReplyOrAddAnAttachment, "请输入回复内容", "Enter a reply or add an attachment")
	Register(KeyCmdHostOffline, "Host 未连接，无法开始录音", "Host offline")
	Register(KeyCmdNoCompletedBotJsonReply, "尚无已完成的 Bot JSON 回复", "No completed Bot JSON reply")
	Register(KeyCmdResolveTheExistingChatDraftFirst, "请先处理主聊天已有草稿", "Resolve the existing chat draft first")
	Register(KeyCmdDraftPreparedReviewAndSend, "已填入主聊天草稿；请审阅后发送", "Draft prepared; review and send")
	Register(KeyCmdImportedForPreviewSaveToApply, "已导入预览；保存后生效", "Imported for preview; save to apply")
	Register(KeyCmdSavedOtherBotCopiesRemainIndependen, "已保存；其他 Bot 的副本保持独立", "Saved; other Bot copies remain independent")
	Register(KeyCmdIndependentCopyAdded, "已添加独立副本", "Independent copy added")
	Register(KeyCmdTemplateUpdatedForFutureCopies, "已更新库模板；已有 Bot 副本不变", "Template updated for future copies")
	Register(KeyCmdRemovedFromThisBotTemplateRetained, "已从本 Bot 移除；可从库重新添加", "Removed from this Bot; template retained")
	Register(KeyCmdExportInstallationHistory, "导出安装历史", "Export installation history")
	Register(KeyCmdPhoneRemoteSettingWasNotSavedRetry, "手机 Remote 设置未保存，请重试。", "Phone Remote setting was not saved; retry.")
	Register(KeyCmdWhaleAssistantIsDisabled, "请先在助手设置启用鲸鱼娘", "Whale assistant is disabled")
	Register(KeyCmdHostOffline2, "Host 未连接，请稍后重试", "Host offline")
	Register(KeyCmdConnectionChanged, "Host 或插件已变化，请重试", "Connection changed")
	Register(KeyCmdConversationChanged, "已切换会话，请重新点击", "Conversation changed")
	Register(KeyCmdPetChatUnavailable, "宠物会话不可用，请检查鲸鱼娘插件", "Pet chat unavailable")
	Register(KeyCmdPetChatOpened, "已打开宠物会话", "Pet chat opened")
	Register(KeyCmdRuntimeChanged, "运行环境已变化，请重试", "Runtime changed")
	Register(KeyCmdCancelled, "用量读取已取消，账本保持不变", "Cancelled")
	Register(KeyCmdScanUnavailableSavedFoodRetained, "刷新未完成，保留已结算粮", "Scan unavailable; saved food retained")
	Register(KeyCmdLocalFoodRefreshed, "粮碗已刷新", "Local food refreshed")
	Register(KeyCmdConnectHostAndEnableWhaleFirst, "请连接 Host 并启用鲸鱼娘", "Connect Host and enable Whale first")
	Register(KeyCmdSelectThePetLookModelAndRefreshIt, "请在设置 → 桌宠选择看看模型并刷新图像目录", "Select the Pet Look model and refresh its image catalog in Settings → Pet")
	Register(KeyCmdLookAtScreen, "看看屏幕", "Look at screen")
	Register(KeyCmdCancel, "取消", "Cancel")
	Register(KeyCmdCaptureAndSend, "截图并发送", "Capture and send")
	Register(KeyCmdCancelledNoScreenshotTaken, "已取消，未截图", "Cancelled; no screenshot taken")
	Register(KeyCmdWindowConversationOrVisionRouteChan, "窗口、会话或视觉配置已变化，请重试", "Window, conversation or vision route changed")
	Register(KeyCmdContextChangedScreenshotWasNotSent, "窗口、会话或视觉配置已变化，截图未发送", "Context changed; screenshot was not sent")
	Register(KeyCmdContextChanged, "窗口、会话或视觉配置已变化", "Context changed")
	Register(KeyCmdLookFailedCheckTheVisionModel, "看看失败，请检查视觉模型后重试", "Look failed; check the vision model")
	Register(KeyCmdHostOfflineReopenAfterConnecting, "Host 未连接。恢复后重新点聊聊。", "Host offline; reopen after connecting.")
	Register(KeyCmdGatewaySettingsSavedApplying, "网关设置已保存，正在应用", "Gateway settings saved; applying")
	Register(KeyCmdPhonePrReview, "手机 PR 审阅", "Phone PR review")
	Register(KeyCmdWorking, "处理中", "Working…")
	Register(KeyCmdPhoneRequestedAPrReviewAndConfirm, "手机请求创建 PR；请在本窗口核对并确认", "Phone requested a PR; review and confirm here")
	Register(KeyCmdCommitPushMayAlreadyBeCompleteCanc, "手机上的提交/推送可能已完成。取消不会撤销这些操作；手机只重试剩余 PR。", "Commit/push may already be complete. Cancel does not undo them; the phone retains the remaining PR step.")
	Register(KeyCmdPrTitle, "PR 标题", "PR title")
	Register(KeyCmdPrBody, "PR 正文", "PR body")
	Register(KeyCmdCreateDraftPr, "创建草稿 PR", "Create draft PR")
	Register(KeyCmdGenerateCandidate, "生成候选文案", "Generate candidate")
	Register(KeyCmdCandidateNotYetApplied, "候选文案未采用", "Candidate not yet applied")
	Register(KeyCmdApplyCandidate, "采用候选文案", "Apply candidate")
	Register(KeyCmdPrepareTargetReview, "准备核对目标", "Prepare target review")
	Register(KeyCmdReviewTheExactContentBelow, "请核对下方完整内容", "Review the exact content below")
	Register(KeyCmdConfirmThisPr, "确认创建此 PR", "Confirm this PR")
	Register(KeyCmdOutcomeUncertainCheckExistingPrBefo, "; 结果不确定，请先检查已有 PR", "Outcome uncertain; check existing PR before retry")
	Register(KeyCmdPublishedHtmlUnavailable, "已发布 HTML 读取不可用：", "Published HTML unavailable: ")
	Register(KeyCmdSavePublishedHtml, "另存已发布 HTML", "Save published HTML")
	Register(KeyCmdIsolatedPreviewHelperIsNotInstalled, "隔离 helper 未随应用安装", "Isolated preview helper is not installed")
}

// Window titles, dialog titles and notifications opened from the desktop
// entrypoint; registered here so cmd never edits the shared map literals.
const (
	KeyCmdBotStatusUnavailable    = "cmd.bot_status_unavailable"
	KeyCmdConfigurationLoadFailed = "cmd.configuration_load_failed"
	KeyCmdCommitStagedChangesOnly = "cmd.commit_staged_changes_only"
	KeyCmdSecondsRemaining        = "cmd.seconds_remaining"
	KeyCmdOpenFile                = "cmd.open_file"
)

func init() {
	Register(KeyCmdBotStatusUnavailable, "%d 个 Bot 状态暂不可用；其余继续观察", "%d Bot activity checks are unavailable; the rest are still monitored")
	Register(KeyCmdConfigurationLoadFailed, "配置读取失败", "Configuration could not be loaded")
	Register(KeyCmdCommitStagedChangesOnly, "仅提交以下已暂存变更：\n", "Commit only these staged changes:\n")
	Register(KeyCmdSecondsRemaining, "剩余 %d 秒", "%d seconds remaining")
	Register(KeyCmdOpenFile, "打开文件", "Open file")
}
