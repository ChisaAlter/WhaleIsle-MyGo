; Directory junctions are restored only after the stage has its final name.
; The embedded windowsgui helper validates every declaration and physical target
; before changing any link. Unknown reparse points retain the original guard.
Var wiOldLinksDetached
Var wiNewLinksRestored
Var wiRuntimeHelperReady

!macro WI_RunRuntimeLinks Root Command
  StrCpy $wiError 0
!if /FileExists "${STAGE_DIR}\runtime\runtime-links.exe"
  ${If} ${FileExists} "${Root}\runtime\vendor\runtime-graph.json"
    ${If} $wiRuntimeHelperReady != "1"
      InitPluginsDir
      File /oname=$PLUGINSDIR\runtime-links.exe "${STAGE_DIR}\runtime\runtime-links.exe"
      StrCpy $wiRuntimeHelperReady "1"
    ${EndIf}
    ClearErrors
    ExecWait '"$PLUGINSDIR\runtime-links.exe" ${Command} "${Root}\runtime\vendor"' $R9
    ${If} ${Errors}
    ${OrIf} $R9 != 0
      StrCpy $wiError 1
    ${EndIf}
  ${EndIf}
!endif
!macroend

!macro WI_RunUninstallRuntimeLinks Root Command
  StrCpy $wiError 0
!if /FileExists "${STAGE_DIR}\runtime\runtime-links.exe"
  ${If} ${FileExists} "${Root}\runtime\vendor\runtime-graph.json"
    ${If} $wiRuntimeHelperReady != "1"
      InitPluginsDir
      File /oname=$PLUGINSDIR\runtime-links.exe "${STAGE_DIR}\runtime\runtime-links.exe"
      StrCpy $wiRuntimeHelperReady "1"
    ${EndIf}
    ClearErrors
    ExecWait '"$PLUGINSDIR\runtime-links.exe" ${Command} "${Root}\runtime\vendor"' $R9
    ${If} ${Errors}
    ${OrIf} $R9 != 0
      StrCpy $wiError 1
    ${EndIf}
  ${EndIf}
!endif
!macroend
