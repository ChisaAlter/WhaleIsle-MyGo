; Directory junctions are restored only after the stage has its final name.
; The embedded windowsgui helper validates every declaration and physical target
; before changing any link. Unknown reparse points retain the original guard.
Var wiOldLinksDetached
Var wiNewLinksRestored
Var wiRuntimeHelperReady

; A stage that declares dependency links without the replayer installs a runtime
; whose package names never resolve: the payload is complete, both capabilities
; stay unreachable, and nothing on the target machine reports it. Refuse it here,
; where the failure costs a build instead of a release.
!ifndef SHELL_ONLY
  !if /FileExists "${STAGE_DIR}\runtime\vendor\runtime-graph.json"
    !if /FileExists "${STAGE_DIR}\runtime\runtime-links.exe"
    !else
      !error "The staged runtime declares dependency links but runtime-links.exe was not built beside it. Build the helper before makensis; an installer without it installs a runtime whose links are never created."
    !endif
  !endif
!endif

!macro WI_TraceRuntimeLinks Line
  Push $9
  DetailPrint "${Line}"
  ClearErrors
  CreateDirectory "$LOCALAPPDATA\mygo"
  FileOpen $9 "$LOCALAPPDATA\mygo\runtime-links.log" a
  ${IfNot} ${Errors}
    FileWrite $9 "${Line}$\r$\n"
    FileClose $9
  ${EndIf}
  Pop $9
!macroend

; The guard reads the promoted root in the extended-length form $wiFinalFS is
; kept in. Neither a false guard nor a missing replayer may pass silently: a
; required restore with no graph fails, and the trace records which happened.
!macro WI_RunRuntimeLinks Root Command Required
  StrCpy $wiError 0
  !ifndef SHELL_ONLY
    ${If} ${FileExists} "${Root}\runtime\vendor\runtime-graph.json"
      ${If} $wiRuntimeHelperReady != "1"
        InitPluginsDir
        File /oname=$PLUGINSDIR\runtime-links.exe "${STAGE_DIR}\runtime\runtime-links.exe"
        StrCpy $wiRuntimeHelperReady "1"
      ${EndIf}
      ClearErrors
      CreateDirectory "$LOCALAPPDATA\mygo"
      ExecWait '"$PLUGINSDIR\runtime-links.exe" ${Command} "${Root}\runtime\vendor" "$LOCALAPPDATA\mygo\runtime-links.log"' $R9
      ${If} ${Errors}
      ${OrIf} $R9 != 0
        StrCpy $wiError 1
      ${EndIf}
      !insertmacro WI_TraceRuntimeLinks "runtime links ${Command}: exit=$R9 error=$wiError root=${Root}"
    ${Else}
      !if ${Required} == 1
        StrCpy $wiError 1
        !insertmacro WI_TraceRuntimeLinks "runtime links ${Command}: no runtime graph under ${Root}"
      !else
        !insertmacro WI_TraceRuntimeLinks "runtime links ${Command}: no runtime graph under ${Root}, nothing to replay"
      !endif
    ${EndIf}
  !endif
!macroend

!macro WI_RunUninstallRuntimeLinks Root Command
  StrCpy $wiError 0
  !ifndef SHELL_ONLY
    ${If} ${FileExists} "${Root}\runtime\vendor\runtime-graph.json"
      ${If} $wiRuntimeHelperReady != "1"
        InitPluginsDir
        File /oname=$PLUGINSDIR\runtime-links.exe "${STAGE_DIR}\runtime\runtime-links.exe"
        StrCpy $wiRuntimeHelperReady "1"
      ${EndIf}
      ClearErrors
      CreateDirectory "$LOCALAPPDATA\mygo"
      ExecWait '"$PLUGINSDIR\runtime-links.exe" ${Command} "${Root}\runtime\vendor" "$LOCALAPPDATA\mygo\runtime-links.log"' $R9
      ${If} ${Errors}
      ${OrIf} $R9 != 0
        StrCpy $wiError 1
      ${EndIf}
      !insertmacro WI_TraceRuntimeLinks "runtime links ${Command}: exit=$R9 error=$wiError root=${Root}"
    ${Else}
      !insertmacro WI_TraceRuntimeLinks "runtime links ${Command}: no runtime graph under ${Root}, nothing to replay"
    ${EndIf}
  !endif
!macroend
