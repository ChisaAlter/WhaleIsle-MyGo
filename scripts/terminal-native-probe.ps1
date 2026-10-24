param([Parameter(Mandatory=$true)][int]$OwnerId)
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r115-terminal-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\terminal-pass259-native-accessible-header'
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class TerminalNativeR115 {
 [StructLayout(LayoutKind.Sequential)] public struct Point{public int X,Y;}
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string name,string title);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] static extern bool SetForegroundWindow(IntPtr h);
 public static uint Owner(IntPtr h){uint p;GetWindowThreadProcessId(h,out p);return p;}
 public static void Verify(IntPtr h,int p){if(h==IntPtr.Zero||Owner(h)!=(uint)p)throw new Exception("Fixture ownership changed");}
 public static IntPtr Surface(IntPtr h){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero||Owner(s)!=Owner(h))throw new Exception("Owned surface absent");return s;}
 public static object Activate(IntPtr h){bool activated=SetForegroundWindow(h);return new {returned=activated,foregroundVerified=GetForegroundWindow()==h,foregroundOwner=Owner(GetForegroundWindow())};}
 public static void Focus(IntPtr h){SendMessage(Surface(h),7,IntPtr.Zero,IntPtr.Zero);}
 public static void Click(IntPtr h,int x,int y){var s=Surface(h);var p=new Point{X=x,Y=y};if(!ScreenToClient(s,ref p))throw new Exception("ScreenToClient failed");var l=new IntPtr((p.Y<<16)|(p.X&65535));SendMessage(s,0x200,IntPtr.Zero,l);SendMessage(s,0x201,new IntPtr(1),l);SendMessage(s,0x202,IntPtr.Zero,l);}
 public static void Key(IntPtr h,int key){var s=Surface(h);SendMessage(s,0x100,new IntPtr(key),new IntPtr(1));SendMessage(s,0x101,new IntPtr(key),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Close(IntPtr h){if(!PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero))throw new Exception("Owned close failed");}
}
'@
[void][TerminalNativeR115]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[TerminalNativeR115]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected WM_SETFOCUS/click/key events on the exact owned MyGoSurface. No OS hardware hover/keyboard or original Electron claim. No clipboard write or displayed command execution.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [TerminalNativeR115]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [TerminalNativeR115]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){
   try{$value=Get-Content -LiteralPath $path -Raw -Encoding UTF8|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{# Bounded next read handles atomic writer completion.
   }
  }
  Start-Sleep -Milliseconds 15
 }
 throw 'No fresh F9 metrics'
}
function Click-Label([string]$name){
 Verify-Owner
 $condition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,$name)
 $elements=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,$condition)
 $match=$elements|Where-Object {$_.Current.BoundingRectangle.Width -gt 0}|Select-Object -First 1
 if($null -eq $match){throw ('Missing visible owned label '+$name)}
 $r=$match.Current.BoundingRectangle
 [TerminalNativeR115]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))
}
function Assert-Near([string]$name,[double]$actual,[double]$expected){
 if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}
}
try {
 $run.activation=[TerminalNativeR115]::Activate($handle)
 [TerminalNativeR115]::Focus($handle)
 $initial=Metrics
 Assert-Near 'collapsed foreground' $initial.row.H 24
 Click-Label 'Pwsh Inspect recorded terminal'
 $open=$null
 for($attempt=0;$attempt -lt 50;$attempt++){$metric=Metrics;if($metric.expanded.foreground -eq $true -and $metric.row.H -gt 400){$open=$metric;break};Start-Sleep -Milliseconds 15}
 if($null -eq $open){throw 'Native disclosure did not open'}
 $border=[Math]::Max(1,[Math]::Floor(.5*$open.scale))/$open.scale
 $expected=24+8+168+$border+248+2*$border+1
 Assert-Near 'expanded terminal row including owner marker' $open.row.H $expected
 if($open.foreground.ExitCode -ne 7 -or $open.foreground.Output -match '\[exit code:'){throw 'Terminal model failed to separate exit marker'}
 $run.states+=@{name='clicked-open';pass=$true;metrics=$open}
 # Click leaves the disclosure focused. Space must close; another Space reopens.
 [TerminalNativeR115]::Key($handle,0x20)
 Start-Sleep -Milliseconds 260
 $closed=Metrics
 Assert-Near 'closed body after presence hold' $closed.row.H 24
 if($closed.expanded.foreground -eq $true){throw 'Space did not close'}
 $run.states+=@{name='space-close';pass=$true;metrics=$closed}
 [TerminalNativeR115]::Key($handle,0x20)
 Start-Sleep -Milliseconds 260
 $reopen=Metrics
 if($reopen.expanded.foreground -ne $true){throw 'Space did not reopen'}
 $run.states+=@{name='space-reopen';pass=$true;metrics=$reopen}
 Click-Label 'Toggle theme'
 $dark=Metrics
 if(-not $dark.dark -or $dark.expanded.foreground -ne $true){throw 'Theme switch reset disclosure'}
 $run.states+=@{name='theme-retains-open';pass=$true;metrics=$dark}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native terminal: mouse disclosure, Space close/reopen, geometry scale'+$open.scale+', exit-marker separation and theme retention')
} catch {
 $run.error=$_.Exception.Message;$run.status='failed';throw
} finally {
 try{Verify-Owner;[TerminalNativeR115]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 16|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
