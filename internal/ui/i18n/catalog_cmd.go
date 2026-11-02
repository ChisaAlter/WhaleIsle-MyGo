package i18n

// Keys for copy that originates below the UI layer: store error and status
// text plus desktop runtime messages. Registered from init so the shared
// catalog stays the single source of truth for the active locale.
const (

	// cmd
	KeyCmdChooseLocalAvatarImage                 = "cmd.choose_local_avatar_image"
	KeyCmdBotCreatedSessionConnectionUnavailab   = "cmd.bot_created_session_connection_unavailab"
	KeyCmdCreatedBotOpenCanonicalSession         = "cmd.created_bot_open_canonical_session"
	KeyCmdCatalogRecordRetainedSessionConnecti   = "cmd.catalog_record_retained_session_connecti"
	KeyCmdOpenInspectedCatalogBot                = "cmd.open_inspected_catalog_bot"
	KeyCmdWatchingThisBotPageSCompleteCatalo     = "cmd.watching_this_bot_page_s_complete_catalo"
	KeyCmdSystemNotificationsUnavailableOrNot    = "cmd.system_notifications_unavailable_or_not"
	KeyCmdCouldNotRestoreTheNotificationTarge    = "cmd.could_not_restore_the_notification_targe"
	KeyCmdNotificationTargetVerifyInVisibility   = "cmd.notification_target_verify_in_visibility"
	KeyCmdSystemNotificationDeliveryFailedChec   = "cmd.system_notification_delivery_failed_chec"
	KeyCmdConversationOpenedReadMarkerFailed     = "cmd.conversation_opened_read_marker_failed"
	KeyCmdOpeningCanonicalConversation           = "cmd.opening_canonical_conversation"
	KeyCmdResolveTheExistingChatDraftBeforeS     = "cmd.resolve_the_existing_chat_draft_before_s"
	KeyCmdCanonicalConversationOpened            = "cmd.canonical_conversation_opened"
	KeyCmdNeedsAttention                         = "cmd.needs_attention"
	KeyCmdCreatedRoomOpenCanonicalSession        = "cmd.created_room_open_canonical_session"
	KeyCmdUnsupportedTriggerAddressNoCredentia   = "cmd.unsupported_trigger_address_no_credentia"
	KeyCmdNoImageOnTheClipboard                  = "cmd.no_image_on_the_clipboard"
	KeyCmdEnterAReplyOrAddAnAttachment           = "cmd.enter_a_reply_or_add_an_attachment"
	KeyCmdHostOffline                            = "cmd.host_offline"
	KeyCmdNoCompletedBotJsonReply                = "cmd.no_completed_bot_json_reply"
	KeyCmdResolveTheExistingChatDraftFirst       = "cmd.resolve_the_existing_chat_draft_first"
	KeyCmdDraftPreparedReviewAndSend             = "cmd.draft_prepared_review_and_send"
	KeyCmdImportedForPreviewSaveToApply          = "cmd.imported_for_preview_save_to_apply"
	KeyCmdSavedOtherBotCopiesRemainIndependen    = "cmd.saved_other_bot_copies_remain_independen"
	KeyCmdIndependentCopyAdded                   = "cmd.independent_copy_added"
	KeyCmdTemplateUpdatedForFutureCopies         = "cmd.template_updated_for_future_copies"
	KeyCmdRemovedFromThisBotTemplateRetained     = "cmd.removed_from_this_bot_template_retained"
	KeyCmdExportInstallationHistory              = "cmd.export_installation_history"
	KeyCmdPhoneRemoteSettingWasNotSavedRetry     = "cmd.phone_remote_setting_was_not_saved_retry"
	KeyCmdWhaleAssistantIsDisabled               = "cmd.whale_assistant_is_disabled"
	KeyCmdHostOffline2                           = "cmd.host_offline_2"
	KeyCmdConnectionChanged                      = "cmd.connection_changed"
	KeyCmdConversationChanged                    = "cmd.conversation_changed"
	KeyCmdPetChatUnavailable                     = "cmd.pet_chat_unavailable"
	KeyCmdPetChatOpened                          = "cmd.pet_chat_opened"
	KeyCmdRuntimeChanged                         = "cmd.runtime_changed"
	KeyCmdCancelled                              = "cmd.cancelled"
	KeyCmdScanUnavailableSavedFoodRetained       = "cmd.scan_unavailable_saved_food_retained"
	KeyCmdLocalFoodRefreshed                     = "cmd.local_food_refreshed"
	KeyCmdConnectHostAndEnableWhaleFirst         = "cmd.connect_host_and_enable_whale_first"
	KeyCmdSelectThePetLookModelAndRefreshIt      = "cmd.select_the_pet_look_model_and_refresh_it"
	KeyCmdLookAtScreen                           = "cmd.look_at_screen"
	KeyCmdCancel                                 = "cmd.cancel"
	KeyCmdCaptureAndSend                         = "cmd.capture_and_send"
	KeyCmdCancelledNoScreenshotTaken             = "cmd.cancelled_no_screenshot_taken"
	KeyCmdWindowConversationOrVisionRouteChan    = "cmd.window_conversation_or_vision_route_chan"
	KeyCmdContextChangedScreenshotWasNotSent     = "cmd.context_changed_screenshot_was_not_sent"
	KeyCmdContextChanged                         = "cmd.context_changed"
	KeyCmdLookFailedCheckTheVisionModel          = "cmd.look_failed_check_the_vision_model"
	KeyCmdHostOfflineReopenAfterConnecting       = "cmd.host_offline_reopen_after_connecting"
	KeyCmdGatewaySettingsSavedApplying           = "cmd.gateway_settings_saved_applying"
	KeyCmdPhonePrReview                          = "cmd.phone_pr_review"
	KeyCmdWorking                                = "cmd.working"
	KeyCmdPhoneRequestedAPrReviewAndConfirm      = "cmd.phone_requested_a_pr_review_and_confirm"
	KeyCmdCommitPushMayAlreadyBeCompleteCanc     = "cmd.commit_push_may_already_be_complete_canc"
	KeyCmdPrTitle                                = "cmd.pr_title"
	KeyCmdPrBody                                 = "cmd.pr_body"
	KeyCmdCreateDraftPr                          = "cmd.create_draft_pr"
	KeyCmdGenerateCandidate                      = "cmd.generate_candidate"
	KeyCmdCandidateNotYetApplied                 = "cmd.candidate_not_yet_applied"
	KeyCmdApplyCandidate                         = "cmd.apply_candidate"
	KeyCmdPrepareTargetReview                    = "cmd.prepare_target_review"
	KeyCmdReviewTheExactContentBelow             = "cmd.review_the_exact_content_below"
	KeyCmdConfirmThisPr                          = "cmd.confirm_this_pr"
	KeyCmdOutcomeUncertainCheckExistingPrBefo    = "cmd.outcome_uncertain_check_existing_pr_befo"
	KeyCmdPublishedHtmlUnavailable               = "cmd.published_html_unavailable"
	KeyCmdSavePublishedHtml                      = "cmd.save_published_html"
	KeyCmdIsolatedPreviewHelperIsNotInstalled    = "cmd.isolated_preview_helper_is_not_installed"
	KeyCmdTrayUnavailable                        = "cmd.tray_unavailable"
	KeyCmdTheSystemTrayCouldNotBeCreatedTheWin   = "cmd.the_system_tray_could_not_be_created_the_win"
	KeyCmdKeepWindow                             = "cmd.keep_window"
	KeyCmdQuitApplication                        = "cmd.quit_application"
	KeyCmdMinimizedToTray                        = "cmd.minimized_to_tray"
	KeyCmdClickTheTrayIconToReopenTheWindowOr    = "cmd.click_the_tray_icon_to_reopen_the_window_or"
	KeyCmdDesktopPet                             = "cmd.desktop_pet"
	KeyCmdDeveloperTools                         = "cmd.developer_tools"
	KeyCmdCheckingForUpdates                     = "cmd.checking_for_updates"
	KeyCmdCouldNotCheckForUpdates                = "cmd.could_not_check_for_updates"
	KeyCmdThisBuildHasNoUpdateFeedConfiguredDev  = "cmd.this_build_has_no_update_feed_configured_dev"
	KeyCmdYouAreOnTheLatestVersion               = "cmd.you_are_on_the_latest_version"
	KeyCmdUpdateAvailable                        = "cmd.update_available"
	KeyCmdCheckComplete                          = "cmd.check_complete"
	KeyCmdOk                                     = "cmd.ok"
	KeyCmdClose                                  = "cmd.close"
	KeyCmdPreparingTheUpdateDownload             = "cmd.preparing_the_update_download"
	KeyCmdDownloadingTheUpdate                   = "cmd.downloading_the_update"
	KeyCmdTheUpdateIsDownloadedAndVerifiedInstal = "cmd.the_update_is_downloaded_and_verified_instal"
	KeyCmdTechnicalDetails                       = "cmd.technical_details"
	KeyCmdDismiss                                = "cmd.dismiss"
	KeyCmdNewVersion                             = "cmd.new_version"
	KeyCmdDownloadRoute                          = "cmd.download_route"
	KeyCmdInstallerSize                          = "cmd.installer_size"
	KeyCmdAnIncrementalPatchPathIsAvailable      = "cmd.an_incremental_patch_path_is_available"
	KeyCmdVersion                                = "cmd.version"
	KeyCmdPlatform                               = "cmd.platform"
	KeyCmdIncludesTheRuntime                     = "cmd.includes_the_runtime"
	KeyCmdMygoUpdateAwaitingConfirmation         = "cmd.mygo_update_awaiting_confirmation"
	KeyCmdIsAvailableReturnToConfirmationDownloa = "cmd.is_available_return_to_confirmation_download"
	KeyCmdMygoUpdateAvailable                    = "cmd.mygo_update_available"
	KeyCmdReviewVersionsAndDownloads             = "cmd.review_versions_and_downloads"
	KeyCmdLater                                  = "cmd.later"
	KeyCmdConfirmingNotice                       = "cmd.confirming_notice"
	KeyCmdCouldNotAcknowledgeRetry               = "cmd.could_not_acknowledge_retry"
	KeyCmdThisAccountCannotAcknowledgeTheNoticeR = "cmd.this_account_cannot_acknowledge_the_notice_r"
	KeyCmdGotIt                                  = "cmd.got_it"
	KeyCmdRetry                                  = "cmd.retry"
	KeyCmdRewardNotice                           = "cmd.reward_notice"
	KeyCmdRequestQuotaExhausted                  = "cmd.request_quota_exhausted"
	KeyCmdNoBalanceAvailable                     = "cmd.no_balance_available"
	KeyCmdMygoCannotStartANewTaskWithThisAccou   = "cmd.mygo_cannot_start_a_new_task_with_this_accou"
	KeyCmdTopUp                                  = "cmd.top_up"
	KeyCmdYouHaveSignedOutOfYourAccountPleaseL   = "cmd.you_have_signed_out_of_your_account_please_l"
	KeyCmdModelUnavailablePleaseSignInAndTryAga  = "cmd.model_unavailable_please_sign_in_and_try_aga"
	KeyCmdSignIn                                 = "cmd.sign_in"
	KeyCmdCouldNotOpenAuthorizationPage          = "cmd.could_not_open_authorization_page"
	KeyCmdCouldNotOpenTopUpPage                  = "cmd.could_not_open_top_up_page"
	KeyCmdCouldNotCancelSignIn                   = "cmd.could_not_cancel_sign_in"
	KeyCmdRecoveredDraftsWereKeptTheWorkspaceCha = "cmd.recovered_drafts_were_kept_the_workspace_cha"
	KeyCmdThePreviousSessionIsNotInTheCurrentL   = "cmd.the_previous_session_is_not_in_the_current_l"
	KeyCmdCouldNotSaveResumeState                = "cmd.could_not_save_resume_state"
	KeyCmdTheOriginalFileWasNotOverwrittenKeepT  = "cmd.the_original_file_was_not_overwritten_keep_t"
	KeyCmdKeepWindowOpen                         = "cmd.keep_window_open"
	KeyCmdQuitWithoutSaving                      = "cmd.quit_without_saving"
	KeyCmdActionDidNotCompleteTheWindowWasKept   = "cmd.action_did_not_complete_the_window_was_kept"
	KeyCmdQuit                                   = "cmd.quit"
	KeyCmdRestart                                = "cmd.restart"
	KeyCmdInstallUpdateAndQuit                   = "cmd.install_update_and_quit"
	KeyCmdChangeWhalebridgeComponent             = "cmd.change_whalebridge_component"
	KeyCmdStopRuntime                            = "cmd.stop_runtime"
	KeyCmdThisStopsTheHostRuntimeTheDesktopAppl  = "cmd.this_stops_the_host_runtime_the_desktop_appl"
	KeyCmdHostTaskProtectionIsUnavailableBackgro = "cmd.host_task_protection_is_unavailable_backgrou"
	KeyCmdKnownLocalRunningWorkSessionsAndTermin = "cmd.known_local_running_work_sessions_and_termin"
	KeyCmdTheseChecksAreUnknown                  = "cmd.these_checks_are_unknown"
	KeyCmdContinuingMayInterruptWorkThisDoesNot  = "cmd.continuing_may_interrupt_work_this_does_not"
	KeyCmdContinue                               = "cmd.continue"
	KeyCmdImportIsStillRunningWaitForCompletion  = "cmd.import_is_still_running_wait_for_completion"
	KeyCmdLocalWorkItemsSessionsAndTerminalsAre  = "cmd.local_work_items_sessions_and_terminals_are"
	KeyCmdInstallUpdate                          = "cmd.install_update"
	KeyCmdTheUpdateHasBeenDownloadedAndVerified  = "cmd.the_update_has_been_downloaded_and_verified"
	KeyCmdInstallAndQuit                         = "cmd.install_and_quit"
	KeyCmdMygoIsAvailable                        = "cmd.mygo_is_available"
	KeyCmdAwaitingConfirmationChooseLaterOrDownl = "cmd.awaiting_confirmation_choose_later_or_downlo"
	KeyCmdLoadingMygoReleaseCatalog              = "cmd.loading_mygo_release_catalog"
	KeyCmdCatalogLookupFailedThePreviousListAnd  = "cmd.catalog_lookup_failed_the_previous_list_and"
	KeyCmdReleaseLookupCancelled                 = "cmd.release_lookup_cancelled"
	KeyCmdLoadedMygoReleases                     = "cmd.loaded_mygo_releases"
	KeyCmdNoCompatibleMygoInstallerHasBeenPublis = "cmd.no_compatible_mygo_installer_has_been_publis"
	KeyCmdWhaleisleVendorOverridesTheRuntimePath = "cmd.whaleisle_vendor_overrides_the_runtime_path"
	KeyCmdSelectingAndVerifyingAMygoInstaller    = "cmd.selecting_and_verifying_a_mygo_installer"
	KeyCmdDownloadingAndVerifyingFullInstaller   = "cmd.downloading_and_verifying_full_installer"
	KeyCmdReusingCachedInstallerAndDownloadingCh = "cmd.reusing_cached_installer_and_downloading_cha"
	KeyCmdInstallationPreparationFailedTheCurren = "cmd.installation_preparation_failed_the_current"
	KeyCmdDownloadCancelledTheCurrentInstallatio = "cmd.download_cancelled_the_current_installation"
	KeyCmdDesktopOnly                            = "cmd.desktop_only"
	KeyCmdIncludesRuntime                        = "cmd.includes_runtime"
	KeyCmdFullPackage                            = "cmd.full_package"
	KeyCmdDeltaReconstruction                    = "cmd.delta_reconstruction"
	KeyCmdInstallerDownloadedAndVerifiedChooseIn = "cmd.installer_downloaded_and_verified_choose_ins"
	KeyCmdDiskImageDownloadedAndVerifiedChooseOp = "cmd.disk_image_downloaded_and_verified_choose_op"
	KeyCmdTheSourceDidNotSupportRangesTheComple  = "cmd.the_source_did_not_support_ranges_the_comple"
	KeyCmdTheInstallerIsRetainedButCachingItFor  = "cmd.the_installer_is_retained_but_caching_it_for"
	KeyCmdThisPlatformCannotStartTheInstaller    = "cmd.this_platform_cannot_start_the_installer"
	KeyCmdVerifiedInstallerRetainedForRetryTheCu = "cmd.verified_installer_retained_for_retry_the_cu"
	KeyCmdVerifyingWhalebridgeVersionAndImpactTh = "cmd.verifying_whalebridge_version_and_impact_the"
	KeyCmdCancelledTheInstalledVersionConnection = "cmd.cancelled_the_installed_version_connections"
	KeyCmdWhalebridgeV                           = "cmd.whalebridge_v"
	KeyCmdWasUninstalledAndDisconnectedChatsAndO = "cmd.was_uninstalled_and_disconnected_chats_and_o"
	KeyCmdProviderSettingsKeysAccountsAndUsageWe = "cmd.provider_settings_keys_accounts_and_usage_we"
	KeyCmdWhalebridgeProviderSettingsKeysAccount = "cmd.whalebridge_provider_settings_keys_accounts"
	KeyCmdSelectANewDefaultModel                 = "cmd.select_a_new_default_model"
	KeyCmdWhalebridgeSwitchedToV                 = "cmd.whalebridge_switched_to_v"
	KeyCmdUserDataAndTheOriginalRunningStateWer  = "cmd.user_data_and_the_original_running_state_wer"
	KeyCmdConfirmWhalebridgeUninstall            = "cmd.confirm_whalebridge_uninstall"
	KeyCmdContinueUninstall                      = "cmd.continue_uninstall"
	KeyCmdTargetWhalebridgeVThisStopsTheService  = "cmd.target_whalebridge_v_this_stops_the_service"
	KeyCmdYourCurrentDefaultModelUsesWhalebridge = "cmd.your_current_default_model_uses_whalebridge"
	KeyCmdConfirmWhalebridgeVersionChange        = "cmd.confirm_whalebridge_version_change"
	KeyCmdSwitchVersion                          = "cmd.switch_version"
	KeyCmdConfirmWhalebridgeRollback             = "cmd.confirm_whalebridge_rollback"
	KeyCmdRollBack                               = "cmd.roll_back"
	KeyCmdTargetWhalebridgeVARunningServiceWill  = "cmd.target_whalebridge_v_a_running_service_will"
	KeyCmdAlsoDeleteWhalebridgeConfiguration     = "cmd.also_delete_whalebridge_configuration"
	KeyCmdTargetNNdeletionPermanentlyRemovesWhal = "cmd.target_n_ndeletion_permanently_removes_whale"
	KeyCmdDeleteConfiguration                    = "cmd.delete_configuration"
	KeyCmdKeepConfiguration                      = "cmd.keep_configuration"
	KeyCmdChooseAMacosHostOfflineBundle          = "cmd.choose_a_macos_host_offline_bundle"
	KeyCmdInstallHostOfflineBundle               = "cmd.install_host_offline_bundle"
	KeyCmdSourceNdestinationNNextractAndInstallT = "cmd.source_ndestination_n_nextract_and_install_t"
	KeyCmdInstall                                = "cmd.install"
	KeyCmdExtractingAndCheckingHostAndItsMatchin = "cmd.extracting_and_checking_host_and_its_matchin"
	KeyCmdHostVWasInstalledInChooseSelectInstal  = "cmd.host_v_was_installed_in_choose_select_instal"
	KeyCmdSelectAnInstalledHostVersionDirectory  = "cmd.select_an_installed_host_version_directory"
	KeyCmdReadingTheSelectedHostVersion          = "cmd.reading_the_selected_host_version"
	KeyCmdHostSelectionWasNotChangedIfTaskProte  = "cmd.host_selection_was_not_changed_if_task_prote"
	KeyCmdHostVIsSelectedTheRuntimeRemainsStopp  = "cmd.host_v_is_selected_the_runtime_remains_stopp"
	KeyCmdHostActionDidNotComplete               = "cmd.host_action_did_not_complete"
	KeyCmdSelectHostVersion                      = "cmd.select_host_version"
	KeyCmdTargetHostVNNtheExistingTaskProtectio  = "cmd.target_host_v_n_nthe_existing_task_protectio"
	KeyCmdStopAndSelect                          = "cmd.stop_and_select"
	KeyCmdSettingsSavedButStartupRegistrationFai = "cmd.settings_saved_but_startup_registration_fail"
	KeyCmdResumeStateNeedsAttention              = "cmd.resume_state_needs_attention"
	KeyCmdTheOriginalFileIsPreservedBackItUpBe   = "cmd.the_original_file_is_preserved_back_it_up_be"
	KeyCmdGenerate                               = "cmd.generate"
	KeyCmdReadTaskHistory                        = "cmd.read_task_history"
	KeyCmdSaveTitle                              = "cmd.save_title"
	KeyCmdSavePrompt                             = "cmd.save_prompt"
	KeyCmdDescribeTheTaskInTheConversationTheAg  = "cmd.describe_the_task_in_the_conversation_the_ag"
	KeyCmdEnableDshIm                            = "cmd.enable_dsh_im"
	KeyCmdDshImEnabledRestartTheAppIfThePageS    = "cmd.dsh_im_enabled_restart_the_app_if_the_page_s"
	KeyCmdRemoteWorkspace                        = "cmd.remote_workspace"
	KeyCmdOpenTheDownloadPage                    = "cmd.open_the_download_page"
	KeyCmdAutoAllowTools                         = "cmd.auto_allow_tools"
	KeyCmdCouldNotOpenInstalledApps              = "cmd.could_not_open_installed_apps"
	KeyCmdOpenWindowsSettingsAppsInstalledAppsMa = "cmd.open_windows_settings_apps_installed_apps_ma"
	KeyCmdNoVerifiedUninstallerFound             = "cmd.no_verified_uninstaller_found"
	KeyCmdThisMayBeAPortableSourceBuildOrAnIn    = "cmd.this_may_be_a_portable_source_build_or_an_in"
	KeyCmdUninstallCurrentNativeApp              = "cmd.uninstall_current_native_app"
	KeyCmdContinuingChecksAndStopsTasksOpensThis = "cmd.continuing_checks_and_stops_tasks_opens_this"
	KeyCmdStartUninstallerAndQuit                = "cmd.start_uninstaller_and_quit"
	KeyCmdTheWhalebridgeInstanceChangedOrIsUnava = "cmd.the_whalebridge_instance_changed_or_is_unava"
	KeyCmdTheWhalebridgeInstanceChangedReviewIts = "cmd.the_whalebridge_instance_changed_review_its"
	KeyCmdUninstallComponent                     = "cmd.uninstall_component"
	KeyCmdStopAndRemoveAndItsProgramFilesConfig  = "cmd.stop_and_remove_and_its_program_files_config"
	KeyCmdUninstall                              = "cmd.uninstall"
	KeyCmdRollBackComponent                      = "cmd.roll_back_component"
	KeyCmdRollBackToVItsRunningProcessWillBeS    = "cmd.roll_back_to_v_its_running_process_will_be_s"
	KeyCmdTheRuntimeIsStartingRetryImportAfterS  = "cmd.the_runtime_is_starting_retry_import_after_s"
	KeyCmdConfirmDataImport                      = "cmd.confirm_data_import"
	KeyCmdBeforeImportingCurrentTasksAreCheckedA = "cmd.before_importing_current_tasks_are_checked_a"
	KeyCmdConfirmAndContinue                     = "cmd.confirm_and_continue"
	KeyCmdCheckingCurrentTasksBeforeStoppingTheR = "cmd.checking_current_tasks_before_stopping_the_r"
	KeyCmdTheRuntimeChangedConfirmTheImportAgain = "cmd.the_runtime_changed_confirm_the_import_again"
	KeyCmdTheRuntimeIsStoppedImportingCompletedI = "cmd.the_runtime_is_stopped_importing_completed_i"
	KeyCmdImportCancelledNoImportDataWasWritten  = "cmd.import_cancelled_no_import_data_was_written"
	KeyCmdImportDidNotStart                      = "cmd.import_did_not_start"
	KeyCmdTheRuntimeRemainsStoppedStartItManuall = "cmd.the_runtime_remains_stopped_start_it_manuall"
	KeyCmdImportRecoveryIsIncompleteAllCopiesWer = "cmd.import_recovery_is_incomplete_all_copies_wer"
	KeyCmdAnInterruptedImportWasRecoveredComplet = "cmd.an_interrupted_import_was_recovered_complete"
	KeyCmdOpeningWhalebridgeSettings             = "cmd.opening_whalebridge_settings"
	KeyCmdOfflineWaitingForNetworkAvailability   = "cmd.offline_waiting_for_network_availability"
	KeyCmdNotConnectedToTheHostYetTryAgainShor   = "cmd.not_connected_to_the_host_yet_try_again_shor"
	KeyCmdFailed                                 = "cmd.failed"
	KeyCmdRefreshTheDeliveryDetailsFirst         = "cmd.refresh_the_delivery_details_first"
	KeyCmdRequestedOpeningTheProjectDirectory    = "cmd.requested_opening_the_project_directory"
	KeyCmdNavigationChangedTheFileWasNotOpened   = "cmd.navigation_changed_the_file_was_not_opened"
	KeyCmdThisFileChangedAfterTheReportOpeningI  = "cmd.this_file_changed_after_the_report_opening_i"
	KeyCmdRefreshTheProjectTeamFirst             = "cmd.refresh_the_project_team_first"
	KeyCmdNavigationChangedTheRetryDraftWasNotI  = "cmd.navigation_changed_the_retry_draft_was_not_i"
	KeyCmdTheLeadHasADraftOrPendingSendItWas     = "cmd.the_lead_has_a_draft_or_pending_send_it_was"
	KeyCmdRetryRequestPlacedInTheOriginalLeadDr  = "cmd.retry_request_placed_in_the_original_lead_dr"
	KeyCmdHostIsNotConnectedRefreshWhenReady     = "cmd.host_is_not_connected_refresh_when_ready"
	KeyCmdProjectHostIsUnavailable               = "cmd.project_host_is_unavailable"
	KeyCmdProjectHostUnavailableRefreshToRetry   = "cmd.project_host_unavailable_refresh_to_retry"
	KeyCmdProjectUnavailableOrArchivingRefreshTo = "cmd.project_unavailable_or_archiving_refresh_to"
	KeyCmdProjectStateUpdatedButListRefreshFaile = "cmd.project_state_updated_but_list_refresh_faile"
	KeyCmdProjectArchivedWorkHasStoppedConversat = "cmd.project_archived_work_has_stopped_conversati"
	KeyCmdProjectRestoredSendANewInstructionInT  = "cmd.project_restored_send_a_new_instruction_in_t"
	KeyCmdAutomaticRefreshFailedShowingPreviousS = "cmd.automatic_refresh_failed_showing_previous_st"
	KeyCmdAutoRefreshEvery5s                     = "cmd.auto_refresh_every_5s"
	KeyCmdThisProjectIsNoLongerInTheHostList     = "cmd.this_project_is_no_longer_in_the_host_list"
	KeyCmdTeamRefreshFailedShowingPreviousState  = "cmd.team_refresh_failed_showing_previous_state"
	KeyCmdMemberDetailRefreshFailed              = "cmd.member_detail_refresh_failed"
	KeyCmdTheMemberRunChangedPreviousRecordsAre  = "cmd.the_member_run_changed_previous_records_are"
	KeyCmdProjectAndTeamStateRefreshAutomaticall = "cmd.project_and_team_state_refresh_automatically"
	KeyCmdRefreshOrRestoreTheOriginalProjectFirs = "cmd.refresh_or_restore_the_original_project_firs"
	KeyCmdNavigationChangedTheRecoveryDraftWasNo = "cmd.navigation_changed_the_recovery_draft_was_no"
	KeyCmdSendingTheRecoveryRequestToTheOriginal = "cmd.sending_the_recovery_request_to_the_original"
	KeyCmdChangesWereRevertedButRefreshingFailed = "cmd.changes_were_reverted_but_refreshing_failed"
	KeyCmdThisRequirementSChangesWereReverted    = "cmd.this_requirement_s_changes_were_reverted"
	KeyCmdProjectHostIsUnavailableRefreshToRetry = "cmd.project_host_is_unavailable_refresh_to_retry"
	KeyCmdProjectIsNoLongerInTheCurrentListRef   = "cmd.project_is_no_longer_in_the_current_list_ref"
	KeyCmdRestoreTheProjectBeforeChangingRequest = "cmd.restore_the_project_before_changing_request"
	KeyCmdCardUpdatedButListRefreshFailedRefresh = "cmd.card_updated_but_list_refresh_failed_refresh"
	KeyCmdThisProjectIsNotInTheCurrentListRefr   = "cmd.this_project_is_not_in_the_current_list_refr"
	KeyCmdResumeStateWasNotSaved                 = "cmd.resume_state_was_not_saved"
	KeyCmdTaskDeleted                            = "cmd.task_deleted"
	KeyCmdCouldNotDeleteTheTask                  = "cmd.could_not_delete_the_task"
	KeyCmdDiagnosticsOrPluginInventoryChangedRef = "cmd.diagnostics_or_plugin_inventory_changed_refr"
	KeyCmdPluginOperationDidNotComplete          = "cmd.plugin_operation_did_not_complete"
	KeyCmdApplied                                = "cmd.applied"
	KeyCmdUpdatedPlugins                         = "cmd.updated_plugins"
	KeyCmdRetryingFullStartupRecoveryCompletesOn = "cmd.retrying_full_startup_recovery_completes_onl"
	KeyCmdThePreviousStartDidNotFullyRecoverAut  = "cmd.the_previous_start_did_not_fully_recover_aut"
	KeyCmdTheRecoveryFileWasPreservedRepairItAn  = "cmd.the_recovery_file_was_preserved_repair_it_an"
	KeyCmdSkipUserPluginsRecoveryIsSavedStartIn  = "cmd.skip_user_plugins_recovery_is_saved_start_in"
	KeyCmdRecoveryModeIsReadyUserPluginsRemainS  = "cmd.recovery_mode_is_ready_user_plugins_remain_s"
	KeyCmdCouldNotSaveStartupOutcome             = "cmd.could_not_save_startup_outcome"
	KeyCmdRecoveryModeIsReadyUserPluginsAreStil  = "cmd.recovery_mode_is_ready_user_plugins_are_stil"
	KeyCmdCouldNotReadTheExistingCredentials     = "cmd.could_not_read_the_existing_credentials"
	KeyCmdWhalebridge                            = "cmd.whalebridge"
	KeyCmdCouldNotOpenTheWhalebridgeWindow       = "cmd.could_not_open_the_whalebridge_window"
	KeyCmdCouldNotCreateTheWhalebridgePage       = "cmd.could_not_create_the_whalebridge_page"
	KeyCmdAccounts                               = "cmd.accounts"
	KeyCmdReload                                 = "cmd.reload"
	KeyCmdOpeningTheWhalebridgeManagementPage    = "cmd.opening_the_whalebridge_management_page"
	KeyCmdRestartHarness                         = "cmd.restart_harness"
	KeyCmdShowWindow                             = "cmd.show_window"
	KeyCmdOpenLauncher                           = "cmd.open_launcher"
	KeyCmdSettings                               = "cmd.settings"
	KeyCmdPluginMarketplace                      = "cmd.plugin_marketplace"
	KeyCmdCloseWindow                            = "cmd.close_window"
	KeyCmdAbout                                  = "cmd.about"
	KeyCmdFile                                   = "cmd.file"
	KeyCmdOpenWorkspace                          = "cmd.open_workspace"
	KeyCmdOpenWorkspaceInExplorer                = "cmd.open_workspace_in_explorer"
	KeyCmdMarketplace                            = "cmd.marketplace"
	KeyCmdSkills                                 = "cmd.skills"
	KeyCmdRun                                    = "cmd.run"
	KeyCmdReloadInterface                        = "cmd.reload_interface"
	KeyCmdView                                   = "cmd.view"
	KeyCmdActualSize                             = "cmd.actual_size"
	KeyCmdZoomIn                                 = "cmd.zoom_in"
	KeyCmdZoomOut                                = "cmd.zoom_out"
	KeyCmdFullscreen                             = "cmd.fullscreen"
	KeyCmdHelp                                   = "cmd.help"
	KeyCmdWebUiGuide                             = "cmd.web_ui_guide"
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
	Register(KeyCmdTrayUnavailable, "托盘不可用", "Tray unavailable")
	Register(KeyCmdTheSystemTrayCouldNotBeCreatedTheWin, "系统托盘未能创建，已保留窗口和后台任务。可以保留窗口，或明确退出应用。", "The system tray could not be created. The window and background work were kept open. Keep the window, or explicitly quit the application.")
	Register(KeyCmdKeepWindow, "保留窗口", "Keep window")
	Register(KeyCmdQuitApplication, "退出应用", "Quit application")
	Register(KeyCmdMinimizedToTray, "已最小化到托盘", "Minimized to tray")
	Register(KeyCmdClickTheTrayIconToReopenTheWindowOr, "点击托盘图标可重新打开窗口，托盘菜单可完全退出。可在设置中更改关闭到托盘。", "Click the tray icon to reopen the window, or choose Quit from its menu. Change close-to-tray in Settings.")
	Register(KeyCmdDesktopPet, "桌面宠物", "Desktop pet")
	Register(KeyCmdDeveloperTools, "开发者工具", "Developer tools")
	Register(KeyCmdCheckingForUpdates, "正在检查更新…", "Checking for updates…")
	Register(KeyCmdCouldNotCheckForUpdates, "检查更新失败：", "Could not check for updates: ")
	Register(KeyCmdThisBuildHasNoUpdateFeedConfiguredDev, "当前构建没有配置更新源（开发版）。发布版会通过官方更新源检查更新。", "This build has no update feed configured (development build). Release builds check the official feed.")
	Register(KeyCmdYouAreOnTheLatestVersion, "当前版本已是最新", "You are on the latest version")
	Register(KeyCmdUpdateAvailable, "发现新版本", "Update available")
	Register(KeyCmdCheckComplete, "检查完成。", "Check complete.")
	Register(KeyCmdOk, "好", "OK")
	Register(KeyCmdClose, "关闭", "Close")
	Register(KeyCmdPreparingTheUpdateDownload, "准备下载更新…", "Preparing the update download…")
	Register(KeyCmdDownloadingTheUpdate, "正在下载更新…", "Downloading the update…")
	Register(KeyCmdTheUpdateIsDownloadedAndVerifiedInstal, "更新已下载并校验，等待确认安装", "The update is downloaded and verified; installation awaits confirmation")
	Register(KeyCmdTechnicalDetails, "技术细节", "Technical details")
	Register(KeyCmdDismiss, "关闭", "Dismiss")
	Register(KeyCmdNewVersion, "新版本：", "New version: ")
	Register(KeyCmdDownloadRoute, "下载线路：", "Download route: ")
	Register(KeyCmdInstallerSize, "安装包大小：", "Installer size: ")
	Register(KeyCmdAnIncrementalPatchPathIsAvailable, "存在增量更新路径", "An incremental patch path is available")
	Register(KeyCmdVersion, "版本：", "Version: ")
	Register(KeyCmdPlatform, "平台：", "Platform: ")
	Register(KeyCmdIncludesTheRuntime, "包含运行时", "Includes the runtime")
	Register(KeyCmdMygoUpdateAwaitingConfirmation, "MyGo 更新待确认", "MyGo update awaiting confirmation")
	Register(KeyCmdIsAvailableReturnToConfirmationDownloa, "%s 已可用。点击返回更新确认；下载和安装仍需明确确认。", "%s is available. Return to confirmation; downloading and installation still need explicit confirmation.")
	Register(KeyCmdMygoUpdateAvailable, "MyGo 更新可用", "MyGo update available")
	Register(KeyCmdReviewVersionsAndDownloads, "查看版本与下载方式", "Review versions and downloads")
	Register(KeyCmdLater, "稍后", "Later")
	Register(KeyCmdConfirmingNotice, "正在确认已读…", "Confirming notice…")
	Register(KeyCmdCouldNotAcknowledgeRetry, "确认失败，可重试：", "Could not acknowledge; retry: ")
	Register(KeyCmdThisAccountCannotAcknowledgeTheNoticeR, "当前账户无法确认这条通知，请刷新账户状态。", "This account cannot acknowledge the notice. Refresh the account state.")
	Register(KeyCmdGotIt, "知道了", "Got it")
	Register(KeyCmdRetry, "重试", "Retry")
	Register(KeyCmdRewardNotice, "奖励通知", "Reward notice")
	Register(KeyCmdRequestQuotaExhausted, "当前请求的额度已用尽", "Request quota exhausted.")
	Register(KeyCmdNoBalanceAvailable, "当前无可用余额", "No balance available")
	Register(KeyCmdMygoCannotStartANewTaskWithThisAccou, "没有可用额度时，DeepSeekHarness Go 无法开始新的任务，是否前往充值？或者可以稍后前往 设置 → 账号与余额 进行充值。", "DeepSeekHarness Go cannot start a new task with this account when no balance is available. Would you like to top up? You can also top up later in Settings → Account.")
	Register(KeyCmdTopUp, "前往充值", "Top up")
	Register(KeyCmdYouHaveSignedOutOfYourAccountPleaseL, "登录信息已失效，请重新登录", "You have signed out of your account, please log in again.")
	Register(KeyCmdModelUnavailablePleaseSignInAndTryAga, "当前模型暂不可用，请登录后再试", "Model unavailable. Please sign in and try again.")
	Register(KeyCmdSignIn, "前往登录", "Sign in")
	Register(KeyCmdCouldNotOpenAuthorizationPage, "无法打开授权页面", "Could not open authorization page")
	Register(KeyCmdCouldNotOpenTopUpPage, "无法打开充值页面", "Could not open top-up page")
	Register(KeyCmdCouldNotCancelSignIn, "取消登录失败", "Could not cancel sign-in")
	Register(KeyCmdRecoveredDraftsWereKeptTheWorkspaceCha, "恢复的草稿已保留；工作区已改变，未自动打开旧会话。", "Recovered drafts were kept; the workspace changed, so the old session was not opened.")
	Register(KeyCmdThePreviousSessionIsNotInTheCurrentL, "上次会话不在当前列表中；草稿已保留。", "The previous session is not in the current list; its draft was kept.")
	Register(KeyCmdCouldNotSaveResumeState, "未能保存恢复状态", "Could not save resume state")
	Register(KeyCmdTheOriginalFileWasNotOverwrittenKeepT, "原文件未被覆盖。请保留窗口、备份草稿并解决磁盘或多实例冲突后重试。直接退出可能丢失本次未保存内容。", "The original file was not overwritten. Keep this window open, back up drafts, and resolve disk or other-instance conflicts before retrying. Quitting now may lose unsaved changes.")
	Register(KeyCmdKeepWindowOpen, "保留窗口", "Keep window open")
	Register(KeyCmdQuitWithoutSaving, "不保存并退出", "Quit without saving")
	Register(KeyCmdActionDidNotCompleteTheWindowWasKept, "操作未完成，已保留当前窗口：", "Action did not complete; the window was kept open: ")
	Register(KeyCmdQuit, "退出", "Quit")
	Register(KeyCmdRestart, "重启", "Restart")
	Register(KeyCmdInstallUpdateAndQuit, "安装更新并退出", "Install update and quit")
	Register(KeyCmdChangeWhalebridgeComponent, "更改鲸桥组件", "Change WhaleBridge component")
	Register(KeyCmdStopRuntime, "停止内核", "Stop runtime")
	Register(KeyCmdThisStopsTheHostRuntimeTheDesktopAppl, "这会停止 Host 内核，桌面应用窗口将保持打开。", "This stops the Host runtime. The desktop application window will stay open.")
	Register(KeyCmdHostTaskProtectionIsUnavailableBackgro, "Host 任务保护不可用，无法确认后台任务、定时任务或在途请求是否已排空。继续将仅依赖本地会话信息，可能中断未显示的工作。", "Host task protection is unavailable. Background work, scheduled tasks and in-flight requests cannot be verified or drained. Continuing relies only on local session information and may interrupt unseen work.")
	Register(KeyCmdKnownLocalRunningWorkSessionsAndTermin, "本地运行工作（会话和终端）：%d；Host 活动和计划工作：未知。", "Known local running work (sessions and terminals): %d; Host active and scheduled work: unknown.")
	Register(KeyCmdTheseChecksAreUnknown, "以下检查状态未知：", "These checks are unknown: ")
	Register(KeyCmdContinuingMayInterruptWorkThisDoesNot, "继续可能中断任务；这不代表任务已经完成。是否仍要", "Continuing may interrupt work; this does not mean it has finished. Continue with ")
	Register(KeyCmdContinue, "仍要", "Continue: ")
	Register(KeyCmdImportIsStillRunningWaitForCompletion, "导入尚未结束，请等待完成或取消后重试。", "Import is still running; wait for completion or cancel it before retrying.")
	Register(KeyCmdLocalWorkItemsSessionsAndTerminalsAre, "当前有 %d 个本地工作（会话和终端）仍在运行。继续会中断任务，仍要继续吗？", "%d local work items (sessions and terminals) are still running. Continuing will interrupt work. Continue?")
	Register(KeyCmdInstallUpdate, "安装更新", "Install update")
	Register(KeyCmdTheUpdateHasBeenDownloadedAndVerified, "更新已下载并校验。继续会启动安装器，停止 Host 并退出桌面应用。", "The update has been downloaded and verified. Continuing starts the installer, stops the Host and quits the desktop application.")
	Register(KeyCmdInstallAndQuit, "安装并退出", "Install and quit")
	Register(KeyCmdMygoIsAvailable, "可安装 MyGo %s", "MyGo %s is available")
	Register(KeyCmdAwaitingConfirmationChooseLaterOrDownl, "，等待确认。可在启动器中选择稍后或下载。", "; awaiting confirmation. Choose Later or Download in the launcher.")
	Register(KeyCmdLoadingMygoReleaseCatalog, "正在读取 MyGo 发布目录…", "Loading MyGo release catalog…")
	Register(KeyCmdCatalogLookupFailedThePreviousListAnd, "目录读取失败；已保留此前列表和下载。", "Catalog lookup failed; the previous list and download were kept.")
	Register(KeyCmdReleaseLookupCancelled, "已取消读取发布目录。", "Release lookup cancelled.")
	Register(KeyCmdLoadedMygoReleases, "已读取 %d 个 MyGo 发布条目。", "Loaded %d MyGo releases.")
	Register(KeyCmdNoCompatibleMygoInstallerHasBeenPublis, "尚未发布兼容的 MyGo 安装包。", "No compatible MyGo installer has been published.")
	Register(KeyCmdWhaleisleVendorOverridesTheRuntimePath, "当前设置了 WHALEISLE_VENDOR；原生安装器不会覆盖自定义运行时目录。请修复该目录或取消此覆盖后重试。", "WHALEISLE_VENDOR overrides the runtime path. The native installer cannot replace that custom directory; repair it or remove the override before retrying.")
	Register(KeyCmdSelectingAndVerifyingAMygoInstaller, "正在选择并校验 MyGo 安装包…", "Selecting and verifying a MyGo installer…")
	Register(KeyCmdDownloadingAndVerifyingFullInstaller, "正在下载并校验完整安装包：", "Downloading and verifying full installer: ")
	Register(KeyCmdReusingCachedInstallerAndDownloadingCh, "正在复用已缓存安装包并下载变化区块：", "Reusing cached installer and downloading changed blocks: ")
	Register(KeyCmdInstallationPreparationFailedTheCurren, "安装准备失败；当前安装、选择和此前已校验下载均保留。", "Installation preparation failed; the current installation, selection and previous verified download were kept.")
	Register(KeyCmdDownloadCancelledTheCurrentInstallatio, "已取消下载，未更改当前安装。", "Download cancelled; the current installation was not changed.")
	Register(KeyCmdDesktopOnly, "仅桌面端", "Desktop only")
	Register(KeyCmdIncludesRuntime, "包含运行时", "Includes runtime")
	Register(KeyCmdFullPackage, "完整包", "Full package")
	Register(KeyCmdDeltaReconstruction, "差量重建", "Delta reconstruction")
	Register(KeyCmdInstallerDownloadedAndVerifiedChooseIn, "安装包已下载并校验。点击“安装并退出”继续；仅安装所选发布明确列出的内容。", "Installer downloaded and verified. Choose Install and quit to install the contents declared by the selected release.")
	Register(KeyCmdDiskImageDownloadedAndVerifiedChooseOp, "磁盘映像已下载并校验。点击“打开磁盘映像并退出”，再将 DeepSeekHarness Go.app 拖到 Applications 完成安装。", "Disk image downloaded and verified. Choose Open disk image and quit, then drag DeepSeekHarness Go.app to Applications to finish installation.")
	Register(KeyCmdTheSourceDidNotSupportRangesTheComple, " 下载源未提供范围请求，已回退完整下载并校验。", " The source did not support ranges; the complete installer was downloaded and verified.")
	Register(KeyCmdTheInstallerIsRetainedButCachingItFor, " 安装包已保留，但未能缓存后续差量更新基包；后续仍可使用完整包。", " The installer is retained, but caching it for future deltas failed; full packages remain available.")
	Register(KeyCmdThisPlatformCannotStartTheInstaller, "当前平台无法启动此安装器。", "This platform cannot start the installer.")
	Register(KeyCmdVerifiedInstallerRetainedForRetryTheCu, "已保留校验后的安装包，可稍后重试安装；当前窗口未关闭。", "Verified installer retained for retry; the current window remains open.")
	Register(KeyCmdVerifyingWhalebridgeVersionAndImpactTh, "正在校验鲸桥版本和操作影响；现有实例保持运行…", "Verifying WhaleBridge version and impact; the current instance stays running…")
	Register(KeyCmdCancelledTheInstalledVersionConnection, "已取消；现有版本、连接和配置保持不变。", "Cancelled; the installed version, connections and configuration are unchanged.")
	Register(KeyCmdWhalebridgeV, "鲸桥 v", "WhaleBridge v")
	Register(KeyCmdWasUninstalledAndDisconnectedChatsAndO, " 已卸载并断开。聊天记录和其他渠道已保留。", " was uninstalled and disconnected. Chats and other channels were retained.")
	Register(KeyCmdProviderSettingsKeysAccountsAndUsageWe, " 供应商配置、密钥、账户和用量数据已保留。", " Provider settings, keys, accounts and usage were retained.")
	Register(KeyCmdWhalebridgeProviderSettingsKeysAccount, " 鲸桥供应商配置、密钥、账户和用量数据已删除。", " WhaleBridge provider settings, keys, accounts and usage were deleted.")
	Register(KeyCmdSelectANewDefaultModel, " 请重新选择默认模型。", " Select a new default model.")
	Register(KeyCmdWhalebridgeSwitchedToV, "鲸桥已切换至 v", "WhaleBridge switched to v")
	Register(KeyCmdUserDataAndTheOriginalRunningStateWer, "；用户数据和原运行状态已保留", "; user data and the original running state were retained")
	Register(KeyCmdConfirmWhalebridgeUninstall, "确认卸载鲸桥", "Confirm WhaleBridge uninstall")
	Register(KeyCmdContinueUninstall, "继续卸载", "Continue uninstall")
	Register(KeyCmdTargetWhalebridgeVThisStopsTheService, "目标：鲸桥 v%s。将停止服务、断开鲸桥渠道并删除程序文件；现有连接会中断。将移除桌面端的鲸桥网关凭据引用。支持外部代理托管的版本还会撤销鲸桥代管的代理接入设置。默认保留供应商配置和密钥、账户及用量；下一步可选择是否删除鲸桥配置。聊天记录和其他渠道会保留。官方断开命令如果中途失败，程序恢复不代表全部接入设置已恢复，请检查默认模型和受管代理。", "Target: WhaleBridge v%s. This stops the service, disconnects its channel and removes program files. Existing connections will be interrupted. The desktop WhaleBridge gateway credential reference is removed. Versions with managed external agents also undo their WhaleBridge-managed connection settings. Provider settings and keys, accounts and usage are retained by default; the next step lets you choose whether to delete WhaleBridge configuration. Chats and other channels are retained. If the official disconnect command fails partway, restoring program files does not restore every connection setting. Check the default model and managed agents.")
	Register(KeyCmdYourCurrentDefaultModelUsesWhalebridge, "当前默认模型使用鲸桥。卸载后必须重新选择默认模型。", "Your current default model uses WhaleBridge. Select a new default model after uninstalling.")
	Register(KeyCmdConfirmWhalebridgeVersionChange, "确认切换鲸桥版本", "Confirm WhaleBridge version change")
	Register(KeyCmdSwitchVersion, "切换版本", "Switch version")
	Register(KeyCmdConfirmWhalebridgeRollback, "确认回滚鲸桥", "Confirm WhaleBridge rollback")
	Register(KeyCmdRollBack, "回滚", "Roll back")
	Register(KeyCmdTargetWhalebridgeVARunningServiceWill, "目标：鲸桥 v%s。正在运行的服务会先停止再重新启动，连接会暂时中断；原本停止的组件会保持停止。配置和用户数据会保留；切换失败会尝试恢复原版本。", "Target: WhaleBridge v%s. A running service will stop and restart, temporarily interrupting connections. A stopped component stays stopped. Configuration and user data are retained; a failed switch attempts to restore the original version.")
	Register(KeyCmdAlsoDeleteWhalebridgeConfiguration, "是否同时删除鲸桥配置？", "Also delete WhaleBridge configuration?")
	Register(KeyCmdTargetNNdeletionPermanentlyRemovesWhal, "目标：%s\n\n删除会永久清除该目录内鲸桥的供应商配置和密钥、账户及用量数据，无法在应用内恢复。保留后重新安装可继续使用。聊天记录和其他渠道不会删除。", "Target: %s\n\nDeletion permanently removes WhaleBridge provider settings and keys, accounts and usage in this directory. It cannot be undone in the app. Keeping it lets you reuse it after reinstalling. Chats and other channels are not deleted.")
	Register(KeyCmdDeleteConfiguration, "删除配置", "Delete configuration")
	Register(KeyCmdKeepConfiguration, "保留配置", "Keep configuration")
	Register(KeyCmdChooseAMacosHostOfflineBundle, "选择 macOS Host 离线包", "Choose a macOS Host offline bundle")
	Register(KeyCmdInstallHostOfflineBundle, "安装 Host 离线包", "Install Host offline bundle")
	Register(KeyCmdSourceNdestinationNNextractAndInstallT, "来源：%s\n目标：%s\n\n将解包并安装这个本地原生 Host 与配套 Node。请确认文件来源可信。安装不会运行包内程序，也不会停止当前 Host；安装后可选择版本并单独启动。", "Source: %s\nDestination: %s\n\nExtract and install this local native Host and its matching Node. Confirm that you trust the source. Installation does not execute the bundle or stop the current Host. Select a version and start it separately after installation.")
	Register(KeyCmdInstall, "安装", "Install")
	Register(KeyCmdExtractingAndCheckingHostAndItsMatchin, "正在解包并校验 Host 与配套 Node；当前 Host 保持运行…", "Extracting and checking Host and its matching Node; the current Host stays running…")
	Register(KeyCmdHostVWasInstalledInChooseSelectInstal, "Host v%s 已安装到 %s。点击“选择已安装版本”，确认选择后再启动。", "Host v%s was installed in %s. Choose “Select installed version”, confirm the selection, then start it.")
	Register(KeyCmdSelectAnInstalledHostVersionDirectory, "选择已安装的 Host 版本目录", "Select an installed Host version directory")
	Register(KeyCmdReadingTheSelectedHostVersion, "正在读取所选 Host 版本…", "Reading the selected Host version…")
	Register(KeyCmdHostSelectionWasNotChangedIfTaskProte, "Host 选择未更改；如操作被任务保护阻止，请查看启动器提示。", "Host selection was not changed. If task protection blocked the action, check the launcher notice.")
	Register(KeyCmdHostVIsSelectedTheRuntimeRemainsStopp, "已选择 Host v%s。内核保持停止；点击“启动 Host”使用该版本。", "Host v%s is selected. The runtime remains stopped; click “Start Host” to use this version.")
	Register(KeyCmdHostActionDidNotComplete, "Host 操作未完成：", "Host action did not complete: ")
	Register(KeyCmdSelectHostVersion, "选择 Host 版本", "Select Host version")
	Register(KeyCmdTargetHostVNNtheExistingTaskProtectio, "目标：Host v%s (%s)\n\n会通过现有任务保护停止当前 Host 和手机连接，再保存该版本供下次启动使用。旧运行时、会话及配置会保留。选择完成后内核保持停止，需单独点击“启动 Host”。", "Target: Host v%s (%s)\n\nThe existing task protection will stop the current Host and phone connection before saving this version for the next start. Previous runtimes, sessions and settings are retained. The runtime stays stopped until you click “Start Host”.")
	Register(KeyCmdStopAndSelect, "停止并选择", "Stop and select")
	Register(KeyCmdSettingsSavedButStartupRegistrationFai, "设置已保存，但开机启动未生效", "Settings saved, but startup registration failed")
	Register(KeyCmdResumeStateNeedsAttention, "恢复状态需要处理", "Resume state needs attention")
	Register(KeyCmdTheOriginalFileIsPreservedBackItUpBe, "原文件已保留，请备份后修复。当前窗口不会自动覆盖该文件。", "The original file is preserved. Back it up before repairing it. This window will not automatically overwrite it.")
	Register(KeyCmdGenerate, "自动生成", "Generate")
	Register(KeyCmdReadTaskHistory, "读取任务历史", "Read task history")
	Register(KeyCmdSaveTitle, "保存标题", "Save title")
	Register(KeyCmdSavePrompt, "保存提示词", "Save prompt")
	Register(KeyCmdDescribeTheTaskInTheConversationTheAg, "在会话中描述要定时的任务，Agent 会创建并显示在此页。", "Describe the task in the conversation; the agent creates it and it appears here.")
	Register(KeyCmdEnableDshIm, "启用 dsh-im", "Enable dsh-im")
	Register(KeyCmdDshImEnabledRestartTheAppIfThePageS, "已启用 dsh-im 插件；若页面仍为空请重启应用", "dsh-im enabled; restart the app if the page stays empty")
	Register(KeyCmdRemoteWorkspace, "远程工作区", "Remote workspace")
	Register(KeyCmdOpenTheDownloadPage, "打开下载页", "Open the download page")
	Register(KeyCmdAutoAllowTools, "自动允许工具", "Auto-allow tools")
	Register(KeyCmdCouldNotOpenInstalledApps, "无法打开已安装应用", "Could not open Installed apps")
	Register(KeyCmdOpenWindowsSettingsAppsInstalledAppsMa, "请手动进入 Windows 设置 → 应用 → 已安装应用，核对应用后再自行卸载。当前应用和安装状态未更改。", "Open Windows Settings > Apps > Installed apps manually, check the application, then uninstall it there. The current app and installation state are unchanged.")
	Register(KeyCmdNoVerifiedUninstallerFound, "未检测到可安全启动的卸载器", "No verified uninstaller found")
	Register(KeyCmdThisMayBeAPortableSourceBuildOrAnIn, "当前程序可能为便携版、源码构建或没有新版原生安装标记。未启动任何卸载器。请使用“已安装应用”并核对安装位置。", "This may be a portable/source build or an installation without the native marker. No uninstaller was started. Use Installed apps and check the installation location.")
	Register(KeyCmdUninstallCurrentNativeApp, "卸载当前原生应用", "Uninstall current native app")
	Register(KeyCmdContinuingChecksAndStopsTasksOpensThis, "继续将检查并停止任务、启动此安装的卸载器并退出应用。卸载器仍会要求确认；在其中取消不会恢复已退出的应用。不会清理用户数据目录。安装位置：", "Continuing checks and stops tasks, opens this installation's uninstaller and quits the app. The uninstaller still asks for confirmation; cancelling there will not reopen the app. User data directories are retained. Install location: ")
	Register(KeyCmdStartUninstallerAndQuit, "启动卸载器并退出", "Start uninstaller and quit")
	Register(KeyCmdTheWhalebridgeInstanceChangedOrIsUnava, "鲸桥实例已改变或暂不可用，未打开其他实例。", "The WhaleBridge instance changed or is unavailable. No other instance was opened.")
	Register(KeyCmdTheWhalebridgeInstanceChangedReviewIts, "鲸桥实例已改变，请重新查看提醒。", "The WhaleBridge instance changed. Review its reminders again.")
	Register(KeyCmdUninstallComponent, "卸载组件", "Uninstall component")
	Register(KeyCmdStopAndRemoveAndItsProgramFilesConfig, "将停止并删除组件「%s」及其程序文件，其配置数据会保留。", "Stop and remove %s and its program files. Configuration data will be retained.")
	Register(KeyCmdUninstall, "卸载", "Uninstall")
	Register(KeyCmdRollBackComponent, "回滚组件", "Roll back component")
	Register(KeyCmdRollBackToVItsRunningProcessWillBeS, "将把组件「%s」回滚到 v%s，当前进程会先停止。", "Roll back %s to v%s. Its running process will be stopped first.")
	Register(KeyCmdTheRuntimeIsStartingRetryImportAfterS, "内核正在启动，请稍后再确认导入。", "The runtime is starting. Retry import after startup settles.")
	Register(KeyCmdConfirmDataImport, "确认导入数据", "Confirm data import")
	Register(KeyCmdBeforeImportingCurrentTasksAreCheckedA, "导入前会检查当前任务并停止 Host 内核；如有运行中的工作，会继续显示现有任务确认。桌面窗口保持打开，导入结束后内核保持停止，可稍后手动启动。", "Before importing, current tasks are checked and the Host runtime is stopped. Active work uses the existing task confirmation. The desktop window stays open and the runtime stays stopped after import until you start it again.")
	Register(KeyCmdConfirmAndContinue, "确认并继续", "Confirm and continue")
	Register(KeyCmdCheckingCurrentTasksBeforeStoppingTheR, "正在检查当前任务，随后停止内核并导入；可取消后续导入。", "Checking current tasks before stopping the runtime and importing. You can cancel the remaining import.")
	Register(KeyCmdTheRuntimeChangedConfirmTheImportAgain, "运行环境已改变，请重新确认导入。", "The runtime changed. Confirm the import again.")
	Register(KeyCmdTheRuntimeIsStoppedImportingCompletedI, "内核已停止，正在导入；已完成项目将保留。", "The runtime is stopped. Importing; completed items will be retained.")
	Register(KeyCmdImportCancelledNoImportDataWasWritten, "已取消导入，未写入导入数据。", "Import cancelled. No import data was written.")
	Register(KeyCmdImportDidNotStart, "导入未开始：", "Import did not start: ")
	Register(KeyCmdTheRuntimeRemainsStoppedStartItManuall, " 内核保持停止，可在导入结果确认后手动启动。", " The runtime remains stopped; start it manually after checking the import result.")
	Register(KeyCmdImportRecoveryIsIncompleteAllCopiesWer, "导入恢复尚未完成，已保留所有副本，暂不能启动内核：", "Import recovery is incomplete. All copies were preserved and runtime startup is blocked: ")
	Register(KeyCmdAnInterruptedImportWasRecoveredComplet, "已恢复中断的导入。已完成项目保留；未完成目录已回到原状。请检查选择后继续导入，或点击启动内核。", "An interrupted import was recovered. Completed items were kept and incomplete directories were restored. Review your selection to continue importing, or start the runtime.")
	Register(KeyCmdOpeningWhalebridgeSettings, "正在打开鲸桥设置…", "Opening WhaleBridge settings…")
	Register(KeyCmdOfflineWaitingForNetworkAvailability, "网络已离线，等待网络恢复。", "Offline. Waiting for network availability.")
	Register(KeyCmdNotConnectedToTheHostYetTryAgainShor, "尚未连接 Host，请稍后重试", "Not connected to the Host yet — try again shortly")
	Register(KeyCmdFailed, "失败", " failed")
	Register(KeyCmdRefreshTheDeliveryDetailsFirst, "请先刷新交付详情。", "Refresh the delivery details first.")
	Register(KeyCmdRequestedOpeningTheProjectDirectory, "已请求打开项目目录。", "Requested opening the Project directory.")
	Register(KeyCmdNavigationChangedTheFileWasNotOpened, "导航已改变，未打开文件。", "Navigation changed; the file was not opened.")
	Register(KeyCmdThisFileChangedAfterTheReportOpeningI, "此文件在报告后已修改，打开的是当前内容。", "This file changed after the report; opening its current contents.")
	Register(KeyCmdRefreshTheProjectTeamFirst, "请先刷新 Project Team。", "Refresh the Project Team first.")
	Register(KeyCmdNavigationChangedTheRetryDraftWasNotI, "导航已改变，尚未写入重试草稿。", "Navigation changed; the retry draft was not inserted.")
	Register(KeyCmdTheLeadHasADraftOrPendingSendItWas, "Lead 中已有草稿或正在发送，已保留。请打开 Lead 处理现有草稿后再准备重试。", "The Lead has a draft or pending send; it was preserved. Open the Lead and finish that draft before preparing a retry.")
	Register(KeyCmdRetryRequestPlacedInTheOriginalLeadDr, "重试指令已放入原 Lead 草稿，尚未发送或重试。", "Retry request placed in the original Lead draft. It has not been sent or retried.")
	Register(KeyCmdHostIsNotConnectedRefreshWhenReady, "Host 尚未连接，请刷新。", "Host is not connected; refresh when ready.")
	Register(KeyCmdProjectHostIsUnavailable, "Project Host 不可用。", "Project Host is unavailable.")
	Register(KeyCmdProjectHostUnavailableRefreshToRetry, "Project Host不可用，请刷新。", "Project Host unavailable; refresh to retry.")
	Register(KeyCmdProjectUnavailableOrArchivingRefreshTo, "Project不可用或正在归档，请刷新。", "Project unavailable or archiving; refresh to retry.")
	Register(KeyCmdProjectStateUpdatedButListRefreshFaile, "Project状态已更新，但列表刷新失败：", "Project state updated, but list refresh failed: ")
	Register(KeyCmdProjectArchivedWorkHasStoppedConversat, "Project已归档。工作已停止，对话、工作目录和成果保留。", "Project archived. Work has stopped; conversation, working directory and results are retained.")
	Register(KeyCmdProjectRestoredSendANewInstructionInT, "Project已恢复。请在原Lead对话发送新指令，旧任务不会自动重启。", "Project restored. Send a new instruction in the original Lead conversation; old work does not restart automatically.")
	Register(KeyCmdAutomaticRefreshFailedShowingPreviousS, "自动刷新失败，保留上次状态：", "Automatic refresh failed; showing previous state: ")
	Register(KeyCmdAutoRefreshEvery5s, "每5秒自动刷新 · ", "Auto-refresh every 5s · ")
	Register(KeyCmdThisProjectIsNoLongerInTheHostList, "此Project已不在Host列表中。", "This Project is no longer in the Host list.")
	Register(KeyCmdTeamRefreshFailedShowingPreviousState, "Team刷新失败，保留上次状态：", "Team refresh failed; showing previous state: ")
	Register(KeyCmdMemberDetailRefreshFailed, "成员详情刷新失败：", "Member detail refresh failed: ")
	Register(KeyCmdTheMemberRunChangedPreviousRecordsAre, "成员执行轮次已变化；保留旧过程，可重新选择实时跟进。", "The member run changed; previous records are preserved. Select live follow again for the new run.")
	Register(KeyCmdProjectAndTeamStateRefreshAutomaticall, "每5秒自动刷新项目与Team状态。", "Project and Team state refresh automatically every 5 seconds.")
	Register(KeyCmdRefreshOrRestoreTheOriginalProjectFirs, "请先刷新或恢复原 Project。", "Refresh or restore the original Project first.")
	Register(KeyCmdNavigationChangedTheRecoveryDraftWasNo, "导航已改变，尚未写入恢复草稿。", "Navigation changed; the recovery draft was not inserted.")
	Register(KeyCmdSendingTheRecoveryRequestToTheOriginal, "正在向原 Lead 发送重新汇总请求；发送结果见主会话。", "Sending the recovery request to the original Lead; the main conversation shows submission status.")
	Register(KeyCmdChangesWereRevertedButRefreshingFailed, "改动已撤销，但读取结果失败；请刷新，不要重复撤销。", "Changes were reverted, but refreshing failed; refresh without repeating the revert.")
	Register(KeyCmdThisRequirementSChangesWereReverted, "已撤销此要求的改动。", "This requirement’s changes were reverted.")
	Register(KeyCmdProjectHostIsUnavailableRefreshToRetry, "Project Host 不可用，请刷新。", "Project Host is unavailable; refresh to retry.")
	Register(KeyCmdProjectIsNoLongerInTheCurrentListRef, "Project 已不在当前列表，请刷新。", "Project is no longer in the current list; refresh to retry.")
	Register(KeyCmdRestoreTheProjectBeforeChangingRequest, "恢复 Project 后才能修改要求卡片。", "Restore the Project before changing request cards.")
	Register(KeyCmdCardUpdatedButListRefreshFailedRefresh, "卡片已更新，但列表刷新失败；请刷新，勿重复操作", "Card updated, but list refresh failed; refresh without repeating the mutation")
	Register(KeyCmdThisProjectIsNotInTheCurrentListRefr, "此 Project 已不在当前列表，请刷新。", "This Project is not in the current list; refresh to retry.")
	Register(KeyCmdResumeStateWasNotSaved, "恢复状态未保存", "Resume state was not saved")
	Register(KeyCmdTaskDeleted, "任务已删除", "Task deleted.")
	Register(KeyCmdCouldNotDeleteTheTask, "无法删除任务", "Could not delete the task.")
	Register(KeyCmdDiagnosticsOrPluginInventoryChangedRef, "诊断或插件库存已变化，请刷新后重试。", "Diagnostics or plugin inventory changed; refresh and retry.")
	Register(KeyCmdPluginOperationDidNotComplete, "插件操作未完成。", "Plugin operation did not complete.")
	Register(KeyCmdApplied, "已应用：", "Applied: ")
	Register(KeyCmdUpdatedPlugins, "已更新插件：", "Updated plugins: ")
	Register(KeyCmdRetryingFullStartupRecoveryCompletesOn, "。正在完整重试；Host 健康就绪后才算恢复完成。", ". Retrying full startup; recovery completes only after the Host is healthy.")
	Register(KeyCmdThePreviousStartDidNotFullyRecoverAut, "上次启动未完整恢复，已暂停自动启动。请查看诊断后重试。", "The previous start did not fully recover. Auto-start is paused; review diagnostics and retry.")
	Register(KeyCmdTheRecoveryFileWasPreservedRepairItAn, "恢复文件已保留，请修复后重新打开应用。", "The recovery file was preserved; repair it and reopen the app.")
	Register(KeyCmdSkipUserPluginsRecoveryIsSavedStartIn, "已保留跳过用户插件的恢复模式。可继续以恢复模式启动，或清除恢复标记后重试全部插件。", "Skip-user-plugins recovery is saved. Start in recovery mode, or clear recovery and retry all plugins.")
	Register(KeyCmdRecoveryModeIsReadyUserPluginsRemainS, "仅恢复模式就绪；用户插件仍被跳过。", "Recovery mode is ready; user plugins remain skipped.")
	Register(KeyCmdCouldNotSaveStartupOutcome, "启动结果未能保存：", "Could not save startup outcome: ")
	Register(KeyCmdRecoveryModeIsReadyUserPluginsAreStil, "恢复模式已就绪；用户插件仍被跳过。请在插件页检查原因并明确恢复。", "Recovery mode is ready; user plugins are still skipped. Review the cause on the Plugins page before restoring them.")
	Register(KeyCmdCouldNotReadTheExistingCredentials, "无法读取现有凭据", "Could not read the existing credentials")
	Register(KeyCmdWhalebridge, "鲸桥 · WhaleBridge", "WhaleBridge")
	Register(KeyCmdCouldNotOpenTheWhalebridgeWindow, "无法打开鲸桥管理窗口", "Could not open the WhaleBridge window")
	Register(KeyCmdCouldNotCreateTheWhalebridgePage, "无法创建鲸桥页面：", "Could not create the WhaleBridge page: ")
	Register(KeyCmdAccounts, "账号页", "Accounts")
	Register(KeyCmdReload, "重新载入", "Reload")
	Register(KeyCmdOpeningTheWhalebridgeManagementPage, "正在打开鲸桥管理页…", "Opening the WhaleBridge management page…")
	Register(KeyCmdRestartHarness, "重启 Harness", "Restart Harness")
	Register(KeyCmdShowWindow, "显示窗口", "Show window")
	Register(KeyCmdOpenLauncher, "打开启动器", "Open launcher")
	Register(KeyCmdSettings, "设置…", "Settings…")
	Register(KeyCmdPluginMarketplace, "插件市场", "Plugin marketplace")
	Register(KeyCmdCloseWindow, "关闭窗口", "Close window")
	Register(KeyCmdAbout, "关于", "About")
	Register(KeyCmdFile, "文件", "File")
	Register(KeyCmdOpenWorkspace, "打开工作区…", "Open workspace…")
	Register(KeyCmdOpenWorkspaceInExplorer, "在资源管理器中打开工作区", "Open workspace in explorer")
	Register(KeyCmdMarketplace, "插件市场…", "Marketplace…")
	Register(KeyCmdSkills, "技能…", "Skills…")
	Register(KeyCmdRun, "运行", "Run")
	Register(KeyCmdReloadInterface, "重新加载界面", "Reload interface")
	Register(KeyCmdView, "视图", "View")
	Register(KeyCmdActualSize, "实际大小", "Actual size")
	Register(KeyCmdZoomIn, "放大", "Zoom in")
	Register(KeyCmdZoomOut, "缩小", "Zoom out")
	Register(KeyCmdFullscreen, "全屏", "Fullscreen")
	Register(KeyCmdHelp, "帮助", "Help")
	Register(KeyCmdWebUiGuide, "Web UI 指南", "Web UI guide")
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
