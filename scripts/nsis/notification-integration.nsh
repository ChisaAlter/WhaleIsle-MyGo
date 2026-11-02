; Owned Go notification cleanup. The helper is a trusted standalone windowsgui
; executable embedded in this uninstaller, not a previously installed desktop.
!ifndef WI_NOTIFICATION_INTEGRATION
!define WI_NOTIFICATION_INTEGRATION

Function un.wiRemoveOwnedNotifications
  ; A matching manifest InstallID/runtime/current location must already own the
  ; uninstall record. Refuse use of a target field from another installation.
  ${If} $wiCanManage != 1
    Return
  ${EndIf}
  Push $1
  ClearErrors
  ReadRegStr $1 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeNotificationTarget"
  ${If} ${Errors}
    ; Old installers have no field. Their normal original path remains safe;
    ; a legacy retired retry intentionally does not infer any COM command.
    StrCpy $1 "$wiUninstallSource\${EXE_NAME}"
  ${EndIf}
  InitPluginsDir
  SetOutPath "$PLUGINSDIR"
  File /oname=whaleisle-integration.exe "${STAGE_DIR}\whaleisle-integration.exe"
  ; The helper preflights exact CLSID command and link target+AppID. Unknown or
  ; user-customized entries are preserved. No shared Programs directory removal.
  ClearErrors
  ; Retired retry locations never replace the original notification target.
  ; The helper validates absolute path/basename and then exact owned objects.
  ExecWait '"$PLUGINSDIR\whaleisle-integration.exe" --remove-owned-notifications "$1"' $0
  ${If} ${Errors}
    DetailPrint "Go notification registration cleanup helper could not run."
  ${ElseIf} $0 != 0
    DetailPrint "Go notification registration cleanup failed; preserved unmatched entries."
  ${EndIf}
  Pop $1
FunctionEnd

!endif
