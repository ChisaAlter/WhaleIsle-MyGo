package i18n

// Remaining UI-surface copy: shells for the personal Bot, room, routine and
// task panels, plus market/plugin manager, projects, schedules, assistant,
// agents, browser previews, pickers, launcher and boot. Keys are declared
// here and registered from init so no feature package edits the shared
// catalog literals in i18n.go.
const (

	// remote
	KeySurfacesSettingsOpenConfig     = "surfaces.settings.open_config"
	KeySurfacesSettingsOpenConfigHint = "surfaces.settings.open_config_hint"
	KeyMiscRemotePairingLinkValid = "misc.remote.pairing_link_valid"
	KeyMiscRemoteScanPrivatelyToPairYourPhone = "misc.remote.scan_privately_to_pair_your_phone"
	KeyMiscRemoteQrUnavailableCopyTheTemporaryLinkP = "misc.remote.qr_unavailable_copy_the_temporary_link_p"
	KeyMiscRemoteNoUsableOfferEnablingStartsTheInst = "misc.remote.no_usable_offer_enabling_starts_the_inst"
	KeyMiscRemoteCopyPairingLink = "misc.remote.copy_pairing_link"
	KeyMiscRemoteOpenPairingLink = "misc.remote.open_pairing_link"

	// im
	KeyMiscImWorkspace = "misc.im.workspace"
	KeyMiscImSaveWorkspace = "misc.im.save_workspace"
	KeyMiscImHostDefault = "misc.im.host_default"
	KeyMiscImCurrentPresetUnavailable = "misc.im.current_preset_unavailable"
	KeyMiscImAppliesToNewSessionsSendNewToSwit = "misc.im.applies_to_new_sessions_send_new_to_swit"
	KeyMiscImReconnectAndSendATestMessageToThe = "misc.im.reconnect_and_send_a_test_message_to_the"
	KeyMiscImLastChecked = "misc.im.last_checked"
	KeyMiscImConnectionError = "misc.im.connection_error"
	KeyMiscImCode = "misc.im.code"
	KeyMiscImLastMessageFailed = "misc.im.last_message_failed"
	KeyMiscImOccurred = "misc.im.occurred"

	// schedule
	KeyMiscScheduleDeliveryCursorExpiredRefreshSavedRe = "misc.schedule.delivery_cursor_expired_refresh_saved_re"
	KeyMiscScheduleTaskDeliveryHistoryNotFound = "misc.schedule.task_delivery_history_not_found"
	KeyMiscScheduleHour = "misc.schedule.hour"
	KeyMiscScheduleMinute = "misc.schedule.minute"
	KeyMiscScheduleSecond = "misc.schedule.second"
	KeyMiscScheduleChooseAValidIanaZone = "misc.schedule.choose_a_valid_iana_zone"
	KeyMiscScheduleTimeZone = "misc.schedule.time_zone"
	KeyMiscScheduleSearchTimeZones = "misc.schedule.search_time_zones"
	KeyMiscScheduleDate = "misc.schedule.date"
	KeyMiscScheduleClose = "misc.schedule.close"
	KeyMiscScheduleTimingIsReadOnly = "misc.schedule.timing_is_read_only"
	KeyMiscScheduleEvery = "misc.schedule.every"
	KeyMiscScheduleMinute2 = "misc.schedule.minute_2"
	KeyMiscSchedulePreview = "misc.schedule.preview"

	// assistant
	KeyMiscAssistantIUnderstandAvatarReplacementPresenta = "misc.assistant.i_understand_avatar_replacement_presenta"
	KeyMiscAssistantPersonalBotAvatar = "misc.assistant.personal_bot_avatar"
	KeyMiscAssistantRefreshAvatar = "misc.assistant.refresh_avatar"
	KeyMiscAssistantReadBackBeforeSaving = "misc.assistant.read_back_before_saving"
	KeyMiscAssistantUnsupportedAvatarUseBotWeb = "misc.assistant.unsupported_avatar_use_bot_web"
	KeyMiscAssistantSavedAvatarPreview = "misc.assistant.saved_avatar_preview"
	KeyMiscAssistantExistingImageUnavailablePreservedUse = "misc.assistant.existing_image_unavailable_preserved_use"
	KeyMiscAssistantWriteAuthorityUnavailable = "misc.assistant.write_authority_unavailable"
	KeyMiscAssistantShape = "misc.assistant.shape"
	KeyMiscAssistantColor = "misc.assistant.color"
	KeyMiscAssistantDraft = "misc.assistant.draft"
	KeyMiscAssistantDraftAvatarPreview = "misc.assistant.draft_avatar_preview"
	KeyMiscAssistantDraftImageCrop = "misc.assistant.draft_image_crop"
	KeyMiscAssistantDiscardAvatarDraft = "misc.assistant.discard_avatar_draft"
	KeyMiscAssistantReviewAvatar = "misc.assistant.review_avatar"
	KeyMiscAssistantAdoptMatchingAvatarReadback = "misc.assistant.adopt_matching_avatar_readback"
	KeyMiscAssistantConfirmAvatarReplacement = "misc.assistant.confirm_avatar_replacement"
	KeyMiscAssistantBeforeAvatarPreview = "misc.assistant.before_avatar_preview"
	KeyMiscAssistantAfter = "misc.assistant.after"
	KeyMiscAssistantAfterAvatarPreview = "misc.assistant.after_avatar_preview"
	KeyMiscAssistantCancelAvatarSave = "misc.assistant.cancel_avatar_save"
	KeyMiscAssistantConfirmAvatarSave = "misc.assistant.confirm_avatar_save"
	KeyMiscAssistantChooseLocalAvatarImage = "misc.assistant.choose_local_avatar_image"
	KeyMiscAssistantLocalImageCrop = "misc.assistant.local_image_crop"
	KeyMiscAssistantLocalCropPreview = "misc.assistant.local_crop_preview"
	KeyMiscAssistantCrop = "misc.assistant.crop"
	KeyMiscAssistantApplyCropToDraft = "misc.assistant.apply_crop_to_draft"
	KeyMiscAssistantCancelImageSelection = "misc.assistant.cancel_image_selection"
	KeyMiscAssistantIUnderstandCapabilityAndObjectiveEx = "misc.assistant.i_understand_capability_and_objective_ex"
	KeyMiscAssistantBotCapabilities = "misc.assistant.bot_capabilities"
	KeyMiscAssistantRefreshCapabilities = "misc.assistant.refresh_capabilities"
	KeyMiscAssistantHostWarning = "misc.assistant.host_warning"
	KeyMiscAssistantReadBackBeforeSaving2 = "misc.assistant.read_back_before_saving_2"
	KeyMiscAssistantWriteAuthorityUnavailable2 = "misc.assistant.write_authority_unavailable_2"
	KeyMiscAssistantCapabilityServiceInactive = "misc.assistant.capability_service_inactive"
	KeyMiscAssistantSkillsServiceUnavailable = "misc.assistant.skills_service_unavailable"
	KeyMiscAssistantUnavailableRemoveBeforeSaving = "misc.assistant.unavailable_remove_before_saving"
	KeyMiscAssistantDiscardCapabilityDraft = "misc.assistant.discard_capability_draft"
	KeyMiscAssistantReviewCapabilityChanges = "misc.assistant.review_capability_changes"
	KeyMiscAssistantReviewCapabilitySave = "misc.assistant.review_capability_save"
	KeyMiscAssistantUnknownOutcomesRequireReadback = "misc.assistant.unknown_outcomes_require_readback"
	KeyMiscAssistantCancelCapabilitySave = "misc.assistant.cancel_capability_save"
	KeyMiscAssistantConfirmCapabilitySave = "misc.assistant.confirm_capability_save"
	KeyMiscAssistantIUnderstandBotAndSessionEffects = "misc.assistant.i_understand_bot_and_session_effects"
	KeyMiscAssistantConfirmCreateAndOpenBot = "misc.assistant.confirm_create_and_open_bot"
	KeyMiscAssistantApplicationDefault = "misc.assistant.application_default"
	KeyMiscAssistantResolvedByHost = "misc.assistant.resolved_by_host"
	KeyMiscAssistantCreatePersonalBot = "misc.assistant.create_personal_bot"
	KeyMiscAssistantCreateABotWithTheApplicationDefaul = "misc.assistant.create_a_bot_with_the_application_defaul"
	KeyMiscAssistantRefreshBotCreationCatalog = "misc.assistant.refresh_bot_creation_catalog"
	KeyMiscAssistantReadCurrentCatalogAndAuthorityFirst = "misc.assistant.read_current_catalog_and_authority_first"
	KeyMiscAssistantWriteAuthorityUnavailable3 = "misc.assistant.write_authority_unavailable_3"
	KeyMiscAssistantOpenCreatedBot = "misc.assistant.open_created_bot"
	KeyMiscAssistantNewPersonalBot = "misc.assistant.new_personal_bot"
	KeyMiscAssistantStableBotName120 = "misc.assistant.stable_bot_name_120"
	KeyMiscAssistantDisplayTitle120 = "misc.assistant.display_title_120"
	KeyMiscAssistantDescription12000 = "misc.assistant.description_12000"
	KeyMiscAssistantMemoryReview = "misc.assistant.memory_review"
	KeyMiscAssistantNotifications = "misc.assistant.notifications"
	KeyMiscAssistantAdoptInspectedBot = "misc.assistant.adopt_inspected_bot"
	KeyMiscAssistantDiscardBotDraft = "misc.assistant.discard_bot_draft"
	KeyMiscAssistantReviewBotCreation = "misc.assistant.review_bot_creation"
	KeyMiscAssistantReviewNewBot = "misc.assistant.review_new_bot"
	KeyMiscAssistantBackToBotDraft = "misc.assistant.back_to_bot_draft"
	KeyMiscAssistantScratchDirectory = "misc.assistant.scratch_directory"
	KeyMiscAssistantPreserve = "misc.assistant.preserve"
	KeyMiscAssistantWorkspace = "misc.assistant.workspace"
	KeyMiscAssistantRegistryNotLoadedKeepTheCurrentWor = "misc.assistant.registry_not_loaded_keep_the_current_wor"
	KeyMiscAssistantModel = "misc.assistant.model"
	KeyMiscAssistantCatalogNotLoadedKeepTheCurrentChoi = "misc.assistant.catalog_not_loaded_keep_the_current_choi"
	KeyMiscAssistantModelDefaultEffort = "misc.assistant.model_default_effort"
	KeyMiscAssistantPreserve2 = "misc.assistant.preserve_2"
	KeyMiscAssistantReasoningEffort = "misc.assistant.reasoning_effort"
	KeyMiscAssistantPermanentlyDeleteReviewedBot = "misc.assistant.permanently_delete_reviewed_bot"
	KeyMiscAssistantIConfirmIrreversibleDeletionNow = "misc.assistant.i_confirm_irreversible_deletion_now"
	KeyMiscAssistantDeleteSelectedBotOrRoom = "misc.assistant.delete_selected_bot_or_room"
	KeyMiscAssistantReadDeletionPreviewOrOutcome = "misc.assistant.read_deletion_preview_or_outcome"
	KeyMiscAssistantFullCurrentPreviewAndAuthorityRequi = "misc.assistant.full_current_preview_and_authority_requi"
	KeyMiscAssistantWriteAuthorityUnavailable4 = "misc.assistant.write_authority_unavailable_4"
	KeyMiscAssistantReviewDestructiveDeletion = "misc.assistant.review_destructive_deletion"
	KeyMiscAssistantDestructiveConfirmationCancelByDefau = "misc.assistant.destructive_confirmation_cancel_by_defau"
	KeyMiscAssistantTypeExactStableName = "misc.assistant.type_exact_stable_name"
	KeyMiscAssistantIConfirmThisSourceAndCopyEffects = "misc.assistant.i_confirm_this_source_and_copy_effects"
	KeyMiscAssistantIConfirmCopiedGoalExecutionEffects = "misc.assistant.i_confirm_copied_goal_execution_effects"
	KeyMiscAssistantConfirmDuplicateBot = "misc.assistant.confirm_duplicate_bot"
	KeyMiscAssistantDuplicateSelectedPersonalBot = "misc.assistant.duplicate_selected_personal_bot"
	KeyMiscAssistantReadDuplicateSource = "misc.assistant.read_duplicate_source"
	KeyMiscAssistantReadTheFullCurrentSourceAndWriteA = "misc.assistant.read_the_full_current_source_and_write_a"
	KeyMiscAssistantWriteAuthorityUnavailable5 = "misc.assistant.write_authority_unavailable_5"
	KeyMiscAssistantOpenReceiptLinkedCopy = "misc.assistant.open_receipt_linked_copy"
	KeyMiscAssistantPrepareDuplicate = "misc.assistant.prepare_duplicate"
	KeyMiscAssistantNewStableName120 = "misc.assistant.new_stable_name_120"
	KeyMiscAssistantNoMatchingCandidateDoesNotEstablish = "misc.assistant.no_matching_candidate_does_not_establish"
	KeyMiscAssistantOpenInspectedCandidate = "misc.assistant.open_inspected_candidate"
	KeyMiscAssistantDiscardDuplicateDraft = "misc.assistant.discard_duplicate_draft"
	KeyMiscAssistantReviewDuplicate = "misc.assistant.review_duplicate"
	KeyMiscAssistantReviewDuplicate2 = "misc.assistant.review_duplicate_2"
	KeyMiscAssistantBackToDuplicateDraft = "misc.assistant.back_to_duplicate_draft"
	KeyMiscAssistantBotLongTermMemory = "misc.assistant.bot_long_term_memory"
	KeyMiscAssistantRefreshBeforeEditingOrRetrying = "misc.assistant.refresh_before_editing_or_retrying"
	KeyMiscAssistantReadOnlyOrWriteAuthorityUnavailable = "misc.assistant.read_only_or_write_authority_unavailable"
	KeyMiscAssistantBot = "misc.assistant.bot"
	KeyMiscAssistantUser = "misc.assistant.user"
	KeyMiscAssistantNoParsedEntries = "misc.assistant.no_parsed_entries"
	KeyMiscAssistantAddMemoryEntry = "misc.assistant.add_memory_entry"
	KeyMiscAssistantDraft2 = "misc.assistant.draft_2"
	KeyMiscAssistantEntryToRemove = "misc.assistant.entry_to_remove"
	KeyMiscAssistantReviewMemorySave = "misc.assistant.review_memory_save"
	KeyMiscAssistantDiscardDraft = "misc.assistant.discard_draft"
	KeyMiscAssistantReviewMemoryReplacement = "misc.assistant.review_memory_replacement"
	KeyMiscAssistantBackToMemoryEdit = "misc.assistant.back_to_memory_edit"
	KeyMiscAssistantConfirmTrackReplacement = "misc.assistant.confirm_track_replacement"
	KeyMiscAssistantIUnderstandModelMemoryAndObjective = "misc.assistant.i_understand_model_memory_and_objective"
	KeyMiscAssistantEnabled = "misc.assistant.enabled"
	KeyMiscAssistantDisabled = "misc.assistant.disabled"
	KeyMiscAssistantAutomaticMemoryLearning = "misc.assistant.automatic_memory_learning"
	KeyMiscAssistantRefreshMemoryLearning = "misc.assistant.refresh_memory_learning"
	KeyMiscAssistantReadBackBeforeSavingAgain = "misc.assistant.read_back_before_saving_again"
	KeyMiscAssistantWriteAuthorityUnavailable6 = "misc.assistant.write_authority_unavailable_6"
	KeyMiscAssistantAllowAutomaticMemoryLearning = "misc.assistant.allow_automatic_memory_learning"
	KeyMiscAssistantDiscardMemoryLearningDraft = "misc.assistant.discard_memory_learning_draft"
	KeyMiscAssistantReviewMemoryLearning = "misc.assistant.review_memory_learning"
	KeyMiscAssistantReviewMemoryLearning2 = "misc.assistant.review_memory_learning_2"
	KeyMiscAssistantCancelMemoryLearning = "misc.assistant.cancel_memory_learning"
	KeyMiscAssistantConfirmMemoryLearning = "misc.assistant.confirm_memory_learning"
	KeyMiscAssistantAllowOriginalBotNotifications = "misc.assistant.allow_original_bot_notifications"
	KeyMiscAssistantIUnderstandNotificationScopeAndProf = "misc.assistant.i_understand_notification_scope_and_prof"
	KeyMiscAssistantEnabled2 = "misc.assistant.enabled_2"
	KeyMiscAssistantDisabled2 = "misc.assistant.disabled_2"
	KeyMiscAssistantPersonalBotNotificationSetting = "misc.assistant.personal_bot_notification_setting"
	KeyMiscAssistantRefreshNotificationSetting = "misc.assistant.refresh_notification_setting"
	KeyMiscAssistantReadBackBeforeSaving3 = "misc.assistant.read_back_before_saving_3"
	KeyMiscAssistantBooleanUnavailable = "misc.assistant.boolean_unavailable"
	KeyMiscAssistantDiscardNotificationDraft = "misc.assistant.discard_notification_draft"
	KeyMiscAssistantReviewNotificationSetting = "misc.assistant.review_notification_setting"
	KeyMiscAssistantAdoptMatchingNotificationReadback = "misc.assistant.adopt_matching_notification_readback"
	KeyMiscAssistantConfirmNotificationSetting = "misc.assistant.confirm_notification_setting"
	KeyMiscAssistantCancelNotificationSave = "misc.assistant.cancel_notification_save"
	KeyMiscAssistantConfirmNotificationSetting2 = "misc.assistant.confirm_notification_setting_2"
	KeyMiscAssistantPauseObjective = "misc.assistant.pause_objective"
	KeyMiscAssistantClearCatalogObjective = "misc.assistant.clear_catalog_objective"
	KeyMiscAssistantLiveGoalNotIncludedRefresh = "misc.assistant.live_goal_not_included_refresh"
	KeyMiscAssistantNoVisibleLiveGoalInThisSnapshot = "misc.assistant.no_visible_live_goal_in_this_snapshot"
	KeyMiscAssistantBotObjective = "misc.assistant.bot_objective"
	KeyMiscAssistantRefreshObjective = "misc.assistant.refresh_objective"
	KeyMiscAssistantHostWarning2 = "misc.assistant.host_warning_2"
	KeyMiscAssistantRefreshBeforeAnotherAction = "misc.assistant.refresh_before_another_action"
	KeyMiscAssistantActivityWarning = "misc.assistant.activity_warning"
	KeyMiscAssistantReview = "misc.assistant.review"
	KeyMiscAssistantRefreshUncertainOutcomesNeverAutoRe = "misc.assistant.refresh_uncertain_outcomes_never_auto_re"
	KeyMiscAssistantCancelObjectiveAction = "misc.assistant.cancel_objective_action"
	KeyMiscAssistantConfirmObjectiveAction = "misc.assistant.confirm_objective_action"
	KeyMiscAssistantIUnderstandSavingMayStartOrResume = "misc.assistant.i_understand_saving_may_start_or_resume"
	KeyMiscAssistantObjectiveConfiguration = "misc.assistant.objective_configuration"
	KeyMiscAssistantRefreshObjectiveConfiguration = "misc.assistant.refresh_objective_configuration"
	KeyMiscAssistantObjective = "misc.assistant.objective"
	KeyMiscAssistantRoundCap1200 = "misc.assistant.round_cap_1_200"
	KeyMiscAssistantDiscardObjectiveDraft = "misc.assistant.discard_objective_draft"
	KeyMiscAssistantReviewObjectiveConfiguration = "misc.assistant.review_objective_configuration"
	KeyMiscAssistantReviewObjectiveConfiguration2 = "misc.assistant.review_objective_configuration_2"
	KeyMiscAssistantCancelObjectiveConfiguration = "misc.assistant.cancel_objective_configuration"
	KeyMiscAssistantConfirmObjectiveConfiguration = "misc.assistant.confirm_objective_configuration"
	KeyMiscAssistantIUnderstandModelDirectoryNewSession = "misc.assistant.i_understand_model_directory_new_session"
	KeyMiscAssistantHostDefault = "misc.assistant.host_default"
	KeyMiscAssistantScratch = "misc.assistant.scratch"
	KeyMiscAssistantPathUnavailable = "misc.assistant.path_unavailable"
	KeyMiscAssistantModelAndWorkspace = "misc.assistant.model_and_workspace"
	KeyMiscAssistantRefreshModelAndWorkspace = "misc.assistant.refresh_model_and_workspace"
	KeyMiscAssistantReadBackBeforeSaving4 = "misc.assistant.read_back_before_saving_4"
	KeyMiscAssistantIncompleteProfileUseBotWeb = "misc.assistant.incomplete_profile_use_bot_web"
	KeyMiscAssistantModelDraft = "misc.assistant.model_draft"
	KeyMiscAssistantModelDiscoveryUnavailable = "misc.assistant.model_discovery_unavailable"
	KeyMiscAssistantPreserveCurrent = "misc.assistant.preserve_current"
	KeyMiscAssistantCurrentHostDefault = "misc.assistant.current_host_default"
	KeyMiscAssistantPreserve3 = "misc.assistant.preserve_3"
	KeyMiscAssistantWorkspaceDraft = "misc.assistant.workspace_draft"
	KeyMiscAssistantPreserveCurrent2 = "misc.assistant.preserve_current_2"
	KeyMiscAssistantHostDirectoriesUnavailable = "misc.assistant.host_directories_unavailable"
	KeyMiscAssistantHistoryPresentOrUnknown = "misc.assistant.history_present_or_unknown"
	KeyMiscAssistantDiscardRouteDraft = "misc.assistant.discard_route_draft"
	KeyMiscAssistantReviewModelAndWorkspace = "misc.assistant.review_model_and_workspace"
	KeyMiscAssistantAdoptMatchingReadback = "misc.assistant.adopt_matching_readback"
	KeyMiscAssistantConfirmModelAndWorkspace = "misc.assistant.confirm_model_and_workspace"
	KeyMiscAssistantCancelRouteSave = "misc.assistant.cancel_route_save"
	KeyMiscAssistantConfirmRouteSave = "misc.assistant.confirm_route_save"
	KeyMiscAssistantIUnderstandSourcePolicyAndExecution = "misc.assistant.i_understand_source_policy_and_execution"
	KeyMiscAssistantIExplicitlyApproveTheAccessExpansio = "misc.assistant.i_explicitly_approve_the_access_expansio"
	KeyMiscAssistantNone = "misc.assistant.none"
	KeyMiscAssistantHidden = "misc.assistant.hidden"
	KeyMiscAssistantSelectedBots = "misc.assistant.selected_bots"
	KeyMiscAssistantMessageAndTaskSources = "misc.assistant.message_and_task_sources"
	KeyMiscAssistantRefreshSourcePolicy = "misc.assistant.refresh_source_policy"
	KeyMiscAssistantIncompleteProfile = "misc.assistant.incomplete_profile"
	KeyMiscAssistantDefaultRule = "misc.assistant.default_rule"
	KeyMiscAssistantDiscardSourceDraft = "misc.assistant.discard_source_draft"
	KeyMiscAssistantReviewSourcePolicy = "misc.assistant.review_source_policy"
	KeyMiscAssistantAdoptSourceReadback = "misc.assistant.adopt_source_readback"
	KeyMiscAssistantConfirmSourcePolicy = "misc.assistant.confirm_source_policy"
	KeyMiscAssistantRetainedObjective = "misc.assistant.retained_objective"
	KeyMiscAssistantCancelSourceSave = "misc.assistant.cancel_source_save"
	KeyMiscAssistantConfirmSourceSave = "misc.assistant.confirm_source_save"
	KeyMiscAssistantIUnderstandAddressPersonaPresentatio = "misc.assistant.i_understand_address_persona_presentatio"
	KeyMiscAssistantEmpty = "misc.assistant.empty"
	KeyMiscAssistantNameDisplayTitleAndPersona = "misc.assistant.name_display_title_and_persona"
	KeyMiscAssistantRefreshTextProfile = "misc.assistant.refresh_text_profile"
	KeyMiscAssistantIncompleteProfileUseBotWeb2 = "misc.assistant.incomplete_profile_use_bot_web_2"
	KeyMiscAssistantName120 = "misc.assistant.name_120"
	KeyMiscAssistantDisplayTitle120Optional = "misc.assistant.display_title_120_optional"
	KeyMiscAssistantCompleteDescriptionPersona12000 = "misc.assistant.complete_description_persona_12000"
	KeyMiscAssistantDiscardTextDraft = "misc.assistant.discard_text_draft"
	KeyMiscAssistantReviewTextProfile = "misc.assistant.review_text_profile"
	KeyMiscAssistantAdoptMatchingTextReadback = "misc.assistant.adopt_matching_text_readback"
	KeyMiscAssistantConfirmCompleteTextProfile = "misc.assistant.confirm_complete_text_profile"
	KeyMiscAssistantCancelTextProfileSave = "misc.assistant.cancel_text_profile_save"
	KeyMiscAssistantConfirmTextProfileSave = "misc.assistant.confirm_text_profile_save"
	KeyMiscAssistantRoom = "misc.assistant.room"
	KeyMiscAssistantTargetAwaitsFreshCatalog = "misc.assistant.target_awaits_fresh_catalog"
	KeyMiscAssistantExactTargetNotFound = "misc.assistant.exact_target_not_found"
	KeyMiscAssistantExactTarget = "misc.assistant.exact_target"
	KeyMiscAssistantShowHiddenLocally = "misc.assistant.show_hidden_locally"
	KeyMiscAssistantReadFullCatalog = "misc.assistant.read_full_catalog"
	KeyMiscAssistantAdoptExactReadback = "misc.assistant.adopt_exact_readback"
	KeyMiscAssistantCancel = "misc.assistant.cancel"
	KeyMiscAssistantIConfirmThisTargetAndEffects = "misc.assistant.i_confirm_this_target_and_effects"
	KeyMiscAssistantConfirmUnhide = "misc.assistant.confirm_unhide"
	KeyMiscAssistantConfirmHide = "misc.assistant.confirm_hide"
	KeyMiscAssistantReviewHide = "misc.assistant.review_hide"
	KeyMiscAssistantReviewUnhide = "misc.assistant.review_unhide"
	KeyMiscAssistantNoMatchingItemsShowHiddenToRecover = "misc.assistant.no_matching_items_show_hidden_to_recover"
	KeyMiscAssistantContacts = "misc.assistant.contacts"
	KeyMiscAssistantRefresh = "misc.assistant.refresh"
	KeyMiscAssistantSearchContacts = "misc.assistant.search_contacts"
	KeyMiscAssistantSource = "misc.assistant.source"
	KeyMiscAssistantAllSources = "misc.assistant.all_sources"
	KeyMiscAssistantNewSection = "misc.assistant.new_section"
	KeyMiscAssistantRenameSection = "misc.assistant.rename_section"
	KeyMiscAssistantSaveSection = "misc.assistant.save_section"
	KeyMiscAssistantRestoreSection = "misc.assistant.restore_section"
	KeyMiscAssistantActive = "misc.assistant.active"
	KeyMiscAssistantUnreadCompletedResult = "misc.assistant.unread_completed_result"
	KeyMiscAssistantMarkRead = "misc.assistant.mark_read"
	KeyMiscAssistantNoMatchingContacts = "misc.assistant.no_matching_contacts"
	KeyMiscAssistantFullHistoryStreamingRepliesApprovals = "misc.assistant.full_history_streaming_replies_approvals"
	KeyMiscAssistantSelect = "misc.assistant.select"
	KeyMiscAssistantOpenTheCanonicalChatToSendFollowR = "misc.assistant.open_the_canonical_chat_to_send_follow_r"
	KeyMiscAssistantIUnderstandRoomAndSessionEffects = "misc.assistant.i_understand_room_and_session_effects"
	KeyMiscAssistantConfirmCreateAndOpen = "misc.assistant.confirm_create_and_open"
	KeyMiscAssistantUnavailable = "misc.assistant.unavailable"
	KeyMiscAssistantCreateBotGroup = "misc.assistant.create_bot_group"
	KeyMiscAssistantRefreshCreationCatalog = "misc.assistant.refresh_creation_catalog"
	KeyMiscAssistantReadTheCurrentCatalogAndWriteAutho = "misc.assistant.read_the_current_catalog_and_write_autho"
	KeyMiscAssistantWriteAuthorityUnavailable7 = "misc.assistant.write_authority_unavailable_7"
	KeyMiscAssistantOpenCreatedGroup = "misc.assistant.open_created_group"
	KeyMiscAssistantNewGroup = "misc.assistant.new_group"
	KeyMiscAssistantGroupName120 = "misc.assistant.group_name_120"
	KeyMiscAssistantRounds13 = "misc.assistant.rounds_1_3"
	KeyMiscAssistantSpeaks110 = "misc.assistant.speaks_1_10"
	KeyMiscAssistantDescription120002 = "misc.assistant.description_12000_2"
	KeyMiscAssistantRemoveUnavailableMember = "misc.assistant.remove_unavailable_member"
	KeyMiscAssistantUseInspectedGroup = "misc.assistant.use_inspected_group"
	KeyMiscAssistantDiscardGroupDraft = "misc.assistant.discard_group_draft"
	KeyMiscAssistantReviewGroupCreation = "misc.assistant.review_group_creation"
	KeyMiscAssistantReviewNewGroup = "misc.assistant.review_new_group"
	KeyMiscAssistantBackToGroupDraft = "misc.assistant.back_to_group_draft"
	KeyMiscAssistantPermanentlyDisbandReviewedRoom = "misc.assistant.permanently_disband_reviewed_room"
	KeyMiscAssistantCancelAndKeepRoom = "misc.assistant.cancel_and_keep_room"
	KeyMiscAssistantTypeExactRoomId = "misc.assistant.type_exact_room_id"
	KeyMiscAssistantCancelAndKeepBot = "misc.assistant.cancel_and_keep_bot"
	KeyMiscAssistantTypeExactBotId = "misc.assistant.type_exact_bot_id"
	KeyMiscAssistantIUnderstandGroupProfileAndSessionE = "misc.assistant.i_understand_group_profile_and_session_e"
	KeyMiscAssistantConfirmGroupUpdate = "misc.assistant.confirm_group_update"
	KeyMiscAssistantGroupProfileAndLimits = "misc.assistant.group_profile_and_limits"
	KeyMiscAssistantReadGroupProfile = "misc.assistant.read_group_profile"
	KeyMiscAssistantReadCompleteProfileAndRevisionFirst = "misc.assistant.read_complete_profile_and_revision_first"
	KeyMiscAssistantEditGroupSettings = "misc.assistant.edit_group_settings"
	KeyMiscAssistantRemovingAndSelectingAgainMovesAMem = "misc.assistant.removing_and_selecting_again_moves_a_mem"
	KeyMiscAssistantRemoveUnavailableMember2 = "misc.assistant.remove_unavailable_member_2"
	KeyMiscAssistantAdoptExactReadback2 = "misc.assistant.adopt_exact_readback_2"
	KeyMiscAssistantDiscardEditDraft = "misc.assistant.discard_edit_draft"
	KeyMiscAssistantReviewGroupChanges = "misc.assistant.review_group_changes"
	KeyMiscAssistantReviewGroupUpdate = "misc.assistant.review_group_update"
	KeyMiscAssistantBackToDraft = "misc.assistant.back_to_draft"
	KeyMiscAssistantOpenMemberSession = "misc.assistant.open_member_session"
	KeyMiscAssistantGroupMemberRuntime = "misc.assistant.group_member_runtime"
	KeyMiscAssistantRefreshMemberRuntime = "misc.assistant.refresh_member_runtime"
	KeyMiscAssistantCurrentRuntimeUnverifiedAnyPrevious = "misc.assistant.current_runtime_unverified_any_previous"
	KeyMiscAssistantNoVisibleMemberRuntimeThisIsNotA = "misc.assistant.no_visible_member_runtime_this_is_not_a"
	KeyMiscAssistantUnknownMember = "misc.assistant.unknown_member"
	KeyMiscAssistantUnknown = "misc.assistant.unknown"
	KeyMiscAssistantNeedsAttention = "misc.assistant.needs_attention"
	KeyMiscAssistantEmpty2 = "misc.assistant.empty_2"
	KeyMiscAssistantRoomName120Utf16 = "misc.assistant.room_name_120_utf_16"
	KeyMiscAssistantCompleteRoomDescription12000Utf16 = "misc.assistant.complete_room_description_12000_utf_16"
	KeyMiscAssistantIUnderstandCommandEffectsAndWorking = "misc.assistant.i_understand_command_effects_and_working"
	KeyMiscAssistantThisRoutineSPreviousOutputSelf = "misc.assistant.this_routine_s_previous_output_self"
	KeyMiscAssistantUnavailableSourceRemoveExplicitlyBef = "misc.assistant.unavailable_source_remove_explicitly_bef"
	KeyMiscAssistantAdvancedSettings = "misc.assistant.advanced_settings"
	KeyMiscAssistantAllowSilentResponse = "misc.assistant.allow_silent_response"
	KeyMiscAssistantUnknownWatchPreserved = "misc.assistant.unknown_watch_preserved"
	KeyMiscAssistantNoWatch = "misc.assistant.no_watch"
	KeyMiscAssistantUrlWatch = "misc.assistant.url_watch"
	KeyMiscAssistantCommandWatch = "misc.assistant.command_watch"
	KeyMiscAssistantFullWatchUrl = "misc.assistant.full_watch_url"
	KeyMiscAssistantFullHostShellCommand = "misc.assistant.full_host_shell_command"
	KeyMiscAssistantTimeoutMs1300000 = "misc.assistant.timeout_ms_1_300000"
	KeyMiscAssistantEffectiveWatchNone = "misc.assistant.effective_watch_none"
	KeyMiscAssistantNotConfigured = "misc.assistant.not_configured"
	KeyMiscAssistantConfirmRoutineAction = "misc.assistant.confirm_routine_action"
	KeyMiscAssistantCreateBotRoutine = "misc.assistant.create_bot_routine"
	KeyMiscAssistantNewRoutine = "misc.assistant.new_routine"
	KeyMiscAssistantRefreshTheRoutineCatalogFirstEmpty = "misc.assistant.refresh_the_routine_catalog_first_empty"
	KeyMiscAssistantSaveScheduleContextSilentRepliesAnd = "misc.assistant.save_schedule_context_silent_replies_and"
	KeyMiscAssistantFrequency = "misc.assistant.frequency"
	KeyMiscAssistantNewRoutineName = "misc.assistant.new_routine_name"
	KeyMiscAssistantMinuteHourDayMonthWeekday = "misc.assistant.minute_hour_day_month_weekday"
	KeyMiscAssistantIntervalAmountPositiveInteger = "misc.assistant.interval_amount_positive_integer"
	KeyMiscAssistantIntervalUnit = "misc.assistant.interval_unit"
	KeyMiscAssistantMinutes = "misc.assistant.minutes"
	KeyMiscAssistantHours = "misc.assistant.hours"
	KeyMiscAssistantDays = "misc.assistant.days"
	KeyMiscAssistantHour0023 = "misc.assistant.hour_00_23"
	KeyMiscAssistantMinute0059 = "misc.assistant.minute_00_59"
	KeyMiscAssistantWeekday = "misc.assistant.weekday"
	KeyMiscAssistantSunday = "misc.assistant.sunday"
	KeyMiscAssistantSun = "misc.assistant.sun"
	KeyMiscAssistantMon = "misc.assistant.mon"
	KeyMiscAssistantTue = "misc.assistant.tue"
	KeyMiscAssistantWed = "misc.assistant.wed"
	KeyMiscAssistantThu = "misc.assistant.thu"
	KeyMiscAssistantFri = "misc.assistant.fri"
	KeyMiscAssistantSat = "misc.assistant.sat"
	KeyMiscAssistantDayOfMonth131 = "misc.assistant.day_of_month_1_31"
	KeyMiscAssistantMaxRuns0Unlimited = "misc.assistant.max_runs_0_unlimited"
	KeyMiscAssistantSpecifyTheIntendedIanaTimezone = "misc.assistant.specify_the_intended_iana_timezone"
	KeyMiscAssistantNewRoutinePrompt = "misc.assistant.new_routine_prompt"
	KeyMiscAssistantNewRoutinePaused = "misc.assistant.new_routine_paused"
	KeyMiscAssistantNewRoutineEnabled = "misc.assistant.new_routine_enabled"
	KeyMiscAssistantThisCreationIsLockedUntilItsOutcom = "misc.assistant.this_creation_is_locked_until_its_outcom"
	KeyMiscAssistantCheckCreationOutcome = "misc.assistant.check_creation_outcome"
	KeyMiscAssistantUseRecord = "misc.assistant.use_record"
	KeyMiscAssistantNoNewRecordReviewAgain = "misc.assistant.no_new_record_review_again"
	KeyMiscAssistantDiscardNewDraft = "misc.assistant.discard_new_draft"
	KeyMiscAssistantRecheckCatalog = "misc.assistant.recheck_catalog"
	KeyMiscAssistantReviewCreation = "misc.assistant.review_creation"
	KeyMiscAssistantReviewCreation2 = "misc.assistant.review_creation_2"
	KeyMiscAssistantBackToDraft2 = "misc.assistant.back_to_draft_2"
	KeyMiscAssistantConfirmNewRoutine = "misc.assistant.confirm_new_routine"
	KeyMiscAssistantDaily = "misc.assistant.daily"
	KeyMiscAssistantHourly = "misc.assistant.hourly"
	KeyMiscAssistantWeekdays = "misc.assistant.weekdays"
	KeyMiscAssistantWeekly = "misc.assistant.weekly"
	KeyMiscAssistantMonthly = "misc.assistant.monthly"
	KeyMiscAssistantAdvanced = "misc.assistant.advanced"
	KeyMiscAssistantOnce = "misc.assistant.once"
	KeyMiscAssistantInterval = "misc.assistant.interval"
	KeyMiscAssistantIApproveObtainingPersistentHostCata = "misc.assistant.i_approve_obtaining_persistent_host_cata"
	KeyMiscAssistantCurrentHostBotCatalogTriggerCredent = "misc.assistant.current_host_bot_catalog_trigger_credent"
	KeyMiscAssistantReviewHostCatalogCredential = "misc.assistant.review_host_catalog_credential"
	KeyMiscAssistantConfirmHostCatalogCredential = "misc.assistant.confirm_host_catalog_credential"
	KeyMiscAssistantWaitingForHostClosingDoesNotRevoke = "misc.assistant.waiting_for_host_closing_does_not_revoke"
	KeyMiscAssistantCopyTriggerAddress = "misc.assistant.copy_trigger_address"
	KeyMiscAssistantCopySecretHeader = "misc.assistant.copy_secret_header"
	KeyMiscAssistantCloseAndClearCredential = "misc.assistant.close_and_clear_credential"
	KeyMiscAssistantReviewSave = "misc.assistant.review_save"
	KeyMiscAssistantPauseRoutine = "misc.assistant.pause_routine"
	KeyMiscAssistantEnableRoutine = "misc.assistant.enable_routine"
	KeyMiscAssistantRunNow = "misc.assistant.run_now"
	KeyMiscAssistantDeleteRoutine = "misc.assistant.delete_routine"
	KeyMiscAssistantBotRoutines = "misc.assistant.bot_routines"
	KeyMiscAssistantRefreshRoutines = "misc.assistant.refresh_routines"
	KeyMiscAssistantSelectABotFirst = "misc.assistant.select_a_bot_first"
	KeyMiscAssistantRefreshBeforeMakingChanges = "misc.assistant.refresh_before_making_changes"
	KeyMiscAssistantWaitingForHost = "misc.assistant.waiting_for_host"
	KeyMiscAssistantNoRoutinesForThisBot = "misc.assistant.no_routines_for_this_bot"
	KeyMiscAssistantDismissMissedRunNotice = "misc.assistant.dismiss_missed_run_notice"
	KeyMiscAssistantSearchRoutinesBotsOrErrors = "misc.assistant.search_routines_bots_or_errors"
	KeyMiscAssistantNoMatchingRoutines = "misc.assistant.no_matching_routines"
	KeyMiscAssistantPaused = "misc.assistant.paused"
	KeyMiscAssistantEnabled3 = "misc.assistant.enabled_3"
	KeyMiscAssistantMissedScheduledRun = "misc.assistant.missed_scheduled_run"
	KeyMiscAssistantPrompt = "misc.assistant.prompt"
	KeyMiscAssistantLastOutput = "misc.assistant.last_output"
	KeyMiscAssistantEditRoutine = "misc.assistant.edit_routine"
	KeyMiscAssistantRunHistoryLatest10 = "misc.assistant.run_history_latest_10"
	KeyMiscAssistantHistoryUnavailable = "misc.assistant.history_unavailable"
	KeyMiscAssistantNoRecordedRuns = "misc.assistant.no_recorded_runs"
	KeyMiscAssistantBackToRoutine = "misc.assistant.back_to_routine"
	KeyMiscAssistantUnlimited = "misc.assistant.unlimited"
	KeyMiscAssistantDraftRevisionIsStale = "misc.assistant.draft_revision_is_stale"
	KeyMiscAssistantRoutineName = "misc.assistant.routine_name"
	KeyMiscAssistantSchedule = "misc.assistant.schedule"
	KeyMiscAssistantTimezone = "misc.assistant.timezone"
	KeyMiscAssistantRoutinePrompt = "misc.assistant.routine_prompt"
	KeyMiscAssistantDraftPaused = "misc.assistant.draft_paused"
	KeyMiscAssistantDraftEnabled = "misc.assistant.draft_enabled"
	KeyMiscAssistantDiscardRoutineDraft = "misc.assistant.discard_routine_draft"
	KeyMiscAssistantPause = "misc.assistant.pause"
	KeyMiscAssistantResume = "misc.assistant.resume"
	KeyMiscAssistantCancelTask = "misc.assistant.cancel_task"
	KeyMiscAssistantRetryTask = "misc.assistant.retry_task"
	KeyMiscAssistantStopBot = "misc.assistant.stop_bot"
	KeyMiscAssistantBotTasks = "misc.assistant.bot_tasks"
	KeyMiscAssistantRefreshTasks = "misc.assistant.refresh_tasks"
	KeyMiscAssistantWaitingForHost2 = "misc.assistant.waiting_for_host_2"
	KeyMiscAssistantRefreshBeforeMakingChanges2 = "misc.assistant.refresh_before_making_changes_2"
	KeyMiscAssistantNoDelegationTasksForThisBot = "misc.assistant.no_delegation_tasks_for_this_bot"
	KeyMiscAssistantSearchTasksParticipantsOrErrors = "misc.assistant.search_tasks_participants_or_errors"
	KeyMiscAssistantTaskStatus = "misc.assistant.task_status"
	KeyMiscAssistantNoMatchingTasks = "misc.assistant.no_matching_tasks"
	KeyMiscAssistantOpenSenderConversation = "misc.assistant.open_sender_conversation"
	KeyMiscAssistantOpenRecipientConversation = "misc.assistant.open_recipient_conversation"
	KeyMiscAssistantTask = "misc.assistant.task"
	KeyMiscAssistantSuccessCriteria = "misc.assistant.success_criteria"
	KeyMiscAssistantResult = "misc.assistant.result"
	KeyMiscAssistantError = "misc.assistant.error"
	KeyMiscAssistantRetryOf = "misc.assistant.retry_of"
	KeyMiscAssistantBack = "misc.assistant.back"
	KeyMiscAssistantConfirmAction = "misc.assistant.confirm_action"
	KeyMiscAssistantWhalePreferences = "misc.assistant.whale_preferences"
	KeyMiscAssistantRefreshPreferences = "misc.assistant.refresh_preferences"
	KeyMiscAssistantFollowSessionDefault = "misc.assistant.follow_session_default"
	KeyMiscAssistantCurrent = "misc.assistant.current"
	KeyMiscAssistantModelCatalog = "misc.assistant.model_catalog"
	KeyMiscAssistantDefaultEffort = "misc.assistant.default_effort"
	KeyMiscAssistantShareConversationWithIm = "misc.assistant.share_conversation_with_im"
	KeyMiscAssistantWhaleSettings = "misc.assistant.whale_settings"
	KeyMiscAssistantRefreshBeforeEditingOrRetrying2 = "misc.assistant.refresh_before_editing_or_retrying_2"
	KeyMiscAssistantLongTermMemory = "misc.assistant.long_term_memory"
	KeyMiscAssistantMemoryMdIsInjectedIntoWhaleSSessi = "misc.assistant.memory_md_is_injected_into_whale_s_sessi"
	KeyMiscAssistantSaveMemory = "misc.assistant.save_memory"
	KeyMiscAssistantRestoreMemoryDraft = "misc.assistant.restore_memory_draft"
	KeyMiscAssistantClearMemory = "misc.assistant.clear_memory"
	KeyMiscAssistantClearLongTermMemoryThisCannotBeUn = "misc.assistant.clear_long_term_memory_this_cannot_be_un"
	KeyMiscAssistantCancelKeepsThisDraft = "misc.assistant.cancel_keeps_this_draft"
	KeyMiscAssistantCancelClear = "misc.assistant.cancel_clear"
	KeyMiscAssistantConfirmClearMemory = "misc.assistant.confirm_clear_memory"
	KeyMiscAssistantExtraPersona = "misc.assistant.extra_persona"
	KeyMiscAssistantAppendedToHerPersonaLeaveBlankToS = "misc.assistant.appended_to_her_persona_leave_blank_to_s"
	KeyMiscAssistantSavePersona = "misc.assistant.save_persona"
	KeyMiscAssistantRestorePersonaDraft = "misc.assistant.restore_persona_draft"
	KeyMiscAssistantNameAndTitle = "misc.assistant.name_and_title"
	KeyMiscAssistantName = "misc.assistant.name"
	KeyMiscAssistantHowSheCallsYou = "misc.assistant.how_she_calls_you"
	KeyMiscAssistantRestoreNameDraft = "misc.assistant.restore_name_draft"
	KeyMiscAssistantUnsavedDraft = "misc.assistant.unsaved_draft"
	KeyMiscAssistantMatchesHost = "misc.assistant.matches_host"

	// browserpreview
	KeyMiscBrowserpreviewHtmlPreviewUnavailableUseSystemOpen = "misc.browserpreview.html_preview_unavailable_use_system_open"

	// custompanel
	KeyMiscCustompanelCustomPanels = "misc.custompanel.custom_panels"
	KeyMiscCustompanelOpenABotSCanonicalConversation = "misc.custompanel.open_a_bot_s_canonical_conversation"
	KeyMiscCustompanelRequirements = "misc.custompanel.requirements"
	KeyMiscCustompanelCreateWithBot = "misc.custompanel.create_with_bot"
	KeyMiscCustompanelSendThePreparedDraftThenImportThe = "misc.custompanel.send_the_prepared_draft_then_import_the"
	KeyMiscCustompanelImportReply = "misc.custompanel.import_reply"
	KeyMiscCustompanelNewJson = "misc.custompanel.new_json"
	KeyMiscCustompanelNewTemplateAndBotCopy = "misc.custompanel.new_template_and_bot_copy"
	KeyMiscCustompanelEditThisBotSCopy = "misc.custompanel.edit_this_bot_s_copy"
	KeyMiscCustompanelPreview = "misc.custompanel.preview"
	KeyMiscCustompanelSave = "misc.custompanel.save"
	KeyMiscCustompanelThisBotSPanels = "misc.custompanel.this_bot_s_panels"
	KeyMiscCustompanelEdit = "misc.custompanel.edit"
	KeyMiscCustompanelRemove = "misc.custompanel.remove"
	KeyMiscCustompanelUpdateLibraryTemplate = "misc.custompanel.update_library_template"
	KeyMiscCustompanelPanelLibrary = "misc.custompanel.panel_library"
	KeyMiscCustompanelAddCopy = "misc.custompanel.add_copy"
	KeyMiscCustompanelEditTemplate = "misc.custompanel.edit_template"
	KeyMiscCustompanelPrepareChatDraftDoesNotSend = "misc.custompanel.prepare_chat_draft_does_not_send"
)

func init() {

	// remote
	Register(KeySurfacesSettingsOpenConfig, "打开配置文件", "Open config file")
	Register(KeySurfacesSettingsOpenConfigHint, "此构建尚未接入配置文件打开", "Opening the config file is not wired in this build")
	Register(KeyMiscRemotePairingLinkValid, "配对链接有效", "Pairing link valid · ")
	Register(KeyMiscRemoteScanPrivatelyToPairYourPhone, "用手机扫码配对，请勿分享二维码。", "Scan privately to pair your phone.")
	Register(KeyMiscRemoteQrUnavailableCopyTheTemporaryLinkP, "二维码不可用，可复制临时配对链接。", "QR unavailable; copy the temporary link privately.")
	Register(KeyMiscRemoteNoUsableOfferEnablingStartsTheInst, "没有可用配对链接。启用会自动启动已安装的原版运行时；失败时请检查运行时后刷新重试。", "No usable offer. Enabling starts the installed runtime automatically; check the runtime and refresh to retry if needed.")
	Register(KeyMiscRemoteCopyPairingLink, "复制配对链接", "Copy pairing link")
	Register(KeyMiscRemoteOpenPairingLink, "打开配对链接", "Open pairing link")

	// im
	Register(KeyMiscImWorkspace, "当前工作区：", "Workspace: ")
	Register(KeyMiscImSaveWorkspace, "保存工作区", "Save workspace")
	Register(KeyMiscImHostDefault, "跟随 Host 默认", "Host default")
	Register(KeyMiscImCurrentPresetUnavailable, "当前 Agent Preset 已不可用，可选择其他 Preset 或跟随 Host 默认。", "Current preset unavailable.")
	Register(KeyMiscImAppliesToNewSessionsSendNewToSwit, "仅影响新会话；切换当前聊天请先发送 /new。", "Applies to new sessions; send /new to switch the current chat.")
	Register(KeyMiscImReconnectAndSendATestMessageToThe, "将重新连接，并可能向 Host 按原版平台规则选择的测试目标发送消息。请到对应会话核对结果；此处不指定新收件人。", "Reconnect and send a test message to the original platform target selected by Host.")
	Register(KeyMiscImLastChecked, "最近检查：", "Last checked: ")
	Register(KeyMiscImConnectionError, "连接错误：", "Connection error: ")
	Register(KeyMiscImCode, "错误码：", "Code: ")
	Register(KeyMiscImLastMessageFailed, "最近消息处理失败", "Last message failed")
	Register(KeyMiscImOccurred, "发生时间：", "Occurred: ")

	// schedule
	Register(KeyMiscScheduleDeliveryCursorExpiredRefreshSavedRe, "分页游标已失效，请刷新已保存的投递记录", "Delivery cursor expired; refresh saved records")
	Register(KeyMiscScheduleTaskDeliveryHistoryNotFound, "找不到原任务的投递记录", "Task delivery history not found")
	Register(KeyMiscScheduleHour, "时", "Hour")
	Register(KeyMiscScheduleMinute, "分", "Minute")
	Register(KeyMiscScheduleSecond, "秒", "Second")
	Register(KeyMiscScheduleChooseAValidIanaZone, "请输入有效 IANA 时区", "Choose a valid IANA zone")
	Register(KeyMiscScheduleTimeZone, "时区：", "Time zone: ")
	Register(KeyMiscScheduleSearchTimeZones, "搜索时区", "Search time zones")
	Register(KeyMiscScheduleDate, "日期：", "Date: ")
	Register(KeyMiscScheduleClose, "关闭", "Close")
	Register(KeyMiscScheduleTimingIsReadOnly, "当前时间规则只读", "Timing is read-only")
	Register(KeyMiscScheduleEvery, "每隔", "Every")
	Register(KeyMiscScheduleMinute2, "分钟", "Minute")
	Register(KeyMiscSchedulePreview, "预览：", "Preview: ")

	// assistant
	Register(KeyMiscAssistantIUnderstandAvatarReplacementPresenta, "我理解头像替换、会话展示及目标可能执行的影响", "I understand avatar replacement, presentation and execution effects")
	Register(KeyMiscAssistantPersonalBotAvatar, "个人 Bot 头像", "Personal Bot avatar")
	Register(KeyMiscAssistantRefreshAvatar, "刷新头像", "Refresh avatar")
	Register(KeyMiscAssistantReadBackBeforeSaving, "先读回核实；未知结果保留并锁定草稿，只读回，不自动重试", "Read back before saving")
	Register(KeyMiscAssistantUnsupportedAvatarUseBotWeb, "头像记录无法无损读取，请用 Bot Web", "Unsupported avatar; use Bot Web")
	Register(KeyMiscAssistantSavedAvatarPreview, "已保存头像", "Saved avatar preview")
	Register(KeyMiscAssistantExistingImageUnavailablePreservedUse, "现有图片无法安全预览，保留原头像；请用 Bot Web 核对并编辑", "Existing image unavailable; preserved, use Bot Web")
	Register(KeyMiscAssistantWriteAuthorityUnavailable, "Host 写权限未核实", "Write authority unavailable")
	Register(KeyMiscAssistantShape, "轮廓：", "Shape: ")
	Register(KeyMiscAssistantColor, "颜色：", "Color: ")
	Register(KeyMiscAssistantDraft, "草稿：", "Draft: ")
	Register(KeyMiscAssistantDraftAvatarPreview, "头像草稿", "Draft avatar preview")
	Register(KeyMiscAssistantDraftImageCrop, "草稿图片显示：", "Draft image crop: ")
	Register(KeyMiscAssistantDiscardAvatarDraft, "丢弃头像草稿", "Discard avatar draft")
	Register(KeyMiscAssistantReviewAvatar, "审阅头像", "Review avatar")
	Register(KeyMiscAssistantAdoptMatchingAvatarReadback, "采用匹配头像读回", "Adopt matching avatar readback")
	Register(KeyMiscAssistantConfirmAvatarReplacement, "确认头像替换", "Confirm avatar replacement")
	Register(KeyMiscAssistantBeforeAvatarPreview, "原头像", "Before avatar preview")
	Register(KeyMiscAssistantAfter, "新头像：", "After: ")
	Register(KeyMiscAssistantAfterAvatarPreview, "新头像", "After avatar preview")
	Register(KeyMiscAssistantCancelAvatarSave, "取消头像保存", "Cancel avatar save")
	Register(KeyMiscAssistantConfirmAvatarSave, "确认保存头像", "Confirm avatar save")
	Register(KeyMiscAssistantChooseLocalAvatarImage, "选择本地头像图片", "Choose local avatar image")
	Register(KeyMiscAssistantLocalImageCrop, "本地图片裁剪", "Local image crop")
	Register(KeyMiscAssistantLocalCropPreview, "本地裁剪预览", "Local crop preview")
	Register(KeyMiscAssistantCrop, "裁剪显示：", "Crop: ")
	Register(KeyMiscAssistantApplyCropToDraft, "应用裁剪到草稿", "Apply crop to draft")
	Register(KeyMiscAssistantCancelImageSelection, "取消图片选择", "Cancel image selection")
	Register(KeyMiscAssistantIUnderstandCapabilityAndObjectiveEx, "我理解能力范围及已有目标可能恢复执行的影响", "I understand capability and objective execution effects")
	Register(KeyMiscAssistantBotCapabilities, "Bot 能力范围", "Bot capabilities")
	Register(KeyMiscAssistantRefreshCapabilities, "刷新能力与资料", "Refresh capabilities")
	Register(KeyMiscAssistantHostWarning, "上次 Host 警告：", "Host warning: ")
	Register(KeyMiscAssistantReadBackBeforeSaving2, "先刷新核实；未确认结果时草稿锁定，禁止再次保存", "Read back before saving")
	Register(KeyMiscAssistantWriteAuthorityUnavailable2, "Host 设置只读或写权限无法核实", "Write authority unavailable")
	Register(KeyMiscAssistantCapabilityServiceInactive, "能力服务未激活，无法编辑", "Capability service inactive")
	Register(KeyMiscAssistantSkillsServiceUnavailable, "Skills 服务不可用；不能保存非空选定 Skills", "Skills service unavailable")
	Register(KeyMiscAssistantUnavailableRemoveBeforeSaving, "已不在发现列表；选定模式须移除后保存", "Unavailable; remove before saving")
	Register(KeyMiscAssistantDiscardCapabilityDraft, "丢弃能力草稿", "Discard capability draft")
	Register(KeyMiscAssistantReviewCapabilityChanges, "审阅能力变更", "Review capability changes")
	Register(KeyMiscAssistantReviewCapabilitySave, "确认能力资料保存", "Review capability save")
	Register(KeyMiscAssistantUnknownOutcomesRequireReadback, "未知回包先读回权威资料，草稿保持锁定，不自动重试。关闭或切换 Host/Bot 会清除此局部草稿和确认。", "Unknown outcomes require readback.")
	Register(KeyMiscAssistantCancelCapabilitySave, "取消能力保存", "Cancel capability save")
	Register(KeyMiscAssistantConfirmCapabilitySave, "确认保存能力", "Confirm capability save")
	Register(KeyMiscAssistantIUnderstandBotAndSessionEffects, "我理解 Bot 创建与会话影响", "I understand Bot and session effects")
	Register(KeyMiscAssistantConfirmCreateAndOpenBot, "确认创建并打开 Bot", "Confirm create and open Bot")
	Register(KeyMiscAssistantApplicationDefault, "继承应用默认模型", "Application default")
	Register(KeyMiscAssistantResolvedByHost, "由 Host 解析", "Resolved by Host")
	Register(KeyMiscAssistantCreatePersonalBot, "新建个人 Bot", "Create personal Bot")
	Register(KeyMiscAssistantCreateABotWithTheApplicationDefaul, "创建 Bot 及专属会话，可继承应用默认模型，或选择当前 Host 已加载的模型。", "Create a Bot with the application default or a cached Host model.")
	Register(KeyMiscAssistantRefreshBotCreationCatalog, "刷新 Bot 创建目录", "Refresh Bot creation catalog")
	Register(KeyMiscAssistantReadCurrentCatalogAndAuthorityFirst, "先读取完整目录、revision 与写权限", "Read current catalog and authority first")
	Register(KeyMiscAssistantWriteAuthorityUnavailable3, "Host 写权限未核实或目录只读", "Write authority unavailable")
	Register(KeyMiscAssistantOpenCreatedBot, "打开已创建 Bot", "Open created Bot")
	Register(KeyMiscAssistantNewPersonalBot, "新建个人 Bot", "New personal Bot")
	Register(KeyMiscAssistantStableBotName120, "稳定名称 (≤120)", "Stable Bot name (≤120)")
	Register(KeyMiscAssistantDisplayTitle120, "显示标题 (≤120)", "Display title (≤120)")
	Register(KeyMiscAssistantDescription12000, "Bot 说明 (≤12000)", "Description (≤12000)")
	Register(KeyMiscAssistantMemoryReview, "允许后续记忆复盘", "Memory review")
	Register(KeyMiscAssistantNotifications, "允许后续完成通知", "Notifications")
	Register(KeyMiscAssistantAdoptInspectedBot, "采用核对后的 Bot", "Adopt inspected Bot")
	Register(KeyMiscAssistantDiscardBotDraft, "丢弃 Bot 草稿", "Discard Bot draft")
	Register(KeyMiscAssistantReviewBotCreation, "审阅 Bot 创建", "Review Bot creation")
	Register(KeyMiscAssistantReviewNewBot, "创建前完整审阅", "Review new Bot")
	Register(KeyMiscAssistantBackToBotDraft, "返回修改 Bot", "Back to Bot draft")
	Register(KeyMiscAssistantScratchDirectory, "临时目录", "Scratch directory")
	Register(KeyMiscAssistantPreserve, "保留当前工作区：", "Preserve: ")
	Register(KeyMiscAssistantWorkspace, "工作区", "Workspace")
	Register(KeyMiscAssistantRegistryNotLoadedKeepTheCurrentWor, "工作区目录尚未加载，可保留当前选择或使用临时目录。", "Registry not loaded; keep the current workspace or use a scratch directory.")
	Register(KeyMiscAssistantModel, "模型", "Model")
	Register(KeyMiscAssistantCatalogNotLoadedKeepTheCurrentChoi, "模型目录尚未加载，保留当前选择或继承应用默认。", "Catalog not loaded; keep the current choice or inherit the application default.")
	Register(KeyMiscAssistantModelDefaultEffort, "模型默认推理强度", "Model default effort")
	Register(KeyMiscAssistantPreserve2, "保留推理强度：", "Preserve: ")
	Register(KeyMiscAssistantReasoningEffort, "推理强度", "Reasoning effort")
	Register(KeyMiscAssistantPermanentlyDeleteReviewedBot, "永久删除此 Bot 及所列关联数据", "Permanently delete reviewed Bot")
	Register(KeyMiscAssistantIConfirmIrreversibleDeletionNow, "我当次确认上述不可恢复删除和全部级联影响", "I confirm irreversible deletion now")
	Register(KeyMiscAssistantDeleteSelectedBotOrRoom, "删除选中 Bot 或群聊", "Delete selected Bot or room")
	Register(KeyMiscAssistantReadDeletionPreviewOrOutcome, "只读刷新删除预览或结果", "Read deletion preview or outcome")
	Register(KeyMiscAssistantFullCurrentPreviewAndAuthorityRequi, "必须读取当前完整预览和写权限", "Full current preview and authority required")
	Register(KeyMiscAssistantWriteAuthorityUnavailable4, "Host 只读或写权限未核实", "Write authority unavailable")
	Register(KeyMiscAssistantReviewDestructiveDeletion, "审阅危险删除", "Review destructive deletion")
	Register(KeyMiscAssistantDestructiveConfirmationCancelByDefau, "删除前当次确认，默认取消", "Destructive confirmation; cancel by default")
	Register(KeyMiscAssistantTypeExactStableName, "输入上方精确稳定名称", "Type exact stable name")
	Register(KeyMiscAssistantIConfirmThisSourceAndCopyEffects, "我确认复制此来源与所列资料", "I confirm this source and copy effects")
	Register(KeyMiscAssistantIConfirmCopiedGoalExecutionEffects, "我确认副本目标可能执行并产生费用或消息", "I confirm copied goal execution effects")
	Register(KeyMiscAssistantConfirmDuplicateBot, "确认复制个人 Bot", "Confirm duplicate Bot")
	Register(KeyMiscAssistantDuplicateSelectedPersonalBot, "复制选中个人 Bot", "Duplicate selected personal Bot")
	Register(KeyMiscAssistantReadDuplicateSource, "读取 Bot 复制来源", "Read duplicate source")
	Register(KeyMiscAssistantReadTheFullCurrentSourceAndWriteA, "请读回完整来源、目录 revision 和写权限", "Read the full current source and write authority")
	Register(KeyMiscAssistantWriteAuthorityUnavailable5, "Host 只读或权限未核实", "Write authority unavailable")
	Register(KeyMiscAssistantOpenReceiptLinkedCopy, "确认打开回执中的副本", "Open receipt-linked copy")
	Register(KeyMiscAssistantPrepareDuplicate, "准备复制此 Bot", "Prepare duplicate")
	Register(KeyMiscAssistantNewStableName120, "副本稳定名称 (≤120)", "New stable name (≤120)")
	Register(KeyMiscAssistantNoMatchingCandidateDoesNotEstablish, "未找到相符候选不证明没有复制，也不允许重试", "No matching candidate does not establish failure")
	Register(KeyMiscAssistantOpenInspectedCandidate, "确认打开目录候选：", "Open inspected candidate: ")
	Register(KeyMiscAssistantDiscardDuplicateDraft, "丢弃复制草稿", "Discard duplicate draft")
	Register(KeyMiscAssistantReviewDuplicate, "审阅 Bot 复制", "Review duplicate")
	Register(KeyMiscAssistantReviewDuplicate2, "复制前完整审阅", "Review duplicate")
	Register(KeyMiscAssistantBackToDuplicateDraft, "返回修改复制", "Back to duplicate draft")
	Register(KeyMiscAssistantBotLongTermMemory, "Bot 长期记忆", "Bot long-term memory")
	Register(KeyMiscAssistantRefreshBeforeEditingOrRetrying, "请读取 Host 当前记忆；下方保留内容不是新的权威回执", "Refresh before editing or retrying")
	Register(KeyMiscAssistantReadOnlyOrWriteAuthorityUnavailable, "Host 写权限未验证或目录只读；可查看，不开放编辑", "Read-only or write authority unavailable")
	Register(KeyMiscAssistantBot, "Bot 经验", "bot")
	Register(KeyMiscAssistantUser, "此 Bot 的用户信息", "user")
	Register(KeyMiscAssistantNoParsedEntries, "暂无可编辑条目", "No parsed entries")
	Register(KeyMiscAssistantAddMemoryEntry, "添加记忆条目", "Add memory entry")
	Register(KeyMiscAssistantDraft2, "保留草稿", "Draft · ")
	Register(KeyMiscAssistantEntryToRemove, "拟删除条目：", "Entry to remove: ")
	Register(KeyMiscAssistantReviewMemorySave, "审阅保存记忆", "Review memory save")
	Register(KeyMiscAssistantDiscardDraft, "放弃草稿并采用已读取内容", "Discard draft")
	Register(KeyMiscAssistantReviewMemoryReplacement, "确认替换记忆", "Review memory replacement")
	Register(KeyMiscAssistantBackToMemoryEdit, "返回记忆编辑", "Back to memory edit")
	Register(KeyMiscAssistantConfirmTrackReplacement, "确认替换此轨记忆", "Confirm track replacement")
	Register(KeyMiscAssistantIUnderstandModelMemoryAndObjective, "我理解模型读取、记忆改写及目标同步影响", "I understand model, memory and objective effects")
	Register(KeyMiscAssistantEnabled, "开启", "enabled")
	Register(KeyMiscAssistantDisabled, "关闭", "disabled")
	Register(KeyMiscAssistantAutomaticMemoryLearning, "自主学习记忆", "Automatic memory learning")
	Register(KeyMiscAssistantRefreshMemoryLearning, "刷新自动记忆配置", "Refresh memory learning")
	Register(KeyMiscAssistantReadBackBeforeSavingAgain, "先读回核实；未知结果下草稿保持锁定", "Read back before saving again")
	Register(KeyMiscAssistantWriteAuthorityUnavailable6, "Host 设置只读或写权限未核实", "Write authority unavailable")
	Register(KeyMiscAssistantAllowAutomaticMemoryLearning, "允许自动整理记忆", "Allow automatic memory learning")
	Register(KeyMiscAssistantDiscardMemoryLearningDraft, "丢弃自动记忆草稿", "Discard memory learning draft")
	Register(KeyMiscAssistantReviewMemoryLearning, "审阅自动记忆配置", "Review memory learning")
	Register(KeyMiscAssistantReviewMemoryLearning2, "确认自动记忆配置", "Review memory learning")
	Register(KeyMiscAssistantCancelMemoryLearning, "取消自动记忆保存", "Cancel memory learning")
	Register(KeyMiscAssistantConfirmMemoryLearning, "确认保存自动记忆配置", "Confirm memory learning")
	Register(KeyMiscAssistantAllowOriginalBotNotifications, "允许原版 Bot 通知", "Allow original Bot notifications")
	Register(KeyMiscAssistantIUnderstandNotificationScopeAndProf, "我理解通知范围与完整资料保存的执行影响", "I understand notification scope and profile-save execution effects")
	Register(KeyMiscAssistantEnabled2, "notify=true · 允许", "enabled")
	Register(KeyMiscAssistantDisabled2, "notify=false · 关闭", "disabled")
	Register(KeyMiscAssistantPersonalBotNotificationSetting, "个人 Bot 通知配置", "Personal Bot notification setting")
	Register(KeyMiscAssistantRefreshNotificationSetting, "刷新通知配置", "Refresh notification setting")
	Register(KeyMiscAssistantReadBackBeforeSaving3, "先读回核实；未知结果保留并锁定，只读回，不自动重试", "Read back before saving")
	Register(KeyMiscAssistantBooleanUnavailable, "notify 字段无法核实，不猜默认值，请用 Bot Web", "Boolean unavailable")
	Register(KeyMiscAssistantDiscardNotificationDraft, "丢弃通知草稿", "Discard notification draft")
	Register(KeyMiscAssistantReviewNotificationSetting, "审阅通知配置", "Review notification setting")
	Register(KeyMiscAssistantAdoptMatchingNotificationReadback, "采用匹配通知读回", "Adopt matching notification readback")
	Register(KeyMiscAssistantConfirmNotificationSetting, "确认通知配置", "Confirm notification setting")
	Register(KeyMiscAssistantCancelNotificationSave, "取消通知保存", "Cancel notification save")
	Register(KeyMiscAssistantConfirmNotificationSetting2, "确认保存通知配置", "Confirm notification setting")
	Register(KeyMiscAssistantPauseObjective, "暂停目标推进…", "Pause objective…")
	Register(KeyMiscAssistantClearCatalogObjective, "清除目录目标…", "Clear catalog objective…")
	Register(KeyMiscAssistantLiveGoalNotIncludedRefresh, "该回执未包含 live 目标状态；请刷新核实", "Live goal not included; refresh")
	Register(KeyMiscAssistantNoVisibleLiveGoalInThisSnapshot, "Host 本次未报告可见 live 目标；冷会话可能仍有目录目标，不能据此判定已停止", "No visible live goal in this snapshot")
	Register(KeyMiscAssistantBotObjective, "Bot 目标循环", "Bot objective")
	Register(KeyMiscAssistantRefreshObjective, "刷新 Bot 目标状态", "Refresh objective")
	Register(KeyMiscAssistantHostWarning2, "上次操作的 Host 警告：", "Host warning: ")
	Register(KeyMiscAssistantRefreshBeforeAnotherAction, "请刷新核实后再操作；旧快照或回执不代表实时状态", "Refresh before another action")
	Register(KeyMiscAssistantActivityWarning, "Host 活动读取提示：", "Activity warning: ")
	Register(KeyMiscAssistantReview, "确认：", "Review: ")
	Register(KeyMiscAssistantRefreshUncertainOutcomesNeverAutoRe, "回包不明时先刷新查看权威状态；不自动重发，也不把刷新当作再次执行。", "Refresh uncertain outcomes; never auto-retry.")
	Register(KeyMiscAssistantCancelObjectiveAction, "取消目标操作", "Cancel objective action")
	Register(KeyMiscAssistantConfirmObjectiveAction, "确认目标操作", "Confirm objective action")
	Register(KeyMiscAssistantIUnderstandSavingMayStartOrResume, "我理解保存可能立即启动或恢复目标执行", "I understand saving may start or resume execution")
	Register(KeyMiscAssistantObjectiveConfiguration, "目标配置", "Objective configuration")
	Register(KeyMiscAssistantRefreshObjectiveConfiguration, "刷新目标配置", "Refresh objective configuration")
	Register(KeyMiscAssistantObjective, "目标", "Objective")
	Register(KeyMiscAssistantRoundCap1200, "轮数上限 (1–200)", "Round cap (1–200)")
	Register(KeyMiscAssistantDiscardObjectiveDraft, "丢弃目标配置草稿", "Discard objective draft")
	Register(KeyMiscAssistantReviewObjectiveConfiguration, "审阅目标配置", "Review objective configuration")
	Register(KeyMiscAssistantReviewObjectiveConfiguration2, "确认目标配置", "Review objective configuration")
	Register(KeyMiscAssistantCancelObjectiveConfiguration, "取消目标配置保存", "Cancel objective configuration")
	Register(KeyMiscAssistantConfirmObjectiveConfiguration, "确认保存目标配置", "Confirm objective configuration")
	Register(KeyMiscAssistantIUnderstandModelDirectoryNewSession, "我理解模型、目录、新会话及目标可能执行的影响", "I understand model, directory, new-session and execution effects")
	Register(KeyMiscAssistantHostDefault, "沿用 Host 默认模型", "Host default")
	Register(KeyMiscAssistantScratch, "无注册工作区：", "Scratch: ")
	Register(KeyMiscAssistantPathUnavailable, " · Host 路径未核实", "Path unavailable")
	Register(KeyMiscAssistantModelAndWorkspace, "模型与工作区", "Model and workspace")
	Register(KeyMiscAssistantRefreshModelAndWorkspace, "刷新模型与工作区", "Refresh model and workspace")
	Register(KeyMiscAssistantReadBackBeforeSaving4, "先读回核实；未知结果保留并锁定草稿，不自动重试", "Read back before saving")
	Register(KeyMiscAssistantIncompleteProfileUseBotWeb, "字段不完整，请使用 Bot Web", "Incomplete profile; use Bot Web")
	Register(KeyMiscAssistantModelDraft, "模型草稿：", "Model draft: ")
	Register(KeyMiscAssistantModelDiscoveryUnavailable, "模型目录不可用；保留原配置，不猜测默认模型", "Model discovery unavailable")
	Register(KeyMiscAssistantPreserveCurrent, "当前保留配置：", "Preserve current: ")
	Register(KeyMiscAssistantCurrentHostDefault, "Host 当前默认值：", "Current Host default: ")
	Register(KeyMiscAssistantPreserve3, "保留原强度：", "Preserve: ")
	Register(KeyMiscAssistantWorkspaceDraft, "目录草稿：", "Workspace draft: ")
	Register(KeyMiscAssistantPreserveCurrent2, "保留现有目录：", "Preserve current: ")
	Register(KeyMiscAssistantHostDirectoriesUnavailable, "真实 Host 目录选项未核实；目录更改禁用", "Host directories unavailable")
	Register(KeyMiscAssistantHistoryPresentOrUnknown, "workspaceLocked：存在消息或历史未知，不能改工作区；请在原入口新建 Bot", "History present or unknown")
	Register(KeyMiscAssistantDiscardRouteDraft, "丢弃模型/目录草稿", "Discard route draft")
	Register(KeyMiscAssistantReviewModelAndWorkspace, "审阅模型与目录", "Review model and workspace")
	Register(KeyMiscAssistantAdoptMatchingReadback, "采用匹配的读回资料", "Adopt matching readback")
	Register(KeyMiscAssistantConfirmModelAndWorkspace, "确认模型与工作区", "Confirm model and workspace")
	Register(KeyMiscAssistantCancelRouteSave, "取消模型/目录保存", "Cancel route save")
	Register(KeyMiscAssistantConfirmRouteSave, "确认保存模型/目录", "Confirm route save")
	Register(KeyMiscAssistantIUnderstandSourcePolicyAndExecution, "我理解来源规则和目标可能执行的影响", "I understand source-policy and execution effects")
	Register(KeyMiscAssistantIExplicitlyApproveTheAccessExpansio, "我明确允许下方列出的消息/任务权限扩大", "I explicitly approve the access expansion shown below")
	Register(KeyMiscAssistantNone, "无", "None")
	Register(KeyMiscAssistantHidden, "（已隐藏）", "(hidden)")
	Register(KeyMiscAssistantSelectedBots, "指定 Bot", "Selected Bots")
	Register(KeyMiscAssistantMessageAndTaskSources, "消息与任务来源", "Message and task sources")
	Register(KeyMiscAssistantRefreshSourcePolicy, "刷新消息与任务来源", "Refresh source policy")
	Register(KeyMiscAssistantIncompleteProfile, "字段不完整，请使用现有 Bot Web", "Incomplete profile")
	Register(KeyMiscAssistantDefaultRule, "默认规则", "Default rule")
	Register(KeyMiscAssistantDiscardSourceDraft, "丢弃来源草稿", "Discard source draft")
	Register(KeyMiscAssistantReviewSourcePolicy, "审阅消息与任务来源", "Review source policy")
	Register(KeyMiscAssistantAdoptSourceReadback, "采用匹配的来源读回", "Adopt source readback")
	Register(KeyMiscAssistantConfirmSourcePolicy, "确认消息与任务来源", "Confirm source policy")
	Register(KeyMiscAssistantRetainedObjective, "保留的目标：", "Retained objective: ")
	Register(KeyMiscAssistantCancelSourceSave, "取消来源保存", "Cancel source save")
	Register(KeyMiscAssistantConfirmSourceSave, "确认保存来源规则", "Confirm source save")
	Register(KeyMiscAssistantIUnderstandAddressPersonaPresentatio, "我理解寻址、人设、会话展示及目标可能执行的影响", "I understand address, persona, presentation and execution effects")
	Register(KeyMiscAssistantEmpty, "（空）", "(empty)")
	Register(KeyMiscAssistantNameDisplayTitleAndPersona, "名称、显示标题与人设", "Name, display title and persona")
	Register(KeyMiscAssistantRefreshTextProfile, "刷新文本资料", "Refresh text profile")
	Register(KeyMiscAssistantIncompleteProfileUseBotWeb2, "文本字段不完整，请使用现有 Bot Web 入口", "Incomplete profile; use Bot Web")
	Register(KeyMiscAssistantName120, "寻址名称 (120)", "Name (120)")
	Register(KeyMiscAssistantDisplayTitle120Optional, "显示标题 (120, optional)", "Display title (120, optional)")
	Register(KeyMiscAssistantCompleteDescriptionPersona12000, "完整描述/人设 (12000)", "Complete description/persona (12000)")
	Register(KeyMiscAssistantDiscardTextDraft, "丢弃文本草稿", "Discard text draft")
	Register(KeyMiscAssistantReviewTextProfile, "审阅文本资料", "Review text profile")
	Register(KeyMiscAssistantAdoptMatchingTextReadback, "采用匹配文本读回", "Adopt matching text readback")
	Register(KeyMiscAssistantConfirmCompleteTextProfile, "确认完整文本资料", "Confirm complete text profile")
	Register(KeyMiscAssistantCancelTextProfileSave, "取消文本资料保存", "Cancel text profile save")
	Register(KeyMiscAssistantConfirmTextProfileSave, "确认保存文本资料", "Confirm text profile save")
	Register(KeyMiscAssistantRoom, "群聊", "Room")
	Register(KeyMiscAssistantTargetAwaitsFreshCatalog, "通知目标待本次完整目录核实", "Target awaits fresh catalog")
	Register(KeyMiscAssistantExactTargetNotFound, "目标未在当前目录精确匹配；不会选择其它项", "Exact target not found")
	Register(KeyMiscAssistantExactTarget, "精确定位：", "Exact target: ")
	Register(KeyMiscAssistantShowHiddenLocally, "本地显示隐藏项", "Show hidden locally")
	Register(KeyMiscAssistantReadFullCatalog, "只读刷新完整目录", "Read full catalog")
	Register(KeyMiscAssistantAdoptExactReadback, "采用完整匹配读回", "Adopt exact readback")
	Register(KeyMiscAssistantCancel, "取消", "Cancel")
	Register(KeyMiscAssistantIConfirmThisTargetAndEffects, "我确认精确目标、完整变更和上述影响", "I confirm this target and effects")
	Register(KeyMiscAssistantConfirmUnhide, "确认恢复显示", "Confirm unhide")
	Register(KeyMiscAssistantConfirmHide, "确认隐藏", "Confirm hide")
	Register(KeyMiscAssistantReviewHide, "审阅隐藏", "Review hide")
	Register(KeyMiscAssistantReviewUnhide, "审阅恢复", "Review unhide")
	Register(KeyMiscAssistantNoMatchingItemsShowHiddenToRecover, "当前筛选无项目；可显示隐藏项", "No matching items; show hidden to recover")
	Register(KeyMiscAssistantContacts, "联系人目录", "Contacts")
	Register(KeyMiscAssistantRefresh, "刷新", "Refresh")
	Register(KeyMiscAssistantSearchContacts, "搜索名称、标题和说明", "Search contacts")
	Register(KeyMiscAssistantSource, "来源", "Source")
	Register(KeyMiscAssistantAllSources, "全部来源", "All sources")
	Register(KeyMiscAssistantNewSection, "新建分组", "New section")
	Register(KeyMiscAssistantRenameSection, "重命名分组", "Rename section")
	Register(KeyMiscAssistantSaveSection, "保存分组", "Save section")
	Register(KeyMiscAssistantRestoreSection, "撤销删除分组", "Restore section")
	Register(KeyMiscAssistantActive, "活跃", "Active")
	Register(KeyMiscAssistantUnreadCompletedResult, "有未读完成结果", "Unread completed result")
	Register(KeyMiscAssistantMarkRead, "标为已读", "Mark read")
	Register(KeyMiscAssistantNoMatchingContacts, "没有符合筛选的联系人", "No matching contacts")
	Register(KeyMiscAssistantFullHistoryStreamingRepliesApprovals, "完整消息历史、流式回复、工具审批与停止请在主会话中查看。", "Full history, streaming replies, approvals and stop live in the main conversation.")
	Register(KeyMiscAssistantSelect, "选择", "Select")
	Register(KeyMiscAssistantOpenTheCanonicalChatToSendFollowR, "打开专属主会话并发送；回复、审批和停止均在主聊天中。", "Open the canonical chat to send, follow replies, approve tools and stop.")
	Register(KeyMiscAssistantIUnderstandRoomAndSessionEffects, "我理解创建会话与后续群聊执行影响", "I understand room and session effects")
	Register(KeyMiscAssistantConfirmCreateAndOpen, "确认创建并打开群聊", "Confirm create and open")
	Register(KeyMiscAssistantUnavailable, "不可用", "unavailable")
	Register(KeyMiscAssistantCreateBotGroup, "新建 Bot 群聊", "Create Bot group")
	Register(KeyMiscAssistantRefreshCreationCatalog, "刷新群聊创建目录", "Refresh creation catalog")
	Register(KeyMiscAssistantReadTheCurrentCatalogAndWriteAutho, "先读取完整目录、revision 与写权限", "Read the current catalog and write authority first")
	Register(KeyMiscAssistantWriteAuthorityUnavailable7, "Host 目录只读或写权限未核实", "Write authority unavailable")
	Register(KeyMiscAssistantOpenCreatedGroup, "打开已有群聊", "Open created group")
	Register(KeyMiscAssistantNewGroup, "新建群聊", "New group")
	Register(KeyMiscAssistantGroupName120, "群聊名称 (≤120)", "Group name (≤120)")
	Register(KeyMiscAssistantRounds13, "轮数 (1–3)", "Rounds (1–3)")
	Register(KeyMiscAssistantSpeaks110, "成员发言上限 (1–10)", "Speaks (1–10)")
	Register(KeyMiscAssistantDescription120002, "群聊说明 (≤12000)", "Description (≤12000)")
	Register(KeyMiscAssistantRemoveUnavailableMember, "已选成员已不可用，请移除：", "Remove unavailable member: ")
	Register(KeyMiscAssistantUseInspectedGroup, "采用核对后的群聊", "Use inspected group")
	Register(KeyMiscAssistantDiscardGroupDraft, "丢弃群聊草稿", "Discard group draft")
	Register(KeyMiscAssistantReviewGroupCreation, "审阅群聊创建", "Review group creation")
	Register(KeyMiscAssistantReviewNewGroup, "创建前完整审阅", "Review new group")
	Register(KeyMiscAssistantBackToGroupDraft, "返回修改群聊", "Back to group draft")
	Register(KeyMiscAssistantPermanentlyDisbandReviewedRoom, "永久解散此群聊及所列关联数据", "Permanently disband reviewed room")
	Register(KeyMiscAssistantCancelAndKeepRoom, "取消，保留此群聊", "Cancel and keep room")
	Register(KeyMiscAssistantTypeExactRoomId, "输入上方精确群聊 ID", "Type exact room ID")
	Register(KeyMiscAssistantCancelAndKeepBot, "取消，保留此 Bot", "Cancel and keep Bot")
	Register(KeyMiscAssistantTypeExactBotId, "输入上方精确 Bot ID", "Type exact Bot ID")
	Register(KeyMiscAssistantIUnderstandGroupProfileAndSessionE, "我理解群资料及会话展示变更的影响", "I understand group profile and session effects")
	Register(KeyMiscAssistantConfirmGroupUpdate, "确认保存群聊设置", "Confirm group update")
	Register(KeyMiscAssistantGroupProfileAndLimits, "群名、说明、成员与限额", "Group profile and limits")
	Register(KeyMiscAssistantReadGroupProfile, "读取群聊资料", "Read group profile")
	Register(KeyMiscAssistantReadCompleteProfileAndRevisionFirst, "先读取完整资料与 revision；旧草稿不会自动覆盖 Host", "Read complete profile and revision first")
	Register(KeyMiscAssistantEditGroupSettings, "编辑群聊设置", "Edit group settings")
	Register(KeyMiscAssistantRemovingAndSelectingAgainMovesAMem, "成员保持选择顺序；移除后再选择会排到末尾", "Removing and selecting again moves a member to the end")
	Register(KeyMiscAssistantRemoveUnavailableMember2, "移除不可用成员：", "Remove unavailable member: ")
	Register(KeyMiscAssistantAdoptExactReadback2, "采用完整匹配的读回", "Adopt exact readback")
	Register(KeyMiscAssistantDiscardEditDraft, "丢弃群聊编辑草稿", "Discard edit draft")
	Register(KeyMiscAssistantReviewGroupChanges, "审阅群聊变更", "Review group changes")
	Register(KeyMiscAssistantReviewGroupUpdate, "保存前完整审阅", "Review group update")
	Register(KeyMiscAssistantBackToDraft, "返回修改群聊设置", "Back to draft")
	Register(KeyMiscAssistantOpenMemberSession, "打开此成员会话", "Open member session")
	Register(KeyMiscAssistantGroupMemberRuntime, "群聊成员运行状态", "Group member runtime")
	Register(KeyMiscAssistantRefreshMemberRuntime, "刷新成员运行状态", "Refresh member runtime")
	Register(KeyMiscAssistantCurrentRuntimeUnverifiedAnyPrevious, "当前状态未核实，已有快照也不是当前运行结果", "Current runtime unverified; any previous snapshot is stale")
	Register(KeyMiscAssistantNoVisibleMemberRuntimeThisIsNotA, "没有可见成员运行记录；不代表成员已完成或健康", "No visible member runtime; this is not a completion or health receipt")
	Register(KeyMiscAssistantUnknownMember, "身份未知", "Unknown member")
	Register(KeyMiscAssistantUnknown, "未知", "unknown")
	Register(KeyMiscAssistantNeedsAttention, "需要留意", "Needs attention")
	Register(KeyMiscAssistantEmpty2, "（空）", "(empty)")
	Register(KeyMiscAssistantRoomName120Utf16, "群名 (≤120 UTF-16)", "Room name (≤120 UTF-16)")
	Register(KeyMiscAssistantCompleteRoomDescription12000Utf16, "完整群说明 (≤12000 UTF-16)", "Complete room description (≤12000 UTF-16)")
	Register(KeyMiscAssistantIUnderstandCommandEffectsAndWorking, "我理解命令影响及目录回退，允许按此配置自动执行", "I understand command effects and working-directory fallback")
	Register(KeyMiscAssistantThisRoutineSPreviousOutputSelf, "本任务上一轮输出 (self)", "This routine's previous output (self)")
	Register(KeyMiscAssistantUnavailableSourceRemoveExplicitlyBef, "来源已不可用；保存前须明确移除：", "Unavailable source; remove explicitly before saving: ")
	Register(KeyMiscAssistantAdvancedSettings, "高级配置", "Advanced settings")
	Register(KeyMiscAssistantAllowSilentResponse, "允许静默完成", "Allow silent response")
	Register(KeyMiscAssistantUnknownWatchPreserved, "未知 watch 配置保持原样；明确选择类型才替换：", "Unknown watch preserved: ")
	Register(KeyMiscAssistantNoWatch, "不检查", "No watch")
	Register(KeyMiscAssistantUrlWatch, "读取 URL", "URL watch")
	Register(KeyMiscAssistantCommandWatch, "执行命令", "Command watch")
	Register(KeyMiscAssistantFullWatchUrl, "完整 URL", "Full watch URL")
	Register(KeyMiscAssistantFullHostShellCommand, "完整 Host shell 命令", "Full Host shell command")
	Register(KeyMiscAssistantTimeoutMs1300000, "超时毫秒 (1–300000)", "Timeout ms (1–300000)")
	Register(KeyMiscAssistantEffectiveWatchNone, "有效 watch", "Effective watch: none")
	Register(KeyMiscAssistantNotConfigured, "（未配置）", "(not configured)")
	Register(KeyMiscAssistantConfirmRoutineAction, "确认例行任务操作", "Confirm routine action")
	Register(KeyMiscAssistantCreateBotRoutine, "新建 Bot 例行任务", "Create Bot routine")
	Register(KeyMiscAssistantNewRoutine, "新建例行任务", "New routine")
	Register(KeyMiscAssistantRefreshTheRoutineCatalogFirstEmpty, "先刷新上方例行任务列表；空列表也可创建", "Refresh the routine catalog first; empty catalogs are supported")
	Register(KeyMiscAssistantSaveScheduleContextSilentRepliesAnd, "创建时一次保存调度、上下文、静默回复及 watch 配置。", "Save schedule, context, silent replies and watch together.")
	Register(KeyMiscAssistantFrequency, "类型：", "Frequency: ")
	Register(KeyMiscAssistantNewRoutineName, "新任务名称", "New routine name")
	Register(KeyMiscAssistantMinuteHourDayMonthWeekday, "五字段 Cron", "minute hour day month weekday")
	Register(KeyMiscAssistantIntervalAmountPositiveInteger, "间隔数量 (positive integer)", "Interval amount (positive integer)")
	Register(KeyMiscAssistantIntervalUnit, "间隔单位：", "Interval unit: ")
	Register(KeyMiscAssistantMinutes, "分钟", "Minutes")
	Register(KeyMiscAssistantHours, "小时", "Hours")
	Register(KeyMiscAssistantDays, "天", "Days")
	Register(KeyMiscAssistantHour0023, "小时 (00–23)", "Hour (00–23)")
	Register(KeyMiscAssistantMinute0059, "分钟 (00–59)", "Minute (00–59)")
	Register(KeyMiscAssistantWeekday, "星期：", "Weekday: ")
	Register(KeyMiscAssistantSunday, " (0=周日)", " (0=Sunday)")
	Register(KeyMiscAssistantSun, "周日", "Sun")
	Register(KeyMiscAssistantMon, "周一", "Mon")
	Register(KeyMiscAssistantTue, "周二", "Tue")
	Register(KeyMiscAssistantWed, "周三", "Wed")
	Register(KeyMiscAssistantThu, "周四", "Thu")
	Register(KeyMiscAssistantFri, "周五", "Fri")
	Register(KeyMiscAssistantSat, "周六", "Sat")
	Register(KeyMiscAssistantDayOfMonth131, "每月日期 (1–31)", "Day of month (1–31)")
	Register(KeyMiscAssistantMaxRuns0Unlimited, "执行次数上限 (0 = unlimited)", "Max runs (0 = unlimited)")
	Register(KeyMiscAssistantSpecifyTheIntendedIanaTimezone, "原版按浏览器本地时区安排；无法识别本机IANA时区时须明确填写，不会猜测。", "Specify the intended IANA timezone.")
	Register(KeyMiscAssistantNewRoutinePrompt, "新任务提示词", "New routine prompt")
	Register(KeyMiscAssistantNewRoutinePaused, "新任务：暂停", "New routine paused")
	Register(KeyMiscAssistantNewRoutineEnabled, "新任务：启用", "New routine enabled")
	Register(KeyMiscAssistantThisCreationIsLockedUntilItsOutcom, "同一创建请求已锁定。刷新只查询目录，不会重复提交；核对新记录后可采用它。", "This creation is locked until its outcome is checked.")
	Register(KeyMiscAssistantCheckCreationOutcome, "核对创建结果", "Check creation outcome")
	Register(KeyMiscAssistantUseRecord, "采用这条记录：", "Use record: ")
	Register(KeyMiscAssistantNoNewRecordReviewAgain, "未发现新记录，重新审阅", "No new record; review again")
	Register(KeyMiscAssistantDiscardNewDraft, "放弃新建草稿", "Discard new draft")
	Register(KeyMiscAssistantRecheckCatalog, "采用当前目录基线", "Recheck catalog")
	Register(KeyMiscAssistantReviewCreation, "审阅新建…", "Review creation…")
	Register(KeyMiscAssistantReviewCreation2, "确认创建", "Review creation")
	Register(KeyMiscAssistantBackToDraft2, "返回修改", "Back to draft")
	Register(KeyMiscAssistantConfirmNewRoutine, "确认保存新例行任务", "Confirm new routine")
	Register(KeyMiscAssistantDaily, "每日", "Daily")
	Register(KeyMiscAssistantHourly, "每小时", "Hourly")
	Register(KeyMiscAssistantWeekdays, "工作日", "Weekdays")
	Register(KeyMiscAssistantWeekly, "每周", "Weekly")
	Register(KeyMiscAssistantMonthly, "每月", "Monthly")
	Register(KeyMiscAssistantAdvanced, "高级 Cron", "Advanced")
	Register(KeyMiscAssistantOnce, "延迟一次", "Once")
	Register(KeyMiscAssistantInterval, "固定间隔", "Interval")
	Register(KeyMiscAssistantIApproveObtainingPersistentHostCata, "我理解可能创建当前 Host 的 Bot 目录持久共享触发凭据，确认获取", "I approve obtaining persistent Host catalog trigger access")
	Register(KeyMiscAssistantCurrentHostBotCatalogTriggerCredent, "当前 Host 的 Bot 目录触发凭据", "Current Host Bot catalog trigger credential")
	Register(KeyMiscAssistantReviewHostCatalogCredential, "审阅获取当前 Host 的 Bot 目录触发凭据…", "Review Host catalog credential…")
	Register(KeyMiscAssistantConfirmHostCatalogCredential, "确认获取/生成 Host 目录凭据", "Confirm Host catalog credential")
	Register(KeyMiscAssistantWaitingForHostClosingDoesNotRevoke, "等待 Host；不会自动重试。已发出请求不能靠关闭撤销，Host 可能已保存凭据。", "Waiting for Host; closing does not revoke an issued request")
	Register(KeyMiscAssistantCopyTriggerAddress, "复制触发地址", "Copy trigger address")
	Register(KeyMiscAssistantCopySecretHeader, "复制敏感鉴权头", "Copy secret header")
	Register(KeyMiscAssistantCloseAndClearCredential, "关闭并清除本页凭据", "Close and clear credential")
	Register(KeyMiscAssistantReviewSave, "核对并保存…", "Review save…")
	Register(KeyMiscAssistantPauseRoutine, "暂停例行任务…", "Pause routine…")
	Register(KeyMiscAssistantEnableRoutine, "启用例行任务…", "Enable routine…")
	Register(KeyMiscAssistantRunNow, "立即运行…", "Run now…")
	Register(KeyMiscAssistantDeleteRoutine, "删除例行任务…", "Delete routine…")
	Register(KeyMiscAssistantBotRoutines, "Bot 例行任务", "Bot routines")
	Register(KeyMiscAssistantRefreshRoutines, "刷新例行任务", "Refresh routines")
	Register(KeyMiscAssistantSelectABotFirst, "先选择当前 Host 的 Bot", "Select a Bot first")
	Register(KeyMiscAssistantRefreshBeforeMakingChanges, "请刷新读取当前 Host。未确认结果前不开放写操作", "Refresh before making changes.")
	Register(KeyMiscAssistantWaitingForHost, "等待 Host…", "Waiting for Host…")
	Register(KeyMiscAssistantNoRoutinesForThisBot, "此 Bot 暂无例行任务", "No routines for this Bot")
	Register(KeyMiscAssistantDismissMissedRunNotice, "知道了", "Dismiss missed-run notice")
	Register(KeyMiscAssistantSearchRoutinesBotsOrErrors, "搜索定时任务、机器人或错误", "Search routines, bots, or errors")
	Register(KeyMiscAssistantNoMatchingRoutines, "没有匹配的定时任务", "No matching routines")
	Register(KeyMiscAssistantPaused, "暂停", "Paused")
	Register(KeyMiscAssistantEnabled3, "启用", "Enabled")
	Register(KeyMiscAssistantMissedScheduledRun, "Host 标记为错过执行", "Missed scheduled run")
	Register(KeyMiscAssistantPrompt, "提示词：", "Prompt: ")
	Register(KeyMiscAssistantLastOutput, "上次输出：", "Last output: ")
	Register(KeyMiscAssistantEditRoutine, "编辑例行任务", "Edit routine")
	Register(KeyMiscAssistantRunHistoryLatest10, "运行历史 (latest 10)", "Run history (latest 10)")
	Register(KeyMiscAssistantHistoryUnavailable, "Host 未提供运行历史", "History unavailable")
	Register(KeyMiscAssistantNoRecordedRuns, "暂无运行记录", "No recorded runs")
	Register(KeyMiscAssistantBackToRoutine, "返回", "Back to routine")
	Register(KeyMiscAssistantUnlimited, "不限", "Unlimited")
	Register(KeyMiscAssistantDraftRevisionIsStale, "草稿基线已过期；核对当前记录后可丢弃并重新编辑。", "Draft revision is stale.")
	Register(KeyMiscAssistantRoutineName, "名称", "Routine name")
	Register(KeyMiscAssistantSchedule, "时间规则", "Schedule")
	Register(KeyMiscAssistantTimezone, "时区", "Timezone")
	Register(KeyMiscAssistantRoutinePrompt, "例行任务提示词", "Routine prompt")
	Register(KeyMiscAssistantDraftPaused, "草稿：暂停", "Draft paused")
	Register(KeyMiscAssistantDraftEnabled, "草稿：启用", "Draft enabled")
	Register(KeyMiscAssistantDiscardRoutineDraft, "丢弃草稿", "Discard routine draft")
	Register(KeyMiscAssistantPause, "暂停任务", "Pause")
	Register(KeyMiscAssistantResume, "恢复任务", "Resume")
	Register(KeyMiscAssistantCancelTask, "取消任务", "Cancel task")
	Register(KeyMiscAssistantRetryTask, "重试任务", "Retry task")
	Register(KeyMiscAssistantStopBot, "停止 Bot 会话", "Stop Bot")
	Register(KeyMiscAssistantBotTasks, "Bot 任务", "Bot tasks")
	Register(KeyMiscAssistantRefreshTasks, "刷新任务", "Refresh tasks")
	Register(KeyMiscAssistantWaitingForHost2, "正在等待 Host…", "Waiting for Host…")
	Register(KeyMiscAssistantRefreshBeforeMakingChanges2, "点击刷新加载任务。上次读取失败时不会开放写操作。", "Refresh before making changes.")
	Register(KeyMiscAssistantNoDelegationTasksForThisBot, "此 Bot 暂无委派任务", "No delegation tasks for this Bot")
	Register(KeyMiscAssistantSearchTasksParticipantsOrErrors, "搜索任务、参与者或错误", "Search tasks, participants, or errors")
	Register(KeyMiscAssistantTaskStatus, "任务状态", "Task status")
	Register(KeyMiscAssistantNoMatchingTasks, "没有匹配的任务", "No matching tasks")
	Register(KeyMiscAssistantOpenSenderConversation, "打开发件人会话", "Open sender conversation")
	Register(KeyMiscAssistantOpenRecipientConversation, "打开收件人会话", "Open recipient conversation")
	Register(KeyMiscAssistantTask, "任务：", "Task: ")
	Register(KeyMiscAssistantSuccessCriteria, "成功标准：", "Success criteria: ")
	Register(KeyMiscAssistantResult, "结果：", "Result: ")
	Register(KeyMiscAssistantError, "错误：", "Error: ")
	Register(KeyMiscAssistantRetryOf, "重试源：", "Retry of: ")
	Register(KeyMiscAssistantBack, "返回", "Back")
	Register(KeyMiscAssistantConfirmAction, "确认执行", "Confirm action")
	Register(KeyMiscAssistantWhalePreferences, "鲸鱼娘 · 模型、IM 与技能", "Whale preferences")
	Register(KeyMiscAssistantRefreshPreferences, "读取模型、IM 与技能", "Refresh preferences")
	Register(KeyMiscAssistantFollowSessionDefault, "跟随会话默认", "Follow session default")
	Register(KeyMiscAssistantCurrent, "当前配置：", "Current: ")
	Register(KeyMiscAssistantModelCatalog, "模型目录读取失败：", "Model catalog: ")
	Register(KeyMiscAssistantDefaultEffort, "默认推理档", "Default effort")
	Register(KeyMiscAssistantShareConversationWithIm, "IM 接入同一对话", "Share conversation with IM")
	Register(KeyMiscAssistantWhaleSettings, "鲸鱼娘 · 人设与长期记忆", "Whale settings")
	Register(KeyMiscAssistantRefreshBeforeEditingOrRetrying2, "请先读取当前 Host 内容。失败或结果不确定时，重新读取后才可提交。", "Refresh before editing or retrying.")
	Register(KeyMiscAssistantLongTermMemory, "长期记忆", "Long-term memory")
	Register(KeyMiscAssistantMemoryMdIsInjectedIntoWhaleSSessi, "MEMORY.md 随鲸鱼娘会话注入。清空会写入 Host 的默认记忆模板。", "MEMORY.md is injected into Whale's session.")
	Register(KeyMiscAssistantSaveMemory, "保存记忆", "Save memory")
	Register(KeyMiscAssistantRestoreMemoryDraft, "还原记忆草稿", "Restore memory draft")
	Register(KeyMiscAssistantClearMemory, "清空记忆…", "Clear memory…")
	Register(KeyMiscAssistantClearLongTermMemoryThisCannotBeUn, "确定清空她的长期记忆？此操作不可撤销。", "Clear long-term memory? This cannot be undone.")
	Register(KeyMiscAssistantCancelKeepsThisDraft, "确认后会替换当前 Host 的 MEMORY.md；取消保留当前草稿。", "Cancel keeps this draft.")
	Register(KeyMiscAssistantCancelClear, "取消清空", "Cancel clear")
	Register(KeyMiscAssistantConfirmClearMemory, "确认清空长期记忆", "Confirm clear memory")
	Register(KeyMiscAssistantExtraPersona, "额外人设", "Extra persona")
	Register(KeyMiscAssistantAppendedToHerPersonaLeaveBlankToS, "追加到人格末尾；留空即不用。", "Appended to her persona; leave blank to skip.")
	Register(KeyMiscAssistantSavePersona, "保存人设", "Save persona")
	Register(KeyMiscAssistantRestorePersonaDraft, "还原人设草稿", "Restore persona draft")
	Register(KeyMiscAssistantNameAndTitle, "名字与称呼", "Name and title")
	Register(KeyMiscAssistantName, "名字", "Name")
	Register(KeyMiscAssistantHowSheCallsYou, "对你的称呼", "How she calls you")
	Register(KeyMiscAssistantRestoreNameDraft, "还原名字草稿", "Restore name draft")
	Register(KeyMiscAssistantUnsavedDraft, "未保存草稿", "Unsaved draft")
	Register(KeyMiscAssistantMatchesHost, "与 Host 一致", "Matches Host")

	// browserpreview
	Register(KeyMiscBrowserpreviewHtmlPreviewUnavailableUseSystemOpen, "HTML 预览不可用，请使用系统打开", "HTML preview unavailable; use system open")

	// custompanel
	Register(KeyMiscCustompanelCustomPanels, "自定义面板", "Custom Panels")
	Register(KeyMiscCustompanelOpenABotSCanonicalConversation, "请打开 Bot 的主会话；面板列表按当前 Bot 隔离。", "Open a Bot's canonical conversation.")
	Register(KeyMiscCustompanelRequirements, "需求", "Requirements")
	Register(KeyMiscCustompanelCreateWithBot, "创建面板：与 Bot 对话", "Create with Bot")
	Register(KeyMiscCustompanelSendThePreparedDraftThenImportThe, "创建会填入主聊天草稿；请发送并与 Bot 讨论，再导入完成的 JSON。", "Send the prepared draft, then import the completed reply.")
	Register(KeyMiscCustompanelImportReply, "导入 Bot 回复", "Import reply")
	Register(KeyMiscCustompanelNewJson, "新建 JSON", "New JSON")
	Register(KeyMiscCustompanelNewTemplateAndBotCopy, "新模板与本 Bot 副本", "New template and Bot copy")
	Register(KeyMiscCustompanelEditThisBotSCopy, "仅修改本 Bot 副本", "Edit this Bot's copy")
	Register(KeyMiscCustompanelPreview, "预览", "Preview")
	Register(KeyMiscCustompanelSave, "保存", "Save")
	Register(KeyMiscCustompanelThisBotSPanels, "本 Bot 的面板", "This Bot's Panels")
	Register(KeyMiscCustompanelEdit, "编辑", "Edit")
	Register(KeyMiscCustompanelRemove, "移除", "Remove")
	Register(KeyMiscCustompanelUpdateLibraryTemplate, "用此副本更新库模板", "Update library template")
	Register(KeyMiscCustompanelPanelLibrary, "面板库", "Panel library")
	Register(KeyMiscCustompanelAddCopy, "添加独立副本", "Add copy")
	Register(KeyMiscCustompanelEditTemplate, "编辑模板", "Edit template")
	Register(KeyMiscCustompanelPrepareChatDraftDoesNotSend, "填入本 Bot 聊天草稿；不会自动发送", "Prepare chat draft; does not send")
}

// Review and detail templates: multi-line blocks that describe a Host
// record before a destructive or model-changing action.
const (
	KeyMiscIMCodeReference = "misc.im.code_reference"
	KeyMiscAssistantSavedAvatarDetail = "misc.assistant.saved_avatar_detail"
	KeyMiscAssistantBeforeAvatarDetail = "misc.assistant.before_avatar_detail"
	KeyMiscAssistantZoomPercent = "misc.assistant.zoom_percent"
	KeyMiscAssistantHorizontalOffset = "misc.assistant.horizontal_offset"
	KeyMiscAssistantVerticalOffset = "misc.assistant.vertical_offset"
	KeyMiscAssistantCapabilityReviewDetail = "misc.assistant.capability_review_detail"
	KeyMiscAssistantCreatedDetail = "misc.assistant.created_detail"
	KeyMiscAssistantScratchCwdRequired = "misc.assistant.scratch_cwd_required"
	KeyMiscAssistantExactMatchingRecord = "misc.assistant.exact_matching_record"
	KeyMiscAssistantPreserveModel = "misc.assistant.preserve_model"
	KeyMiscAssistantDeletePreviewDetail = "misc.assistant.delete_preview_detail"
	KeyMiscAssistantReceiptLinkedCopy = "misc.assistant.receipt_linked_copy"
	KeyMiscAssistantScratchCwdOverridden = "misc.assistant.scratch_cwd_overridden"
	KeyMiscAssistantUnattributedCandidate = "misc.assistant.unattributed_candidate"
	KeyMiscAssistantHostRawContent = "misc.assistant.host_raw_content"
	KeyMiscAssistantEditEntry = "misc.assistant.edit_entry"
	KeyMiscAssistantDeleteEntry = "misc.assistant.delete_entry"
	KeyMiscAssistantBeforeContent = "misc.assistant.before_content"
	KeyMiscAssistantAfterContent = "misc.assistant.after_content"
	KeyMiscAssistantMemoryReviewDetail = "misc.assistant.memory_review_detail"
	KeyMiscAssistantNotifyDetail = "misc.assistant.notify_detail"
	KeyMiscAssistantNotifyReviewDetail = "misc.assistant.notify_review_detail"
	KeyMiscAssistantCatalogObjectiveDetail = "misc.assistant.catalog_objective_detail"
	KeyMiscAssistantObjectiveReviewDetail = "misc.assistant.objective_review_detail"
	KeyMiscAssistantCurrentObjectiveDetail = "misc.assistant.current_objective_detail"
	KeyMiscAssistantObjectiveConfigDiff = "misc.assistant.objective_config_diff"
	KeyMiscAssistantRetainedConfig = "misc.assistant.retained_config"
	KeyMiscAssistantRouteDetail = "misc.assistant.route_detail"
	KeyMiscAssistantRouteReviewDetail = "misc.assistant.route_review_detail"
	KeyMiscAssistantDefaultEmptyList = "misc.assistant.default_empty_list"
	KeyMiscAssistantSenderPolicySummary = "misc.assistant.sender_policy_summary"
	KeyMiscAssistantCurrentRuleDetail = "misc.assistant.current_rule_detail"
	KeyMiscAssistantDraftRule = "misc.assistant.draft_rule"
	KeyMiscAssistantSenderPolicyDiff = "misc.assistant.sender_policy_diff"
	KeyMiscAssistantRetainedModel = "misc.assistant.retained_model"
	KeyMiscAssistantProfileDetail = "misc.assistant.profile_detail"
	KeyMiscAssistantProfileDiff = "misc.assistant.profile_diff"
	KeyMiscAssistantBeforeDescription = "misc.assistant.before_description"
	KeyMiscAssistantVisibilityDiff = "misc.assistant.visibility_diff"
	KeyMiscAssistantLastConfirmedResult = "misc.assistant.last_confirmed_result"
	KeyMiscAssistantGroupSession = "misc.assistant.group_session"
	KeyMiscAssistantSelectedMembers = "misc.assistant.selected_members"
	KeyMiscAssistantMatchingRecord = "misc.assistant.matching_record"
	KeyMiscAssistantFullBefore = "misc.assistant.full_before"
	KeyMiscAssistantFullAfter = "misc.assistant.full_after"
	KeyMiscAssistantRoomDetail = "misc.assistant.room_detail"
	KeyMiscAssistantDraftOrder = "misc.assistant.draft_order"
	KeyMiscAssistantRoomEditDiff = "misc.assistant.room_edit_diff"
	KeyMiscAssistantRuntimeFailure = "misc.assistant.runtime_failure"
	KeyMiscAssistantPendingItems = "misc.assistant.pending_items"
	KeyMiscAssistantCompleteMemberRecord = "misc.assistant.complete_member_record"
	KeyMiscAssistantCompleteRuntimeResponse = "misc.assistant.complete_runtime_response"
	KeyMiscAssistantRoomTextDiff = "misc.assistant.room_text_diff"
	KeyMiscAssistantContextSources = "misc.assistant.context_sources"
	KeyMiscAssistantEffectiveContextSources = "misc.assistant.effective_context_sources"
	KeyMiscAssistantEffectiveWatchUnrecognized = "misc.assistant.effective_watch_unrecognized"
	KeyMiscAssistantEffectiveWatch = "misc.assistant.effective_watch"
	KeyMiscAssistantDelayLabel = "misc.assistant.delay_label"
	KeyMiscAssistantTimezoneLabel = "misc.assistant.timezone_label"
	KeyMiscAssistantNewRecord = "misc.assistant.new_record"
	KeyMiscAssistantMissedRuns = "misc.assistant.missed_runs"
	KeyMiscAssistantLocalDraft = "misc.assistant.local_draft"
	KeyMiscAssistantConstraints = "misc.assistant.constraints"
	KeyMiscAssistantWhaleSkills = "misc.assistant.whale_skills"
)

func init() {
	Register(KeyMiscIMCodeReference, "错误码：%s · 参考号：%s", "Code: %s · Reference: %s")
	Register(KeyMiscAssistantSavedAvatarDetail, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n当前已保存头像：%s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nSaved avatar: %s")
	Register(KeyMiscAssistantBeforeAvatarDetail, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n原头像：%s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nBefore: %s")
	Register(KeyMiscAssistantZoomPercent, "缩放：%.2f", "Zoom: %.2f")
	Register(KeyMiscAssistantHorizontalOffset, "水平位置：%.2f", "X: %.2f")
	Register(KeyMiscAssistantVerticalOffset, "垂直位置：%.2f", "Y: %.2f")
	Register(KeyMiscAssistantCapabilityReviewDetail, "Host: %s\nBot: %s · Session: %s · revision %d\n保持工作区：%s\n已有完整目标：%s\nMax rounds: %s", "Host: %s\nBot: %s · Session: %s · revision %d\nWorkspace: %s\nExisting objective: %s\nMax rounds: %s")
	Register(KeyMiscAssistantCreatedDetail, "已创建：%s (%s)\nCanonical session: %s", "Created: %s (%s)\nCanonical session: %s")
	Register(KeyMiscAssistantScratchCwdRequired, "临时会话目录（无工作区时必填）", "Scratch cwd (required without a workspace)")
	Register(KeyMiscAssistantExactMatchingRecord, "精确匹配目录记录：%s (%s)\nCanonical session: %s", "Exact matching record: %s (%s)\nCanonical session: %s")
	Register(KeyMiscAssistantPreserveModel, "保留当前模型：%s · %s", "Preserve: %s · %s")
	Register(KeyMiscAssistantDeletePreviewDetail, "Host: %s · catalog revision %d · kind: %s\n精确目标 ID: %s\n稳定名称：%s\n显示名：%s\n原 canonical session: %s\n完整当前删除影响：\n%s", "Host: %s · catalog revision %d · kind: %s\nExact target ID: %s\nExact stable name: %s\nDisplay title: %s\nOriginal canonical session: %s\nComplete current deletion effects:\n%s")
	Register(KeyMiscAssistantReceiptLinkedCopy, "成功回执关联的目录副本：%s (%s)\nCanonical session: %s\n%s", "Receipt-linked copy: %s (%s)\nCanonical session: %s\n%s")
	Register(KeyMiscAssistantScratchCwdOverridden, "临时会话目录（来源有 workspace 时被其覆盖）", "Scratch cwd (overridden when the source has a workspace)")
	Register(KeyMiscAssistantUnattributedCandidate, "目录候选，未经因果确认：%s (%s)\nCanonical session: %s\n%s", "Unattributed catalog candidate: %s (%s)\nCanonical session: %s\n%s")
	Register(KeyMiscAssistantHostRawContent, "Host 原始内容：\n", "Host raw content:\n")
	Register(KeyMiscAssistantEditEntry, "编辑记忆条目 %d", "Edit entry %d")
	Register(KeyMiscAssistantDeleteEntry, "删除记忆条目 %d…", "Delete entry %d…")
	Register(KeyMiscAssistantBeforeContent, "替换前：\n", "Before:\n")
	Register(KeyMiscAssistantAfterContent, "\n替换后：\n", "\nAfter:\n")
	Register(KeyMiscAssistantMemoryReviewDetail, "Host: %s\nBot: %s · Session: %s · revision %d\nmemoryReview: %s → %s\n配置模型：%s\n已有完整目标：%s\nMax rounds: %s", "Host: %s\nBot: %s · Session: %s · revision %d\nmemoryReview: %s → %s\nConfigured model: %s\nExisting objective: %s\nMax rounds: %s")
	Register(KeyMiscAssistantNotifyDetail, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n已保存：%s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nSaved: %s")
	Register(KeyMiscAssistantNotifyReviewDetail, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n之前：%s\n之后：%s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nBefore: %s\nAfter: %s")
	Register(KeyMiscAssistantCatalogObjectiveDetail, "Host: %s\nBot: %s (%s) · Session: %s · revision %d\n目录目标：%s\n目录轮数上限：%d", "Host: %s\nBot: %s (%s) · Session: %s · revision %d\nCatalog objective: %s\nCatalog max rounds: %d")
	Register(KeyMiscAssistantObjectiveReviewDetail, "Host: %s\nBot: %s · Session: %s · revision %d\n完整目录目标：%s\nMax rounds: %d", "Host: %s\nBot: %s · Session: %s · revision %d\nObjective: %s\nMax rounds: %d")
	Register(KeyMiscAssistantCurrentObjectiveDetail, "Host: %s\nBot: %s · Session: %s · revision %d\n当前完整目录目标：%s\n当前轮数上限：%d", "Host: %s\nBot: %s · Session: %s · revision %d\nCurrent objective: %s\nCurrent round cap: %d")
	Register(KeyMiscAssistantObjectiveConfigDiff, "Host: %s\nBot: %s · Session: %s · revision %d\n原完整目标：\n%s\n新完整目标（已去除首尾空白）：\n%s\n轮数上限：%d → %d", "Host: %s\nBot: %s · Session: %s · revision %d\nBefore objective:\n%s\nAfter objective (trimmed):\n%s\nRound cap: %d → %d")
	Register(KeyMiscAssistantRetainedConfig, "沿用模型配置：%s\n沿用 workspaceId: %s\n沿用能力策略：%s", "Configured model: %s\nworkspaceId: %s\nCapability policy: %s")
	Register(KeyMiscAssistantRouteDetail, "Host: %s\nBot: %s (%s) · Session: %s · revision %d\n当前模型：%s\n当前目录：%s", "Host: %s\nBot: %s (%s) · Session: %s · revision %d\nCurrent model: %s\nCurrent workspace: %s")
	Register(KeyMiscAssistantRouteReviewDetail, "Host: %s\nBot: %s · canonical Session: %s · revision %d\n原完整模型：%s\n新完整模型：%s\n原目录：%s\n新目录：%s\nHost default at review: %s", "Host: %s\nBot: %s · canonical Session: %s · revision %d\nBefore: %s\nAfter: %s\nBefore workspace: %s\nAfter workspace: %s\nHost default at review: %s")
	Register(KeyMiscAssistantDefaultEmptyList, "默认空名单：普通消息允许同 catalog 当前及未来所有其他个人 Bot；直接任务只允许同群 Bot。", "Default empty list: ordinary messages allow every other personal Bot in this catalog, now and in the future; direct tasks allow only Bots in the same group.")
	Register(KeyMiscAssistantSenderPolicySummary, "%s\n完整 allowedSenderIds: %q\n普通消息：\n%s\n直接任务委派：\n%s", "%s\nAll allowedSenderIds: %q\nOrdinary messages:\n%s\nDirect task delegation:\n%s")
	Register(KeyMiscAssistantCurrentRuleDetail, "Host: %s\nBot: %s (%s) · Session: %s · revision %d\n当前规则：\n%s", "Host: %s\nBot: %s (%s) · Session: %s · revision %d\nCurrent rule:\n%s")
	Register(KeyMiscAssistantDraftRule, "草稿完整规则：\n", "Draft rule:\n")
	Register(KeyMiscAssistantSenderPolicyDiff, "Host: %s\nBot: %s · Session: %s · revision %d\n修改前：\n%s\n修改后：\n%s", "Host: %s\nBot: %s · Session: %s · revision %d\nBefore:\n%s\nAfter:\n%s")
	Register(KeyMiscAssistantRetainedModel, "\n保留的模型：", "\nRetained model: ")
	Register(KeyMiscAssistantProfileDetail, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n当前寻址名称：%s\n当前显示标题：%s\n当前完整描述/人设：\n%s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nCurrent name: %s\nCurrent title: %s\nCurrent complete description:\n%s")
	Register(KeyMiscAssistantProfileDiff, "Host: %s\nBot ID: %s · Session ID: %s · revision %d\n原寻址名称：\n%s\n新寻址名称（已trim）：\n%s\n原显示标题：\n%s\n新显示标题（已trim）：\n%s\n最终显示名称：%s → %s", "Host: %s\nBot ID: %s · Session ID: %s · revision %d\nBefore name:\n%s\nAfter name:\n%s\nBefore title:\n%s\nAfter title:\n%s\nDisplay name: %s → %s")
	Register(KeyMiscAssistantBeforeDescription, "原完整描述/人设：\n", "Before complete description:\n")
	Register(KeyMiscAssistantVisibilityDiff, "Host: %s · catalog revision %d\n%s\n完整变更：hidden %t → %t; updatedAt %s → %d\n保留目标其它字段及未知嵌套值: %s\n完整 items 数组 %d 项及原顺序保留；其它每项所有字段不变。", "Host: %s · catalog revision %d\n%s\nComplete changes: hidden %t → %t; updatedAt %s → %d\nOther target fields and unknown nested values retained: %s\nAll %d items and their order retained; every other field of each item is unchanged.")
	Register(KeyMiscAssistantLastConfirmedResult, "最近已确认结果：\n", "Last confirmed result:\n")
	Register(KeyMiscAssistantGroupSession, "群聊：%s (%s)\nCanonical session: %s", "Group: %s (%s)\nCanonical session: %s")
	Register(KeyMiscAssistantSelectedMembers, "Host: %s · revision %d\n已选成员：%d (2–6)", "Host: %s · revision %d\nSelected members: %d (2–6)")
	Register(KeyMiscAssistantMatchingRecord, "同身份目录记录：%s (%s)\nSession: %s\nDescription: %s\nWorkspace: %s\nMembers: %s\nRounds: %d · Speaks: %d", "Matching record: %s (%s)\nSession: %s\nDescription: %s\nWorkspace: %s\nMembers: %s\nRounds: %d · Speaks: %d")
	Register(KeyMiscAssistantFullBefore, "保存前完整资料：\n", "Full before:\n")
	Register(KeyMiscAssistantFullAfter, "\n\n保存后完整资料：\n", "\n\nFull after (Host will set updatedAt):\n")
	Register(KeyMiscAssistantRoomDetail, "Host: %s · revision %d\nGroup: %s (%s)\nCanonical session: %s\nWorkspace: %s\n当前轮数：%d · 发言上限：%d\n当前成员（有序）：\n%s", "Host: %s · revision %d\nGroup: %s (%s)\nCanonical session: %s\nWorkspace: %s\nRounds: %d · Speaks: %d\nCurrent members (ordered):\n%s")
	Register(KeyMiscAssistantDraftOrder, "草稿成员顺序：\n", "Draft order:\n")
	Register(KeyMiscAssistantRoomEditDiff, "Host: %s · revision %d\nGroup: %s (%s) · Session: %s\n轮数：%d → %d\n发言上限：%d → %d\n原成员：\n%s\n新成员：\n%s", "Host: %s · revision %d\nGroup: %s (%s) · Session: %s\nRounds: %d → %d\nSpeaks: %d → %d\nBefore members:\n%s\nAfter members:\n%s")
	Register(KeyMiscAssistantRuntimeFailure, "group/runtime 读取或打开失败：\n", "Runtime read or open failure:\n")
	Register(KeyMiscAssistantPendingItems, "待处理：\n", "Pending:\n")
	Register(KeyMiscAssistantCompleteMemberRecord, "完整成员记录：\n", "Complete member record:\n")
	Register(KeyMiscAssistantCompleteRuntimeResponse, "完整运行回包（含扩展字段）：\n", "Complete runtime response (with extension fields):\n")
	Register(KeyMiscAssistantRoomTextDiff, "群名：%q → %q\n原完整群说明：\n%s\n新完整群说明（已 trim，可清空）：\n%s\n%s", "Room name: %q → %q\nBefore complete room description:\n%s\nAfter complete room description (trimmed, may be empty):\n%s\n%s")
	Register(KeyMiscAssistantContextSources, "上下文来源（最多 3 项）", "Context sources (max 3)")
	Register(KeyMiscAssistantEffectiveContextSources, "有效上下文来源：\n", "Effective context sources:\n")
	Register(KeyMiscAssistantEffectiveWatchUnrecognized, "有效 watch：未知配置，未编辑保持原样：\n", "Effective watch is unrecognized and preserved:\n")
	Register(KeyMiscAssistantEffectiveWatch, "有效 watch：%s\n完整目标：\n%s\nTimeout: %s ms", "Effective watch: %s\nFull target:\n%s\nTimeout: %s ms")
	Register(KeyMiscAssistantDelayLabel, "延迟（例如 30m、1h、1d）", "Delay (for example 30m, 1h, 1d)")
	Register(KeyMiscAssistantTimezoneLabel, "时区（IANA 时区，例如 Asia/Shanghai）", "IANA timezone (for example Asia/Shanghai)")
	Register(KeyMiscAssistantNewRecord, "候选记录：%s (%s)\n%s\n%s · %s · enabled %t", "New record: %s (%s)\n%s\n%s · %s · enabled %t")
	Register(KeyMiscAssistantMissedRuns, "%d 个定时任务错过运行", "%d scheduled runs missed")
	Register(KeyMiscAssistantLocalDraft, "本地草稿 · revision %d", "Local draft · revision %d")
	Register(KeyMiscAssistantConstraints, "约束：\n- ", "Constraints:\n- ")
	Register(KeyMiscAssistantWhaleSkills, "她的技能 · 已启用 %d / %d", "Whale skills · %d / %d enabled")
}

// Mixed preview blocks that the original wrote in both languages at once.
const (
	KeyMiscAssistantBotCreateConfig = "misc.assistant.bot_create_config"
	KeyMiscAssistantBotDuplicateConfig = "misc.assistant.bot_duplicate_config"
	KeyMiscAssistantVisibilityTitle = "misc.assistant.visibility_title"
)

func init() {
	Register(KeyMiscAssistantBotCreateConfig, "Host: %s · revision %d\nBot 名称：%s\n显示标题：%s\n说明：%s\n工作区 ID：%s\n临时目录：%s\nBot ID：%s\nCanonical session：%s\n模型提供方：%s\n模型 ID：%s\n推理强度：%s\nmemoryReview: %t · notify: %t\n头像：blob circle/blue\n工具、技能与 MCP：全部（空名称筛选）\n消息与任务来源：[]\n目标：空 · objectiveMaxRounds: 16\n分区：无 · pinned: false · hidden: false\n个人 Bot 默认：memberBotIds [] · maxRounds 3 · maxSpeaks 10", "Host: %s · revision %d\nBot name: %s\nDisplay title: %s\nDescription: %s\nWorkspace ID: %s\nScratch cwd: %s\nBot ID: %s\nCanonical session: %s\nModel provider: %s\nModel ID: %s\nReasoning effort: %s\nmemoryReview: %t · notify: %t\nAvatar: blob circle/blue\nTools, skills and MCP: all (empty name filters)\nPeer message/task sources: []\nobjective: empty · objectiveMaxRounds: 16\nSection: none · pinned: false · hidden: false\nPersonal Bot defaults: memberBotIds [] · maxRounds 3 · maxSpeaks 10")
	Register(KeyMiscAssistantBotDuplicateConfig, "Host: %s · revision %d\n来源：%s (%s)\n原 canonical session：%s\n新名称：%s\n临时目录：%s\n新 Bot ID 与 canonical session：由 Host 新生成，提交前未知\n完整复制配置：\n%s", "Host: %s · revision %d\nSource: %s (%s)\nOriginal canonical session: %s\nNew name: %s\nScratch cwd: %s\nNew Bot ID and canonical session: generated by the Host, unknown before submit\nFull copied configuration:\n%s")
	Register(KeyMiscAssistantVisibilityTitle, "隐藏与恢复 Bot 或群聊", "Visibility management")
}
