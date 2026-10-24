param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final','-verified','-release')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r122-image-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-image-pass266-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class ImageNativeR122 {
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
[void][ImageNativeR122]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[ImageNativeR122]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key on exact owned MyGoSurface and UIAutomation link Invoke. No Host, browser opening, web requests, code execution, OS clipboard or hardware input. Recorded local PNG and injected attachment loader only.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [ImageNativeR122]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [ImageNativeR122]::Key($handle,0x78)
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
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[ImageNativeR122]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Near([string]$name,[double]$actual,[double]$expected){if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}}
function Assert-Open([object]$m){if(-not $m.expanded.'|tool:native-image'){throw 'Image action toggled outer disclosure'}}
try {
 [ImageNativeR122]::Focus($handle)
 $initial=Metrics;Assert-Near 'collapsed row' $initial.row.H 24
 Click-Label 'argument.png';$header=Metrics
 if(@($header.files).Count -ne 1 -or $header.files[0] -ne '/work/recorded.png' -or $header.expanded.'|tool:native-image'){throw 'Header file target or disclosure'}
 $r=(Element-Label 'Read image argument.png').Current.BoundingRectangle
 [ImageNativeR122]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $loaded=Metrics;Assert-Open $loaded
 [void](Element-Label '640x360px, 4096 bytes');[void](Element-Label "  Recorded extra text`nSecond raw line")
 Click-Label 'Loaded sample, click to view original';$open=Metrics;Assert-Open $open
 if(-not $open.lightbox -or @($open.opened).Count -ne 1){throw 'Thumbnail did not open modal'}
 [void](Element-Label 'Original image preview');[void](Element-Label 'Close original image preview')
 [ImageNativeR122]::Key($handle,0x1B);$escape=Metrics;Assert-Open $escape
 if($escape.lightbox){throw 'Escape did not close'}
 [ImageNativeR122]::Key($handle,0x20);$space=Metrics
 if(-not $space.lightbox -or @($space.opened).Count -ne 2){throw 'Space did not restore opener'}
 Click-Label 'Close original image preview';$closed=Metrics
 if($closed.lightbox){throw 'Close did not dismiss'}
 [ImageNativeR122]::Key($handle,0x0D);$enter=Metrics
 if(-not $enter.lightbox -or @($enter.opened).Count -ne 3){throw 'Enter did not restore opener'}
 [ImageNativeR122]::Key($handle,0x1B);$escape=Metrics
 (Element-Label 'Loaded sample, click to view original').GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke();$uia=Metrics
 if(-not $uia.lightbox -or @($uia.opened).Count -ne 4){throw 'UIA Invoke did not open image'}
 [ImageNativeR122]::Key($handle,0x1B);$escape=Metrics
 $run.states+=@{name='click-Space-Enter-UIA-modal-Escape';pass=$true;metrics=$escape}
 $rows=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,(New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,'Read image argument.png')))
 $r=($rows|Sort-Object {$_.Current.BoundingRectangle.Y}|Select-Object -Last 1).Current.BoundingRectangle
 [ImageNativeR122]::Click($handle,[int]($r.Right-8),[int]($r.Y+12*$initial.scale))
 $failed=Metrics
 [void](Element-Label 'Image failed to load; click to retry')
 if($failed.loads -ne 1){throw 'Initial failed load repeated'}
 Click-Label 'Image failed to load; click to retry'
 for($attempt=0;$attempt -lt 60;$attempt++){try{$retryElement=Element-Label 'Retry sample, click to view original';break}catch{Start-Sleep -Milliseconds 20}}
 $recovered=Metrics
 if($recovered.loads -ne 2 -or -not $recovered.expanded.'|tool:native-image-retry'){throw 'Explicit retry count/disclosure'}
 Click-Label 'Retry sample, click to view original';$retryOpen=Metrics
 if(-not $retryOpen.lightbox -or @($retryOpen.opened).Count -ne 5){throw 'Recovered image not openable'}
 [ImageNativeR122]::Key($handle,0x1B);$escape=Metrics
 $run.states+=@{name='failure-keeps-text-explicit-retry';pass=$true;metrics=$escape}
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Open $dark
 if(-not $dark.dark){throw 'Theme failed'};Assert-Near 'theme image height' $dark.row.H $loaded.row.H
 $run.states+=@{name='theme-keeps-image-card';pass=$true;metrics=$dark}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native image: metadata, thumbnail click Space Enter UIA, modal Escape/focus, explicit retry, theme scale'+$dark.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[ImageNativeR122]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
