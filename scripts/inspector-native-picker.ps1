param(
 [Parameter(Mandatory=$true)][int]$OwnerId,
 [switch]$CaptureFinal
)
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r111-inspector-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-pass255-native-picker'
$protectedIds=@(20012,16008)
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in $protectedIds){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;using System.Collections.Generic;using System.Text;using System.Runtime.InteropServices;using System.Drawing;using System.Drawing.Imaging;
public class InspectorPickerR111 {
 public struct Point{public int X,Y;}
 public struct Rect{public int L,T,R,B;}
 [StructLayout(LayoutKind.Sequential)] public struct CursorInfo{public int Size;public uint Flags;public IntPtr Cursor;public Point Position;}
 [StructLayout(LayoutKind.Sequential,CharSet=CharSet.Unicode)] public struct MonitorInfo{public int Size;public Rect Monitor,Work;public uint Flags;[MarshalAs(UnmanagedType.ByValTStr,SizeConst=32)]public string Device;}
 delegate bool MonitorEnum(IntPtr h,IntPtr dc,ref Rect r,IntPtr data);
 [DllImport("kernel32.dll")] static extern void SetLastError(uint e);
 [DllImport("kernel32.dll")] static extern uint GetCurrentThreadId();
 [DllImport("user32.dll",SetLastError=true)] public static extern bool ShowWindow(IntPtr h,int c);
 [DllImport("user32.dll",SetLastError=true)] static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] static extern IntPtr GetThreadDpiAwarenessContext();
 [DllImport("user32.dll")] static extern int GetAwarenessFromDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr FindWindowEx(IntPtr p,IntPtr a,string c,string t);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h,ref Point p);
 [DllImport("user32.dll",SetLastError=true)] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll",SetLastError=true)] static extern bool GetCursorPos(out Point p);
 [DllImport("user32.dll",SetLastError=true)] static extern bool SetCursorPos(int x,int y);
 [DllImport("user32.dll",SetLastError=true)] static extern bool GetClipCursor(out Rect r);
 [DllImport("user32.dll",SetLastError=true)] static extern bool GetCursorInfo(ref CursorInfo c);
 [DllImport("user32.dll")] static extern IntPtr LoadCursor(IntPtr instance,IntPtr name);
 [DllImport("user32.dll")] static extern IntPtr WindowFromPoint(Point p);
 [DllImport("user32.dll")] static extern IntPtr GetAncestor(IntPtr h,uint flags);
 [DllImport("user32.dll")] static extern IntPtr GetWindow(IntPtr h,uint command);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h,StringBuilder s,int n);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder s,int n);
 [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsIconic(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsWindowEnabled(IntPtr h);
 [DllImport("user32.dll")] static extern bool PrintWindow(IntPtr h,IntPtr dc,uint f);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out Rect r);
 [DllImport("user32.dll",SetLastError=true)] static extern bool SetWindowPos(IntPtr h,IntPtr after,int x,int y,int w,int ht,uint flags);
 [DllImport("user32.dll")] static extern uint GetDpiForWindow(IntPtr h);
 [DllImport("user32.dll")] static extern IntPtr MonitorFromWindow(IntPtr h,uint flags);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern bool GetMonitorInfo(IntPtr h,ref MonitorInfo m);
 [DllImport("user32.dll")] static extern bool EnumDisplayMonitors(IntPtr dc,IntPtr clip,MonitorEnum cb,IntPtr data);
 [DllImport("user32.dll")] static extern int GetSystemMetrics(int index);
 [DllImport("user32.dll")] static extern IntPtr GetThreadDesktop(uint id);
 [DllImport("user32.dll")] static extern IntPtr GetProcessWindowStation();
 [DllImport("user32.dll",SetLastError=true)] static extern IntPtr OpenInputDesktop(uint flags,bool inherit,uint access);
 [DllImport("user32.dll")] static extern bool CloseDesktop(IntPtr h);
 [DllImport("user32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern bool GetUserObjectInformation(IntPtr h,int index,StringBuilder s,int bytes,out int needed);
 [DllImport("dwmapi.dll")] static extern int DwmGetWindowAttribute(IntPtr h,uint attribute,out int v,int bytes);
 static int Error(bool ok){return ok?0:Marshal.GetLastWin32Error();}
 public static uint Owner(IntPtr h){uint p;GetWindowThreadProcessId(h,out p);return p;}
 public static void Verify(IntPtr h,int pid){if(h==IntPtr.Zero||Owner(h)!=(uint)pid)throw new Exception("Fixture window ownership changed");}
 static string Text(IntPtr h,bool title){var s=new StringBuilder(256);if(title)GetWindowText(h,s,256);else GetClassName(h,s,256);return s.ToString();}
 static string ObjectName(IntPtr h){var s=new StringBuilder(256);int needed;return GetUserObjectInformation(h,2,s,512,out needed)?s.ToString():"unavailable: "+Marshal.GetLastWin32Error();}
 public static object Describe(IntPtr h){Rect r;GetWindowRect(h,out r);int cloaked;int hr=DwmGetWindowAttribute(h,14,out cloaked,4);var root=GetAncestor(h,2);var rootOwner=GetAncestor(h,3);return new {handle=h.ToInt64(),pid=Owner(h),title=Text(h,true),className=Text(h,false),visible=IsWindowVisible(h),iconic=IsIconic(h),enabled=IsWindowEnabled(h),rect=r,root=root.ToInt64(),rootPid=Owner(root),rootOwner=rootOwner.ToInt64(),rootOwnerPid=Owner(rootOwner),owner=GetWindow(h,4).ToInt64(),cloaked=hr==0?(int?)cloaked:null};}
 public static IntPtr Surface(IntPtr h){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero||Owner(s)!=Owner(h))throw new Exception("Owned surface absent");return s;}
 static IntPtr Pos(IntPtr s,int x,int y){var p=new Point{X=x,Y=y};if(!ScreenToClient(s,ref p))throw new Exception("ScreenToClient failed");return new IntPtr((p.Y<<16)|(p.X&65535));}
 public static Point Origin(IntPtr h){var p=new Point();if(!ClientToScreen(Surface(h),ref p))throw new Exception("ClientToScreen failed");return p;}
 public static object Top(IntPtr h){SetLastError(0);bool ok=SetWindowPos(h,new IntPtr(-1),0,0,0,0,0x13);int error=Error(ok);return new {ok=ok,error=error};}
 public static object Activate(IntPtr h){SetLastError(0);bool ok=SetForegroundWindow(h);int error=Error(ok);return new {returned=ok,error=error,foregroundVerified=GetForegroundWindow()==h,foreground=Describe(GetForegroundWindow())};}
 // This deliberately injects WM_SETFOCUS; it is not a verified foreground activation.
 public static void FocusEvent(IntPtr h){SendMessage(Surface(h),7,IntPtr.Zero,IntPtr.Zero);}
 public static object Desktop(IntPtr h){Rect clip;SetLastError(0);bool clipOK=GetClipCursor(out clip);int clipError=Error(clipOK);SetLastError(0);var input=OpenInputDesktop(0,false,1);int inputError=input==IntPtr.Zero?Marshal.GetLastWin32Error():0;string inputName=input==IntPtr.Zero?null:ObjectName(input);if(input!=IntPtr.Zero)CloseDesktop(input);var monitors=new List<object>();MonitorEnum cb=delegate(IntPtr mh,IntPtr dc,ref Rect r,IntPtr data){var mi=new MonitorInfo{Size=Marshal.SizeOf(typeof(MonitorInfo))};bool ok=GetMonitorInfo(mh,ref mi);monitors.Add(new {handle=mh.ToInt64(),ok=ok,value=mi});return true;};EnumDisplayMonitors(IntPtr.Zero,IntPtr.Zero,cb,IntPtr.Zero);return new {fixture=Describe(h),surface=Describe(Surface(h)),dpi=GetDpiForWindow(h),threadDpiAwareness=GetAwarenessFromDpiAwarenessContext(GetThreadDpiAwarenessContext()),windowStation=ObjectName(GetProcessWindowStation()),threadDesktop=ObjectName(GetThreadDesktop(GetCurrentThreadId())),inputDesktop=new {name=inputName,error=inputError},clip=new {ok=clipOK,error=clipError,rect=clip},virtualScreen=new {x=GetSystemMetrics(76),y=GetSystemMetrics(77),width=GetSystemMetrics(78),height=GetSystemMetrics(79)},remoteSession=GetSystemMetrics(0x1000)!=0,monitors=monitors,foreground=Describe(GetForegroundWindow())};}
 public static object ProbeCursor(IntPtr h,int x,int y){Point before,after;SetLastError(0);bool beforeOK=GetCursorPos(out before);int beforeError=Error(beforeOK);SetLastError(0);bool setOK=SetCursorPos(x,y);int setError=Error(setOK);SetLastError(0);bool afterOK=GetCursorPos(out after);int afterError=Error(afterOK);var target=new Point{X=x,Y=y};var targetHit=WindowFromPoint(target);var actualHit=afterOK?WindowFromPoint(after):IntPtr.Zero;bool exact=afterOK&&after.X==x&&after.Y==y;return new {target=target,before=new {ok=beforeOK,error=beforeError,point=before},set=new {ok=setOK,error=setError},after=new {ok=afterOK,error=afterError,point=after},exactPositionVerified=exact,requestedHit=Describe(targetHit),actualHit=Describe(actualHit),nativeTargetVerified=setOK&&exact&&GetAncestor(targetHit,2)==h&&GetAncestor(actualHit,2)==h};}
 public static void Move(IntPtr h,int x,int y){var s=Surface(h);SendMessage(s,0x200,IntPtr.Zero,Pos(s,x+1,y));SendMessage(s,0x200,IntPtr.Zero,Pos(s,x,y));}
 // Probe the cursor selected by the surface's normal WM_SETCURSOR handler.
 // A matching handle under injection does not prove hardware input or native hit ownership.
 public static object Cursor(IntPtr h){var s=Surface(h);var handled=SendMessage(s,0x20,s,new IntPtr((0x200<<16)|1));var ci=new CursorInfo{Size=Marshal.SizeOf(typeof(CursorInfo))};SetLastError(0);bool ok=GetCursorInfo(ref ci);int error=Error(ok);var cross=LoadCursor(IntPtr.Zero,new IntPtr(32515));return new {ok=ok,error=error,handled=handled.ToInt64(),handle=ci.Cursor.ToInt64(),crosshairHandle=cross.ToInt64(),isCrosshair=ok&&ci.Cursor==cross,flags=ci.Flags,position=ci.Position,scope="injected WM_SETCURSOR on owned MyGoSurface"};}
 public static void Click(IntPtr h,int x,int y,bool right){var s=Surface(h);var l=Pos(s,x,y);SendMessage(s,0x200,IntPtr.Zero,l);SendMessage(s,right?0x204u:0x201u,new IntPtr(right?2:1),l);SendMessage(s,right?0x205u:0x202u,IntPtr.Zero,l);}
 public static void Wheel(IntPtr h,int x,int y,int delta){SendMessage(Surface(h),0x20a,new IntPtr(delta<<16),new IntPtr((y<<16)|(x&65535)));}
 public static void Key(IntPtr h,int k){var s=Surface(h);SendMessage(s,0x100,new IntPtr(k),new IntPtr(1));SendMessage(s,0x101,new IntPtr(k),new IntPtr(unchecked((int)0xc0000001)));}
 public static void ResizeHeight(IntPtr h,int height){Rect r;GetWindowRect(h,out r);if(!SetWindowPos(h,IntPtr.Zero,0,0,r.R-r.L,height,0x16))throw new Exception("Owned resize failed: "+Marshal.GetLastWin32Error());}
 public static void Close(IntPtr h){if(!PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero))throw new Exception("Owned close failed");}
 public static void Capture(IntPtr h,string path){Rect r;GetWindowRect(h,out r);using(var b=new Bitmap(r.R-r.L,r.B-r.T)){using(var g=Graphics.FromImage(b)){var dc=g.GetHdc();try{if(!PrintWindow(h,dc,2))throw new Exception("Capture failed");}finally{g.ReleaseHdc(dc);}}b.Save(path,ImageFormat.Png);}}
}
'@
[void][InspectorPickerR111]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[InspectorPickerR111]::Verify($handle,$OwnerId)
$run=[ordered]@{
 schema=2;runId=[guid]::NewGuid().ToString();status='running';strictPassed=$false;startedAt=[datetime]::UtcNow.ToString('o');finishedAt=$null
 fixture=@{pid=$OwnerId;executable=$fixturePath;startTimeUtc=$ownedStart.ToString('o');closeRequested=$false;exited=$false}
 scope='Win32 messages injected into the owned MyGoSurface; WM_SETFOCUS is synthetic; no hardware input claim'
 assertions='strict: failed assertions are retained and cause a nonzero exit; independent checks continue for diagnosis'
 desktop=$null;activation=$null;checks=@();states=@();cursorSamples=@();error=$null;screenshot=$null
 limitations=@('Native hover is proven only if SetCursorPos and GetCursorPos agree and WindowFromPoint root belongs to the fixture.','Injected WM_SETFOCUS does not establish OS foreground focus.')
}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId -ErrorAction Stop
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'Owned PID start time changed'}
 [InspectorPickerR111]::Verify($handle,$OwnerId)
}
function Write-Evidence {
 $run | ConvertTo-Json -Depth 24 | Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
function Save([string]$name,$metrics,$extra=$null){
 $run.states+=@{name=$name;metrics=$metrics;extra=$extra}
 Write-Evidence
}
function Check([string]$name,[bool]$passed,[string]$detail){
 $run.checks+=@{name=$name;passed=$passed;detail=$detail}
 Write-Host ((@{$true='PASS';$false='FAIL'}[$passed])+' '+$name+': '+$detail)
 Write-Evidence
}
function Find([string]$name){
 $all=@($root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)|Where-Object {$_.Current.Name -eq $name -and $_.Current.BoundingRectangle.Width -gt 0})
 if($all.Count -eq 0){throw ('Missing '+$name)}
 $button=$all|Where-Object {$_.Current.ControlType -eq [System.Windows.Automation.ControlType]::Button}|Select-Object -First 1
 if($null -ne $button){return $button}
 return $all[0]
}
function Click([string]$name){
 Verify-Owner
 $r=(Find $name).Current.BoundingRectangle
 [InspectorPickerR111]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2),$false)
 Start-Sleep -Milliseconds 180
}
function Key([int]$key){Verify-Owner;[InspectorPickerR111]::Key($handle,$key);Start-Sleep -Milliseconds 180}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [InspectorPickerR111]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){
   try{$value=Get-Content -LiteralPath $path -Raw|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{}
  }
  Start-Sleep -Milliseconds 30
 }
 throw 'Fixture did not acknowledge fresh F9 metrics'
}
function Point([string]$id,$m){
 $r=$m.reveal.pickRows.$id
 if($null -eq $r -or $r.W -le 0 -or $r.H -le 0){throw ('No painted pick row '+$id)}
 $origin=[InspectorPickerR111]::Origin($handle)
 @{x=[int]($origin.X+($r.X+[Math]::Min(20,$r.W/2))*$m.reveal.scale);y=[int]($origin.Y+($r.Y+$r.H/2)*$m.reveal.scale)}
}
function Geometry-Changes($before,$after){
 $changes=@()
 $names=@(@($before.PSObject.Properties.Name)+@($after.PSObject.Properties.Name)|Sort-Object -Unique)
 foreach($name in $names){if(($before.$name|ConvertTo-Json -Compress) -ne ($after.$name|ConvertTo-Json -Compress)){$changes+=$name}}
 return $changes
}
function Hover([string]$label,[string]$id,$baseline){
 Verify-Owner
 $pos=Point $id $baseline
 $native=[InspectorPickerR111]::ProbeCursor($handle,$pos.x,$pos.y)
 [InspectorPickerR111]::Move($handle,$pos.x,$pos.y)
 $cursor=[InspectorPickerR111]::Cursor($handle)
 Start-Sleep -Milliseconds 150
 $m=Metrics
 $run.cursorSamples+=@{name=$label;native=$native;surfaceCursor=$cursor}
 $row=$m.reveal.pickRows.$id;$area=$m.reveal.pickArea;$preview=$m.reveal.pickPreview
 $expected=@{X=[Math]::Max($row.X,$area.X);Y=[Math]::Max($row.Y,$area.Y);W=0;H=0}
 $expected.W=[Math]::Max(0,[Math]::Min($row.X+$row.W,$area.X+$area.W)-$expected.X)
 $expected.H=[Math]::Max(0,[Math]::Min($row.Y+$row.H,$area.Y+$area.H)-$expected.Y)
 $geometry=$preview.W -gt 0 -and $preview.H -gt 0
 foreach($coordinate in 'X','Y','W','H'){if([Math]::Abs($preview.$coordinate-$expected.$coordinate) -gt 0.01){$geometry=$false}}
 Check ($label+'-hover-identity') ($m.picking -and $m.reveal.pickPreviewID -eq $id -and $geometry) ('expected '+$id+'; actual '+$m.reveal.pickPreviewID+'; preview='+($preview|ConvertTo-Json -Compress))
 Check ($label+'-crosshair') ([bool]$cursor.isCrosshair) 'Surface WM_SETCURSOR must select IDC_CROSS; injection is not physical cursor proof'
 Save ($label+'-hover') $m @{native=$native;surfaceCursor=$cursor;expectedPreview=$expected}
 return @{metrics=$m;position=$pos}
}
$completed=$false
try {
 Verify-Owner
 [void][InspectorPickerR111]::ShowWindow($handle,9)
 $top=[InspectorPickerR111]::Top($handle)
 $run.activation=[InspectorPickerR111]::Activate($handle)
 [InspectorPickerR111]::FocusEvent($handle)
 $root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
 $run.desktop=[InspectorPickerR111]::Desktop($handle)
 $run.desktop | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath ($prefix+'-cursor-diagnostic.json') -Encoding UTF8
 Key 0x1b
 Key 0x75
 $before=Metrics
 Save 'baseline-before-picker' $before @{top=$top;syntheticFocus=$true}
 Click 'Pick an element in Chat View'
 $m=Metrics
 Check 'picker-started' ([bool]$m.picking) 'Toolbar activates picker'
 $activationChanges=@(Geometry-Changes $before.reveal.pickRows $m.reveal.pickRows)
 Check 'picker-activation-no-reflow' ($activationChanges.Count -eq 0) ('Changed rows: '+($activationChanges -join ', '))
 $group=$m.groups.PSObject.Properties|Where-Object {$m.reveal.pickRows.($_.Name).H -gt 0}|Select-Object -First 1
 if($null -eq $group){throw 'No visible group'}
 $hover=Hover 'group' $group.Name $m
 $changes=@(Geometry-Changes $before.reveal.pickRows $hover.metrics.reveal.pickRows)
 Check 'group-hover-no-reflow' ($changes.Count -eq 0) ('Changed rows: '+($changes -join ', '))
 $pos=$hover.position
 Verify-Owner
 [InspectorPickerR111]::Click($handle,$pos.x,$pos.y,$true)
 Start-Sleep -Milliseconds 150
 $right=Metrics
 Check 'right-click-suppressed' ($right.picking -and $right.selected -eq '' -and ($right.expanded|ConvertTo-Json -Compress) -eq ($before.expanded|ConvertTo-Json -Compress)) 'Right click must keep picker, selection, and group disclosure unchanged'
 Save 'right-click-suppressed' $right
 Verify-Owner
 [InspectorPickerR111]::Click($handle,$pos.x,$pos.y,$false)
 Start-Sleep -Milliseconds 180
 $picked=Metrics
 Check 'group-picked-event10-no-disclosure' (!$picked.picking -and $picked.selected -eq 'event:10' -and ($picked.expanded|ConvertTo-Json -Compress) -eq ($before.expanded|ConvertTo-Json -Compress)) ('Selected '+$picked.selected+'; picking='+$picked.picking)
 Save 'group-picked-no-disclosure' $picked
 Click 'Close'
 Key 0x75
 Click 'Pick an element in Chat View'
 $m=Metrics
 $hover=Hover 'response' 'm40/text/2' $m
 $pos=$hover.position
 Verify-Owner
 [InspectorPickerR111]::Click($handle,$pos.x,$pos.y,$false)
 Start-Sleep -Milliseconds 180
 $m=Metrics
 Check 'response-picked-event40' (!$m.picking -and $m.selected -eq 'event:40') ('Selected '+$m.selected+'; picking='+$m.picking)
 Save 'response-picked' $m
 Click 'Close'
 Click 'Pick an element in Chat View'
 Click 'Outside composer'
 $m=Metrics
 $composer=Find 'Outside composer'
 $composerFocus=[bool]$composer.Current.HasKeyboardFocus
 Check 'outside-composer-keeps-picker' ([bool]$m.picking) 'Composer click must not cancel picker'
 Check 'outside-composer-focus' $composerFocus 'UIA HasKeyboardFocus after Win32-injected click and initial synthetic WM_SETFOCUS'
 Save 'outside-composer-focused' $m @{hasKeyboardFocus=$composerFocus;syntheticWindowFocus=$true}
 Key 0x1b
 $m=Metrics
 Check 'escape-from-composer' (!$m.picking -and $m.reveal.pickPreview.H -le 0 -and $m.reveal.pickPreviewID -eq '') 'Escape cancels picker and removes preview from composer focus'
 Save 'escape-from-composer' $m
 Key 0x75
 # The default 720-DIP fixture fits these short records. Resize only the owned
 # fixture so the wheel assertion actually exercises a nonzero scroll range.
 Verify-Owner
 [InspectorPickerR111]::ResizeHeight($handle,[int](430*$m.reveal.scale))
 Start-Sleep -Milliseconds 300
 Click 'Work details'
 Click 'Used tools'
 Start-Sleep -Milliseconds 500
 Click 'Pick an element in Chat View'
 $m=Metrics
 Save 'expanded-reasoning-with-scroll-range' $m
 $hover=Hover 'reasoning' 'm40/reasoning/0' $m
 $m=$hover.metrics;$pos=$hover.position
 Check 'wheel-has-scroll-range' ($m.max -gt 0) ('scroll='+$m.scroll+'; max='+$m.max)
 Verify-Owner
 [InspectorPickerR111]::Wheel($handle,$pos.x,$pos.y,-120)
 Start-Sleep -Milliseconds 180
 $scrolled=Metrics
 Check 'wheel-clears-preview-keeps-picker' ($scrolled.picking -and $scrolled.reveal.pickPreview.H -le 0 -and $scrolled.reveal.pickPreviewID -eq '') 'Wheel clears preview while keeping picker active'
 Check 'wheel-scrolls-transcript' ($m.max -gt 0 -and $scrolled.scroll -gt $m.scroll) ('before='+$m.scroll+'; after='+$scrolled.scroll+'; max='+$m.max)
 Save 'wheel-keeps-picker' $scrolled
 Key 0x1b
 $final=Metrics
 Check 'final-escape' (!$final.picking -and $final.reveal.pickPreview.H -le 0) 'Final Escape retires picker'
 Save 'final-idle' $final
 $completed=$true
} catch {
 $run.error=$_.Exception.Message
 Write-Output ('ERROR '+$run.error)
} finally {
 $failed=@($run.checks|Where-Object {!$_.passed})
 $nativeVerified=@($run.cursorSamples|Where-Object {$_.native.nativeTargetVerified})
 $blocked=@($run.cursorSamples|Where-Object {!$_.native.nativeTargetVerified})
 $run.strictPassed=$completed -and $failed.Count -eq 0
 $run.status=if($run.strictPassed){'pass-strict-injected-surface'}elseif($completed -and $blocked.Count -gt 0){'partial-environment-blocked'}else{'failed'}
 $run['nativeDesktopHover']=@{verified=$run.cursorSamples.Count -gt 0 -and $nativeVerified.Count -eq $run.cursorSamples.Count;blockedSamples=$blocked.Count;scope='SetCursorPos/GetCursorPos/WindowFromPoint ownership; no hardware-input claim'}
 if($CaptureFinal -and $completed){
  try {Verify-Owner;$capturePath=$prefix+$(if($run.strictPassed){'-final.png'}else{'-final-partial.png'});[InspectorPickerR111]::Capture($handle,$capturePath);$run.screenshot=@{path=$capturePath;scope='Final idle fixture capture after every input check; not hover proof'}}catch{$run.screenshot=@{error=$_.Exception.Message}}
 }
 try {Verify-Owner;[InspectorPickerR111]::Close($handle);$run.fixture.closeRequested=$true;for($wait=0;$wait -lt 100;$wait++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 50}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 Write-Evidence
 $result=[ordered]@{runId=$run.runId;status=$run.status;strictPassed=$run.strictPassed;scope=$run.scope;completedSequence=$completed;passedChecks=@($run.checks|Where-Object passed).Count;failedChecks=$failed;nativeDesktopHover=$run.nativeDesktopHover;fixture=$run.fixture;error=$run.error;startedAt=$run.startedAt;finishedAt=$run.finishedAt;evidence=@(($prefix+'-state.json'),($prefix+'-cursor-diagnostic.json'),($prefix+'-metrics.json'))}
 $result | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath ($prefix+'-result.json') -Encoding UTF8
 $lines=@(('Status: '+$run.status),('Strict passed: '+$run.strictPassed),('Scope: '+$run.scope),('Native desktop hover verified: '+$run.nativeDesktopHover.verified))
 $lines+=@($run.checks|ForEach-Object {$(if($_.passed){'PASS '}else{'FAIL '})+$_.name+': '+$_.detail})
 if($run.error){$lines+='ERROR '+$run.error}
 $lines|ForEach-Object {$_.TrimEnd()}|Set-Content -LiteralPath ($prefix+'-run.txt') -Encoding UTF8
}
if(!$run.strictPassed){throw ('Strict native picker acceptance did not pass: '+$run.status+'; see '+$prefix+'-result.json')}
'PASS strict Win32-injected surface: hover/crosshair/no reflow, right suppression, group event10 + response event40, composer focus/Escape, reasoning, wheel'
