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
}

// migratedCmdFiles still contain the call sites for that slice.
var migratedCmdFiles = []string{
	"desktop_tray.go", "desktop_tray_menu.go", "application_menu.go",
	"update_check_lifecycle.go", "update_progress.go", "update_attention.go", "launcher_update_check.go",
	"account_bonus_notice.go", "account_quota_notice.go", "account_status_notices.go", "account_return.go",
	"recovery.go",
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
