package nsis

import (
	"strings"
	"testing"
)

func TestNotificationTargetSharesInstallRegistrationTransaction(t *testing.T) {
	installer := transactionSource(t, "whaleisle.nsi")
	install := sourceRegion(t, installer, `Section "Install"`, "SectionEnd")
	requireSourceOrder(t, install, "Call wiPromote", `"NativeInstallID" "$wiInstallID"`, `"NativeNotificationTarget" "$wiFinal\${EXE_NAME}"`, "Call wiCommitInstallation")
	helper := transactionSource(t, "payload-transaction.nsh")
	remember := sourceRegion(t, helper, "Function wiRememberRegistration", "FunctionEnd")
	if !strings.Contains(remember, `ReadRegStr $wiOldNotificationTarget HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeNotificationTarget"`) {
		t.Fatal("original notification target is not remembered with the installation record")
	}
	restore := sourceRegion(t, helper, "Function wiRestoreRegistration", "FunctionEnd")
	if !strings.Contains(restore, `"NativeNotificationTarget" $wiOldNotificationTarget`) {
		t.Fatal("rollback does not restore the original notification target")
	}
	retry := sourceRegion(t, helper, "Function un.wiRetainRetry", "FunctionEnd")
	if strings.Contains(retry, "NativeNotificationTarget") {
		t.Fatal("retired retry changes the original notification target")
	}
	uninstall := sourceRegion(t, installer, `Section "Uninstall"`, "SectionEnd")
	requireSourceOrder(t, uninstall, "Call un.wiRemovePayload", "Call un.wiOwnsRegistration", "${If} $wiCanManage == 1", "Call un.wiRemoveOwnedNotifications", "DeleteRegKey")
}

func TestNotificationCleanupUsesVerifiedRecordTargetBeforeLegacyFallback(t *testing.T) {
	cleanup := transactionSource(t, "notification-integration.nsh")
	requireSourceOrder(t, cleanup, "${If} $wiCanManage != 1", "Return", "ReadRegStr $1", `"NativeNotificationTarget"`, "${If} ${Errors}", `StrCpy $1 "$wiUninstallSource\${EXE_NAME}"`, "ExecWait", `--remove-owned-notifications "$1"`)
	if strings.Contains(cleanup, "WriteReg") || strings.Contains(cleanup, "DeleteReg") {
		t.Fatal("notification cleanup include mutates registration outside its helper")
	}
}
