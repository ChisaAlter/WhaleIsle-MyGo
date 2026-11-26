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

func TestHealthIsConfirmedBeforeRegistrationAndOldPayloadRetirement(t *testing.T) {
	install := sourceRegion(t, transactionSource(t, "whaleisle.nsi"), `Section "Install"`, "SectionEnd")
	requireSourceOrder(t, install, "Call wiPromote", "Call wiConfirmHealth", `StrCpy $wiRegistryTouched "1"`, "WriteRegStr", "Call wiCommitInstallation")
	health := sourceRegion(t, transactionSource(t, "payload-transaction.nsh"), "Function wiConfirmHealth", "FunctionEnd")
	requireSourceOrder(t, health, "!ifndef SHELL_ONLY", "whaleisle-integration.exe", "ExecWait", "--check-installation", "$R9 != 0", "SetErrorLevel 2", "Abort")
	if strings.Contains(health, "wiCommitted") {
		t.Fatal("health failure can bypass rollback")
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
	guiEnd := sourceRegion(t, helper, "Function .onGUIEnd", "FunctionEnd")
	requireSourceOrder(t, guiEnd, `${If} $wiCommitted != "1"`, "Call wiRollback", "Call wiRestoreRegistration", "${EndIf}")
}

func TestPayloadOutputDoesNotChangeProcessWorkingDirectory(t *testing.T) {
	helper := transactionSource(t, "payload-transaction.nsh")
	output := sourceRegion(t, helper, "!macro WI_SetPayloadOutPath Path", "!macroend")
	requireSourceOrder(t, output, "ClearErrors", `CreateDirectory "${Path}"`, "WI_CheckExtract", `StrCpy $OUTDIR "${Path}"`)
	if strings.Contains(output, "SetOutPath") {
		t.Fatal("payload output must not call SetCurrentDirectoryW through SetOutPath")
	}
	install := sourceRegion(t, transactionSource(t, "whaleisle.nsi"), `Section "Install"`, "SectionEnd")
	if strings.Contains(install, "SetOutPath ") || strings.Count(install, "!insertmacro WI_SetPayloadOutPath ") != 10 {
		t.Fatal("root, resources, runtime and pet output must use the long-path extraction macro")
	}
	for _, function := range []string{"Function wiRollback", "Function wiPromote", "Function un.wiRemovePayload"} {
		body := sourceRegion(t, helper, function, "FunctionEnd")
		if !strings.Contains(body, "SetOutPath $TEMP") {
			t.Fatalf("%s no longer releases the payload working directory before moving it", function)
		}
	}
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

func TestRetiredUninstallerCannotManageAnotherInstallation(t *testing.T) {
	helper := transactionSource(t, "payload-transaction.nsh")
	owner := sourceRegion(t, helper, "Function un.wiOwnsRegistration", "FunctionEnd")
	requireSourceOrder(t, owner, `"NativeInstallID"`, "$R0 != $wiUninstallID", `"NativeRuntime"`,
		`"InstallLocation"`, "$R0 == $wiUninstallSource", "$R1 == $wiUninstallSource", "StrCpy $wiCanManage 1")
	retry := sourceRegion(t, helper, "Function un.wiRetainRetry", "FunctionEnd")
	requireSourceOrder(t, retry, "Call un.wiOwnsRegistration", "${If} $wiCanManage == 1", "WriteRegStr")
	uninstall := sourceRegion(t, transactionSource(t, "whaleisle.nsi"), `Section "Uninstall"`, "SectionEnd")
	requireSourceOrder(t, uninstall, "Call un.wiRemovePayload", "Call un.wiOwnsRegistration",
		"${If} $wiCanManage == 1", "Delete ", "DeleteRegKey")
	create := sourceRegion(t, helper, "Function wiCreateManifest", "FunctionEnd")
	if !strings.Contains(create, "I|$wiInstallID") {
		t.Fatal("payload does not retain its unique installation identity")
	}
}

func TestLegacyRegisteredUninstallerRetiresBeforePromotion(t *testing.T) {
	helper := transactionSource(t, "payload-transaction.nsh")
	capture := sourceRegion(t, helper, "Function wiCaptureOldManagement", "FunctionEnd")
	requireSourceOrder(t, capture, "$wiOldLocation != $wiFinal", "$wiOldUninstall", "${GetParent}",
		"$1 != $wiFinal", "${GetFileName}", "Call wiReadManifestIdentity")
	retire := sourceRegion(t, helper, "Function wiRetireLegacyManagement", "FunctionEnd")
	requireSourceOrder(t, retire, "GetFileAttributesW", "0x410", "Rename", ".retired-uninstaller", "StrCpy $wiOldManagementRetired")
	promote := sourceRegion(t, helper, "Function wiPromote", "FunctionEnd")
	requireSourceOrder(t, promote, `Rename "$wiFinalFS" "$wiOldFS"`, "Call wiRetireLegacyManagement", `Rename "$wiStageFS" "$wiFinalFS"`)
	rollback := sourceRegion(t, helper, "Function wiRollback", "FunctionEnd")
	requireSourceOrder(t, rollback, ".retired-uninstaller", `Rename "$wiOldFS" "$wiFinalFS"`)
}

func TestUpgradeRequiresNativeIdentityAndRejectsElectron(t *testing.T) {
	helper := transactionSource(t, "payload-transaction.nsh")
	begin := sourceRegion(t, helper, "Function wiBeginStage", "FunctionEnd")
	requireSourceOrder(t, begin, `"$wiFinalFS\Whale Isle.exe"`, "This is an Electron Whale Isle installation",
		`"$wiFinalFS\whaleisle.exe"`, "Call wiReadManifestIdentity", `$wiOldNative != "WhaleIsle-MyGo"`,
		"$wiOldLocation != $wiFinal", "Call wiCaptureOldManagement", "CoCreateGuid")
	if strings.Contains(begin, `${AndIfNot} ${FileExists} "$wiFinalFS\Whale Isle.exe"`) {
		t.Fatal("Electron executable still grants upgrade ownership")
	}
}

// An installer compiled from a stage that declares dependency links but carries
// no replayer installs a runtime whose package names never resolve, and reports
// nothing. The stage state is a compile-time fact, so it is refused at build.
func TestStagedRuntimeLinksCannotCompileWithoutTheirReplayer(t *testing.T) {
	links := transactionSource(t, "runtime-links.nsh")
	requireSourceOrder(t, links, `!ifndef SHELL_ONLY`,
		`!if /FileExists "${STAGE_DIR}\runtime\vendor\runtime-graph.json"`,
		`!if /FileExists "${STAGE_DIR}\runtime\runtime-links.exe"`,
		"!else", "!error", "!endif")
	if strings.Count(links, "!error") != 1 {
		t.Fatal("only the missing replayer may stop a compile")
	}
}

// A replay that runs must say so. The trace is the only evidence a silent
// install leaves, and it has to carry the guard result and the helper exit code.
func TestRuntimeLinkReplayRecordsItsGuardAndExit(t *testing.T) {
	links := transactionSource(t, "runtime-links.nsh")
	for _, name := range []string{"WI_RunRuntimeLinks", "WI_RunUninstallRuntimeLinks"} {
		macro := sourceRegion(t, links, "!macro "+name+" ", "!macroend")
		requireSourceOrder(t, macro, `"${Root}\runtime\vendor\runtime-graph.json"`,
			`ExecWait '"$PLUGINSDIR\runtime-links.exe" ${Command} "${Root}\runtime\vendor"`,
			`"$LOCALAPPDATA\mygo\runtime-links.log"`, "$R9", "StrCpy $wiError 1")
		if !strings.Contains(macro, "WI_TraceRuntimeLinks") {
			t.Fatalf("%s leaves no trace of its guard and exit code", name)
		}
	}
}

// Promotion must fail when the payload it just renamed carries no graph, and an
// upgrade must repair a tree promoted before replay existed instead of refusing
// every installation that predates the fix.
func TestPromotionRequiresItsGraphAndRepairsThePreviousTree(t *testing.T) {
	helper := transactionSource(t, "payload-transaction.nsh")
	promote := sourceRegion(t, helper, "Function wiPromote", "FunctionEnd")
	requireSourceOrder(t, promote, `WI_RunRuntimeLinks "$wiFinalFS" verify 0`,
		`WI_RunRuntimeLinks "$wiFinalFS" restore 0`, `WI_RunRuntimeLinks "$wiFinalFS" verify 0`,
		`WI_RunRuntimeLinks "$wiFinalFS" detach 0`, `Rename "$wiStageFS" "$wiFinalFS"`,
		`WI_RunRuntimeLinks "$wiFinalFS" restore 1`,
		"The new runtime dependencies could not be verified or restored")
	links := transactionSource(t, "runtime-links.nsh")
	macro := sourceRegion(t, links, "!macro WI_RunRuntimeLinks ", "!macroend")
	requireSourceOrder(t, macro, "!if ${Required} == 1", "StrCpy $wiError 1", "!else")
}

func TestEveryReplayCallSiteStatesWhetherItsGraphIsRequired(t *testing.T) {
	for _, name := range []string{"payload-transaction.nsh", "runtime-links.nsh"} {
		for _, line := range strings.Split(transactionSource(t, name), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "!insertmacro WI_RunRuntimeLinks ") {
				continue
			}
			if last := line[len(line)-1]; last != '0' && last != '1' {
				t.Fatalf("replay call site does not state its requirement: %s", line)
			}
		}
	}
}
