; Whale Isle NSIS installer (M11).
; Requires staged artifacts from scripts/package.ps1:
;   dist/whaleisle/whaleisle.exe
;   dist/whaleisle/resources/ (optional)
;
; Usage:
;   makensis /DVERSION=0.0.0-dev scripts\nsis\whaleisle.nsi
;
; Cover-upgrade target matches Electron appId ai.deepseek.harness.gui
; default install dir under %LOCALAPPDATA%\Programs\Deepseek-Harness-Desktop.

!ifndef VERSION
  !define VERSION "0.0.0-dev"
!endif

!define PRODUCT_NAME "Whale Isle"
!define APP_ID "ai.deepseek.harness.gui"
!define EXE_NAME "whaleisle.exe"
!define STAGE_DIR "..\..\dist\whaleisle"

Name "${PRODUCT_NAME}"
OutFile "..\..\dist\WhaleIsle-Setup-${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Programs\Deepseek-Harness-Desktop"
InstallDirRegKey HKCU "Software\${APP_ID}" "InstallLocation"
RequestExecutionLevel user
Unicode true
SetCompressor /SOLID lzma

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File "${STAGE_DIR}\${EXE_NAME}"
  IfFileExists "${STAGE_DIR}\resources" stage_resources skip_resources
stage_resources:
  CreateDirectory "$INSTDIR\resources"
  File /r "${STAGE_DIR}\resources\*.*"
skip_resources:
!if /FileExists "${STAGE_DIR}\.runtime-staged"
  CreateDirectory "$INSTDIR\runtime"
  File /r "${STAGE_DIR}\runtime\*.*"
!endif

  WriteRegStr HKCU "Software\${APP_ID}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "Publisher" "DeepSeek"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\${EXE_NAME}"
  RMDir /r "$INSTDIR\resources"
  RMDir /r "$INSTDIR\runtime"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"
  DeleteRegKey HKCU "Software\${APP_ID}"
SectionEnd
