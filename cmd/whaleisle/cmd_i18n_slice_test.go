package main

import (
	"os"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// migratedCmdKeys is the round179 slice: tray and application menu labels, the
// update notices, account/bonus/quota notices and the recovery dialog copy.
var migratedCmdKeys = []string{
	"cmd.choose_local_avatar_image",
	"cmd.bot_created_session_connection_unavailab",
	"cmd.created_bot_open_canonical_session",
	"cmd.catalog_record_retained_session_connecti",
	"cmd.open_inspected_catalog_bot",
	"cmd.watching_this_bot_page_s_complete_catalo",
	"cmd.system_notifications_unavailable_or_not",
	"cmd.could_not_restore_the_notification_targe",
	"cmd.notification_target_verify_in_visibility",
	"cmd.system_notification_delivery_failed_chec",
	"cmd.conversation_opened_read_marker_failed",
	"cmd.opening_canonical_conversation",
	"cmd.resolve_the_existing_chat_draft_before_s",
	"cmd.canonical_conversation_opened",
	"cmd.needs_attention",
	"cmd.created_room_open_canonical_session",
	"cmd.unsupported_trigger_address_no_credentia",
	"cmd.no_image_on_the_clipboard",
	"cmd.enter_a_reply_or_add_an_attachment",
	"cmd.host_offline",
	"cmd.no_completed_bot_json_reply",
	"cmd.resolve_the_existing_chat_draft_first",
	"cmd.draft_prepared_review_and_send",
	"cmd.imported_for_preview_save_to_apply",
	"cmd.saved_other_bot_copies_remain_independen",
	"cmd.independent_copy_added",
	"cmd.template_updated_for_future_copies",
	"cmd.removed_from_this_bot_template_retained",
	"cmd.export_installation_history",
	"cmd.phone_remote_setting_was_not_saved_retry",
	"cmd.whale_assistant_is_disabled",
	"cmd.host_offline_2",
	"cmd.connection_changed",
	"cmd.conversation_changed",
	"cmd.pet_chat_unavailable",
	"cmd.pet_chat_opened",
	"cmd.runtime_changed",
	"cmd.cancelled",
	"cmd.scan_unavailable_saved_food_retained",
	"cmd.local_food_refreshed",
	"cmd.connect_host_and_enable_whale_first",
	"cmd.select_the_pet_look_model_and_refresh_it",
	"cmd.look_at_screen",
	"cmd.cancel",
	"cmd.capture_and_send",
	"cmd.cancelled_no_screenshot_taken",
	"cmd.window_conversation_or_vision_route_chan",
	"cmd.context_changed_screenshot_was_not_sent",
	"cmd.context_changed",
	"cmd.look_failed_check_the_vision_model",
	"cmd.host_offline_reopen_after_connecting",
	"cmd.gateway_settings_saved_applying",
	"cmd.phone_pr_review",
	"cmd.working",
	"cmd.phone_requested_a_pr_review_and_confirm",
	"cmd.commit_push_may_already_be_complete_canc",
	"cmd.pr_title",
	"cmd.pr_body",
	"cmd.create_draft_pr",
	"cmd.generate_candidate",
	"cmd.candidate_not_yet_applied",
	"cmd.apply_candidate",
	"cmd.prepare_target_review",
	"cmd.review_the_exact_content_below",
	"cmd.confirm_this_pr",
	"cmd.outcome_uncertain_check_existing_pr_befo",
	"cmd.published_html_unavailable",
	"cmd.save_published_html",
	"cmd.isolated_preview_helper_is_not_installed",
	"cmd.tray_unavailable",
	"cmd.the_system_tray_could_not_be_created_the_win",
	"cmd.keep_window",
	"cmd.quit_application",
	"cmd.minimized_to_tray",
	"cmd.click_the_tray_icon_to_reopen_the_window_or",
	"cmd.desktop_pet",
	"cmd.developer_tools",
	"cmd.checking_for_updates",
	"cmd.could_not_check_for_updates",
	"cmd.this_build_has_no_update_feed_configured_dev",
	"cmd.you_are_on_the_latest_version",
	"cmd.update_available",
	"cmd.check_complete",
	"cmd.ok",
	"cmd.close",
	"cmd.preparing_the_update_download",
	"cmd.downloading_the_update",
	"cmd.the_update_is_downloaded_and_verified_instal",
	"cmd.technical_details",
	"cmd.dismiss",
	"cmd.new_version",
	"cmd.download_route",
	"cmd.installer_size",
	"cmd.an_incremental_patch_path_is_available",
	"cmd.version",
	"cmd.platform",
	"cmd.includes_the_runtime",
	"cmd.mygo_update_awaiting_confirmation",
	"cmd.is_available_return_to_confirmation_download",
	"cmd.mygo_update_available",
	"cmd.review_versions_and_downloads",
	"cmd.later",
	"cmd.confirming_notice",
	"cmd.could_not_acknowledge_retry",
	"cmd.this_account_cannot_acknowledge_the_notice_r",
	"cmd.got_it",
	"cmd.retry",
	"cmd.reward_notice",
	"cmd.request_quota_exhausted",
	"cmd.no_balance_available",
	"cmd.mygo_cannot_start_a_new_task_with_this_accou",
	"cmd.top_up",
	"cmd.you_have_signed_out_of_your_account_please_l",
	"cmd.model_unavailable_please_sign_in_and_try_aga",
	"cmd.sign_in",
	"cmd.could_not_open_authorization_page",
	"cmd.could_not_open_top_up_page",
	"cmd.could_not_cancel_sign_in",
	"cmd.recovered_drafts_were_kept_the_workspace_cha",
	"cmd.the_previous_session_is_not_in_the_current_l",
	"cmd.could_not_save_resume_state",
	"cmd.the_original_file_was_not_overwritten_keep_t",
	"cmd.keep_window_open",
	"cmd.quit_without_saving",
	"cmd.action_did_not_complete_the_window_was_kept",
	"cmd.quit",
	"cmd.restart",
	"cmd.install_update_and_quit",
	"cmd.change_whalebridge_component",
	"cmd.stop_runtime",
	"cmd.this_stops_the_host_runtime_the_desktop_appl",
	"cmd.host_task_protection_is_unavailable_backgrou",
	"cmd.known_local_running_work_sessions_and_termin",
	"cmd.these_checks_are_unknown",
	"cmd.continuing_may_interrupt_work_this_does_not",
	"cmd.continue",
	"cmd.import_is_still_running_wait_for_completion",
	"cmd.local_work_items_sessions_and_terminals_are",
	"cmd.install_update",
	"cmd.the_update_has_been_downloaded_and_verified",
	"cmd.install_and_quit",
	"cmd.mygo_is_available",
	"cmd.awaiting_confirmation_choose_later_or_downlo",
	"cmd.loading_mygo_release_catalog",
	"cmd.catalog_lookup_failed_the_previous_list_and",
	"cmd.release_lookup_cancelled",
	"cmd.loaded_mygo_releases",
	"cmd.no_compatible_mygo_installer_has_been_publis",
	"cmd.whaleisle_vendor_overrides_the_runtime_path",
	"cmd.selecting_and_verifying_a_mygo_installer",
	"cmd.downloading_and_verifying_full_installer",
	"cmd.reusing_cached_installer_and_downloading_cha",
	"cmd.installation_preparation_failed_the_current",
	"cmd.download_cancelled_the_current_installation",
	"cmd.desktop_only",
	"cmd.includes_runtime",
	"cmd.full_package",
	"cmd.delta_reconstruction",
	"cmd.installer_downloaded_and_verified_choose_ins",
	"cmd.disk_image_downloaded_and_verified_choose_op",
	"cmd.the_source_did_not_support_ranges_the_comple",
	"cmd.the_installer_is_retained_but_caching_it_for",
	"cmd.this_platform_cannot_start_the_installer",
	"cmd.verified_installer_retained_for_retry_the_cu",
	"cmd.verifying_whalebridge_version_and_impact_the",
	"cmd.cancelled_the_installed_version_connections",
	"cmd.whalebridge_v",
	"cmd.was_uninstalled_and_disconnected_chats_and_o",
	"cmd.provider_settings_keys_accounts_and_usage_we",
	"cmd.whalebridge_provider_settings_keys_accounts",
	"cmd.select_a_new_default_model",
	"cmd.whalebridge_switched_to_v",
	"cmd.user_data_and_the_original_running_state_wer",
	"cmd.confirm_whalebridge_uninstall",
	"cmd.continue_uninstall",
	"cmd.target_whalebridge_v_this_stops_the_service",
	"cmd.your_current_default_model_uses_whalebridge",
	"cmd.confirm_whalebridge_version_change",
	"cmd.switch_version",
	"cmd.confirm_whalebridge_rollback",
	"cmd.roll_back",
	"cmd.target_whalebridge_v_a_running_service_will",
	"cmd.also_delete_whalebridge_configuration",
	"cmd.target_n_ndeletion_permanently_removes_whale",
	"cmd.delete_configuration",
	"cmd.keep_configuration",
	"cmd.choose_a_macos_host_offline_bundle",
	"cmd.install_host_offline_bundle",
	"cmd.source_ndestination_n_nextract_and_install_t",
	"cmd.install",
	"cmd.extracting_and_checking_host_and_its_matchin",
	"cmd.host_v_was_installed_in_choose_select_instal",
	"cmd.select_an_installed_host_version_directory",
	"cmd.reading_the_selected_host_version",
	"cmd.host_selection_was_not_changed_if_task_prote",
	"cmd.host_v_is_selected_the_runtime_remains_stopp",
	"cmd.host_action_did_not_complete",
	"cmd.select_host_version",
	"cmd.target_host_v_n_nthe_existing_task_protectio",
	"cmd.stop_and_select",
	"cmd.settings_saved_but_startup_registration_fail",
	"cmd.resume_state_needs_attention",
	"cmd.the_original_file_is_preserved_back_it_up_be",
	"cmd.generate",
	"cmd.read_task_history",
	"cmd.save_title",
	"cmd.save_prompt",
	"cmd.describe_the_task_in_the_conversation_the_ag",
	"cmd.enable_dsh_im",
	"cmd.dsh_im_enabled_restart_the_app_if_the_page_s",
	"cmd.remote_workspace",
	"cmd.open_the_download_page",
	"cmd.auto_allow_tools",
	"cmd.bot_status_unavailable",
	"cmd.configuration_load_failed",
	"cmd.commit_staged_changes_only",
	"cmd.seconds_remaining",
	"cmd.open_file",
}

// migratedCmdFiles still contain the call sites for that slice.
var migratedCmdFiles = []string{
	"desktop_tray.go",
	"desktop_tray_menu.go",
	"application_menu.go",
	"update_check_lifecycle.go",
	"update_progress.go",
	"update_attention.go",
	"launcher_update_check.go",
	"account_bonus_notice.go",
	"account_quota_notice.go",
	"account_status_notices.go",
	"account_return.go",
	"recovery.go",
	"task_protection.go",
	"native_releases.go",
	"component_changes.go",
	"host_bundle.go",
	"main.go",
}

func cmdVerbCount(text string) int {
	n := 0
	for i := 0; i+1 < len(text); i++ {
		if text[i] == '%' && strings.ContainsRune("sdvq", rune(text[i+1])) {
			n++
		}
	}
	return n
}

// TestCmdI18nSliceCatalog keeps both locales registered for every migrated key
// and keeps their format verbs in step, so a translated string cannot drop an
// argument.
func TestCmdI18nSliceCatalog(t *testing.T) {
	for _, key := range migratedCmdKeys {
		zh := i18n.Lookup(i18n.LocaleZH, key)
		en := i18n.Lookup(i18n.LocaleEN, key)
		if zh == "" || en == "" {
			t.Fatalf("%s is missing a locale: zh=%q en=%q", key, zh, en)
		}
		if zh == en {
			t.Fatalf("%s has identical zh/en copy", key)
		}
		if cmdVerbCount(zh) != cmdVerbCount(en) {
			t.Fatalf("%s format verbs differ: zh=%q en=%q", key, zh, en)
		}
	}
}

// TestCmdI18nSliceCallSites proves the migrated files read the catalog instead
// of keeping the inline pair: no migrated Chinese copy may remain inside a
// resumeText call.
func TestCmdI18nSliceCallSites(t *testing.T) {
	sources := make(map[string]string, len(migratedCmdFiles))
	for _, name := range migratedCmdFiles {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		sources[name] = string(data)
	}
	for _, key := range migratedCmdKeys {
		zh := i18n.Lookup(i18n.LocaleZH, key)
		if zh == "" || strings.ContainsAny(zh, "\\\"") {
			continue
		}
		inline := "resumeText(\"" + zh + "\""
		for name, src := range sources {
			if strings.Contains(src, inline) {
				t.Fatalf("%s still holds inline copy for %s: %s", name, key, zh)
			}
		}
	}
}
