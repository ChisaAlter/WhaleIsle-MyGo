package i18n

import (
	"strings"
	"testing"
	"unicode"
)

// cmdCopyAuditKeys are the keys added by the 2026-10-26 cmd/ inline-text audit.
// The test fails if a key loses either locale, both locales are identical, the
// format verbs drift, or the English copy still carries Chinese characters.
var cmdCopyAuditKeys = []string{
	KeyCmdConfirmPushDestination,
	KeyCmdConfirmPush,
	KeyCmdPushReviewCommitted,
	KeyCmdPushReviewDefaultBranch,
	KeyCmdSessionPluginTitle,
	KeyCmdSessionPluginHint,
	KeyCmdSessionContentUnavailable,
	KeyCmdConnectingSessionContent,
	KeyCmdSessionOwnerUnavailable,
	KeyCmdNoSessionPluginContent,
	KeyCmdSessionPluginContent,
	KeyCmdCreatorNeedsHost,
	KeyCmdCreatorUnavailable,
	KeyCmdCreatorChecking,
	KeyCmdCreatorPresetsUnreadable,
	KeyCmdCreatorResult,
	KeyCmdCreatorOpenExisting,
	KeyCmdOfficePreviewReady,
	KeyCmdOfficePreviewMissingFonts,
	KeyCmdOfficePreviewUnavailable,
	KeyCmdOfficePreviewCancelled,
	KeyCmdOfficePreviewTooLarge,
	KeyCmdOfficePreviewInvalid,
	KeyCmdOfficePreviewTimeout,
	KeyCmdOfficePreviewBusy,
	KeyCmdOfficePreviewSourceChanged,
	KeyCmdOfficePreviewFailed,
	KeyCmdPreviewSwitchSessionWorkspace,
	KeyCmdPreviewSourceClosed,
	KeyCmdPreviewSessionChanged,
	KeyCmdPreviewChildUnsupported,
	KeyCmdPreviewRestoreUnsupported,
	KeyCmdMediaPathRequired,
	KeyCmdMediaHostServiceRequired,
	KeyCmdMediaReadingSnapshot,
	KeyCmdMediaPreviewUnavailable,
	KeyCmdMediaSnapshotOpened,
	KeyCmdImagePreviewUnavailable,
	KeyCmdImportBusy,
	KeyCmdImportScanFirst,
	KeyCmdImportNothingSelected,
	KeyCmdImportRestartBlocked,
	KeyCmdChooseProjectDirectory,
	KeyCmdPluginContentOpen,
	KeyCmdPluginContentCollapse,
}

func cmdCopyVerbCount(text string) int {
	n := 0
	for i := 0; i+1 < len(text); i++ {
		if text[i] == '%' && strings.ContainsRune("sdvq", rune(text[i+1])) {
			n++
		}
	}
	return n
}

func cmdCopyHasHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func TestCmdCopyAuditKeysHaveBothLocales(t *testing.T) {
	if len(cmdCopyAuditKeys) < 30 || len(cmdCopyAuditKeys) > 50 {
		t.Fatalf("audit slice has %d keys, want the agreed 30-50 migration", len(cmdCopyAuditKeys))
	}
	seen := map[string]bool{}
	for _, key := range cmdCopyAuditKeys {
		if key == "" {
			t.Fatal("empty key in the audit slice")
		}
		if seen[key] {
			t.Fatalf("duplicate key %q in the audit slice", key)
		}
		seen[key] = true
		if !Registered(key) {
			t.Fatalf("%s is not registered in both catalogs", key)
		}
		zh := Lookup(LocaleZH, key)
		en := Lookup(LocaleEN, key)
		if zh == "" || en == "" {
			t.Fatalf("%s is missing a locale: zh=%q en=%q", key, zh, en)
		}
		if zh == en {
			t.Fatalf("%s has identical zh/en copy", key)
		}
		if cmdCopyHasHan(en) {
			t.Fatalf("%s English copy still has Chinese characters: %q", key, en)
		}
		if cmdCopyVerbCount(zh) != cmdCopyVerbCount(en) {
			t.Fatalf("%s format verbs differ: zh=%q en=%q", key, zh, en)
		}
	}
}
