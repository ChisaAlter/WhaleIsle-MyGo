param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r117-summary-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-summary-pass261-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class SummaryNativeR117 {
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
[void][SummaryNativeR117]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[SummaryNativeR117]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key messages on exact owned MyGoSurface plus UIAutomation Invoke. Recorded callback only: no files opened, Host calls, clipboard or hardware input.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [SummaryNativeR117]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [SummaryNativeR117]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){
   try{$value=Get-Content -LiteralPath $path -Raw -Encoding UTF8|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{}
  }
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
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[SummaryNativeR117]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Path([object]$metrics,[int]$count,[bool]$open){
 if(@($metrics.opened).Count -ne $count -or [bool]$metrics.expanded.'|tool:read-summary' -ne $open){throw ('Path/expansion mismatch '+($metrics|ConvertTo-Json -Compress -Depth 6))}
 foreach($value in $metrics.opened){if($value -ne '/recorded/src/read.go:37'){throw ('Wrong recorded path/line '+$value)}}
}
try {
 [SummaryNativeR117]::Focus($handle)
 $initial=Metrics;Assert-Path $initial 0 $false
 Click-Label 'src/read.go';$click=Metrics;Assert-Path $click 1 $false
 [SummaryNativeR117]::Key($handle,0x0D);$enter=Metrics;Assert-Path $enter 2 $false
 [SummaryNativeR117]::Key($handle,0x20);$space=Metrics;Assert-Path $space 3 $false
 $run.states+=@{name='path-click-enter-space';pass=$true;metrics=$space}
 $row=Element-Label 'Read src/read.go'
 $r=$row.Current.BoundingRectangle
 [SummaryNativeR117]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $open=Metrics;Assert-Path $open 3 $true
 if($open.rows.'read-summary'.H -le 24){throw 'Disclosure failed to mount output'}
 Click-Label 'src/read.go';$pathOpen=Metrics;Assert-Path $pathOpen 4 $true
 # Invoke targets the exposed path control even under an actionable disclosure.
 $pattern=(Element-Label 'src/read.go').GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
 $pattern.Invoke();$invoked=Metrics;Assert-Path $invoked 5 $true
 $run.states+=@{name='open-row-path-and-accessible-invoke';pass=$true;metrics=$invoked}
 [void](Element-Label '1KB');[void](Element-Label '+2 -1')
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Path $dark 5 $true
 if(-not $dark.dark){throw 'Theme switch failed'}
 $run.states+=@{name='theme-keeps-disclosure-and-path';pass=$true;metrics=$dark}
 $r=(Element-Label 'Read src/read.go').Current.BoundingRectangle
 [SummaryNativeR117]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 Start-Sleep -Milliseconds 260
 $closed=Metrics;Assert-Path $closed 5 $false
 [SummaryNativeR117]::Key($handle,0x20);$reopened=Metrics;Assert-Path $reopened 5 $true
 $run.states+=@{name='disclosure-space';pass=$true;metrics=$reopened}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS file summary: path click/Enter/Space, independent disclosure, UIAutomation Invoke, totals and theme at scale'+$initial.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[SummaryNativeR117]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
