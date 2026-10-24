param([Parameter(Mandatory=$true)][int]$OwnerId)
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r116-code-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-code-pass260-native'
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class CodeNativeR116 {
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
[void][CodeNativeR116]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[CodeNativeR116]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus, click and key messages into exact owned MyGoSurface. No OS hardware input/hover, no original app interaction, no clipboard mutation or code execution.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [CodeNativeR116]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [CodeNativeR116]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){
   try{$value=Get-Content -LiteralPath $path -Raw -Encoding UTF8|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{# Retry the read while the fixture completes its bounded small write.
   }
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
 if($null -eq $match){throw ('Missing visible owned label '+$name)}
 return $match
}
function Click-Label([string]$name){
 $element=Element-Label $name
 $r=$element.Current.BoundingRectangle
 [CodeNativeR116]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))
}
function Assert-Near([string]$name,[double]$actual,[double]$expected){
 if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}
}
try {
 [CodeNativeR116]::Focus($handle)
 $initial=Metrics
 Assert-Near 'collapsed foreground' $initial.row.H 24
 Click-Label 'Code Inspect recorded program'
 $open=Metrics
 if($open.expanded.code -ne $true -or $open.row.H -lt 120){throw 'Native disclosure did not mount program+OUT'}
 $run.states+=@{name='clicked-open';pass=$true;metrics=$open}
 Click-Label 'Wrap lines'
 $unwrapped=Metrics
 if($unwrapped.expanded.code -ne $true -or $unwrapped.row.H -ge $open.row.H){throw 'Wrap action failed or closed row'}
 $run.states+=@{name='unwrap';pass=$true;metrics=$unwrapped}
 [CodeNativeR116]::Key($handle,0x0D)
 $rewrapped=Metrics
 Assert-Near 'focused Enter restores wrapped layout' $rewrapped.row.H $open.row.H
 if($rewrapped.expanded.code -ne $true){throw 'Toolbar Enter toggled disclosure'}
 $run.states+=@{name='wrap-enter';pass=$true;metrics=$rewrapped}
 Click-Label 'Toggle theme'
 $dark=Metrics
 if(-not $dark.dark -or $dark.expanded.code -ne $true){throw 'Theme switch reset disclosure'}
 Assert-Near 'theme retains wrap layout' $dark.row.H $open.row.H
 $run.states+=@{name='theme-retains-open';pass=$true;metrics=$dark}
 Click-Label 'Code Inspect recorded program'
 Start-Sleep -Milliseconds 260
 $closed=Metrics
 Assert-Near 'collapsed after presence hold' $closed.row.H 24
 [CodeNativeR116]::Key($handle,0x20)
 $reopened=Metrics
 if($reopened.expanded.code -ne $true){throw 'Disclosure Space did not reopen'}
 $run.states+=@{name='disclosure-space';pass=$true;metrics=$reopened}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native run_code: disclosure, wrap/Enter, theme retention, Space reopen at scale'+$open.scale)
} catch {
 $run.error=$_.Exception.Message;$run.status='failed';throw
} finally {
 try{Verify-Owner;[CodeNativeR116]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
