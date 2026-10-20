package storelocale

// Keys for copy that originates below the UI layer: store error and status
// text plus desktop runtime messages. Registered from init so the shared
// catalog stays the single source of truth for the active locale.
const (

	// store/assistant
	KeyStoreAssistantInterruptedRefresh                  = "store.assistant.interrupted_refresh"
	KeyStoreAssistantSaveNotVerifiedRefreshToCompare     = "store.assistant.save_not_verified_refresh_to_compare"
	KeyStoreAssistantWhaleSettingsLoaded                 = "store.assistant.whale_settings_loaded"
	KeyStoreAssistantSavedAndReadBack                    = "store.assistant.saved_and_read_back"
	KeyStoreAssistantInterruptedRefreshToVerify          = "store.assistant.interrupted_refresh_to_verify"
	KeyStoreAssistantDraftRestoredFromHost               = "store.assistant.draft_restored_from_host"
	KeyStoreAssistantWhaleIdentityChanged                = "store.assistant.whale_identity_changed"
	KeyStoreAssistantLoadedUnsavedDraftsPreserved        = "store.assistant.loaded_unsaved_drafts_preserved"
	KeyStoreAssistantMemoryExceeds40000Utf16Units        = "store.assistant.memory_exceeds_40000_utf_16_units"
	KeyStoreAssistantWhaleIdentityChangedRefresh         = "store.assistant.whale_identity_changed_refresh"
	KeyStoreAssistantHostContentChangedRefreshAndCompare = "store.assistant.host_content_changed_refresh_and_compare"
	KeyStoreAssistantHostConfirmedTheOperationRefreshTo  = "store.assistant.host_confirmed_the_operation_refresh_to"
	KeyStoreAssistantNotConfirmedDraftRetainedRefreshBef = "store.assistant.not_confirmed_draft_retained_refresh_bef"
	KeyStoreAssistantWhaleIdentityChangedAfterSubmission = "store.assistant.whale_identity_changed_after_submission"
	KeyStoreAssistantHostAcceptedTheWriteButReadBackDi   = "store.assistant.host_accepted_the_write_but_read_back_di"
	KeyStoreAssistantSavedAndReadBackFromHost            = "store.assistant.saved_and_read_back_from_host"
	KeyStoreAssistantClearConfirmedByHostCurrentContents = "store.assistant.clear_confirmed_by_host_current_contents"

	// store/dshbot
	KeyStoreDshbotReadUnavailableDraftRetained          = "store.dshbot.read_unavailable_draft_retained"
	KeyStoreDshbotMatchingProfileFoundInspectAndAdopt   = "store.dshbot.matching_profile_found_inspect_and_adopt"
	KeyStoreDshbotUnknownOutcomeRemainsLocked           = "store.dshbot.unknown_outcome_remains_locked"
	KeyStoreDshbotReadBackReviewRemainingChanges        = "store.dshbot.read_back_review_remaining_changes"
	KeyStoreDshbotNotSentRefreshAndReview               = "store.dshbot.not_sent_refresh_and_review"
	KeyStoreDshbotNotSentRefresh                        = "store.dshbot.not_sent_refresh"
	KeyStoreDshbotUnknownReadBackOnly                   = "store.dshbot.unknown_read_back_only"
	KeyStoreDshbotProfileSavedVerifyPresentationWarnin  = "store.dshbot.profile_saved_verify_presentation_warnin"
	KeyStoreDshbotAdoptedCatalogPresentationRenameAnd   = "store.dshbot.adopted_catalog_presentation_rename_and"
	KeyStoreDshbotExactCatalogRecordFoundInspectAndA    = "store.dshbot.exact_catalog_record_found_inspect_and_a"
	KeyStoreDshbotOutcomeRemainsUnknownNoRetry          = "store.dshbot.outcome_remains_unknown_no_retry"
	KeyStoreDshbotNotSentDraftRetained                  = "store.dshbot.not_sent_draft_retained"
	KeyStoreDshbotRejectedBeforeWriteDraftRetained      = "store.dshbot.rejected_before_write_draft_retained"
	KeyStoreDshbotUnknownOutcomeReadBackWithoutRetry    = "store.dshbot.unknown_outcome_read_back_without_retry"
	KeyStoreDshbotBotCreatedOpeningFailureIsSeparate    = "store.dshbot.bot_created_opening_failure_is_separate"
	KeyStoreDshbotAdoptedExistingBotOpeningVerifiesRe   = "store.dshbot.adopted_existing_bot_opening_verifies_re"
	KeyStoreDshbotReadbackDoesNotSettleTheUnknownAtt    = "store.dshbot.readback_does_not_settle_the_unknown_att"
	KeyStoreDshbotPreviewReadDeletionNotAuthorized      = "store.dshbot.preview_read_deletion_not_authorized"
	KeyStoreDshbotDeletionPreviewExpired                = "store.dshbot.deletion_preview_expired"
	KeyStoreDshbotCancelledBeforeDispatch               = "store.dshbot.cancelled_before_dispatch"
	KeyStoreDshbotDeletionNotSent                       = "store.dshbot.deletion_not_sent"
	KeyStoreDshbotRejectedBeforeEffects                 = "store.dshbot.rejected_before_effects"
	KeyStoreDshbotUnknownSideEffectsMayAlreadyHaveOc    = "store.dshbot.unknown_side_effects_may_already_have_oc"
	KeyStoreDshbotCandidatesDoNotEstablishCausalityNo   = "store.dshbot.candidates_do_not_establish_causality_no"
	KeyStoreDshbotUnknownDuplicateOutcomeCandidatesAre  = "store.dshbot.unknown_duplicate_outcome_candidates_are"
	KeyStoreDshbotProfileSessionReceiptConfirmedInspec  = "store.dshbot.profile_session_receipt_confirmed_inspec"
	KeyStoreDshbotExactReadbackInspectAndAdopt          = "store.dshbot.exact_readback_inspect_and_adopt"
	KeyStoreDshbotStillUnknownNoRetry                   = "store.dshbot.still_unknown_no_retry"
	KeyStoreDshbotNotSentRefreshAndReview2              = "store.dshbot.not_sent_refresh_and_review_2"
	KeyStoreDshbotRejectedBeforeWrite                   = "store.dshbot.rejected_before_write"
	KeyStoreDshbotUnknownOutcomeNoRetry                 = "store.dshbot.unknown_outcome_no_retry"
	KeyStoreDshbotConfirmedVisibilityHiddenItemsRemain  = "store.dshbot.confirmed_visibility_hidden_items_remain"
	KeyStoreDshbotAdoptedExactReadbackNoResend          = "store.dshbot.adopted_exact_readback_no_resend"
	KeyStoreDshbotReadFailedDraftRetainedAndLocked      = "store.dshbot.read_failed_draft_retained_and_locked"
	KeyStoreDshbotReadBackReviewPreservedDraftAgainst   = "store.dshbot.read_back_review_preserved_draft_against"
	KeyStoreDshbotOutcomeUncertainReadBackBeforeSavin   = "store.dshbot.outcome_uncertain_read_back_before_savin"
	KeyStoreDshbotPreflightChangedNoSaveSent            = "store.dshbot.preflight_changed_no_save_sent"
	KeyStoreDshbotSavedReceiptReadBackToReconcile       = "store.dshbot.saved_receipt_read_back_to_reconcile"
	KeyStoreDshbotRefreshedCatalogConfirmsTheOperation  = "store.dshbot.refreshed_catalog_confirms_the_operation"
	KeyStoreDshbotOutcomeUnknownRefreshWithoutResendin  = "store.dshbot.outcome_unknown_refresh_without_resendin"
	KeyStoreDshbotOutcomeUnknownRefreshToReconcile      = "store.dshbot.outcome_unknown_refresh_to_reconcile"
	KeyStoreDshbotDirectorySaved                        = "store.dshbot.directory_saved"
	KeyStoreDshbotUnassigned                            = "store.dshbot.unassigned"
	KeyStoreDshbotMemoryUnavailableDraftRetained        = "store.dshbot.memory_unavailable_draft_retained"
	KeyStoreDshbotBotIdentityChanged                    = "store.dshbot.bot_identity_changed"
	KeyStoreDshbotReadbackMatchesSubmittedContentNoRe   = "store.dshbot.readback_matches_submitted_content_no_re"
	KeyStoreDshbotRefreshedWithoutRetryingOrRebasingD   = "store.dshbot.refreshed_without_retrying_or_rebasing_d"
	KeyStoreDshbotMemoryConflictNoAutomaticRebase       = "store.dshbot.memory_conflict_no_automatic_rebase"
	KeyStoreDshbotMemoryRevisionConflict                = "store.dshbot.memory_revision_conflict"
	KeyStoreDshbotEntryMustNotBeEmpty                   = "store.dshbot.entry_must_not_be_empty"
	KeyStoreDshbotNoMemoryChanges                       = "store.dshbot.no_memory_changes"
	KeyStoreDshbotSaveUncertainRefreshToVerify          = "store.dshbot.save_uncertain_refresh_to_verify"
	KeyStoreDshbotHostConfirmedMemoryReplacement        = "store.dshbot.host_confirmed_memory_replacement"
	KeyStoreDshbotReadUnavailableDraftRetainedAndLock   = "store.dshbot.read_unavailable_draft_retained_and_lock"
	KeyStoreDshbotReadBackReviewAnyRemainingDraft       = "store.dshbot.read_back_review_any_remaining_draft"
	KeyStoreDshbotOutcomeUncertainReadBackBeforeSavin2  = "store.dshbot.outcome_uncertain_read_back_before_savin_2"
	KeyStoreDshbotPreflightChangedNoSaveSent2           = "store.dshbot.preflight_changed_no_save_sent_2"
	KeyStoreDshbotReceiptReceivedReadBackToVerify       = "store.dshbot.receipt_received_read_back_to_verify"
	KeyStoreDshbotReadUnavailableDraftRetained2         = "store.dshbot.read_unavailable_draft_retained_2"
	KeyStoreDshbotMatchingProfileFoundInspectAndAdopt2  = "store.dshbot.matching_profile_found_inspect_and_adopt_2"
	KeyStoreDshbotProfileSavedVerifyPresentationWarnin2 = "store.dshbot.profile_saved_verify_presentation_warnin_2"
	KeyStoreDshbotReadUnavailableDraftRetainedAndLock2  = "store.dshbot.read_unavailable_draft_retained_and_lock_2"
	KeyStoreDshbotObjectiveStateUnavailableRefreshBefo  = "store.dshbot.objective_state_unavailable_refresh_befo"
	KeyStoreDshbotRefreshedWithoutStartingABot          = "store.dshbot.refreshed_without_starting_a_bot"
	KeyStoreDshbotOutcomeUncertainRefreshBeforeAnother  = "store.dshbot.outcome_uncertain_refresh_before_another"
	KeyStoreDshbotPreflightChangedOrUnavailableNoActi   = "store.dshbot.preflight_changed_or_unavailable_no_acti"
	KeyStoreDshbotHostReceiptReceivedRefreshToVerify    = "store.dshbot.host_receipt_received_refresh_to_verify"
	KeyStoreDshbotLimitsMustBeIntegers                  = "store.dshbot.limits_must_be_integers"
	KeyStoreDshbotRoomCreatedOpeningFailureIsSeparate   = "store.dshbot.room_created_opening_failure_is_separate"
	KeyStoreDshbotAdoptedExistingRoomOpeningVerifiesR   = "store.dshbot.adopted_existing_room_opening_verifies_r"
	KeyStoreDshbotExactReviewedProfileFoundInspectAnd   = "store.dshbot.exact_reviewed_profile_found_inspect_and"
	KeyStoreDshbotNotConfirmedNoRetry                   = "store.dshbot.not_confirmed_no_retry"
	KeyStoreDshbotLimitsMustBeIntegers2                 = "store.dshbot.limits_must_be_integers_2"
	KeyStoreDshbotNoChangesToSave                       = "store.dshbot.no_changes_to_save"
	KeyStoreDshbotNotSentDraftRetained2                 = "store.dshbot.not_sent_draft_retained_2"
	KeyStoreDshbotNotSentDraftRetained3                 = "store.dshbot.not_sent_draft_retained_3"
	KeyStoreDshbotUnknownOutcomeReadBackOnly            = "store.dshbot.unknown_outcome_read_back_only"
	KeyStoreDshbotHostConfirmedGroupUpdate              = "store.dshbot.host_confirmed_group_update"
	KeyStoreDshbotAdoptedMatchingCatalogPresentationSi  = "store.dshbot.adopted_matching_catalog_presentation_si"
	KeyStoreDshbotRuntimeUnavailableAnyOldSnapshotIs    = "store.dshbot.runtime_unavailable_any_old_snapshot_is"
	KeyStoreDshbotReadOnlySnapshotNoAutomaticMemberA    = "store.dshbot.read_only_snapshot_no_automatic_member_a"
	KeyStoreDshbotOpenFailedRefreshBeforeAnotherExpli   = "store.dshbot.open_failed_refresh_before_another_expli"
	KeyStoreDshbotReadUnavailableDraftRetained3         = "store.dshbot.read_unavailable_draft_retained_3"
	KeyStoreDshbotMatchingProfileFoundInspectAndAdopt3  = "store.dshbot.matching_profile_found_inspect_and_adopt_3"
	KeyStoreDshbotSavedUseExistingEntryToOpenAfterR     = "store.dshbot.saved_use_existing_entry_to_open_after_r"
	KeyStoreDshbotAdoptedCatalogSessionAndLiveSynchro   = "store.dshbot.adopted_catalog_session_and_live_synchro"
	KeyStoreDshbotContextSourceIsNotOwnedByThisBot      = "store.dshbot.context_source_is_not_owned_by_this_bot"
	KeyStoreDshbotAtMost3ContextSources                 = "store.dshbot.at_most_3_context_sources"
	KeyStoreDshbotUnsupportedWatchKind                  = "store.dshbot.unsupported_watch_kind"
	KeyStoreDshbotSelectASupportedWatchTypeFirst        = "store.dshbot.select_a_supported_watch_type_first"
	KeyStoreDshbotWatchTimeoutMustBeIntegerMillisecon   = "store.dshbot.watch_timeout_must_be_integer_millisecon"
	KeyStoreDshbotRoutinesRefreshed                     = "store.dshbot.routines_refreshed"
	KeyStoreDshbotMissedRunStatusUnavailable            = "store.dshbot.missed_run_status_unavailable"
	KeyStoreDshbotCatalogChangedReviewCurrentValuesBe   = "store.dshbot.catalog_changed_review_current_values_be"
	KeyStoreDshbotRunLimitMustBeAnInteger               = "store.dshbot.run_limit_must_be_an_integer"
	KeyStoreDshbotLimitMustExceedCurrentRunCount        = "store.dshbot.limit_must_exceed_current_run_count"
	KeyStoreDshbotAPreviousRunIsPendingOrRunning        = "store.dshbot.a_previous_run_is_pending_or_running"
	KeyStoreDshbotOutcomeMayHaveAppliedRefreshBefore    = "store.dshbot.outcome_may_have_applied_refresh_before"
	KeyStoreDshbotHostReturnedCatalog                   = "store.dshbot.host_returned_catalog"
	KeyStoreDshbotCheckStatusAndHistoryThisIsNotCom     = "store.dshbot.check_status_and_history_this_is_not_com"
	KeyStoreDshbotBotSessionChangedInspectTheOriginal   = "store.dshbot.bot_session_changed_inspect_the_original"
	KeyStoreDshbotBotIdentityChanged2                   = "store.dshbot.bot_identity_changed_2"
	KeyStoreDshbotCurrentCatalogAdoptedReviewCreation   = "store.dshbot.current_catalog_adopted_review_creation"
	KeyStoreDshbotCatalogChangedRecheckBeforeReviewing  = "store.dshbot.catalog_changed_recheck_before_reviewing"
	KeyStoreDshbotRunLimitMustBeAnInteger2              = "store.dshbot.run_limit_must_be_an_integer_2"
	KeyStoreDshbotSpecifyTheIntendedIanaTimezone        = "store.dshbot.specify_the_intended_iana_timezone"
	KeyStoreDshbotInvalidIanaTimezone                   = "store.dshbot.invalid_iana_timezone"
	KeyStoreDshbotCreationNotSentReviewTheRetainedDr    = "store.dshbot.creation_not_sent_review_the_retained_dr"
	KeyStoreDshbotCreationRejectedBeforeWriteDraftRet   = "store.dshbot.creation_rejected_before_write_draft_ret"
	KeyStoreDshbotOutcomeUncertainInspectTheCatalogBe   = "store.dshbot.outcome_uncertain_inspect_the_catalog_be"
	KeyStoreDshbotCreatedRoutine                        = "store.dshbot.created_routine"
	KeyStoreDshbotUsingTheInspectedExistingRecord       = "store.dshbot.using_the_inspected_existing_record"
	KeyStoreDshbotCronNeedsFiveFieldsAtMost200Chara     = "store.dshbot.cron_needs_five_fields_at_most_200_chara"
	KeyStoreDshbotEachCronFieldMayContainAtMost40C      = "store.dshbot.each_cron_field_may_contain_at_most_40_c"
	KeyStoreDshbotEnterAValidCalendarTime               = "store.dshbot.enter_a_valid_calendar_time"
	KeyStoreDshbotSelectWeekday06                       = "store.dshbot.select_weekday_0_6"
	KeyStoreDshbotDayOfMonthMustBe131                   = "store.dshbot.day_of_month_must_be_1_31"
	KeyStoreDshbotSelectASupportedFrequency             = "store.dshbot.select_a_supported_frequency"
	KeyStoreDshbotDurationMustBeAPositiveIntegerWith    = "store.dshbot.duration_must_be_a_positive_integer_with"
	KeyStoreDshbotDurationMustBe1525600Minutes          = "store.dshbot.duration_must_be_1_525600_minutes"
	KeyStoreDshbotEnterAValidDailyTime                  = "store.dshbot.enter_a_valid_daily_time"
	KeyStoreDshbotEnterAOneShotDelaySuchAs30m1hOr       = "store.dshbot.enter_a_one_shot_delay_such_as_30m_1h_or"
	KeyStoreDshbotRefreshBeforeFurtherChangesRefreshC   = "store.dshbot.refresh_before_further_changes_refresh_c"
	KeyStoreDshbotCredentialOutcomeUncertainNoAutomati  = "store.dshbot.credential_outcome_uncertain_no_automati"
	KeyStoreDshbotReadUnavailableDraftRetained4         = "store.dshbot.read_unavailable_draft_retained_4"
	KeyStoreDshbotMatchingProfileFoundInspectAndAdopt4  = "store.dshbot.matching_profile_found_inspect_and_adopt_4"
	KeyStoreDshbotSavedProfileLiveSynchronizationAndE   = "store.dshbot.saved_profile_live_synchronization_and_e"
	KeyStoreDshbotAdoptedCatalogLiveSynchronizationAnd  = "store.dshbot.adopted_catalog_live_synchronization_and"
	KeyStoreDshbotTasksRefreshed                        = "store.dshbot.tasks_refreshed"
	KeyStoreDshbotCatalogChangedReviewTheActionAgain    = "store.dshbot.catalog_changed_review_the_action_again"
	KeyStoreDshbotHostAccepted                          = "store.dshbot.host_accepted"
	KeyStoreDshbotStateSavedWakeFailed                  = "store.dshbot.state_saved_wake_failed"
	KeyStoreDshbotWakeNotConfirmedRefreshToInspect      = "store.dshbot.wake_not_confirmed_refresh_to_inspect"
	KeyStoreDshbotReadUnavailableDraftRetained5         = "store.dshbot.read_unavailable_draft_retained_5"
	KeyStoreDshbotMatchingProfileFoundInspectAndAdopt5  = "store.dshbot.matching_profile_found_inspect_and_adopt_5"
	KeyStoreDshbotProfileSavedVerifyPresentationWarnin3 = "store.dshbot.profile_saved_verify_presentation_warnin_3"

	// store/imonboard
	KeyStoreImonboardBotUnavailableRefreshToRetry         = "store.imonboard.bot_unavailable_refresh_to_retry"
	KeyStoreImonboardEnterAnAbsoluteWorkspacePath         = "store.imonboard.enter_an_absolute_workspace_path"
	KeyStoreImonboardPresetUnavailableOnThisHost          = "store.imonboard.preset_unavailable_on_this_host"
	KeyStoreImonboardLark                                 = "store.imonboard.lark"
	KeyStoreImonboardConnectionStatusRefreshFailedRefresh = "store.imonboard.connection_status_refresh_failed_refresh"
	KeyStoreImonboardIncompleteHostConnectionStatus       = "store.imonboard.incomplete_host_connection_status"

	// store/remote
	KeyStoreRemoteRemovalCouldNotBeVerifiedTheHostS  = "store.remote.removal_could_not_be_verified_the_host_s"
	KeyStoreRemoteStatusRefreshFailedRefreshToCheckT = "store.remote.status_refresh_failed_refresh_to_check_t"

	// store/schedule
	KeyStoreScheduleEndedNoFutureDeliveries      = "store.schedule.ended_no_future_deliveries"
	KeyStoreScheduleUnknownStatusRefreshToVerify = "store.schedule.unknown_status_refresh_to_verify"
	KeyStoreScheduleNextRunUnavailableRefresh    = "store.schedule.next_run_unavailable_refresh"
	KeyStoreScheduleNextRun                      = "store.schedule.next_run"
	KeyStoreScheduleRuleZoneUnavailable          = "store.schedule.rule_zone_unavailable"
	KeyStoreScheduleAllSessions                  = "store.schedule.all_sessions"
	KeyStoreScheduleCurrentSessionOnly           = "store.schedule.current_session_only"
	KeyStoreScheduleHostCatalogUnavailable       = "store.schedule.host_catalog_unavailable"

	// store/session
	KeyStoreSessionChildPromptSubmitted                  = "store.session.child_prompt_submitted"
	KeyStoreSessionInterruptRequested                    = "store.session.interrupt_requested"
	KeyStoreSessionLiveSequenceChangedResynchronizing    = "store.session.live_sequence_changed_resynchronizing"
	KeyStoreSessionRestoredTargetFailedSourceCatalogVe   = "store.session.restored_target_failed_source_catalog_ve"
	KeyStoreSessionCreatorConfirmationExpired            = "store.session.creator_confirmation_expired"
	KeyStoreSessionConnectAHostFirst                     = "store.session.connect_a_host_first"
	KeyStoreSessionChooseAWorkspaceFirst                 = "store.session.choose_a_workspace_first"
	KeyStoreSessionCannotAllocateATaskIdentity           = "store.session.cannot_allocate_a_task_identity"
	KeyStoreSessionCouldNotVerifyCordisNoTaskWasCrea     = "store.session.could_not_verify_cordis_no_task_was_crea"
	KeyStoreSessionCordisIsUnavailableNoTaskWasCreate    = "store.session.cordis_is_unavailable_no_task_was_create"
	KeyStoreSessionNavigationChangedNoTaskWasCreated     = "store.session.navigation_changed_no_task_was_created"
	KeyStoreSessionCreationOutcomeUnknownLocateTheReta   = "store.session.creation_outcome_unknown_locate_the_reta"
	KeyStoreSessionTaskCreatedNavigationChangedBeforeC   = "store.session.task_created_navigation_changed_before_c"
	KeyStoreSessionTaskCreatedCordisOutcomeUnconfirmed   = "store.session.task_created_cordis_outcome_unconfirmed"
	KeyStoreSessionCordisAppliedNewerNavigationPreserve  = "store.session.cordis_applied_newer_navigation_preserve"
	KeyStoreSessionCouldNotLocateTaskIdRetained          = "store.session.could_not_locate_task_id_retained"
	KeyStoreSessionHostOrPageChangedReadbackNotAdopte    = "store.session.host_or_page_changed_readback_not_adopte"
	KeyStoreSessionExistingTaskLocatedInspectItsPreset   = "store.session.existing_task_located_inspect_its_preset"
	KeyStoreSessionThisIdIsNotConfirmedInTheListNo       = "store.session.this_id_is_not_confirmed_in_the_list_no"
	KeyStoreSessionCommandOwnerChanged                   = "store.session.command_owner_changed"
	KeyStoreSessionExplicitPermissionRiskAcknowledgement = "store.session.explicit_permission_risk_acknowledgement"
	KeyStoreSessionSkipped                               = "store.session.skipped"
	KeyStoreSessionAnswer                                = "store.session.answer"
	KeyStoreSessionPermanentlyDeleteThisSkillDirectory   = "store.session.permanently_delete_this_skill_directory"
	KeyStoreSessionSpeechHostOffline                     = "store.session.speech_host_offline"
	KeyStoreSessionSpeechUnavailable                     = "store.session.speech_unavailable"
	KeyStoreSessionPrepareTheSelectedSpeechProvider      = "store.session.prepare_the_selected_speech_provider"
	KeyStoreSessionInvalidSpeechLimits                   = "store.session.invalid_speech_limits"
	KeyStoreSessionSpeechLimitTooSmall                   = "store.session.speech_limit_too_small"
	KeyStoreSessionSpeechSessionChanged                  = "store.session.speech_session_changed"
	KeyStoreSessionTranscriptionFailed                   = "store.session.transcription_failed"
	KeyStoreSessionNoSpeechRecognized                    = "store.session.no_speech_recognized"
	KeyStoreSessionTranscriptTooLarge                    = "store.session.transcript_too_large"
	KeyStoreSessionInvalidRecording                      = "store.session.invalid_recording"
	KeyStoreSessionPublishedHtmlUnavailable              = "store.session.published_html_unavailable"
	KeyStoreSessionLivePreview                           = "store.session.live_preview"

	// store/settings
	KeyStoreSettingsCustomInstructionsMustNotExceed1500 = "store.settings.custom_instructions_must_not_exceed_1500"

	// store/usage
	KeyStoreUsagePricesMustBeFiniteNumbersIn010000 = "store.usage.prices_must_be_finite_numbers_in_0_10000"
	KeyStoreUsagePricesSaved                       = "store.usage.prices_saved"
	KeyStoreUsagePostRepairStatisticsRescanFailed  = "store.usage.post_repair_statistics_rescan_failed"
	KeyStoreUsageUsageStatisticsUnavailable        = "store.usage.usage_statistics_unavailable"

	// desktop/botnotify
	KeyDesktopBotnotifyBotNeedsYourAttention             = "desktop.botnotify.bot_needs_your_attention"
	KeyDesktopBotnotifyBotObjectiveIsBlocked             = "desktop.botnotify.bot_objective_is_blocked"
	KeyDesktopBotnotifyRoutineDidNotCompleteSuccessfully = "desktop.botnotify.routine_did_not_complete_successfully"
	KeyDesktopBotnotifyRoutineCompleted                  = "desktop.botnotify.routine_completed"

	// desktop/pet
	KeyDesktopPetCouldNotSaveHideTryAgain             = "desktop.pet.could_not_save_hide_try_again"
	KeyDesktopPetOpeningPetChat                       = "desktop.pet.opening_pet_chat"
	KeyDesktopPetMissingOrChangedTha4AssetsFallback   = "desktop.pet.missing_or_changed_tha4_assets_fallback"
	KeyDesktopPetFilesStayUntouched                   = "desktop.pet.files_stay_untouched"
	KeyDesktopPetChooseOneFile                        = "desktop.pet.choose_one_file"
	KeyDesktopPetFeedOnceFileStaysUntouched           = "desktop.pet.feed_once_file_stays_untouched"
	KeyDesktopPetFileSelectionFailed                  = "desktop.pet.file_selection_failed"
	KeyDesktopPetSaveFailedNotApplied                 = "desktop.pet.save_failed_not_applied"
	KeyDesktopPetFed                                  = "desktop.pet.fed"
	KeyDesktopPetFed2                                 = "desktop.pet.fed_2"
	KeyDesktopPetLedgerUnavailable                    = "desktop.pet.ledger_unavailable"
	KeyDesktopPetLocalGameOnly                        = "desktop.pet.local_game_only"
	KeyDesktopPetReadingLocalUsage                    = "desktop.pet.reading_local_usage"
	KeyDesktopPetCancelled                            = "desktop.pet.cancelled"
	KeyDesktopPetSaveFailedUnchanged                  = "desktop.pet.save_failed_unchanged"
	KeyDesktopPetSaveFailedUnchanged2                 = "desktop.pet.save_failed_unchanged_2"
	KeyDesktopPetNoLocalFood                          = "desktop.pet.no_local_food"
	KeyDesktopPetInteractionWasNotSaved               = "desktop.pet.interaction_was_not_saved"
	KeyDesktopPetPleaseWaitBeforeAnotherGlance        = "desktop.pet.please_wait_before_another_glance"
	KeyDesktopPetWaitingForScreenshotApproval         = "desktop.pet.waiting_for_screenshot_approval"
	KeyDesktopPetLooking                              = "desktop.pet.looking"
	KeyDesktopPetPetConfigurationUnavailable          = "desktop.pet.pet_configuration_unavailable"
	KeyDesktopPetRefreshTheImageCatalogOrTurnLookO    = "desktop.pet.refresh_the_image_catalog_or_turn_look_o"
	KeyDesktopPetPetPositionWasNotSaved               = "desktop.pet.pet_position_was_not_saved"
	KeyDesktopPetSaveFailedUnchanged3                 = "desktop.pet.save_failed_unchanged_3"
	KeyDesktopPetPatted                               = "desktop.pet.patted"
	KeyDesktopPetLoadingSharedConversation            = "desktop.pet.loading_shared_conversation"
	KeyDesktopPetSendUsesDesktopApiSettings           = "desktop.pet.send_uses_desktop_api_settings"
	KeyDesktopPetHostUnavailable                      = "desktop.pet.host_unavailable"
	KeyDesktopPetWhaleQuickChat                       = "desktop.pet.whale_quick_chat"
	KeyDesktopPetQuickChatWindowCouldNotOpen          = "desktop.pet.quick_chat_window_could_not_open"
	KeyDesktopPetConnectionChangedReopenQuickChat     = "desktop.pet.connection_changed_reopen_quick_chat"
	KeyDesktopPetSharedHistoryUnavailable             = "desktop.pet.shared_history_unavailable"
	KeyDesktopPetSharedWithMainConversation           = "desktop.pet.shared_with_main_conversation"
	KeyDesktopPetPetSessionChanged                    = "desktop.pet.pet_session_changed"
	KeyDesktopPetModelCatalogUnavailable              = "desktop.pet.model_catalog_unavailable"
	KeyDesktopPetHostModelCatalogUnavailable          = "desktop.pet.host_model_catalog_unavailable"
	KeyDesktopPetEnterUpTo2000Utf16Units              = "desktop.pet.enter_up_to_2000_utf_16_units"
	KeyDesktopPetWaitingForSharedSession              = "desktop.pet.waiting_for_shared_session"
	KeyDesktopPetTurnCompletedSyncingHistory          = "desktop.pet.turn_completed_syncing_history"
	KeyDesktopPetPetSessionChangedBeforeSend          = "desktop.pet.pet_session_changed_before_send"
	KeyDesktopPetCompletionUnconfirmedCheckHistory    = "desktop.pet.completion_unconfirmed_check_history"
	KeyDesktopPetWaitCancelledApiMayContinue          = "desktop.pet.wait_cancelled_api_may_continue"
	KeyDesktopPetWaitCancelledHostMayContinue         = "desktop.pet.wait_cancelled_host_may_continue"
	KeyDesktopPetChangingSharedSessionModel           = "desktop.pet.changing_shared_session_model"
	KeyDesktopPetModelUpdated                         = "desktop.pet.model_updated"
	KeyDesktopPetModelChangeUnconfirmed               = "desktop.pet.model_change_unconfirmed"
	KeyDesktopPetStoppingTheCurrentPetTurn            = "desktop.pet.stopping_the_current_pet_turn"
	KeyDesktopPetStopAcceptedQueuedMessagesRemain     = "desktop.pet.stop_accepted_queued_messages_remain"
	KeyDesktopPetStopUnconfirmedCheckMainConversation = "desktop.pet.stop_unconfirmed_check_main_conversation"
	KeyDesktopPetSessionChangedOrUnavailable          = "desktop.pet.session_changed_or_unavailable"
	KeyDesktopPetSendingToConfiguredApi               = "desktop.pet.sending_to_configured_api"
	KeyDesktopPetDesktopApiCredentialsUnavailable     = "desktop.pet.desktop_api_credentials_unavailable"
	KeyDesktopPetApiEndpointIsInvalidOrInsecure       = "desktop.pet.api_endpoint_is_invalid_or_insecure"
	KeyDesktopPetSettingsChangedCloseAndReopen        = "desktop.pet.settings_changed_close_and_reopen"
	KeyDesktopPetApiResultUnconfirmedAvoidDuplicateS  = "desktop.pet.api_result_unconfirmed_avoid_duplicate_s"
	KeyDesktopPetTemporaryReplyReceived               = "desktop.pet.temporary_reply_received"
	KeyDesktopPetOpen                                 = "desktop.pet.open"
	KeyDesktopPetCloseQuickChat                       = "desktop.pet.close_quick_chat"
	KeyDesktopPetClosingOnlyCancelsTheLocalWait       = "desktop.pet.closing_only_cancels_the_local_wait"
	KeyDesktopPetTemporaryOnlineApiChat               = "desktop.pet.temporary_online_api_chat"
	KeyDesktopPetOneSharedConversation                = "desktop.pet.one_shared_conversation"
	KeyDesktopPetPetSessionId                         = "desktop.pet.pet_session_id"
	KeyDesktopPetLoadingSharedHistory                 = "desktop.pet.loading_shared_history"
	KeyDesktopPetClickSendToApiToSend                 = "desktop.pet.click_send_to_api_to_send"
	KeyDesktopPetNoMessagesYet                        = "desktop.pet.no_messages_yet"
	KeyDesktopPetConnectionUnavailable                = "desktop.pet.connection_unavailable"
	KeyDesktopPetSharedHistory                        = "desktop.pet.shared_history"
	KeyDesktopPetTemporaryApiHistory                  = "desktop.pet.temporary_api_history"
	KeyDesktopPetYou                                  = "desktop.pet.you"
	KeyDesktopPetStatus                               = "desktop.pet.status"
	KeyDesktopPetSession                              = "desktop.pet.session"
	KeyDesktopPetQuickChatDraft                       = "desktop.pet.quick_chat_draft"
	KeyDesktopPetEnterSendsShiftEnterNewline          = "desktop.pet.enter_sends_shift_enter_newline"
	KeyDesktopPetSend                                 = "desktop.pet.send"
	KeyDesktopPetSendToApi                            = "desktop.pet.send_to_api"
	KeyDesktopPetCancelWait                           = "desktop.pet.cancel_wait"
	KeyDesktopPetStopTurn                             = "desktop.pet.stop_turn"
	KeyDesktopPetRefresh                              = "desktop.pet.refresh"
	KeyDesktopPetModel                                = "desktop.pet.model"
	KeyDesktopPetSelectSharedModel                    = "desktop.pet.select_shared_model"
	KeyDesktopPetReasoning                            = "desktop.pet.reasoning"
	KeyDesktopPetDefault                              = "desktop.pet.default"
	KeyDesktopPetSelectReasoningEffort                = "desktop.pet.select_reasoning_effort"
	KeyDesktopPetSleep                                = "desktop.pet.sleep"
	KeyDesktopPetAwake                                = "desktop.pet.awake"
	KeyDesktopPetDozing                               = "desktop.pet.dozing"
	KeyDesktopPetWake                                 = "desktop.pet.wake"
	KeyDesktopPetResting                              = "desktop.pet.resting"
	KeyDesktopPetSaveFailedStillResting               = "desktop.pet.save_failed_still_resting"
	KeyDesktopPetUnknown                              = "desktop.pet.unknown"
	KeyDesktopPetRunning                              = "desktop.pet.running"
	KeyDesktopPetIdle                                 = "desktop.pet.idle"
	KeyDesktopPetOffline                              = "desktop.pet.offline"
	KeyDesktopPetOtherConversationsAreWorking         = "desktop.pet.other_conversations_are_working"

	// desktop/pluginrpc
	KeyDesktopPluginrpcUnsupportedUnicodeUseExistingWebEdi = "desktop.pluginrpc.unsupported_unicode_use_existing_web_edi"
	KeyDesktopPluginrpcProfileCannotBeSafelyRoundTripped   = "desktop.pluginrpc.profile_cannot_be_safely_round_tripped"
	KeyDesktopPluginrpcRefreshTheContactIdentity           = "desktop.pluginrpc.refresh_the_contact_identity"
	KeyDesktopPluginrpcRefreshTheCatalogFirst              = "desktop.pluginrpc.refresh_the_catalog_first"
	KeyDesktopPluginrpcSectionName1120Characters           = "desktop.pluginrpc.section_name_1_120_characters"
	KeyDesktopPluginrpcSectionNameAlreadyExists            = "desktop.pluginrpc.section_name_already_exists"
	KeyDesktopPluginrpcNoSectionSnapshot                   = "desktop.pluginrpc.no_section_snapshot"
	KeyDesktopPluginrpcUnsupportedContactsAction           = "desktop.pluginrpc.unsupported_contacts_action"
	KeyDesktopPluginrpcNotificationCatalogExceedsTheHostLi = "desktop.pluginrpc.notification_catalog_exceeds_the_host_li"
	KeyDesktopPluginrpcUseClearCatalogObjectiveToRemoveIt  = "desktop.pluginrpc.use_clear_catalog_objective_to_remove_it"
	KeyDesktopPluginrpcObjectiveExceeds2000Utf16Units      = "desktop.pluginrpc.objective_exceeds_2_000_utf_16_units"
	KeyDesktopPluginrpcRoundCapMustBeAnIntegerFrom1To2     = "desktop.pluginrpc.round_cap_must_be_an_integer_from_1_to_2"
	KeyDesktopPluginrpcMemberOwnershipUnavailable          = "desktop.pluginrpc.member_ownership_unavailable"
	KeyDesktopPluginrpcHistoricalMemberNotInCurrentRoom    = "desktop.pluginrpc.historical_member_not_in_current_room"
	KeyDesktopPluginrpcCurrentBotIdentityUnavailable       = "desktop.pluginrpc.current_bot_identity_unavailable"
	KeyDesktopPluginrpcMemberSessionUnavailable            = "desktop.pluginrpc.member_session_unavailable"
	KeyDesktopPluginrpcSessionDoesNotIdentifyThisRoomMemb  = "desktop.pluginrpc.session_does_not_identify_this_room_memb"
	KeyDesktopPluginrpcAmbiguousRuntimeIdentity            = "desktop.pluginrpc.ambiguous_runtime_identity"
	KeyDesktopPluginrpcInvalidRoomTextUnicode              = "desktop.pluginrpc.invalid_room_text_unicode"
	KeyDesktopPluginrpcRoomNameRequiredMaximum120Utf16Un   = "desktop.pluginrpc.room_name_required_maximum_120_utf_16_un"
	KeyDesktopPluginrpcRoomDescriptionMaximum12000Utf16Un  = "desktop.pluginrpc.room_description_maximum_12000_utf_16_un"
	KeyDesktopPluginrpcInvalidUnicode                      = "desktop.pluginrpc.invalid_unicode"
	KeyDesktopPluginrpcNameIsRequiredMaximum120Utf16Unit   = "desktop.pluginrpc.name_is_required_maximum_120_utf_16_unit"
	KeyDesktopPluginrpcTitle120Description12000Utf16Units  = "desktop.pluginrpc.title_120_description_12000_utf_16_units"

	// desktop/previewprocess
	KeyDesktopPreviewprocessIsolatedInteractivePreview = "desktop.previewprocess.isolated_interactive_preview"

	// desktop/screenglance
	KeyDesktopScreenglanceCaptureThroughTheSystemScreenshotDi  = "desktop.screenglance.capture_through_the_system_screenshot_di"
	KeyDesktopScreenglanceCaptureAndSendOneScreenshotOfTheP    = "desktop.screenglance.capture_and_send_one_screenshot_of_the_p"
	KeyDesktopScreenglanceTheImageMayContainPrivateInformatio  = "desktop.screenglance.the_image_may_contain_private_informatio"
	KeyDesktopScreenglanceTheImageMayContainPrivateInformatio2 = "desktop.screenglance.the_image_may_contain_private_informatio_2"
	KeyDesktopScreenglanceTheImageMayContainPrivateInformatio3 = "desktop.screenglance.the_image_may_contain_private_informatio_3"
	KeyDesktopScreenglanceScreenshotUnavailableCheckThisAppS   = "desktop.screenglance.screenshot_unavailable_check_this_app_s"
	KeyDesktopScreenglanceScreenshotCancelledDeniedOrUnavailab = "desktop.screenglance.screenshot_cancelled_denied_or_unavailab"
	KeyDesktopScreenglanceScreenshotUnavailable                = "desktop.screenglance.screenshot_unavailable"

	// desktop/voiceinput
	KeyDesktopVoiceinputOpeningMicrophone           = "desktop.voiceinput.opening_microphone"
	KeyDesktopVoiceinputMicrophoneUnavailable       = "desktop.voiceinput.microphone_unavailable"
	KeyDesktopVoiceinputRecording                   = "desktop.voiceinput.recording"
	KeyDesktopVoiceinputTranscribing                = "desktop.voiceinput.transcribing"
	KeyDesktopVoiceinputRecordingFailed             = "desktop.voiceinput.recording_failed"
	KeyDesktopVoiceinputDraftChangedInsertOrDiscard = "desktop.voiceinput.draft_changed_insert_or_discard"
)

func init() {

	// store/assistant
	register(KeyStoreAssistantInterruptedRefresh, "操作中断，请刷新", "Interrupted; refresh")
	register(KeyStoreAssistantSaveNotVerifiedRefreshToCompare, "保存结果未核实，请刷新后核对", "Save not verified; refresh to compare")
	register(KeyStoreAssistantWhaleSettingsLoaded, "已读取 Whale 设置", "Whale settings loaded")
	register(KeyStoreAssistantSavedAndReadBack, "已保存并读回 Whale 设置", "Saved and read back")
	register(KeyStoreAssistantInterruptedRefreshToVerify, "操作已中断，请重新读取核对", "Interrupted; refresh to verify")
	register(KeyStoreAssistantDraftRestoredFromHost, "草稿已恢复为当前 Host 内容", "Draft restored from Host")
	register(KeyStoreAssistantWhaleIdentityChanged, "Whale 身份已变更，已隔离之前的草稿", "Whale identity changed")
	register(KeyStoreAssistantLoadedUnsavedDraftsPreserved, "已读取；未保存的草稿保持不变", "Loaded; unsaved drafts preserved")
	register(KeyStoreAssistantMemoryExceeds40000Utf16Units, "长期记忆超过 40000 字符限制", "Memory exceeds 40000 UTF-16 units")
	register(KeyStoreAssistantWhaleIdentityChangedRefresh, "Whale 身份已改变，请重新读取", "Whale identity changed; refresh")
	register(KeyStoreAssistantHostContentChangedRefreshAndCompare, "Host 内容已变化，未提交；请重新读取并核对草稿", "Host content changed; refresh and compare your draft")
	register(KeyStoreAssistantHostConfirmedTheOperationRefreshTo, "Host 已确认操作；重新读取后才能继续", "Host confirmed the operation; refresh to verify")
	register(KeyStoreAssistantNotConfirmedDraftRetainedRefreshBef, "未确认完成；草稿已保留，先重新读取再决定是否重试", "Not confirmed; draft retained. Refresh before retrying")
	register(KeyStoreAssistantWhaleIdentityChangedAfterSubmission, "操作后 Whale 身份已变化，请重新读取核对", "Whale identity changed after submission")
	register(KeyStoreAssistantHostAcceptedTheWriteButReadBackDi, "Host 已接受写入，但读回内容不一致；草稿已保留，请核对", "Host accepted the write but read-back differs; compare the preserved draft")
	register(KeyStoreAssistantSavedAndReadBackFromHost, "已保存并从 Host 重新读取", "Saved and read back from Host")
	register(KeyStoreAssistantClearConfirmedByHostCurrentContents, "清空已由 Host 确认；已读取当前内容", "Clear confirmed by Host; current contents loaded")

	// store/dshbot
	register(KeyStoreDshbotReadUnavailableDraftRetained, "头像未核实；保留草稿，请刷新", "Read unavailable; draft retained")
	register(KeyStoreDshbotMatchingProfileFoundInspectAndAdopt, "完整资料已匹配审阅值，核对完整头像后可采用读回", "Matching profile found; inspect and adopt")
	register(KeyStoreDshbotUnknownOutcomeRemainsLocked, "结果仍未知；仅保留读回，不允许重发。请在 Bot Web 核对", "Unknown outcome remains locked")
	register(KeyStoreDshbotReadBackReviewRemainingChanges, "已读回；剩余草稿需要重新审阅", "Read back; review remaining changes")
	register(KeyStoreDshbotNotSentRefreshAndReview, "未发送；草稿保留，刷新重新审阅", "Not sent; refresh and review")
	register(KeyStoreDshbotNotSentRefresh, "未发送；刷新重新审阅", "Not sent; refresh")
	register(KeyStoreDshbotUnknownReadBackOnly, "结果未知；保留并锁定草稿，只读回，不自动重试", "Unknown; read back only")
	register(KeyStoreDshbotProfileSavedVerifyPresentationWarnin, "Host 已返回完整头像回执；请刷新核对展示、警告与目标状态", "Profile saved; verify presentation, warnings and goal state")
	register(KeyStoreDshbotAdoptedCatalogPresentationRenameAnd, "已采用完整匹配资料，没有重发；会话展示/重命名及目标同步需另行核实", "Adopted catalog; presentation, rename and goal sync remain unverified")
	register(KeyStoreDshbotExactCatalogRecordFoundInspectAndA, "目录已有同身份记录；核对后采用，再单独打开 canonical 会话", "Exact catalog record found; inspect and adopt")
	register(KeyStoreDshbotOutcomeRemainsUnknownNoRetry, "读回未确认完整匹配记录。结果仍未知，不解锁重建；可使用 Bot Web 入口核对", "Outcome remains unknown; no retry")
	register(KeyStoreDshbotNotSentDraftRetained, "未发送创建；草稿保留，刷新后重新审阅", "Not sent; draft retained")
	register(KeyStoreDshbotRejectedBeforeWriteDraftRetained, "写入前拒绝；保留草稿，请刷新后重新审阅", "Rejected before write; draft retained")
	register(KeyStoreDshbotUnknownOutcomeReadBackWithoutRetry, "创建结果未知，草稿与身份已锁定。仅可读回，不重复创建", "Unknown outcome; read back without retry")
	register(KeyStoreDshbotBotCreatedOpeningFailureIsSeparate, "Host 已确认Bot和专属会话创建。打开失败不代表创建失败", "Bot created; opening failure is separate")
	register(KeyStoreDshbotAdoptedExistingBotOpeningVerifiesRe, "已采用目录中的同身份Bot，未重发创建。打开会核实或恢复专属会话", "Adopted existing bot; opening verifies/restores its canonical session")
	register(KeyStoreDshbotReadbackDoesNotSettleTheUnknownAtt, "只读核验不能证明停止、目录、任务或 notepad 的最终结果；原删除仍未知，禁止重发", "Readback does not settle the unknown attempt")
	register(KeyStoreDshbotPreviewReadDeletionNotAuthorized, "完整预览已读取；尚未授权删除", "Preview read; deletion not authorized")
	register(KeyStoreDshbotDeletionPreviewExpired, "删除预览已失效，请重新读取", "Deletion preview expired")
	register(KeyStoreDshbotCancelledBeforeDispatch, "已取消，未发送删除；下次需重新读取完整预览", "Cancelled before dispatch")
	register(KeyStoreDshbotDeletionNotSent, "未发送删除；需重新读取预览并当次确认", "Deletion not sent")
	register(KeyStoreDshbotRejectedBeforeEffects, "Host 在副作用前明确拒绝；请重新读取预览并确认", "Rejected before effects")
	register(KeyStoreDshbotUnknownSideEffectsMayAlreadyHaveOc, "删除结果未知；可能已停止代理、取消任务、改目录或删除 notepad。只能只读核验，禁止自动重试或假定回滚", "Unknown; side effects may already have occurred")
	register(KeyStoreDshbotCandidatesDoNotEstablishCausalityNo, "仅列出新出现且配置相符的目录候选；不能证明是此请求创建，原 attempt 保持未知且不可重发。Bot 记忆复制结果也未知", "Candidates do not establish causality; no retry")
	register(KeyStoreDshbotUnknownDuplicateOutcomeCandidatesAre, "复制结果未知，原来源与本次输入已锁定。Host 自生新身份；只能读回候选，不能自动归因或重试", "Unknown duplicate outcome; candidates are not causal receipts")
	register(KeyStoreDshbotProfileSessionReceiptConfirmedInspec, "Host 已确认副本资料与新会话创建；记忆复制状态请看完整 warning。不会自动打开，请单独选择", "Profile/session receipt confirmed; inspect warnings before opening")
	register(KeyStoreDshbotExactReadbackInspectAndAdopt, "完整目录读回匹配原 attempt；核对后显式采用，不会重发", "Exact readback; inspect and adopt")
	register(KeyStoreDshbotStillUnknownNoRetry, "仍未确认原写入；保留锁定，只读核对或使用 Web 管理", "Still unknown; no retry")
	register(KeyStoreDshbotNotSentRefreshAndReview2, "未发送；刷新目录后重新审阅", "Not sent; refresh and review")
	register(KeyStoreDshbotRejectedBeforeWrite, "Host 写入前拒绝；刷新后重新审阅", "Rejected before write")
	register(KeyStoreDshbotUnknownOutcomeNoRetry, "隐藏/恢复结果未知；保留原身份，只读回，不重发", "Unknown outcome; no retry")
	register(KeyStoreDshbotConfirmedVisibilityHiddenItemsRemain, "Host 已确认精确隐藏状态；可在此显示隐藏项并恢复", "Confirmed visibility; hidden items remain manageable here")
	register(KeyStoreDshbotAdoptedExactReadbackNoResend, "已采用完整匹配读回；未重发", "Adopted exact readback; no resend")
	register(KeyStoreDshbotReadFailedDraftRetainedAndLocked, "未能核实当前能力资料；草稿保留且禁写，请刷新", "Read failed; draft retained and locked")
	register(KeyStoreDshbotReadBackReviewPreservedDraftAgainst, "已读回当前 Host 资料；保留的草稿需按当前版本重新审阅", "Read back; review preserved draft against current state")
	register(KeyStoreDshbotOutcomeUncertainReadBackBeforeSavin, "保存结果未知；草稿锁定，先读回核实，不自动重试", "Outcome uncertain; read back before saving again")
	register(KeyStoreDshbotPreflightChangedNoSaveSent, "发送前资料或权限已变化，未保存；草稿保留，请刷新重新审阅", "Preflight changed; no save sent")
	register(KeyStoreDshbotSavedReceiptReadBackToReconcile, "Host 已返回保存回执；草稿保留，刷新核实实际配置与警告", "Saved receipt; read back to reconcile")
	register(KeyStoreDshbotRefreshedCatalogConfirmsTheOperation, "读回已确认操作结果", "Refreshed catalog confirms the operation")
	register(KeyStoreDshbotOutcomeUnknownRefreshWithoutResendin, "原操作结果尚未确认；保留操作并只读刷新，不会自动重发", "Outcome unknown; refresh without resending")
	register(KeyStoreDshbotOutcomeUnknownRefreshToReconcile, "操作结果未知；请刷新目录核实，未重发", "Outcome unknown; refresh to reconcile")
	register(KeyStoreDshbotDirectorySaved, "目录已保存", "Directory saved")
	register(KeyStoreDshbotUnassigned, "未分组", "Unassigned")
	register(KeyStoreDshbotMemoryUnavailableDraftRetained, "读取记忆失败；保留草稿，重新读取后才可保存", "Memory unavailable; draft retained")
	register(KeyStoreDshbotBotIdentityChanged, "Bot 会话身份已改变，旧编辑已隔离", "Bot identity changed")
	register(KeyStoreDshbotReadbackMatchesSubmittedContentNoRe, "刷新已核实 Host 当前内容与提交内容一致，未重复保存", "Readback matches submitted content; no repeat write")
	register(KeyStoreDshbotRefreshedWithoutRetryingOrRebasingD, "已读取 Host 当前内容；草稿未自动覆盖或重试，请重新核对", "Refreshed without retrying or rebasing draft")
	register(KeyStoreDshbotMemoryConflictNoAutomaticRebase, "此轨记忆已改变；保留草稿但禁止静默覆盖。请先核对当前内容，再明确放弃旧草稿后重新编辑", "Memory conflict; no automatic rebase")
	register(KeyStoreDshbotMemoryRevisionConflict, "记忆版本冲突，保留草稿；请核对 Host 当前内容", "Memory revision conflict")
	register(KeyStoreDshbotEntryMustNotBeEmpty, "记忆条目不能为空", "Entry must not be empty")
	register(KeyStoreDshbotNoMemoryChanges, "内容未改变", "No memory changes")
	register(KeyStoreDshbotSaveUncertainRefreshToVerify, "保存结果未确认；草稿保留且不自动重试。请刷新核实 Host 当前内容", "Save uncertain; refresh to verify")
	register(KeyStoreDshbotHostConfirmedMemoryReplacement, "Host 已确认此轨记忆替换；未启动 Bot", "Host confirmed memory replacement")
	register(KeyStoreDshbotReadUnavailableDraftRetainedAndLock, "自动记忆配置未能核实；保留并锁定草稿，请刷新", "Read unavailable; draft retained and locked")
	register(KeyStoreDshbotReadBackReviewAnyRemainingDraft, "已读回配置；不同于 Host 的草稿须重新审阅", "Read back; review any remaining draft")
	register(KeyStoreDshbotOutcomeUncertainReadBackBeforeSavin2, "保存结果未知；保留并锁定草稿，先读回，不自动重试", "Outcome uncertain; read back before saving")
	register(KeyStoreDshbotPreflightChangedNoSaveSent2, "发送前资料或权限变化，未保存；刷新重新审阅", "Preflight changed; no save sent")
	register(KeyStoreDshbotReceiptReceivedReadBackToVerify, "Host 返回保存回执；刷新核实配置与警告", "Receipt received; read back to verify")
	register(KeyStoreDshbotReadUnavailableDraftRetained2, "通知配置未核实；保留草稿，请刷新", "Read unavailable; draft retained")
	register(KeyStoreDshbotMatchingProfileFoundInspectAndAdopt2, "完整资料已匹配审阅值，核对通知开关及Host/Bot身份后可采用读回", "Matching profile found; inspect and adopt")
	register(KeyStoreDshbotProfileSavedVerifyPresentationWarnin2, "Host 已返回完整通知配置回执；请刷新核对展示、警告与目标状态", "Profile saved; verify presentation, warnings and goal state")
	register(KeyStoreDshbotReadUnavailableDraftRetainedAndLock2, "目标配置未能核实；保留并锁定草稿，请刷新", "Read unavailable; draft retained and locked")
	register(KeyStoreDshbotObjectiveStateUnavailableRefreshBefo, "目标状态未能核实；保留旧快照但禁止操作，请刷新", "Objective state unavailable; refresh before actions")
	register(KeyStoreDshbotRefreshedWithoutStartingABot, "已读取当前 Host 的目录目标和 live 摘要；未运行 Bot", "Refreshed without starting a Bot")
	register(KeyStoreDshbotOutcomeUncertainRefreshBeforeAnother, "目标操作结果未确认；未自动重试。先刷新状态，不要重复发送", "Outcome uncertain; refresh before another action")
	register(KeyStoreDshbotPreflightChangedOrUnavailableNoActi, "发送前目标或连接状态无法核实，未发送目标操作；请刷新重新审阅", "Preflight changed or unavailable; no action sent")
	register(KeyStoreDshbotHostReceiptReceivedRefreshToVerify, "Host 已返回操作回执；这不证明目标完成或运行已中断。请刷新核对当前状态", "Host receipt received; refresh to verify live state")
	register(KeyStoreDshbotLimitsMustBeIntegers, "轮数和发言次数须为整数", "Limits must be integers")
	register(KeyStoreDshbotRoomCreatedOpeningFailureIsSeparate, "Host 已确认群聊和专属会话创建。打开失败不代表创建失败", "Room created; opening failure is separate")
	register(KeyStoreDshbotAdoptedExistingRoomOpeningVerifiesR, "已采用目录中的同身份群聊，未重发创建。打开会核实或恢复专属会话", "Adopted existing room; opening verifies/restores its canonical session")
	register(KeyStoreDshbotExactReviewedProfileFoundInspectAnd, "读回已匹配审阅后的完整资料；核对后采用", "Exact reviewed profile found; inspect and adopt")
	register(KeyStoreDshbotNotConfirmedNoRetry, "读回尚未完整匹配；草稿仍锁定，不重发。可通过 Bot Web 核对", "Not confirmed; no retry")
	register(KeyStoreDshbotLimitsMustBeIntegers2, "轮数和发言上限须为整数", "Limits must be integers")
	register(KeyStoreDshbotNoChangesToSave, "草稿没有变更", "No changes to save")
	register(KeyStoreDshbotNotSentDraftRetained2, "未发送保存，草稿保留；刷新后重新审阅", "Not sent; draft retained")
	register(KeyStoreDshbotNotSentDraftRetained3, "本地校验拒绝，未发送；草稿保留", "Not sent; draft retained")
	register(KeyStoreDshbotUnknownOutcomeReadBackOnly, "保存结果未知；保留并锁定草稿，仅可读回，不自动重试", "Unknown outcome; read back only")
	register(KeyStoreDshbotHostConfirmedGroupUpdate, "Host 已确认群聊资料保存", "Host confirmed group update")
	register(KeyStoreDshbotAdoptedMatchingCatalogPresentationSi, "已采用读回的完整匹配资料；没有重发更新。会话展示副作用不能仅凭目录核实", "Adopted matching catalog; presentation side effects remain unverified")
	register(KeyStoreDshbotRuntimeUnavailableAnyOldSnapshotIs, "运行态不可用；旧快照不代表当前状态", "Runtime unavailable; any old snapshot is stale")
	register(KeyStoreDshbotReadOnlySnapshotNoAutomaticMemberA, "只读快照；不会自动 follow 或启动成员", "Read-only snapshot; no automatic member activation")
	register(KeyStoreDshbotOpenFailedRefreshBeforeAnotherExpli, "打开失败；未创建或执行成员会话，请重新读取状态", "Open failed; refresh before another explicit open")
	register(KeyStoreDshbotReadUnavailableDraftRetained3, "模型/工作区未核实；保留草稿，请刷新", "Read unavailable; draft retained")
	register(KeyStoreDshbotMatchingProfileFoundInspectAndAdopt3, "完整资料已匹配审阅值，核对新会话后可采用读回", "Matching profile found; inspect and adopt")
	register(KeyStoreDshbotSavedUseExistingEntryToOpenAfterR, "Host 已返回完整保存回执；尚未打开新会话，刷新后可从现有入口打开", "Saved; use existing entry to open after refresh")
	register(KeyStoreDshbotAdoptedCatalogSessionAndLiveSynchro, "已采用完整匹配资料，没有重发；会话创建及目标同步需另行核实", "Adopted catalog; session and live synchronization remain unverified")
	register(KeyStoreDshbotContextSourceIsNotOwnedByThisBot, "上下文来源不属于此 Bot", "Context source is not owned by this Bot")
	register(KeyStoreDshbotAtMost3ContextSources, "最多选择 3 个上下文来源", "At most 3 context sources")
	register(KeyStoreDshbotUnsupportedWatchKind, "不支持的 watch 类型", "Unsupported watch kind")
	register(KeyStoreDshbotSelectASupportedWatchTypeFirst, "未知 watch 配置须先明确选择已支持类型", "Select a supported watch type first")
	register(KeyStoreDshbotWatchTimeoutMustBeIntegerMillisecon, "watch 超时须为整数毫秒", "Watch timeout must be integer milliseconds")
	register(KeyStoreDshbotRoutinesRefreshed, "例行任务已从当前 Host 刷新", "Routines refreshed")
	register(KeyStoreDshbotMissedRunStatusUnavailable, "；错过记录不可用：", "Missed-run status unavailable: ")
	register(KeyStoreDshbotCatalogChangedReviewCurrentValuesBe, "目录已变化。草稿保留；请核对当前记录后丢弃旧草稿再编辑", "Catalog changed; review current values before discarding the stale draft")
	register(KeyStoreDshbotRunLimitMustBeAnInteger, "执行次数上限必须为整数", "Run limit must be an integer")
	register(KeyStoreDshbotLimitMustExceedCurrentRunCount, "启用时次数上限必须大于已执行次数", "Limit must exceed current run count")
	register(KeyStoreDshbotAPreviousRunIsPendingOrRunning, "上一轮仍在排队或运行", "A previous run is pending or running")
	register(KeyStoreDshbotOutcomeMayHaveAppliedRefreshBefore, "；结果可能已生效，先刷新核实再操作", "Outcome may have applied; refresh before another action")
	register(KeyStoreDshbotHostReturnedCatalog, "Host 已返回目录：", "Host returned catalog: ")
	register(KeyStoreDshbotCheckStatusAndHistoryThisIsNotCom, "；执行结果请看状态与历史，不代表已完成", "Check status and history; this is not completion")
	register(KeyStoreDshbotBotSessionChangedInspectTheOriginal, "Bot 会话已更换。创建结果待核对，不能重发", "Bot session changed; inspect the original creation")
	register(KeyStoreDshbotBotIdentityChanged2, "Bot 会话已更换，请先核对原任务", "Bot identity changed")
	register(KeyStoreDshbotCurrentCatalogAdoptedReviewCreation, "已采用当前目录基线，请重新审阅保存", "Current catalog adopted; review creation again")
	register(KeyStoreDshbotCatalogChangedRecheckBeforeReviewing, "目录已变化；先核对当前目录再审阅创建", "Catalog changed; recheck before reviewing")
	register(KeyStoreDshbotRunLimitMustBeAnInteger2, "执行次数上限须为整数", "Run limit must be an integer")
	register(KeyStoreDshbotSpecifyTheIntendedIanaTimezone, "请明确填写 IANA 时区", "Specify the intended IANA timezone")
	register(KeyStoreDshbotInvalidIanaTimezone, "时区无效：", "Invalid IANA timezone: ")
	register(KeyStoreDshbotCreationNotSentReviewTheRetainedDr, "未发送创建请求；草稿保留，请重新审阅", "Creation not sent; review the retained draft again")
	register(KeyStoreDshbotCreationRejectedBeforeWriteDraftRet, "Host 在写入前拒绝创建；草稿保留，修正或刷新后重新审阅", "Creation rejected before write; draft retained")
	register(KeyStoreDshbotOutcomeUncertainInspectTheCatalogBe, "创建结果待核对，已锁定此请求与草稿；刷新只读目录，勿重复保存", "Outcome uncertain; inspect the catalog before another creation")
	register(KeyStoreDshbotCreatedRoutine, "Host 已返回新例行任务：", "Created routine: ")
	register(KeyStoreDshbotUsingTheInspectedExistingRecord, "已采用核对后的已有记录，未发送新建请求", "Using the inspected existing record")
	register(KeyStoreDshbotCronNeedsFiveFieldsAtMost200Chara, "Cron 须为五字段，最多 200 字符", "Cron needs five fields, at most 200 characters")
	register(KeyStoreDshbotEachCronFieldMayContainAtMost40C, "Cron 每字段最多 40 字符", "Each cron field may contain at most 40 characters")
	register(KeyStoreDshbotEnterAValidCalendarTime, "请填写有效时间（00–23时，00–59分）", "Enter a valid calendar time")
	register(KeyStoreDshbotSelectWeekday06, "请选择星期（0=周日，6=周六）", "Select weekday 0–6")
	register(KeyStoreDshbotDayOfMonthMustBe131, "每月日期须为 1–31", "Day of month must be 1–31")
	register(KeyStoreDshbotSelectASupportedFrequency, "请选择支持的任务类型", "Select a supported frequency")
	register(KeyStoreDshbotDurationMustBeAPositiveIntegerWith, "时长须为正整数和 m/h/d 单位", "Duration must be a positive integer with m/h/d")
	register(KeyStoreDshbotDurationMustBe1525600Minutes, "时长须为 1–525600 分钟（最多 365 天）", "Duration must be 1–525600 minutes")
	register(KeyStoreDshbotEnterAValidDailyTime, "请填写有效时间（00–23时，00–59分）", "Enter a valid daily time")
	register(KeyStoreDshbotEnterAOneShotDelaySuchAs30m1hOr, "一次性延迟格式须为 30m、1h 或 1d", "Enter a one-shot delay such as 30m, 1h or 1d")
	register(KeyStoreDshbotRefreshBeforeFurtherChangesRefreshC, "获取凭据可能改变目录版本；其他写操作前请刷新。刷新会清除本页明文", "Refresh before further changes; refresh clears this credential view")
	register(KeyStoreDshbotCredentialOutcomeUncertainNoAutomati, "未能取得可用凭据；Host 可能已创建它，未自动重试。关闭后先刷新，再独立确认获取", "Credential outcome uncertain; no automatic retry")
	register(KeyStoreDshbotReadUnavailableDraftRetained4, "消息与任务来源未核实；保留草稿，请刷新", "Read unavailable; draft retained")
	register(KeyStoreDshbotMatchingProfileFoundInspectAndAdopt4, "完整资料已匹配审阅值，核对来源规则后可采用读回", "Matching profile found; inspect and adopt")
	register(KeyStoreDshbotSavedProfileLiveSynchronizationAndE, "Host 已返回完整保存回执；目标同步与实际执行尚未核实", "Saved profile; live synchronization and execution unverified")
	register(KeyStoreDshbotAdoptedCatalogLiveSynchronizationAnd, "已采用完整匹配资料，没有重发；目标同步与实际执行需另行核实", "Adopted catalog; live synchronization and execution remain unverified")
	register(KeyStoreDshbotTasksRefreshed, "任务已从当前 Host 刷新", "Tasks refreshed")
	register(KeyStoreDshbotCatalogChangedReviewTheActionAgain, "目录已变化。关闭确认框并重新核对操作", "Catalog changed; review the action again")
	register(KeyStoreDshbotHostAccepted, "Host 已确认：", "Host accepted: ")
	register(KeyStoreDshbotStateSavedWakeFailed, "；任务状态已保存，但唤醒失败：", "State saved; wake failed: ")
	register(KeyStoreDshbotWakeNotConfirmedRefreshToInspect, "；Host 未确认唤醒，请刷新检查", "Wake not confirmed; refresh to inspect")
	register(KeyStoreDshbotReadUnavailableDraftRetained5, "文本资料未核实；保留草稿，请刷新", "Read unavailable; draft retained")
	register(KeyStoreDshbotMatchingProfileFoundInspectAndAdopt5, "完整资料已匹配审阅值，核对三项完整文本后可采用读回", "Matching profile found; inspect and adopt")
	register(KeyStoreDshbotProfileSavedVerifyPresentationWarnin3, "Host 已返回完整文本资料回执；请刷新核对展示、警告与目标状态", "Profile saved; verify presentation, warnings and goal state")

	// store/imonboard
	register(KeyStoreImonboardBotUnavailableRefreshToRetry, "机器人已不可用，请刷新后重试", "Bot unavailable; refresh to retry")
	register(KeyStoreImonboardEnterAnAbsoluteWorkspacePath, "请输入工作区绝对路径", "Enter an absolute workspace path")
	register(KeyStoreImonboardPresetUnavailableOnThisHost, "Agent Preset 不在当前 Host 公开目录中", "Preset unavailable on this Host")
	register(KeyStoreImonboardLark, "飞书", "Lark")
	register(KeyStoreImonboardConnectionStatusRefreshFailedRefresh, "连接状态刷新失败，可手动刷新重试", "Connection status refresh failed; refresh to retry")
	register(KeyStoreImonboardIncompleteHostConnectionStatus, "Host 状态格式不完整", "Incomplete Host connection status")

	// store/remote
	register(KeyStoreRemoteRemovalCouldNotBeVerifiedTheHostS, "删除未能确认：Host 仍返回此机器，请刷新后重试。", "Removal could not be verified: the Host still lists this machine; refresh before retrying.")
	register(KeyStoreRemoteStatusRefreshFailedRefreshToCheckT, "；状态刷新失败，请刷新。", "Status refresh failed; refresh to check the remaining state.")

	// store/schedule
	register(KeyStoreScheduleEndedNoFutureDeliveries, "已结束，不再执行", "Ended; no future deliveries")
	register(KeyStoreScheduleUnknownStatusRefreshToVerify, "状态未知，请刷新", "Unknown status; refresh to verify")
	register(KeyStoreScheduleNextRunUnavailableRefresh, "下次执行时间不可用，请刷新", "Next run unavailable; refresh")
	register(KeyStoreScheduleNextRun, "下次执行：", "Next run: ")
	register(KeyStoreScheduleRuleZoneUnavailable, " · 规则时区无法显示：", "Rule zone unavailable: ")
	register(KeyStoreScheduleAllSessions, "全部会话", "All sessions · ")
	register(KeyStoreScheduleCurrentSessionOnly, "仅当前会话：", "Current session only: ")
	register(KeyStoreScheduleHostCatalogUnavailable, " · 全局目录不可用", "Host catalog unavailable")

	// store/session
	register(KeyStoreSessionChildPromptSubmitted, "子代理输入已提交", "Child prompt submitted")
	register(KeyStoreSessionInterruptRequested, "中断请求已提交", "Interrupt requested")
	register(KeyStoreSessionLiveSequenceChangedResynchronizing, "实时序列不连续，正在重新同步…", "Live sequence changed; resynchronizing…")
	register(KeyStoreSessionRestoredTargetFailedSourceCatalogVe, "恢复目标未通过来源目录验证", "Restored target failed source catalog verification")
	register(KeyStoreSessionCreatorConfirmationExpired, "Creator 确认已失效；请在当前 Host 和页面重新准备", "Creator confirmation expired")
	register(KeyStoreSessionConnectAHostFirst, "请先连接 Host", "Connect a Host first")
	register(KeyStoreSessionChooseAWorkspaceFirst, "请选择工作目录，或等待 Host 提供无目录工作区", "Choose a workspace first")
	register(KeyStoreSessionCannotAllocateATaskIdentity, "无法生成新任务身份", "Cannot allocate a task identity")
	register(KeyStoreSessionCouldNotVerifyCordisNoTaskWasCrea, "无法确认 cordis 是否可用；尚未创建任务", "Could not verify cordis; no task was created")
	register(KeyStoreSessionCordisIsUnavailableNoTaskWasCreate, "当前 Host 的 cordis 缺失、隐藏或损坏；尚未创建任务", "cordis is unavailable; no task was created")
	register(KeyStoreSessionNavigationChangedNoTaskWasCreated, "页面或 Host 已改变，未创建任务", "Navigation changed; no task was created")
	register(KeyStoreSessionCreationOutcomeUnknownLocateTheReta, "创建结果未确认；保留目标 ID，请定位已有任务，勿重复创建", "Creation outcome unknown; locate the retained ID before creating again")
	register(KeyStoreSessionTaskCreatedNavigationChangedBeforeC, "任务已创建，页面或 Host 已改变；未应用 cordis", "Task created; navigation changed before cordis was applied")
	register(KeyStoreSessionTaskCreatedCordisOutcomeUnconfirmed, "任务已创建，cordis 应用结果未确认；可打开该任务检查，不会自动重试", "Task created; cordis outcome unconfirmed; open the existing task to inspect")
	register(KeyStoreSessionCordisAppliedNewerNavigationPreserve, "已在该任务应用 cordis；页面已改变，未切换当前会话", "cordis applied; newer navigation preserved")
	register(KeyStoreSessionCouldNotLocateTaskIdRetained, "读取任务列表失败；目标 ID 保留", "Could not locate task; ID retained")
	register(KeyStoreSessionHostOrPageChangedReadbackNotAdopte, "Host 或页面已改变；定位结果未采纳", "Host or page changed; readback not adopted")
	register(KeyStoreSessionExistingTaskLocatedInspectItsPreset, "已定位此目标；cordis 状态未确认，请打开检查", "Existing task located; inspect its preset in the composer")
	register(KeyStoreSessionThisIdIsNotConfirmedInTheListNo, "列表尚未确认此目标；不会据此重建", "This ID is not confirmed in the list; no replacement was created")
	register(KeyStoreSessionCommandOwnerChanged, "命令所属会话或扩展已变更", "Command owner changed")
	register(KeyStoreSessionExplicitPermissionRiskAcknowledgement, "请确认权限风险", "Explicit permission risk acknowledgement required")
	register(KeyStoreSessionSkipped, "未作答", "Skipped")
	register(KeyStoreSessionAnswer, "回答：", "Answer: ")
	register(KeyStoreSessionPermanentlyDeleteThisSkillDirectory, "永久删除此 Skill 目录及其中内容，无法在应用内撤销。", "Permanently delete this Skill directory and its contents; there is no in-app undo. Review the exact target, then confirm or cancel.")
	register(KeyStoreSessionSpeechHostOffline, "语音 Host 未连接", "Speech Host offline")
	register(KeyStoreSessionSpeechUnavailable, "语音服务不可用，请检查语音插件", "Speech unavailable")
	register(KeyStoreSessionPrepareTheSelectedSpeechProvider, "请先在语音输入插件设置准备所选提供方", "Prepare the selected speech provider")
	register(KeyStoreSessionInvalidSpeechLimits, "语音服务限额无效", "Invalid speech limits")
	register(KeyStoreSessionSpeechLimitTooSmall, "语音服务限额过小", "Speech limit too small")
	register(KeyStoreSessionSpeechSessionChanged, "语音会话已变化", "Speech session changed")
	register(KeyStoreSessionTranscriptionFailed, "转写失败，原草稿未改动", "Transcription failed")
	register(KeyStoreSessionNoSpeechRecognized, "未识别到文字", "No speech recognized")
	register(KeyStoreSessionTranscriptTooLarge, "转写文字过长", "Transcript too large")
	register(KeyStoreSessionInvalidRecording, "无有效录音或超出限额", "Invalid recording")
	register(KeyStoreSessionPublishedHtmlUnavailable, "无法读取已发布 HTML：", "Published HTML unavailable: ")
	register(KeyStoreSessionLivePreview, " · 实时临时输出", "live preview")

	// store/settings
	register(KeyStoreSettingsCustomInstructionsMustNotExceed1500, "自定义指令不能超过 1500 字符", "Custom instructions must not exceed 1500 UTF-16 units")

	// store/usage
	register(KeyStoreUsagePricesMustBeFiniteNumbersIn010000, "价格须为 0–1000000 有限数值", "Prices must be finite numbers in [0, 1000000]")
	register(KeyStoreUsagePricesSaved, "价格已保存", "Prices saved")
	register(KeyStoreUsagePostRepairStatisticsRescanFailed, "修复后的统计重扫失败：", "Post-repair statistics rescan failed: ")
	register(KeyStoreUsageUsageStatisticsUnavailable, "用量统计后端不可用：", "Usage statistics unavailable: ")

	// desktop/botnotify
	register(KeyDesktopBotnotifyBotNeedsYourAttention, "Bot 需要处理待确认事项", "Bot needs your attention")
	register(KeyDesktopBotnotifyBotObjectiveIsBlocked, "Bot 目标遇到阻碍", "Bot objective is blocked")
	register(KeyDesktopBotnotifyRoutineDidNotCompleteSuccessfully, "例行任务结束但未成功", "Routine did not complete successfully")
	register(KeyDesktopBotnotifyRoutineCompleted, "例行任务已完成", "Routine completed")

	// desktop/pet
	register(KeyDesktopPetCouldNotSaveHideTryAgain, "无法保存隐藏设置，请重试", "Could not save Hide; try again")
	register(KeyDesktopPetOpeningPetChat, "正在打开宠物会话…", "Opening pet chat…")
	register(KeyDesktopPetMissingOrChangedTha4AssetsFallback, "THA4 资源不可用，已回退", "Missing or changed THA4 assets; fallback")
	register(KeyDesktopPetFilesStayUntouched, "不读取内容，不修改文件", "Files stay untouched")
	register(KeyDesktopPetChooseOneFile, "选择一个文件投喂", "Choose one file")
	register(KeyDesktopPetFeedOnceFileStaysUntouched, "投喂一次（不读取或修改文件）", "Feed once; file stays untouched")
	register(KeyDesktopPetFileSelectionFailed, "文件选择失败", "File selection failed")
	register(KeyDesktopPetSaveFailedNotApplied, "保存失败，未计入状态", "Save failed; not applied")
	register(KeyDesktopPetFed, "已投喂：饱食 +2，心情 +1（上限 100）", "Fed")
	register(KeyDesktopPetFed2, "」的心意！饱食 +2，心情 +1；文件未读取或改动", "Fed")
	register(KeyDesktopPetLedgerUnavailable, "成长账本不可用，未改动存档", "Ledger unavailable")
	register(KeyDesktopPetLocalGameOnly, "刷新实际用量；首次仅建基线", "Local game only")
	register(KeyDesktopPetReadingLocalUsage, "读取本应用实际用量…", "Reading local usage…")
	register(KeyDesktopPetCancelled, "读取已取消，未改动账本", "Cancelled")
	register(KeyDesktopPetSaveFailedUnchanged, "账本保存失败，粮与成长未改动", "Save failed; unchanged")
	register(KeyDesktopPetSaveFailedUnchanged2, "保存失败，粮与成长均未扣改", "Save failed; unchanged")
	register(KeyDesktopPetNoLocalFood, "粮碗空了，先刷新实际用量", "No local food")
	register(KeyDesktopPetInteractionWasNotSaved, "互动保存失败", "Interaction was not saved")
	register(KeyDesktopPetPleaseWaitBeforeAnotherGlance, "请稍等再看看", "Please wait before another glance")
	register(KeyDesktopPetWaitingForScreenshotApproval, "等待本次截图确认", "Waiting for screenshot approval")
	register(KeyDesktopPetLooking, "正在看看…", "Looking…")
	register(KeyDesktopPetPetConfigurationUnavailable, "桌宠配置不可用", "Pet configuration unavailable")
	register(KeyDesktopPetRefreshTheImageCatalogOrTurnLookO, "所选视觉路由不在当前图像目录中，请刷新或关闭看看", "Refresh the image catalog or turn Look off")
	register(KeyDesktopPetPetPositionWasNotSaved, "桌宠位置保存失败：", "Pet position was not saved: ")
	register(KeyDesktopPetSaveFailedUnchanged3, "保存失败，未计入互动", "Save failed; unchanged")
	register(KeyDesktopPetPatted, "已摸摸头：心情 +10；亲密按同类冷却计入", "Patted")
	register(KeyDesktopPetLoadingSharedConversation, "正在读取共同会话…", "Loading shared conversation…")
	register(KeyDesktopPetSendUsesDesktopApiSettings, " 联网发送使用桌面 API 设置", "Send uses desktop API settings")
	register(KeyDesktopPetHostUnavailable, "Host 地址不可用", "Host unavailable")
	register(KeyDesktopPetWhaleQuickChat, "鲸鱼娘 · 聊聊", "Whale quick chat")
	register(KeyDesktopPetQuickChatWindowCouldNotOpen, "聊天卡片创建失败", "Quick chat window could not open")
	register(KeyDesktopPetConnectionChangedReopenQuickChat, "Host 或设置已变化，草稿保留。重新点聊聊连接。", "Connection changed; reopen quick chat.")
	register(KeyDesktopPetSharedHistoryUnavailable, "共同会话暂不可读，请检查 Host", "Shared history unavailable")
	register(KeyDesktopPetSharedWithMainConversation, "与主窗口使用同一持久会话", "Shared with main conversation")
	register(KeyDesktopPetPetSessionChanged, "宠物会话已更新；旧请求已停止等待，草稿保留", "Pet session changed")
	register(KeyDesktopPetModelCatalogUnavailable, "模型目录暂不可用", "Model catalog unavailable")
	register(KeyDesktopPetHostModelCatalogUnavailable, "Host 未提供模型目录", "Host model catalog unavailable")
	register(KeyDesktopPetEnterUpTo2000Utf16Units, "请输入 1–2000 字；表情可能占两字", "Enter up to 2000 UTF-16 units")
	register(KeyDesktopPetWaitingForSharedSession, "正在等待共同会话回复…", "Waiting for shared session…")
	register(KeyDesktopPetTurnCompletedSyncingHistory, "本轮已完成，正在同步记录", "Turn completed; syncing history")
	register(KeyDesktopPetPetSessionChangedBeforeSend, "会话已改变，消息未发送；请重新查看", "Pet session changed before send")
	register(KeyDesktopPetCompletionUnconfirmedCheckHistory, "未确认本轮完成；请查看共同记录，勿重复发送", "Completion unconfirmed; check history")
	register(KeyDesktopPetWaitCancelledApiMayContinue, "已取消本地等待；API 可能已收到请求，请勿直接重复发送", "Wait cancelled; API may continue")
	register(KeyDesktopPetWaitCancelledHostMayContinue, "已取消本地等待。Host 可能仍在运行或排队。", "Wait cancelled; Host may continue.")
	register(KeyDesktopPetChangingSharedSessionModel, "正在切换共同会话模型…", "Changing shared session model…")
	register(KeyDesktopPetModelUpdated, "模型已更新", "Model updated")
	register(KeyDesktopPetModelChangeUnconfirmed, "模型未确认切换，请刷新后重试", "Model change unconfirmed")
	register(KeyDesktopPetStoppingTheCurrentPetTurn, "请求停止宠物会话当前轮…", "Stopping the current pet turn…")
	register(KeyDesktopPetStopAcceptedQueuedMessagesRemain, "Host 已接受停止当前轮；排队消息请到主会话管理", "Stop accepted; queued messages remain")
	register(KeyDesktopPetStopUnconfirmedCheckMainConversation, "未确认停止，请到主会话查看", "Stop unconfirmed; check main conversation")
	register(KeyDesktopPetSessionChangedOrUnavailable, "会话身份已变化或不可用，请刷新", "Session changed or unavailable")
	register(KeyDesktopPetSendingToConfiguredApi, "正在请求已配置 API；临时聊天不会写入共同会话", "Sending to configured API…")
	register(KeyDesktopPetDesktopApiCredentialsUnavailable, "桌面 API 凭据不可用，请在设置中检查", "Desktop API credentials unavailable")
	register(KeyDesktopPetApiEndpointIsInvalidOrInsecure, "API 地址无效或使用非本机明文 HTTP", "API endpoint is invalid or insecure")
	register(KeyDesktopPetSettingsChangedCloseAndReopen, "API 或人设设置已变化；关闭后重新点聊聊", "Settings changed; close and reopen")
	register(KeyDesktopPetApiResultUnconfirmedAvoidDuplicateS, "未确认 API 回复，请勿直接重复发送；服务端可能已收到", "API result unconfirmed; avoid duplicate sends")
	register(KeyDesktopPetTemporaryReplyReceived, "临时回复已收到；关闭窗口后清除", "Temporary reply received")
	register(KeyDesktopPetOpen, "↗ 主会话", "Open")
	register(KeyDesktopPetCloseQuickChat, "关闭聊天卡片", "Close quick chat")
	register(KeyDesktopPetClosingOnlyCancelsTheLocalWait, "关闭只取消本地等待，服务端可能继续", "Closing only cancels the local wait")
	register(KeyDesktopPetTemporaryOnlineApiChat, "联网临时聊天 · deepseek-chat · 使用桌面 API 设置；仅保留本窗口最近 6 轮，不写入共同会话", "Temporary online API chat")
	register(KeyDesktopPetOneSharedConversation, "与主窗口共享模型、上下文和持久记录", "One shared conversation")
	register(KeyDesktopPetPetSessionId, "宠物会话 ID", "Pet session ID")
	register(KeyDesktopPetLoadingSharedHistory, "正在读取共同记录…", "Loading shared history…")
	register(KeyDesktopPetClickSendToApiToSend, "输入后点联网发送才会请求 API", "Click Send to API to send")
	register(KeyDesktopPetNoMessagesYet, "还没有文字记录", "No messages yet")
	register(KeyDesktopPetConnectionUnavailable, "连接不可用；草稿仍保留", "Connection unavailable")
	register(KeyDesktopPetSharedHistory, "共同会话记录", "Shared history")
	register(KeyDesktopPetTemporaryApiHistory, "临时 API 聊天记录", "Temporary API history")
	register(KeyDesktopPetYou, "你", "You")
	register(KeyDesktopPetStatus, "状态", "Status")
	register(KeyDesktopPetSession, "会话", "Session")
	register(KeyDesktopPetQuickChatDraft, "聊天草稿", "Quick chat draft")
	register(KeyDesktopPetEnterSendsShiftEnterNewline, "说点什么… Enter 发送，Shift+Enter 换行", "Enter sends; Shift+Enter newline")
	register(KeyDesktopPetSend, "发送", "Send")
	register(KeyDesktopPetSendToApi, "联网发送", "Send to API")
	register(KeyDesktopPetCancelWait, "取消等待", "Cancel wait")
	register(KeyDesktopPetStopTurn, "停止当前轮", "Stop turn")
	register(KeyDesktopPetRefresh, "刷新", "Refresh")
	register(KeyDesktopPetModel, "模型", "Model")
	register(KeyDesktopPetSelectSharedModel, "选择共同会话模型", "Select shared model")
	register(KeyDesktopPetReasoning, "思考：", "Reasoning: ")
	register(KeyDesktopPetDefault, "默认", "Default")
	register(KeyDesktopPetSelectReasoningEffort, "选择思考强度", "Select reasoning effort")
	register(KeyDesktopPetSleep, "睡觉", "Sleep")
	register(KeyDesktopPetAwake, "清醒", "Awake")
	register(KeyDesktopPetDozing, "打盹中", "Dozing")
	register(KeyDesktopPetWake, "叫醒", "Wake")
	register(KeyDesktopPetResting, "休眠中", "Resting")
	register(KeyDesktopPetSaveFailedStillResting, "唤醒保存失败，仍保持休眠", "Save failed; still resting")
	register(KeyDesktopPetUnknown, "状态未知", "Unknown")
	register(KeyDesktopPetRunning, "运行中", "Running")
	register(KeyDesktopPetIdle, "未运行", "Idle")
	register(KeyDesktopPetOffline, "离线", "Offline")
	register(KeyDesktopPetOtherConversationsAreWorking, "其他会话正在运行", "Other conversations are working")

	// desktop/pluginrpc
	register(KeyDesktopPluginrpcUnsupportedUnicodeUseExistingWebEdi, "资料包含不能无损解码的 Unicode，保留 Web 入口", "Unsupported Unicode; use existing Web editor")
	register(KeyDesktopPluginrpcProfileCannotBeSafelyRoundTripped, "无法证明资料无损，保留 Web 编辑入口：", "Profile cannot be safely round-tripped: ")
	register(KeyDesktopPluginrpcRefreshTheContactIdentity, "联系人已变化，请刷新目录", "Refresh the contact identity")
	register(KeyDesktopPluginrpcRefreshTheCatalogFirst, "请先刷新目录", "Refresh the catalog first")
	register(KeyDesktopPluginrpcSectionName1120Characters, "分组名称需为 1–120 字符", "Section name: 1–120 characters")
	register(KeyDesktopPluginrpcSectionNameAlreadyExists, "分组名称已存在", "Section name already exists")
	register(KeyDesktopPluginrpcNoSectionSnapshot, "没有可恢复的分组快照", "No section snapshot")
	register(KeyDesktopPluginrpcUnsupportedContactsAction, "不支持的联系人操作", "Unsupported contacts action")
	register(KeyDesktopPluginrpcNotificationCatalogExceedsTheHostLi, "Bot 通知不可用：目录超过 200 个条目", "Notification catalog exceeds the Host limit of 200 items")
	register(KeyDesktopPluginrpcUseClearCatalogObjectiveToRemoveIt, "目标不能为空；清除请使用已有清除目录目标入口", "Use Clear catalog objective to remove it")
	register(KeyDesktopPluginrpcObjectiveExceeds2000Utf16Units, "目标最多 2,000 UTF-16 单元", "Objective exceeds 2,000 UTF-16 units")
	register(KeyDesktopPluginrpcRoundCapMustBeAnIntegerFrom1To2, "轮数上限须为 1–200 的整数", "Round cap must be an integer from 1 to 200")
	register(KeyDesktopPluginrpcMemberOwnershipUnavailable, "成员归属未核实", "Member ownership unavailable")
	register(KeyDesktopPluginrpcHistoricalMemberNotInCurrentRoom, "已不在当前群聊成员中", "Historical member; not in current room")
	register(KeyDesktopPluginrpcCurrentBotIdentityUnavailable, "当前 Bot 身份不可用", "Current Bot identity unavailable")
	register(KeyDesktopPluginrpcMemberSessionUnavailable, "成员会话不可用", "Member session unavailable")
	register(KeyDesktopPluginrpcSessionDoesNotIdentifyThisRoomMemb, "会话身份无法对应此群聊成员", "Session does not identify this room member")
	register(KeyDesktopPluginrpcAmbiguousRuntimeIdentity, "运行身份重复，禁止打开", "Ambiguous runtime identity")
	register(KeyDesktopPluginrpcInvalidRoomTextUnicode, "群文本含无效 Unicode", "Invalid room text Unicode")
	register(KeyDesktopPluginrpcRoomNameRequiredMaximum120Utf16Un, "群名必填且最多 120 UTF-16 单元", "Room name required, maximum 120 UTF-16 units")
	register(KeyDesktopPluginrpcRoomDescriptionMaximum12000Utf16Un, "群说明最多 12000 UTF-16 单元", "Room description maximum 12000 UTF-16 units")
	register(KeyDesktopPluginrpcInvalidUnicode, "文本含无效 Unicode", "Invalid Unicode")
	register(KeyDesktopPluginrpcNameIsRequiredMaximum120Utf16Unit, "寻址名称必填且最多120个UTF-16单元", "Name is required, maximum 120 UTF-16 units")
	register(KeyDesktopPluginrpcTitle120Description12000Utf16Units, "显示标题最多120，人设描述最多12000个UTF-16单元", "Title ≤120; description ≤12000 UTF-16 units")

	// desktop/previewprocess
	register(KeyDesktopPreviewprocessIsolatedInteractivePreview, "隔离交互预览", "Isolated interactive preview")

	// desktop/screenglance
	register(KeyDesktopScreenglanceCaptureThroughTheSystemScreenshotDi, "通过系统截图对话框截取一张图片并发送给视觉模型？请在系统界面选择要分享的显示器或区域，桌面可能截取整个屏幕。", "Capture through the system screenshot dialog and send one image to the vision model? Select the display or area to share; the desktop may capture the entire screen.")
	register(KeyDesktopScreenglanceCaptureAndSendOneScreenshotOfTheP, "截取桌宠所在显示器的一张截图并发送给视觉模型？", "Capture and send one screenshot of the pet's display?")
	register(KeyDesktopScreenglanceTheImageMayContainPrivateInformatio, "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。系统会创建临时截图；读取后尝试删除，但只读 portal 文件可能由系统保留。", "The image may contain private information and be retained in Whale chat. The system creates a temporary screenshot; removal is attempted after reading, but a read-only portal image may remain under desktop management.")
	register(KeyDesktopScreenglanceTheImageMayContainPrivateInformatio2, "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。macOS 截图会暂存于本机私有临时目录，读取后删除。需要你在系统设置中授予屏幕录制权限。", "The image may contain private information and be retained in Whale chat. macOS capture uses a private temporary directory, removed after reading. Screen Recording permission must be granted by you in System Settings.")
	register(KeyDesktopScreenglanceTheImageMayContainPrivateInformatio3, "截图可能包含私人信息，并可能保存在鲸鱼娘会话中。本机不另存截图文件。", "The image may contain private information and may be retained in Whale chat. No separate local screenshot file is saved.")
	register(KeyDesktopScreenglanceScreenshotUnavailableCheckThisAppS, "截图不可用：请检查 macOS 系统设置 → 隐私与安全性 → 屏幕录制中的应用权限后重试", "Screenshot unavailable; check this app's Screen Recording permission in macOS System Settings and retry")
	register(KeyDesktopScreenglanceScreenshotCancelledDeniedOrUnavailab, "截图已取消、被拒绝或系统截图服务不可用；需要支持交互截图的 XDG Desktop Portal（v2+）", "Screenshot cancelled, denied or unavailable; an interactive XDG Screenshot portal (v2+) is required")
	register(KeyDesktopScreenglanceScreenshotUnavailable, "本次截图不可用", "Screenshot unavailable")

	// desktop/voiceinput
	register(KeyDesktopVoiceinputOpeningMicrophone, "正在检查语音服务与麦克风权限", "Opening microphone…")
	register(KeyDesktopVoiceinputMicrophoneUnavailable, "无法打开麦克风，请检查系统权限和设备", "Microphone unavailable")
	register(KeyDesktopVoiceinputRecording, "录音中：停止并转写，或取消；到时自动转写", "Recording")
	register(KeyDesktopVoiceinputTranscribing, "正在转写，结果只填入草稿", "Transcribing…")
	register(KeyDesktopVoiceinputRecordingFailed, "录音失败，原草稿未改动", "Recording failed")
	register(KeyDesktopVoiceinputDraftChangedInsertOrDiscard, "草稿已修改；点击插入或取消", "Draft changed; insert or discard")
}
