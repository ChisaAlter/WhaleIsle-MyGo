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

; Same MUI2 page and resource contract as the original assisted installer.
; Resource provenance is recorded beside these byte-for-byte source assets.
!include "MUI2.nsh"
!define MUI_ICON "branding\icon.ico"
!define MUI_UNICON "branding\icon.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_RIGHT
!define MUI_HEADERIMAGE_BITMAP "branding\installerHeader.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "branding\installerSidebar.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "branding\uninstallerSidebar.bmp"
BrandingText "${PRODUCT_NAME} ${VERSION}"

!define MUI_WELCOMEPAGE_TITLE_3LINES
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "branding\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TITLE_3LINES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE_NAME}"
!define MUI_FINISHPAGE_LINK "github.com/ChisaAlter/Deepseek-Harness-Desktop"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/ChisaAlter/Deepseek-Harness-Desktop"
!insertmacro MUI_PAGE_FINISH

; MUI clears welcome settings after page insertion; set this again for the
; original uninstall title's third line, not just the install welcome title.
!define MUI_WELCOMEPAGE_TITLE_3LINES
!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Original installerLanguages order is zh_CN, en_US. NSIS selects the OS
; language automatically; the original does not request a language dialog.
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "${STAGE_DIR}\${EXE_NAME}"
  ; Required companion, built from the same source. Never launched by NSIS.
  File "${STAGE_DIR}\whaleisle-preview.exe"
  ; Required native terminal ABI: staged and hashed by tools/stage-terminal.
  File "${STAGE_DIR}\ghostty-vt.dll"
  File "${STAGE_DIR}\ghostty-vt.dll.LICENSE"
  File "${STAGE_DIR}\terminal-runtime.json"
  ; Resources are selected at compile time, not by a build-machine path at install time.
  SetOutPath "$INSTDIR\resources"
  File /nonfatal /r "${STAGE_DIR}\resources\*.*"
  SetOutPath "$INSTDIR"
!ifndef SHELL_ONLY
!if /FileExists "${STAGE_DIR}\.runtime-staged"
!else
  !error "Host runtime not staged. Run package.ps1; SHELL_ONLY is for explicit development packages only."
!endif
!if /FileExists "${STAGE_DIR}\runtime\node\node.exe"
!else
  !error "Required Host runtime/node/node.exe is missing."
!endif
  SetOutPath "$INSTDIR\runtime"
  File /r "${STAGE_DIR}\runtime\*.*"
  SetOutPath "$INSTDIR"
!endif

!ifndef SHELL_ONLY
!if /FileExists "${STAGE_DIR}\.pet-render-staged"
!else
  !error "THA4 resources not staged. Run package.ps1 with WHALEISLE_PET_RENDER_ASSETS."
!endif
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
  File "${STAGE_DIR}\pet-render-assets\provenance.json"
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
  ; Distinguish this native product from the legacy Electron app sharing APP_ID.
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeRuntime" "WhaleIsle-MyGo"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\${EXE_NAME}"
  Delete "$INSTDIR\whaleisle-preview.exe"
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

