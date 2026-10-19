package nsis

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These are static packaging contract checks. They never compile or launch an
// installer, touch an installed application, or certify Windows runtime behavior.
func transactionSource(t *testing.T, name string) string {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(source), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func sourceRegion(t *testing.T, source, start, end string) string {
	t.Helper()
	first := strings.Index(source, start)
	if first < 0 {
		t.Fatalf("missing source boundary %q", start)
	}
	last := strings.Index(source[first+len(start):], end)
	if last < 0 {
		t.Fatalf("missing source boundary %q", end)
	}
	return source[first : first+len(start)+last]
}

func requireSourceOrder(t *testing.T, source string, steps ...string) {
	t.Helper()
	last := 0
	for _, step := range steps {
		next := strings.Index(source[last:], step)
		if next < 0 {
			t.Fatalf("missing or reordered transaction step %q", step)
		}
		last += next + len(step)
	}
}

func TestInstallStagesBeforeReplacingAndRegistering(t *testing.T) {
	source := transactionSource(t, "whaleisle.nsi")
	install := sourceRegion(t, source, `Section "Install"`, "SectionEnd")
	requireSourceOrder(t, install, "Call wiBeginStage", `File "${STAGE_DIR}\${EXE_NAME}"`,
		"Call wiCreateManifest", "$wiError != 0", "Abort", "Call wiPromote",
		`WriteRegStr HKCU "Software\${APP_ID}"`, "CreateShortCut", "Call wiCommitInstallation")
	helper := transactionSource(t, "payload-transaction.nsh")
	rollback := sourceRegion(t, helper, "Function wiRollback", "FunctionEnd")
	requireSourceOrder(t, rollback, `Rename "$wiFinalFS" "$wiStageFS"`, `Rename "$wiOldFS" "$wiFinalFS"`)
	failure := sourceRegion(t, helper, "Function .onInstFailed", "FunctionEnd")
	requireSourceOrder(t, failure, "Call wiRollback", "Call wiRestoreRegistration")
}

func TestUninstallFailureKeepsRetryRegistrationAndShortcuts(t *testing.T) {
	source := transactionSource(t, "whaleisle.nsi")
	uninstall := sourceRegion(t, source, `Section "Uninstall"`, "SectionEnd")
	requireSourceOrder(t, uninstall, "Call un.wiRemovePayload", "Delete ", "DeleteRegKey")
	helper := transactionSource(t, "payload-transaction.nsh")
	remove := sourceRegion(t, helper, "Function un.wiRemovePayload", "FunctionEnd")
	requireSourceOrder(t, remove, "retry-manifest", `Rename "$wiRootFS" "$0"`,
		"Call un.wiCleanOwnedPayload", "Call un.wiRetainRetry", `Delete "$wiRootFS\Uninstall.exe"`)
	retry := sourceRegion(t, helper, "Function un.wiRetainRetry", "FunctionEnd")
	requireSourceOrder(t, retry, "retry-uninstall.exe", `"InstallLocation" "$wiRoot"`,
		`"UninstallString"`, "SetErrorLevel 2", "Abort")
	if strings.Contains(retry, "DeleteRegKey") || strings.Contains(retry, `Delete "$DESKTOP`) {
		t.Fatal("failed cleanup removes its retry management")
	}
}

func TestExistingTreesRequireAnOwnedManifestAndNeverRecurse(t *testing.T) {
	for _, name := range []string{"whaleisle.nsi", "payload-transaction.nsh"} {
		source := transactionSource(t, name)
		for _, line := range strings.Split(source, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "RMDir /r") {
				t.Fatalf("%s recursively deletes an existing directory", name)
			}
		}
	}
	helper := transactionSource(t, "payload-transaction.nsh")
	clean := sourceRegion(t, helper, "Function ${Prefix}wiCleanOwnedPayload", "FunctionEnd")
	requireSourceOrder(t, clean, "Call ${Prefix}wiCheckRoot", "GetFileAttributesW", "0x410",
		"FileReadUTF16LE", "WhaleIsle-Payload-1", "Call ${Prefix}wiOwnedPath", `Delete "$wiRootFS\$R2"`)
	owned := sourceRegion(t, helper, "Function ${Prefix}wiOwnedPath", "FunctionEnd")
	for _, guard := range []string{`$R2 == ".."`, `$R2 == "."`, `"resources"`, `"runtime"`, `"pet-render-assets"`, "0x400"} {
		if !strings.Contains(owned, guard) {
			t.Fatalf("missing ownership/no-follow constraint %s", guard)
		}
	}
}

