param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final','-verified','-release')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r121-web-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-web-pass265-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class WebNativeR121 {
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
[void][WebNativeR121]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[WebNativeR121]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key on exact owned MyGoSurface and UIAutomation link Invoke. No Host, browser opening, web requests, code execution, OS clipboard or hardware input.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [WebNativeR121]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [WebNativeR121]::Key($handle,0x78)
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
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[WebNativeR121]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Near([string]$name,[double]$actual,[double]$expected){if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}}
function Assert-Open([object]$m){if(-not $m.expanded.'|tool:native-web-search' -or -not $m.expanded.'|tool:native-web-fetch'){throw 'Link action toggled an outer disclosure'}}
try {
 [WebNativeR121]::Focus($handle)
 $initial=Metrics;Assert-Near 'collapsed search' $initial.search.H 24;Assert-Near 'collapsed fetch' $initial.fetch.H 24
 Click-Label 'https://original.example.test/a';$header=Metrics
 if(@($header.opened).Count -ne 1 -or $header.opened[0] -ne 'https://original.example.test/a' -or $header.expanded.'|tool:native-web-fetch'){throw 'Header link opened wrong target or expanded row'}
 [WebNativeR121]::Key($handle,0x0D);$headerEnter=Metrics
 if(@($headerEnter.opened).Count -ne 2){throw 'Header Enter did not open once'}
 [WebNativeR121]::Key($handle,0x20);$headerSpace=Metrics
 if(@($headerSpace.opened).Count -ne 2 -or $headerSpace.expanded.'|tool:native-web-fetch'){throw 'Header Space opened link or toggled row'}
 $r=(Element-Label 'Search recorded query').Current.BoundingRectangle
 [WebNativeR121]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $r=(Element-Label 'Fetch https://original.example.test/a').Current.BoundingRectangle
 [WebNativeR121]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $both=Metrics;Assert-Open $both
 [void](Element-Label 'Recorded answer');[void](Element-Label 'Full recorded snippet.');[void](Element-Label 'Source list truncated');[void](Element-Label 'HTTP 200');[void](Element-Label 'Content truncated')
 $run.states+=@{name='both-structured-cards';pass=$true;metrics=$both}
 Click-Label 'Recorded source';$source=Metrics;Assert-Open $source
 if(@($source.opened).Count -ne 3 -or $source.opened[2] -ne 'https://source.example.test/a'){throw 'Search source target'}
 [WebNativeR121]::Key($handle,0x0D);$sourceEnter=Metrics;Assert-Open $sourceEnter
 if(@($sourceEnter.opened).Count -ne 4){throw 'Source Enter activation'}
 [WebNativeR121]::Key($handle,0x20);$sourceSpace=Metrics;Assert-Open $sourceSpace
 if(@($sourceSpace.opened).Count -ne 4){throw 'Source Space activated anchor'}
 Click-Label 'https://final.example.test/b';$final=Metrics;Assert-Open $final
 if(@($final.opened).Count -ne 5 -or $final.opened[4] -ne 'https://final.example.test/b'){throw 'Final URL target'}
 $run.states+=@{name='original-final-source-click-enter-space';pass=$true;metrics=$final}
 $element=Element-Label 'https://final.example.test/b'
 $pattern=$element.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
 $pattern.Invoke();$accessible=Metrics;Assert-Open $accessible
 if(@($accessible.opened).Count -ne 6 -or $accessible.opened[5] -ne 'https://final.example.test/b'){throw 'Accessible link invoke target'}
 $run.states+=@{name='link-UIA-Invoke';pass=$true;metrics=$accessible}
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Open $dark
 if(-not $dark.dark){throw 'Theme failed'};Assert-Near 'theme search height' $dark.search.H $both.search.H;Assert-Near 'theme fetch height' $dark.fetch.H $both.fetch.H
 $run.states+=@{name='theme-keeps-bodies';pass=$true;metrics=$dark}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native web: original/final/source links, click Enter Space UIA Invoke, theme scale'+$both.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[WebNativeR121]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
