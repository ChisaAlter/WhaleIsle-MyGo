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
  ; Required native terminal ABI: staged and hashed by tools/stage-terminal.
  File "${STAGE_DIR}\ghostty-vt.dll"
  File "${STAGE_DIR}\ghostty-vt.dll.LICENSE"
  File "${STAGE_DIR}\terminal-runtime.json"
  IfFileExists "${STAGE_DIR}\resources" stage_resources skip_resources
stage_resources:
  CreateDirectory "$INSTDIR\resources"
  File /r "${STAGE_DIR}\resources\*.*"
skip_resources:
!if /FileExists "${STAGE_DIR}\.runtime-staged"
  CreateDirectory "$INSTDIR\runtime"
  File /r "${STAGE_DIR}\runtime\*.*"
!endif

!if /FileExists "${STAGE_DIR}\.pet-render-staged"
  ; Exact manifest selection only; do not package arbitrary sidecar files.
  SetOutPath "$INSTDIR\pet-render-assets\avatar"
  File "${STAGE_DIR}\pet-render-assets\avatar\character.png"
  File "${STAGE_DIR}\pet-render-assets\avatar\model.onnx"
  SetOutPath "$INSTDIR\pet-render-assets\ort"
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.jsep.mjs"
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.jsep.wasm"
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.mjs"
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.wasm"
  File "${STAGE_DIR}\pet-render-assets\ort\ort.all.min.js"
  SetOutPath "$INSTDIR\pet-render-assets"
  File "${STAGE_DIR}\pet-render-assets\LICENSE"
  SetOutPath "$INSTDIR\pet-render-assets\ort"
  File "${STAGE_DIR}\pet-render-assets\ort\LICENSE"
  SetOutPath "$INSTDIR"
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
  Delete "$INSTDIR\ghostty-vt.dll"
  Delete "$INSTDIR\ghostty-vt.dll.LICENSE"
  Delete "$INSTDIR\terminal-runtime.json"
  RMDir /r "$INSTDIR\resources"
  RMDir /r "$INSTDIR\runtime"
  RMDir /r "$INSTDIR\pet-render-assets"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"
  DeleteRegKey HKCU "Software\${APP_ID}"
SectionEnd
