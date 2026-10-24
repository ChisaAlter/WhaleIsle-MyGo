param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final','-verified','-release')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r119-diff-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-diff-pass263-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class DiffNativeR119 {
 [StructLayout(LayoutKind.Sequential)] public struct Point{public int X,Y;}
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string name,string title);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 public static uint Owner(IntPtr h){uint p;GetWindowThreadProcessId(h,out p);return p;}
 public static void Verify(IntPtr h,int p){if(h==IntPtr.Zero||Owner(h)!=(uint)p)throw new Exception("Fixture ownership changed");}
 public static IntPtr Surface(IntPtr h){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero||Owner(s)!=Owner(h))throw new Exception("Owned surface absent");return s;}
 public static void Focus(IntPtr h){SendMessage(Surface(h),7,IntPtr.Zero,IntPtr.Zero);}
 public static void Click(IntPtr h,int x,int y){var s=Surface(h);var p=new Point{X=x,Y=y};if(!ScreenToClient(s,ref p))throw new Exception("ScreenToClient failed");var l=new IntPtr((p.Y<<16)|(p.X&65535));SendMessage(s,0x200,IntPtr.Zero,l);SendMessage(s,0x201,new IntPtr(1),l);SendMessage(s,0x202,IntPtr.Zero,l);}
 public static void Key(IntPtr h,int key){var s=Surface(h);SendMessage(s,0x100,new IntPtr(key),new IntPtr(1));SendMessage(s,0x101,new IntPtr(key),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Close(IntPtr h){if(!PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero))throw new Exception("Owned close failed");}
}
'@
[void][DiffNativeR119]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[DiffNativeR119]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key on exact owned MyGoSurface and UIAutomation ExpandCollapse. No Host, file opening, code execution, OS clipboard or hardware input.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [DiffNativeR119]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [DiffNativeR119]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){try{$value=Get-Content -LiteralPath $path -Raw -Encoding UTF8|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{}}
  Start-Sleep -Milliseconds 15
 }
 throw 'No fresh F9 metrics'
}
function Element-Label([string]$name){
 Verify-Owner
 $condition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,$name)
 $elements=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,$condition)
 $match=$elements|Where-Object {$_.Current.BoundingRectangle.Width -gt 0}|Select-Object -First 1
 if($null -eq $match){throw ('Missing owned accessible label '+$name)}
 return $match
}
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[DiffNativeR119]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Near([string]$name,[double]$actual,[double]$expected){if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}}
function Assert-Open([object]$m){if(-not $m.expanded.'|tool:native-diff' -or @($m.opened).Count -ne 0){throw 'Body action toggled header or opened file'}}
try {
 [DiffNativeR119]::Focus($handle)
 $initial=Metrics;Assert-Near 'collapsed row' $initial.row.H 24
 $r=(Element-Label 'Write src/create.go').Current.BoundingRectangle
 [DiffNativeR119]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $folded=Metrics;Assert-Open $folded
 [void](Element-Label 'Diff content card');[void](Element-Label 'Copy')
 $fold=Element-Label 'Expand 5 more diff lines';$foldHeight=$fold.Current.BoundingRectangle.Height/$folded.scale
 # UIA bounds are integer device pixels. One edge may round by half a pixel.
 if([Math]::Abs($foldHeight-19) -gt (1/$folded.scale+.001)){throw ('Fold height exceeds one device pixel rounding: '+$foldHeight)}
 $run.states+=@{name='diff-open-folded';pass=$true;metrics=$folded}
 Click-Label 'Expand 5 more diff lines';$expanded=Metrics;Assert-Open $expanded
 Assert-Near 'five more rows' ($expanded.row.H-$folded.row.H) 95
 [void](Element-Label '+ created-06')
 [DiffNativeR119]::Key($handle,0x20);$spaceFolded=Metrics;Assert-Open $spaceFolded;Assert-Near 'focused Space folds' $spaceFolded.row.H $folded.row.H
 [DiffNativeR119]::Key($handle,0x0D);$enterExpanded=Metrics;Assert-Open $enterExpanded;Assert-Near 'focused Enter expands' $enterExpanded.row.H $expanded.row.H
 $run.states+=@{name='fold-click-space-enter';pass=$true;metrics=$enterExpanded}
 (Element-Label 'Collapse diff').GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Collapse()
 $invoked=Metrics;Assert-Open $invoked;Assert-Near 'accessible Collapse pattern' $invoked.row.H $folded.row.H
 Click-Label 'Wrap lines';$wrapped=Metrics;Assert-Open $wrapped
 if($wrapped.row.H -le $folded.row.H){throw 'Wrapping did not increase long-line height'}
 [DiffNativeR119]::Key($handle,0x0D);$unwrapped=Metrics;Assert-Open $unwrapped;Assert-Near 'wrap Enter restores columns' $unwrapped.row.H $folded.row.H
 $run.states+=@{name='wrap-click-enter';pass=$true;metrics=$unwrapped}
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Open $dark
 if(-not $dark.dark){throw 'Theme failed'};Assert-Near 'theme retains fold and wrap' $dark.row.H $folded.row.H
 $run.states+=@{name='dark-keeps-fold-wrap';pass=$true;metrics=$dark}
 # Unmount a changed body: a later reopen must restore both initial states.
 Click-Label 'Expand 5 more diff lines';Click-Label 'Wrap lines';$changed=Metrics;Assert-Open $changed
 $r=(Element-Label 'Write src/create.go').Current.BoundingRectangle
 [DiffNativeR119]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$dark.scale))
 Start-Sleep -Milliseconds 260
 $closed=Metrics;Assert-Near 'closed after presence' $closed.row.H 24
 [DiffNativeR119]::Key($handle,0x20);$reopened=Metrics;Assert-Open $reopened;Assert-Near 'reopen folds fresh body' $reopened.row.H $folded.row.H
 $run.states+=@{name='row-space-reopens';pass=$true;metrics=$reopened}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native diff: fold click/Space/Enter/UIA, wrap/Enter, theme and parent reopen at scale'+$folded.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[DiffNativeR119]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
