param([Parameter(Mandatory=$true)][int]$OwnerId,[ValidateSet('','-final','-verified','-release')][string]$Suffix='')
$ErrorActionPreference='Stop'
$fixturePath='C:\AI\WhaleIsle-MyGo\.devdata\r123-details-fixture.exe'
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-details-pass267-native'+$Suffix
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008,12516){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class DetailsNativeR123 {
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
[void][DetailsNativeR123]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[DetailsNativeR123]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');status='running';scope='Injected focus/click/key on exact owned MyGoSurface and UIAutomation link Invoke. No Host, browser opening, web requests, code execution, OS clipboard or hardware input. Recorded JSON only; navigation captures payloads without a Host or goal mutation.';fixture=@{pid=$OwnerId;path=$fixturePath;startUtc=$ownedStart.ToString('o');closed=$false;exited=$false};states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'PID recycled'}
 [DetailsNativeR123]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [DetailsNativeR123]::Key($handle,0x78)
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
function Click-Label([string]$name){$r=(Element-Label $name).Current.BoundingRectangle;[DetailsNativeR123]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Near([string]$name,[double]$actual,[double]$expected){if([Math]::Abs($actual-$expected) -gt .03){throw ($name+': '+$actual+' expected '+$expected)}}
function Inspect-Element([int]$index=0){
 Verify-Owner
 $condition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,'Inspect')
 $elements=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,$condition)|Where-Object {$_.Current.ControlType -eq [System.Windows.Automation.ControlType]::Button -and $_.Current.BoundingRectangle.Width -gt 0}|Sort-Object {$_.Current.BoundingRectangle.Y}
 if(@($elements).Count -le $index){throw 'Missing owned Inspect button'}
 return @($elements)[$index]
}
function Click-Element([object]$e){$r=$e.Current.BoundingRectangle;[DetailsNativeR123]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Assert-Goal([object]$m,[int]$count){
 if(-not $m.expanded.'goal-message|tool:native-goal' -or @($m.inspected).Count -ne $count){throw 'Inspect action toggled owner or count changed'}
 foreach($event in $m.inspected){if($event.CallID -ne 'native-goal' -or $event.MessageID -ne 'goal-message' -or $event.Seq -ne 207){throw 'Inspect lost wire identity'}}
}
try {
 [DetailsNativeR123]::Focus($handle)
 $initial=Metrics;Assert-Near 'collapsed goal' $initial.row.H 24
 Click-Label 'View goal Recorded objective';$opened=Metrics;Assert-Goal $opened 0
 [void](Element-Label 'Status');[void](Element-Label 'Awaiting continuation');[void](Element-Label 'Rounds');[void](Element-Label '123 / 256')
 $r=(Inspect-Element).Current.BoundingRectangle
 $actualY=($r.Y-$root.Current.BoundingRectangle.Y)/$opened.scale
 # UIA exposes integer physical-pixel bounds for both owner and child. Their
 # difference permits the sum of two one-pixel rounding errors; logical boxes
 # are checked without this allowance in the software layout regression.
 $inspectY=@{actual=$actualY;expected=($opened.row.Y+36);maxError=(2/$opened.scale)}
 if([Math]::Abs($inspectY.actual-$inspectY.expected) -gt $inspectY.maxError){throw 'Inspect UIA position exceeds physical-pixel rounding'}
 $run.inspectUIAPosition=$inspectY
 Click-Element (Inspect-Element);$clicked=Metrics;Assert-Goal $clicked 1
 [DetailsNativeR123]::Key($handle,0x20);$space=Metrics;Assert-Goal $space 2
 [DetailsNativeR123]::Key($handle,0x0D);$enter=Metrics;Assert-Goal $enter 3
 (Inspect-Element).GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke();$invoked=Metrics;Assert-Goal $invoked 4
 $run.states+=@{name='details-click-Space-Enter-UIA-wire-identity';pass=$true;metrics=$invoked}
 Click-Label 'View goal No goal';$empty=Metrics
 if(-not $empty.expanded.'empty-message|tool:native-empty'){throw 'Empty goal not expandable'}
 [void](Element-Label 'No goal')
 $run.states+=@{name='empty-goal-card-and-Inspect';pass=$true;metrics=$empty}
 Click-Label 'Toggle width';$narrow=Metrics;Assert-Goal $narrow 4
 Assert-Near 'narrow width' $narrow.row.W 320
 Click-Label 'Toggle theme';$dark=Metrics;Assert-Goal $dark 4
 if(-not $dark.dark){throw 'Theme not changed'}
 $run.states+=@{name='narrow-theme-keeps-navigation';pass=$true;metrics=$dark}
 Click-Label ('Tool call unknown_tool '+[char]0xB7+' recorded');$generic=Metrics
 [void](Element-Label 'IN');[void](Element-Label 'OUT')
 $last=Inspect-Element 2
 Click-Element $last;$genericClick=Metrics
 if(@($genericClick.inspected).Count -ne 5 -or $genericClick.inspected[4].CallID -ne 'native-generic' -or $genericClick.inspected[4].MessageID -ne 'generic-message' -or $genericClick.inspected[4].Seq -ne 209){throw 'Generic Inspect lost identity'}
 if(-not $genericClick.expanded.'generic-message|tool:native-generic'){throw 'Generic Inspect folded owner'}
 $run.states+=@{name='generic-inline-Inspect';pass=$true;metrics=$genericClick}
 $run.status='pass-injected-native-surface'
 Write-Output ('PASS native goal details: recorded fields, click Space Enter UIA identity, empty state, narrow/theme, generic Inspect scale'+$dark.scale)
} catch {$run.error=$_.Exception.Message;$run.status='failed';throw} finally {
 try{Verify-Owner;[DetailsNativeR123]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
