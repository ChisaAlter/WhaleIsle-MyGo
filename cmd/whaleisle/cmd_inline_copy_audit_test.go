package main

import (
	"os"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// auditedCmdCopyFiles are the files whose user-visible copy moved into the
// catalog in the 2026-10-26 audit. The test fails if one drops out of the
// catalog or keeps a second inline copy next to the i18n lookup.
var auditedCmdCopyFiles = []string{
	"git_push_review.go",
	"plugin_session_host.go",
	"plugin_creator_task.go",
	"office_document.go",
	"workspace_document_preview.go",
	"media_preview.go",
	"image_document.go",
	"import_flow.go",
	"project_directory.go",
	"plugin_contributions.go",
}

// auditedCmdCopyKeys is the migration list. It duplicates the i18n package's
// own slice on purpose: that test guards the catalog, this one guards the call
// sites, and neither can silently shrink the audit.
var auditedCmdCopyKeys = []string{
	i18n.KeyCmdConfirmPushDestination,
	i18n.KeyCmdConfirmPush,
	i18n.KeyCmdPushReviewCommitted,
	i18n.KeyCmdPushReviewDefaultBranch,
	i18n.KeyCmdSessionPluginTitle,
	i18n.KeyCmdSessionPluginHint,
	i18n.KeyCmdSessionContentUnavailable,
	i18n.KeyCmdConnectingSessionContent,
	i18n.KeyCmdSessionOwnerUnavailable,
	i18n.KeyCmdNoSessionPluginContent,
	i18n.KeyCmdSessionPluginContent,
	i18n.KeyCmdCreatorNeedsHost,
	i18n.KeyCmdCreatorUnavailable,
	i18n.KeyCmdCreatorChecking,
	i18n.KeyCmdCreatorPresetsUnreadable,
	i18n.KeyCmdCreatorResult,
	i18n.KeyCmdCreatorOpenExisting,
	i18n.KeyCmdOfficePreviewReady,
	i18n.KeyCmdOfficePreviewMissingFonts,
	i18n.KeyCmdOfficePreviewUnavailable,
	i18n.KeyCmdOfficePreviewCancelled,
	i18n.KeyCmdOfficePreviewTooLarge,
	i18n.KeyCmdOfficePreviewInvalid,
	i18n.KeyCmdOfficePreviewTimeout,
	i18n.KeyCmdOfficePreviewBusy,
	i18n.KeyCmdOfficePreviewSourceChanged,
	i18n.KeyCmdOfficePreviewFailed,
	i18n.KeyCmdPreviewSwitchSessionWorkspace,
	i18n.KeyCmdPreviewSourceClosed,
	i18n.KeyCmdPreviewSessionChanged,
	i18n.KeyCmdPreviewChildUnsupported,
	i18n.KeyCmdPreviewRestoreUnsupported,
	i18n.KeyCmdMediaPathRequired,
	i18n.KeyCmdMediaHostServiceRequired,
	i18n.KeyCmdMediaReadingSnapshot,
	i18n.KeyCmdMediaPreviewUnavailable,
	i18n.KeyCmdMediaSnapshotOpened,
	i18n.KeyCmdImagePreviewUnavailable,
	i18n.KeyCmdImportBusy,
	i18n.KeyCmdImportScanFirst,
	i18n.KeyCmdImportNothingSelected,
	i18n.KeyCmdImportRestartBlocked,
	i18n.KeyCmdChooseProjectDirectory,
	i18n.KeyCmdPluginContentOpen,
	i18n.KeyCmdPluginContentCollapse,
}

// TestCmdInlineCopyUsesCatalog proves the migrated call sites read the shared
// catalog. A key whose Chinese text reappears as a quoted literal in cmd/ means
// the edit was reverted or a new inline copy was added beside the lookup.
func TestCmdInlineCopyUsesCatalog(t *testing.T) {
	if got := len(auditedCmdCopyKeys); got < 30 || got > 50 {
		t.Fatalf("audit migrated %d keys, want the agreed 30-50", got)
	}
	sources := make(map[string]string, len(auditedCmdCopyFiles))
	for _, name := range auditedCmdCopyFiles {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		sources[name] = string(data)
		if !strings.Contains(sources[name], "i18n.T(") {
			t.Fatalf("%s has no catalog lookup", name)
		}
	}
	for _, key := range auditedCmdCopyKeys {
		zh := i18n.Lookup(i18n.LocaleZH, key)
		en := i18n.Lookup(i18n.LocaleEN, key)
		if zh == "" || en == "" {
			t.Fatalf("%s is missing a locale: zh=%q en=%q", key, zh, en)
		}
		patterns := []string{
			`"` + zh + `"`,
			`pluginCopy("` + zh + `"`,
			`pluginCreatorText("` + zh + `"`,
			`gitActionLabel("` + zh + `"`,
		}
		for name, src := range sources {
			for _, pattern := range patterns {
				if strings.Contains(src, pattern) {
					t.Fatalf("%s still holds inline copy for %s: %s", name, key, pattern)
				}
			}
		}
	}
}

// TestCmdInlineTextAuditDocumentExists keeps the audit deliverable in tree and
// makes sure every migrated key is listed there, so the "30-50 migrated" claim
// is traceable without rerunning the scanner.
func TestCmdInlineTextAuditDocumentExists(t *testing.T) {
	const path = "../../docs/parity/cmd-inline-text-audit-20261026.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit document: %v", err)
	}
	doc := string(data)
	for _, want := range []string{
		"# cmd/ inline Chinese text audit",
		"## 1. How the scan works",
		"## 2. Per-file inventory",
		"## 3. Migrated to internal/ui/i18n",
		"## 4. Remaining items and why",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("audit document is missing section %q", want)
		}
	}
	for _, key := range auditedCmdCopyKeys {
		if !strings.Contains(doc, key) {
			t.Fatalf("audit document does not list migrated key %s", key)
		}
	}
}
