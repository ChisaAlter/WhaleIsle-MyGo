; Installation payload ownership and directory transaction helpers.
; Only fresh stage trees may be traversed. Existing trees are cleaned exclusively
; through the manifest written from that stage, never with RMDir /r.
!include "LogicLib.nsh"
!include "FileFunc.nsh"

!define WI_MANIFEST ".whaleisle-owned-v1"
Var wiFinal
Var wiFinalFS
Var wiStage
Var wiStageFS
Var wiOld
Var wiOldFS
Var wiOldMoved
Var wiNewMoved
Var wiCommitted
Var wiRecoveryStopped
Var wiRoot
Var wiRootFS
Var wiError
Var wiManifestHandle
Var wiRetired
Var wiOldLocation
Var wiOldUninstall
Var wiOldDisplay
Var wiOldVersion
Var wiOldPublisher
Var wiOldNative
Var wiOldProductLocation
Var wiRegistryTouched
Var wiHadDesktopLink
Var wiHadMenuLink

!macro WI_CheckExtract
  ${If} ${Errors}
    SetErrorLevel 2
    Abort "Installation could not complete. Transaction recovery will retain the previous installation or report its backup path."
  ${EndIf}
!macroend

; Match the original installer's long-path handling, including UNC shares.
!macro WI_LongPath Source Destination Scratch
  StrCpy ${Destination} "${Source}"
  StrCpy ${Scratch} ${Destination} 4
  ${If} ${Scratch} != "\\?\"
    StrCpy ${Scratch} ${Destination} 2
    ${If} ${Scratch} == "\\"
      StrCpy ${Destination} ${Destination} "" 2
      StrCpy ${Destination} "\\?\UNC\${Destination}"
    ${Else}
      StrCpy ${Destination} "\\?\${Destination}"
    ${EndIf}
  ${EndIf}
!macroend
; Relative paths may name only installer-owned payload roots. Reject traversal
; and every linked ancestor before touching an existing installation file.
!macro WI_DefineOwnershipFunctions Prefix
Function ${Prefix}wiCheckRoot
  Push $R0
  Push $R1
  Push $R2
  StrCpy $R0 $wiRoot 8
  ${If} $R0 == "\\?\UNC\"
    StrCpy $wiRoot $wiRoot "" 8
    StrCpy $wiRoot "\\$wiRoot"
  ${Else}
    StrCpy $R0 $wiRoot 4
    ${If} $R0 == "\\?\"
      StrCpy $wiRoot $wiRoot "" 4
    ${EndIf}
  ${EndIf}
  StrCpy $R0 $wiRoot 2
  StrCpy $R1 $wiRoot 2 1
  ${If} $R0 != "\\"
  ${AndIf} $R1 != ":\"
    StrCpy $wiError 1
    Goto wi_root_done
  ${EndIf}
  GetFullPathName $wiRoot "$wiRoot"
  ; Normalize a trailing separator before deriving sibling transaction paths.
  wi_root_trim:
    StrLen $R0 $wiRoot
    ${If} $R0 > 3
      StrCpy $R1 $wiRoot 1 -1
      ${If} $R1 == "\"
        StrCpy $wiRoot $wiRoot -1
        Goto wi_root_trim
      ${EndIf}
    ${EndIf}
    ${If} $R0 < 4
      StrCpy $wiError 1
      Goto wi_root_done
    ${EndIf}
  StrCpy $R0 $wiRoot 2
  ${If} $R0 == "\\"
    ; A UNC share itself is a volume root, not an application directory.
    StrCpy $R0 2
    StrCpy $R2 0
    wi_unc_next:
      StrCpy $R1 $wiRoot 1 $R0
      ${If} $R1 == ""
        ${If} $R2 < 2
          StrCpy $wiError 1
          Goto wi_root_done
        ${EndIf}
        Goto wi_unc_done
      ${EndIf}
      ${If} $R1 == "\"
        IntOp $R2 $R2 + 1
      ${EndIf}
      IntOp $R0 $R0 + 1
      Goto wi_unc_next
    wi_unc_done:
  ${EndIf}
  ${GetParent} "$wiRoot" $R0
  ${If} $R0 == ""
    StrCpy $wiError 1
    Goto wi_root_done
  ${EndIf}
  StrCpy $R0 $wiRoot
  wi_root_next:
    !insertmacro WI_LongPath "$R0" $R1 $R2
    System::Call 'kernel32::GetFileAttributesW(w "$R1") i.r11'
    ${If} $R1 != -1
      IntOp $R1 $R1 & 0x410
      ${If} $R1 != 0x10
        StrCpy $wiError 1
        Goto wi_root_done
      ${EndIf}
    ${Else}
      System::Call 'kernel32::GetLastError() i.r11'
      ${If} $R1 != 2
      ${AndIf} $R1 != 3
        StrCpy $wiError 1
        Goto wi_root_done
      ${EndIf}
    ${EndIf}
    ${GetParent} "$R0" $R2
    ${If} $R2 == ""
    ${OrIf} $R2 == $R0
      Goto wi_root_done
    ${EndIf}
    StrCpy $R0 $R2
    Goto wi_root_next
  wi_root_done:
    !insertmacro WI_LongPath "$wiRoot" $wiRootFS $R2
    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd

Function ${Prefix}wiOwnedPath
  Exch $R0
  Push $R1
  Push $R2
  Push $R3
  Push $R4
  Push $R5
  StrCpy $R1 0
  StrCpy $R2 ""
  StrCpy $R4 "$wiRootFS"
  StrCpy $R5 ""
  wi_path_next:
    StrCpy $R3 $R0 1 $R1
    ${If} $R3 == ":"
    ${OrIf} $R3 == "/"
    ${OrIf} $R3 == "*"
    ${OrIf} $R3 == "?"
      Goto wi_path_bad
    ${EndIf}
    ${If} $R3 == "\"
    ${OrIf} $R3 == ""
      ${If} $R2 == ""
      ${OrIf} $R2 == "."
      ${OrIf} $R2 == ".."
        Goto wi_path_bad
      ${EndIf}
      ${If} $R5 == ""
        StrCpy $R5 $R2
        ${If} $R2 != "resources"
        ${AndIf} $R2 != "runtime"
        ${AndIf} $R2 != "pet-render-assets"
        ${AndIf} $R2 != "whaleisle.exe"
        ${AndIf} $R2 != "whaleisle-preview.exe"
        ${AndIf} $R2 != "ghostty-vt.dll"
        ${AndIf} $R2 != "ghostty-vt.dll.LICENSE"
        ${AndIf} $R2 != "terminal-runtime.json"
        ${AndIf} $R2 != "Uninstall.exe"
          Goto wi_path_bad
        ${EndIf}
        ${If} $R3 == "\"
          ${If} $R2 != "resources"
          ${AndIf} $R2 != "runtime"
          ${AndIf} $R2 != "pet-render-assets"
            Goto wi_path_bad
          ${EndIf}
        ${EndIf}
      ${EndIf}
      StrCpy $R4 "$R4\$R2"
      System::Call 'kernel32::GetFileAttributesW(w "$R4") i.r12'
      ${If} $R2 != -1
        IntOp $R2 $R2 & 0x400
        ${If} $R2 != 0
          Goto wi_path_bad
        ${EndIf}
      ${Else}
        System::Call 'kernel32::GetLastError() i.r12'
        ${If} $R2 != 2
        ${AndIf} $R2 != 3
          Goto wi_path_bad
        ${EndIf}
      ${EndIf}
      StrCpy $R2 ""
      ${If} $R3 == ""
        Goto wi_path_done
      ${EndIf}
    ${Else}
      StrCpy $R2 "$R2$R3"
    ${EndIf}
    IntOp $R1 $R1 + 1
    Goto wi_path_next
  wi_path_bad:
    StrCpy $wiError 1
  wi_path_done:
    Pop $R5
    Pop $R4
    Pop $R3
    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd

; The manifest is read twice: validate every record before deleting any file.
; Directory records are post-order, so RMDir removes only empty directories.
; Uninstall.exe and the manifest remain available until the caller commits.
Function ${Prefix}wiCleanOwnedPayload
  Push $R0
  Push $R1
  Push $R2
  Push $R3
  Push $R4
  Push $R5
  StrCpy $wiError 0
  Call ${Prefix}wiCheckRoot
  ${If} $wiError != 0
    Goto wi_clean_bad
  ${EndIf}
  System::Call 'kernel32::GetFileAttributesW(w "$wiRootFS") i.r10'
  ${If} $R0 == -1
    Goto wi_clean_bad
  ${EndIf}
  IntOp $R0 $R0 & 0x410
  ${If} $R0 != 0x10
    Goto wi_clean_bad
  ${EndIf}
  System::Call 'kernel32::GetFileAttributesW(w "$wiRootFS\${WI_MANIFEST}") i.r10'
  ${If} $R0 == -1
    Goto wi_clean_bad
  ${EndIf}
  IntOp $R0 $R0 & 0x410
  ${If} $R0 != 0
    Goto wi_clean_bad
  ${EndIf}
  ClearErrors
  FileOpen $R1 "$wiRootFS\${WI_MANIFEST}" r
  ${If} ${Errors}
    Goto wi_clean_bad
  ${EndIf}
  StrCpy $R5 0
  wi_clean_pass:
    FileSeek $R1 0 SET
    ClearErrors
    FileReadUTF16LE $R1 $R2
    ${If} $R2 != "WhaleIsle-Payload-1$\r$\n"
      Goto wi_clean_close_bad
    ${EndIf}
  wi_clean_next:
    ClearErrors
    FileReadUTF16LE $R1 $R2
    ${If} ${Errors}
      Goto wi_clean_end_pass
    ${EndIf}
    StrCpy $R3 $R2 2
    ${If} $R3 != "F|"
    ${AndIf} $R3 != "D|"
      Goto wi_clean_close_bad
    ${EndIf}
    StrLen $R0 $R2
    ${If} $R0 < 5
      Goto wi_clean_close_bad
    ${EndIf}
    StrCpy $R4 $R2 2 -2
    ${If} $R4 != "$\r$\n"
      Goto wi_clean_close_bad
    ${EndIf}
    IntOp $R0 $R0 - 4
    StrCpy $R2 $R2 $R0 2
    Push $R2
    Call ${Prefix}wiOwnedPath
    ${If} $wiError != 0
      Goto wi_clean_close_bad
    ${EndIf}
    ${If} $R5 == 0
      Goto wi_clean_next
    ${EndIf}
    ${If} $R2 == "Uninstall.exe"
      Goto wi_clean_next
    ${EndIf}
    System::Call 'kernel32::GetFileAttributesW(w "$wiRootFS\$R2") i.r10'
    ${If} $R0 == -1
      System::Call 'kernel32::GetLastError() i.r10'
      ${If} $R0 == 2
      ${OrIf} $R0 == 3
        Goto wi_clean_next
      ${EndIf}
      Goto wi_clean_close_bad
    ${EndIf}
    IntOp $R0 $R0 & 0x10
    ClearErrors
    ${If} $R3 == "D|"
      ${If} $R0 != 0x10
        Goto wi_clean_close_bad
      ${EndIf}
      RMDir "$wiRootFS\$R2"
    ${Else}
      ${If} $R0 != 0
        Goto wi_clean_close_bad
      ${EndIf}
      Delete "$wiRootFS\$R2"
    ${EndIf}
    ${If} ${Errors}
      Goto wi_clean_close_bad
    ${EndIf}
    Goto wi_clean_next
  wi_clean_end_pass:
    ${If} $R5 == 0
      StrCpy $R5 1
      Goto wi_clean_pass
    ${EndIf}
    FileClose $R1
    ; Unknown files are never removed and keep the installation manageable.
    ClearErrors
    FindFirst $R1 $R2 "$wiRootFS\*.*"
    ${If} ${Errors}
      Goto wi_clean_bad
    ${EndIf}
  wi_clean_remaining:
    ${If} $R2 == ""
      FindClose $R1
      Goto wi_clean_done
    ${EndIf}
    ${If} $R2 != "."
    ${AndIf} $R2 != ".."
    ${AndIf} $R2 != "${WI_MANIFEST}"
    ${AndIf} $R2 != "Uninstall.exe"
      FindClose $R1
      Goto wi_clean_bad
    ${EndIf}
    FindNext $R1 $R2
    Goto wi_clean_remaining
  wi_clean_close_bad:
    FileClose $R1
  wi_clean_bad:
    StrCpy $wiError 1
  wi_clean_done:
    Pop $R5
    Pop $R4
    Pop $R3
    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd
!macroend

!insertmacro WI_DefineOwnershipFunctions ""
!insertmacro WI_DefineOwnershipFunctions "un."

; Only called against the unique stage freshly created by this installer.
; Refuse reparse points even there, rather than following their targets.
Function wiWriteStageManifestTree
  Exch $R0
  Push $R1
  Push $R2
  Push $R3
  Push $R4
  StrCpy $R4 $R0
  ${If} $R4 != ""
    StrCpy $R4 "$R4\"
  ${EndIf}
  ClearErrors
  FindFirst $R1 $R2 "$wiStageFS\$R4*.*"
  ${If} ${Errors}
    StrCpy $wiError 1
    Goto wi_manifest_done
  ${EndIf}
  wi_manifest_next:
    ${If} $R2 == ""
      Goto wi_manifest_close
    ${EndIf}
    ${If} $R2 != "."
    ${AndIf} $R2 != ".."
    ${AndIf} $R2 != "${WI_MANIFEST}"
      StrCpy $R3 "$R4$R2"
      Push $R3
      Call wiOwnedPath
      ${If} $wiError != 0
        Goto wi_manifest_close
      ${EndIf}
      System::Call 'kernel32::GetFileAttributesW(w "$wiStageFS\$R3") i.r12'
      ${If} $R2 == -1
        StrCpy $wiError 1
        Goto wi_manifest_close
      ${EndIf}
      IntOp $R2 $R2 & 0x10
      ${If} $R2 == 0x10
        Push $R3
        Call wiWriteStageManifestTree
        ${If} $wiError != 0
          Goto wi_manifest_close
        ${EndIf}
        ClearErrors
        FileWriteUTF16LE $wiManifestHandle "D|$R3$\r$\n"
      ${Else}
        ClearErrors
        FileWriteUTF16LE $wiManifestHandle "F|$R3$\r$\n"
      ${EndIf}
      ${If} ${Errors}
        StrCpy $wiError 1
        Goto wi_manifest_close
      ${EndIf}
    ${EndIf}
    FindNext $R1 $R2
    Goto wi_manifest_next
  wi_manifest_close:
    FindClose $R1
  wi_manifest_done:
    Pop $R4
    Pop $R3
    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd

Function wiCreateManifest
  StrCpy $wiRoot $wiStage
  StrCpy $wiRootFS $wiStageFS
  StrCpy $wiError 0
  ClearErrors
  FileOpen $wiManifestHandle "$wiStageFS\${WI_MANIFEST}" w
  ${If} ${Errors}
    StrCpy $wiError 1
    Return
  ${EndIf}
  FileWriteUTF16LE $wiManifestHandle "WhaleIsle-Payload-1$\r$\n"
  ${If} ${Errors}
    StrCpy $wiError 1
  ${Else}
    Push ""
    Call wiWriteStageManifestTree
  ${EndIf}
  ClearErrors
  FileClose $wiManifestHandle
  ${If} ${Errors}
    StrCpy $wiError 1
  ${EndIf}
FunctionEnd

; Rename-based rollback never recursively removes the previous installation.
Function wiRollback
  SetOutPath $TEMP
  ${If} $wiCommitted == "1"
  ${OrIf} $wiRecoveryStopped == "1"
    Return
  ${EndIf}
  ${If} $wiNewMoved == "1"
    ClearErrors
    Rename "$wiFinalFS" "$wiStageFS"
    ${If} ${Errors}
      StrCpy $wiRecoveryStopped "1"
      MessageBox MB_OK|MB_ICONEXCLAMATION "Cannot move the incomplete new application aside. Previous installation backup: $wiOld. Nothing was deleted." /SD IDOK
      SetErrorLevel 2
      Return
    ${EndIf}
    StrCpy $wiNewMoved ""
  ${EndIf}
  ${If} $wiOldMoved == "1"
    ClearErrors
    Rename "$wiOldFS" "$wiFinalFS"
    ${If} ${Errors}
      StrCpy $wiRecoveryStopped "1"
      MessageBox MB_OK|MB_ICONEXCLAMATION "Cannot restore the previous application. Complete backup: $wiOld. Close programs using the directories and rename the backup to $wiFinal." /SD IDOK
      SetErrorLevel 2
      Return
    ${EndIf}
    StrCpy $wiOldMoved ""
  ${EndIf}
  StrCpy $INSTDIR $wiFinal
  ; A failed extraction may have no complete manifest. Preserve that unique
  ; stage instead of inventing ownership from a partially written listing.
  ${If} $wiStage != ""
  ${AndIf} ${FileExists} "$wiStageFS\*.*"
    DetailPrint "Uncommitted stage retained: $wiStage"
  ${EndIf}
FunctionEnd

Function wiBeginStage
  StrCpy $wiFinal $INSTDIR
  StrCpy $wiOldMoved ""
  StrCpy $wiNewMoved ""
  StrCpy $wiCommitted ""
  StrCpy $wiRecoveryStopped ""
  StrCpy $wiRegistryTouched ""
  StrCpy $wiRoot $wiFinal
  StrCpy $wiError 0
  Call wiCheckRoot
  ${If} $wiError != 0
    Abort "Choose an absolute, unlinked application directory beneath a filesystem root."
  ${EndIf}
  StrCpy $wiFinal $wiRoot
  !insertmacro WI_LongPath "$wiFinal" $wiFinalFS $1
  ${If} $wiFinal == $PROFILE
  ${OrIf} $wiFinal == $LOCALAPPDATA
  ${OrIf} $wiFinal == $APPDATA
  ${OrIf} $wiFinal == $PROGRAMFILES
  ${OrIf} $wiFinal == $WINDIR
  ${OrIf} $wiFinal == $SYSDIR
    Abort "Choose a dedicated application directory, not a system or user-data directory."
  ${EndIf}
  ; Never move a nonempty unrelated directory. Legacy product backups without
  ; an ownership manifest are kept after upgrade, never guessed or traversed.
  StrCpy $2 0
  ClearErrors
  FindFirst $0 $1 "$wiFinalFS\*.*"
  ${If} ${Errors}
    System::Call 'kernel32::GetFileAttributesW(w "$wiFinalFS") i.r10'
    ${If} $R0 != -1
      Abort "Cannot inspect the previous installation directory. No existing files were moved or removed."
    ${EndIf}
    System::Call 'kernel32::GetLastError() i.r10'
    ${If} $R0 != 2
    ${AndIf} $R0 != 3
      Abort "Cannot inspect the installation path. No existing files were moved or removed."
    ${EndIf}
  ${Else}
    wi_existing_next:
      ${If} $1 != ""
        ${If} $1 != "."
        ${AndIf} $1 != ".."
          StrCpy $2 1
        ${EndIf}
        FindNext $0 $1
        Goto wi_existing_next
      ${EndIf}
    FindClose $0
  ${EndIf}
  ${If} $2 == 1
    ${IfNot} ${FileExists} "$wiFinalFS\whaleisle.exe"
    ${AndIfNot} ${FileExists} "$wiFinalFS\Whale Isle.exe"
    ${AndIfNot} ${FileExists} "$wiFinalFS\Deepseek-Harness-Desktop.exe"
    ${AndIfNot} ${FileExists} "$wiFinalFS\Whale Isle Launcher.exe"
    ${AndIfNot} ${FileExists} "$wiFinalFS\Deepseek-Harness-Launcher.exe"
      Abort "This directory contains unrelated files. Choose a dedicated application directory."
    ${EndIf}
  ${EndIf}
  System::Call 'ole32::CoCreateGuid(g .r0) i.r1'
  ${If} $1 != 0
    Abort "Cannot allocate the installation transaction."
  ${EndIf}
  StrCpy $wiStage "$wiFinal.new-$0"
  StrCpy $wiOld "$wiFinal.old-$0"
  !insertmacro WI_LongPath "$wiFinal" $wiFinalFS $1
  !insertmacro WI_LongPath "$wiStage" $wiStageFS $1
  !insertmacro WI_LongPath "$wiOld" $wiOldFS $1
  ${If} ${FileExists} "$wiStage"
  ${OrIf} ${FileExists} "$wiOld"
    Abort "The installation transaction path already exists."
  ${EndIf}
  ClearErrors
  CreateDirectory "$wiStageFS"
  !insertmacro WI_CheckExtract
  StrCpy $INSTDIR $wiStageFS
FunctionEnd

Function wiPromote
  SetOutPath $TEMP
  ClearErrors
  System::Call 'kernel32::GetFileAttributesW(w "$wiFinalFS") i.r0'
  ${If} $0 != -1
    Rename "$wiFinalFS" "$wiOldFS"
    ${If} ${Errors}
      SetErrorLevel 2
      Abort "Cannot move the previous application. Close it and retry; its files and registration were retained."
    ${EndIf}
    StrCpy $wiOldMoved "1"
  ${EndIf}
  ClearErrors
  Rename "$wiStageFS" "$wiFinalFS"
  ${If} ${Errors}
    SetErrorLevel 2
    Abort "Cannot promote the staged application. The previous installation will be restored."
  ${EndIf}
  StrCpy $wiNewMoved "1"
  StrCpy $INSTDIR $wiFinal
FunctionEnd



!macro WI_RestoreValue Key Name Value
  ClearErrors
  ${If} ${Value} == ""
    DeleteRegValue HKCU "${Key}" "${Name}"
  ${Else}
    WriteRegStr HKCU "${Key}" "${Name}" "${Value}"
  ${EndIf}
  ${If} ${Errors}
    StrCpy $wiError 1
  ${EndIf}
!macroend

Function wiRememberRegistration
  InitPluginsDir
  ReadRegStr $wiOldProductLocation HKCU "Software\${APP_ID}" "InstallLocation"
  ReadRegStr $wiOldLocation HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation"
  ReadRegStr $wiOldUninstall HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString"
  ReadRegStr $wiOldDisplay HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayName"
  ReadRegStr $wiOldVersion HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayVersion"
  ReadRegStr $wiOldPublisher HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "Publisher"
  ReadRegStr $wiOldNative HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeRuntime"
  StrCpy $wiHadDesktopLink ""
  StrCpy $wiHadMenuLink ""
  ${If} ${FileExists} "$DESKTOP\${PRODUCT_NAME}.lnk"
    ClearErrors
    CopyFiles /SILENT "$DESKTOP\${PRODUCT_NAME}.lnk" "$PLUGINSDIR\desktop-before.lnk"
    !insertmacro WI_CheckExtract
    StrCpy $wiHadDesktopLink "1"
  ${EndIf}
  ${If} ${FileExists} "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    ClearErrors
    CopyFiles /SILENT "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$PLUGINSDIR\menu-before.lnk"
    !insertmacro WI_CheckExtract
    StrCpy $wiHadMenuLink "1"
  ${EndIf}
FunctionEnd

Function wiRestoreRegistration
  ${If} $wiRegistryTouched != "1"
    Return
  ${EndIf}
  StrCpy $wiError 0
  !insertmacro WI_RestoreValue "Software\${APP_ID}" "InstallLocation" $wiOldProductLocation
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation" $wiOldLocation
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString" $wiOldUninstall
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayName" $wiOldDisplay
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayVersion" $wiOldVersion
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "Publisher" $wiOldPublisher
  !insertmacro WI_RestoreValue "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeRuntime" $wiOldNative
  DeleteRegKey /ifempty HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"
  DeleteRegKey /ifempty HKCU "Software\${APP_ID}"
  ${If} $wiHadDesktopLink == "1"
    ClearErrors
    CopyFiles /SILENT "$PLUGINSDIR\desktop-before.lnk" "$DESKTOP\${PRODUCT_NAME}.lnk"
    ${If} ${Errors}
      StrCpy $wiError 1
    ${EndIf}
  ${Else}
    Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  ${EndIf}
  ${If} $wiHadMenuLink == "1"
    ClearErrors
    CopyFiles /SILENT "$PLUGINSDIR\menu-before.lnk" "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    ${If} ${Errors}
      StrCpy $wiError 1
    ${EndIf}
  ${Else}
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  ${EndIf}
  ${If} $wiError != 0
    MessageBox MB_OK|MB_ICONEXCLAMATION "Installation failed and the previous registration could not be completely restored. Previous application backup (if present): $wiOld. No user data was deleted." /SD IDOK
  ${EndIf}
  StrCpy $wiRegistryTouched ""
FunctionEnd

Function wiCommitInstallation
  ; Promotion and registration have committed. Failure to retire old files must
  ; not roll back a new working installation or erase unknown user content.
  StrCpy $wiCommitted "1"
  ${If} $wiOldMoved == "1"
    StrCpy $wiOldMoved ""
    StrCpy $wiRoot $wiOld
    Call wiCleanOwnedPayload
    ${If} $wiError == 0
      ClearErrors
      Delete "$wiOldFS\Uninstall.exe"
      Delete "$wiOldFS\${WI_MANIFEST}"
      RMDir "$wiOldFS"
      ${If} ${Errors}
        StrCpy $wiError 1
      ${EndIf}
    ${EndIf}
    ${If} $wiError != 0
      DetailPrint "New application installed. Previous/unknown files retained: $wiOld"
      MessageBox MB_OK|MB_ICONEXCLAMATION "The new application is installed. Previous files could not be fully retired, or have no ownership manifest. They were retained at $wiOld. User files were not removed." /SD IDOK
    ${EndIf}
  ${EndIf}
FunctionEnd

Function .onInstFailed
  Call wiRollback
  Call wiRestoreRegistration
FunctionEnd

Function .onGUIEnd
  ${If} $wiCommitted != "1"
    Call wiRollback
  ${EndIf}
FunctionEnd

Function un.wiRetainRetry
  ; Cleanup may already have deleted some owned files. Do not promise a full
  ; restore. Keep management at the actual retired location for another attempt.
  CopyFiles /SILENT "$PLUGINSDIR\retry-uninstall.exe" "$wiRootFS\Uninstall.exe"
  CopyFiles /SILENT "$PLUGINSDIR\retry-manifest" "$wiRootFS\${WI_MANIFEST}"
  ClearErrors
  WriteRegStr HKCU "Software\${APP_ID}" "InstallLocation" "$wiRoot"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation" "$wiRoot"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString" '"$wiRoot\Uninstall.exe"'
  ${If} ${Errors}
    DetailPrint "Could not update the uninstall record; retry directly from $wiRoot\Uninstall.exe"
  ${EndIf}
  SetErrorLevel 2
  Abort "Uninstall incomplete. Some application files may already be removed; unknown/locked files were retained at $wiRoot. Registration and shortcuts were kept. Close processes or move your own remaining files aside, then retry $wiRoot\Uninstall.exe."
FunctionEnd

Function un.wiRemovePayload
  InitPluginsDir
  SetOutPath $TEMP
  StrCpy $wiRoot $INSTDIR
  StrCpy $wiError 0
  Call un.wiCheckRoot
  ${If} $wiError != 0
    SetErrorLevel 2
    Abort "Cannot uninstall from a linked or invalid directory. Registration and files were kept."
  ${EndIf}
  ClearErrors
  CopyFiles /SILENT "$wiRootFS\Uninstall.exe" "$PLUGINSDIR\retry-uninstall.exe"
  ${If} ${Errors}
    SetErrorLevel 2
    Abort "Cannot preserve the uninstall retry program. No files or registration were removed."
  ${EndIf}
  CopyFiles /SILENT "$wiRootFS\${WI_MANIFEST}" "$PLUGINSDIR\retry-manifest"
  ${If} ${Errors}
    SetErrorLevel 2
    Abort "The application ownership manifest is missing or unreadable. No files or registration were removed."
  ${EndIf}
  System::Call 'ole32::CoCreateGuid(g .r0) i.r1'
  ${If} $1 != 0
    SetErrorLevel 2
    Abort "Cannot allocate the uninstall transaction. No files or registration were removed."
  ${EndIf}
  StrCpy $wiRetired "$wiRoot.uninstall-$0"
  ClearErrors
  !insertmacro WI_LongPath "$wiRetired" $0 $1
  Rename "$wiRootFS" "$0"
  ${If} ${Errors}
    SetErrorLevel 2
    Abort "Cannot move the application directory. Close the application and retry. Files and registration were kept."
  ${EndIf}
  StrCpy $wiRoot $wiRetired
  Call un.wiCleanOwnedPayload
  ${If} $wiError != 0
    Call un.wiRetainRetry
  ${EndIf}
  ClearErrors
  Delete "$wiRootFS\Uninstall.exe"
  ${If} ${Errors}
    Call un.wiRetainRetry
  ${EndIf}
  Delete "$wiRootFS\${WI_MANIFEST}"
  ${If} ${Errors}
    Call un.wiRetainRetry
  ${EndIf}
  RMDir "$wiRootFS"
  ${If} ${Errors}
    Call un.wiRetainRetry
  ${EndIf}
  ClearErrors
FunctionEnd






