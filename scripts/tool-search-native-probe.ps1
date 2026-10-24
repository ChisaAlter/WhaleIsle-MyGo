param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final','-verified','-release')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r120-search-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-search-pass264-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class SearchNativeR120 {
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
[void][SearchNativeR120]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[SearchNativeR120]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key on exact owned MyGoSurface and UIAutomation ExpandCollapse. No Host, file opening, code execution, OS clipboard or hardware input.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [SearchNativeR120]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [SearchNativeR120]::Key($handle,0x78)
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
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[SearchNativeR120]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Near([string]$name,[double]$actual,[double]$expected){if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}}
function Assert-Open([object]$m){if(-not $m.expanded.'|tool:native-search' -or @($m.opened).Count -ne 0){throw 'Body action toggled header or opened file'}}
try {
 [SearchNativeR120]::Focus($handle)
 $initial=Metrics;Assert-Near 'collapsed row' $initial.row.H 24
 $r=(Element-Label 'Grep needle').Current.BoundingRectangle
 [SearchNativeR120]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $folded=Metrics;Assert-Open $folded
 [void](Element-Label 'Search results card');[void](Element-Label 'Showing 10 of 30 matches · 2 files');[void](Element-Label 'Copy')
 $fold=Element-Label 'Expand 4 more result lines';$foldHeight=$fold.Current.BoundingRectangle.Height/$folded.scale
 if([Math]::Abs($foldHeight-19) -gt (1/$folded.scale+.001)){throw ('Fold height exceeds one physical pixel rounding: '+$foldHeight)}
 [void](Element-Label 'src/b.go 6');[void](Element-Label '23: second-03')
 $run.states+=@{name='search-tail-attributed';pass=$true;metrics=$folded}
 Click-Label 'src/b.go 6';$group=Metrics;Assert-Open $group
 Assert-Near 'group collapse changes flat rows' ($folded.row.H-$group.row.H) 63
 [SearchNativeR120]::Key($handle,0x20);$groupSpace=Metrics;Assert-Open $groupSpace;Assert-Near 'Space restores group' $groupSpace.row.H $folded.row.H
 [SearchNativeR120]::Key($handle,0x0D);$groupEnter=Metrics;Assert-Open $groupEnter;Assert-Near 'Enter collapses group' $groupEnter.row.H $group.row.H
 (Element-Label 'src/b.go 6').GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Expand()
 $groupUIA=Metrics;Assert-Open $groupUIA;Assert-Near 'UIA Expand restores group' $groupUIA.row.H $folded.row.H
 $run.states+=@{name='group-click-space-enter-UIA';pass=$true;metrics=$groupUIA}
 Click-Label 'Expand 4 more result lines';$expanded=Metrics;Assert-Open $expanded
 Assert-Near 'four restored match rows' ($expanded.row.H-$folded.row.H) 88
 [void](Element-Label '22: second-02')
 [SearchNativeR120]::Key($handle,0x20);$spaceFolded=Metrics;Assert-Open $spaceFolded;Assert-Near 'focused Space folds' $spaceFolded.row.H $folded.row.H
 [SearchNativeR120]::Key($handle,0x0D);$enterExpanded=Metrics;Assert-Open $enterExpanded;Assert-Near 'focused Enter expands' $enterExpanded.row.H $expanded.row.H
 (Element-Label 'Collapse results').GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Collapse()
 $accessible=Metrics;Assert-Open $accessible;Assert-Near 'accessible fold pattern' $accessible.row.H $folded.row.H
 $run.states+=@{name='fold-click-space-enter-UIA';pass=$true;metrics=$accessible}
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Open $dark
 if(-not $dark.dark){throw 'Theme failed'};Assert-Near 'theme retains groups and fold' $dark.row.H $folded.row.H
 $run.states+=@{name='dark-keeps-fold-groups';pass=$true;metrics=$dark}
 Click-Label 'src/a.go 4';$changed=Metrics;Assert-Open $changed
 $r=(Element-Label 'Grep needle').Current.BoundingRectangle
 [SearchNativeR120]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$dark.scale))
 Start-Sleep -Milliseconds 260
 $closed=Metrics;Assert-Near 'closed after presence' $closed.row.H 24
 [SearchNativeR120]::Key($handle,0x20);$reopened=Metrics;Assert-Open $reopened;Assert-Near 'reopen resets group fold' $reopened.row.H $folded.row.H
 $run.states+=@{name='row-space-reopens-fresh';pass=$true;metrics=$reopened}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native search: tail attribution, group/fold click Space Enter UIA, theme/reopen scale'+$folded.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[SearchNativeR120]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
