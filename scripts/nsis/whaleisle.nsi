; Whale Isle NSIS installer (M11).
; Requires staged artifacts from scripts/package.ps1:
;   dist/whaleisle/whaleisle.exe
;   dist/whaleisle/resources/ (optional)
;
; Usage:
;   makensis /DVERSION=0.0.0-dev scripts\nsis\whaleisle.nsi
;
; Side-by-side native install identity separate from Electron appId
; ai.deepseek.harness.gui, defaulting to %LOCALAPPDATA%\Programs\WhaleIsle-MyGo.

!ifndef VERSION
  !define VERSION "0.0.0-dev"
!endif

!ifndef PRODUCT_NAME
  !define PRODUCT_NAME "鲸屿 MyGo"
!endif
!ifndef APP_ID
  !define APP_ID "ai.deepseek.harness.gui.mygo"
!endif
!define EXE_NAME "whaleisle.exe"
!ifndef STAGE_DIR
  !define STAGE_DIR "..\..\dist\whaleisle"
!endif
!ifndef SETUP_OUT
  !define SETUP_OUT "..\..\dist\WhaleIsle-MyGo-Setup-${VERSION}.exe"
!endif

Name "${PRODUCT_NAME}"
OutFile "${SETUP_OUT}"
InstallDir "$LOCALAPPDATA\Programs\WhaleIsle-MyGo"
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
!define MUI_FINISHPAGE_LINK "github.com/ChisaAlter/WhaleIsle-MyGo"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/ChisaAlter/WhaleIsle-MyGo"
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

!include "payload-transaction.nsh"
!include "notification-integration.nsh"

Section "Install"
  Call wiRememberRegistration
  Call wiBeginStage
  !insertmacro WI_SetPayloadOutPath "$INSTDIR"
  File "${STAGE_DIR}\${EXE_NAME}"
  !insertmacro WI_CheckExtract
  ; Required companion, built from the same source. Never launched by NSIS.
  File "${STAGE_DIR}\whaleisle-preview.exe"
  !insertmacro WI_CheckExtract
  ; Required native terminal ABI: staged and hashed by tools/stage-terminal.
  File "${STAGE_DIR}\ghostty-vt.dll"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\ghostty-vt.dll.LICENSE"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\terminal-runtime.json"
  !insertmacro WI_CheckExtract
  ; Resources are selected at compile time, not by a build-machine path at install time.
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\resources"
  File /nonfatal /r "${STAGE_DIR}\resources\*.*"
  !insertmacro WI_CheckExtract
  !insertmacro WI_SetPayloadOutPath "$INSTDIR"
!ifndef SHELL_ONLY
!if /FileExists "${STAGE_DIR}\.runtime-staged"
!else
  !error "Host runtime not staged. Run package.ps1; SHELL_ONLY is for explicit development packages only."
!endif
!if /FileExists "${STAGE_DIR}\runtime\node\node.exe"
!else
  !error "Required Host runtime/node/node.exe is missing."
!endif
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\runtime"
!ifdef RUNTIME_INCLUDE
  !include "${RUNTIME_INCLUDE}"
!else
!if /FileExists "${STAGE_DIR}\runtime\vendor\runtime-graph.json"
  !error "Closed runtime graph requires the generated RUNTIME_INCLUDE; do not recurse through dependency aliases."
!endif
  File /r "${STAGE_DIR}\runtime\*.*"
  !insertmacro WI_CheckExtract
!endif
  !insertmacro WI_SetPayloadOutPath "$INSTDIR"
!endif

!ifndef SHELL_ONLY
!if /FileExists "${STAGE_DIR}\.pet-render-staged"
!else
  !error "THA4 resources not staged. Run package.ps1 with WHALEISLE_PET_RENDER_ASSETS."
!endif
  ; Exact manifest selection only; do not package arbitrary sidecar files.
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\pet-render-assets\avatar"
  File "${STAGE_DIR}\pet-render-assets\avatar\character.png"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\avatar\model.onnx"
  !insertmacro WI_CheckExtract
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\pet-render-assets\ort"
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.jsep.mjs"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.jsep.wasm"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.mjs"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\ort\ort-wasm-simd-threaded.wasm"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\ort\ort.all.min.js"
  !insertmacro WI_CheckExtract
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\pet-render-assets"
  File "${STAGE_DIR}\pet-render-assets\LICENSE"
  !insertmacro WI_CheckExtract
  File "${STAGE_DIR}\pet-render-assets\provenance.json"
  !insertmacro WI_CheckExtract
  !insertmacro WI_SetPayloadOutPath "$INSTDIR\pet-render-assets\ort"
  File "${STAGE_DIR}\pet-render-assets\ort\LICENSE"
  !insertmacro WI_CheckExtract
  !insertmacro WI_SetPayloadOutPath "$INSTDIR"
!endif

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  !insertmacro WI_CheckExtract
  Call wiCreateManifest
  ${If} $wiError != 0
    SetErrorLevel 2
    Abort "Cannot create the application ownership manifest. The previous installation was retained."
  ${EndIf}
  Call wiPromote
  ClearErrors
  StrCpy $wiRegistryTouched "1"
  WriteRegStr HKCU "Software\${APP_ID}" "InstallLocation" "$INSTDIR"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayName" "${PRODUCT_NAME}"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayVersion" "${VERSION}"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "Publisher" "WhaleIsle"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayIcon" "$INSTDIR\${EXE_NAME},0"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation" "$INSTDIR"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  !insertmacro WI_CheckExtract
  ; Distinguish this native product from the legacy Electron app sharing APP_ID.
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeInstallID" "$wiInstallID"
  !insertmacro WI_CheckExtract
  ; Retry cleanup may move the payload. Keep its original notification target
  ; with the same installation identity instead of deriving it from the retry path.
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeNotificationTarget" "$wiFinal\${EXE_NAME}"
  !insertmacro WI_CheckExtract
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NativeRuntime" "WhaleIsle-MyGo"
  !insertmacro WI_CheckExtract

  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  !insertmacro WI_CheckExtract
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  !insertmacro WI_CheckExtract
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  !insertmacro WI_CheckExtract
  Call wiCommitInstallation
SectionEnd

Section "Uninstall"
  ; Do not remove registration or shortcuts unless owned-payload cleanup
  ; succeeds. Failed cleanup retargets the retained record to its retry path.
  Call un.wiRemovePayload
  Call un.wiOwnsRegistration
  ${If} $wiCanManage == 1
  Call un.wiRemoveOwnedNotifications
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"
  DeleteRegKey HKCU "Software\${APP_ID}"
  ${EndIf}
SectionEnd
