param([Parameter(Mandatory=$true)][int]$OwnerId,[switch]$VerifyTreeLayout,[switch]$VerifyToolRows)
$ErrorActionPreference='Stop'
if($VerifyTreeLayout -and $VerifyToolRows){throw 'Choose one fixture revision'}
$fixturePath=if($VerifyToolRows){'C:\AI\WhaleIsle-MyGo\.devdata\r114-inspector-fixture.exe'}elseif($VerifyTreeLayout){'C:\AI\WhaleIsle-MyGo\.devdata\r113-inspector-fixture.exe'}else{'C:\AI\WhaleIsle-MyGo\.devdata\r112-inspector-fixture.exe'}
$prefix=if($VerifyToolRows){'C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-row-pass258-native'}elseif($VerifyTreeLayout){'C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\tool-tree-pass257-native-layout'}else{'C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-pass256-native-nested-tools'}
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId='+$OwnerId)
if($null -eq $owner -or $owner.ExecutablePath -ne $fixturePath -or $OwnerId -in 20012,16008){throw 'Wrong owned fixture'}
$ownedStart=(Get-Process -Id $OwnerId).StartTime.ToUniversalTime()
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class InspectorNestedR112 {
 [StructLayout(LayoutKind.Sequential)] public struct Point{public int X,Y;}
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string name,string title);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h,int n);
 [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] static extern bool SetForegroundWindow(IntPtr h);
 public static uint Owner(IntPtr h){uint p;GetWindowThreadProcessId(h,out p);return p;}
 public static void Verify(IntPtr h,int p){if(h==IntPtr.Zero||Owner(h)!=(uint)p)throw new Exception("Fixture window ownership changed");}
 public static IntPtr Surface(IntPtr h){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero||Owner(s)!=Owner(h))throw new Exception("Owned surface absent");return s;}
 public static object Activate(IntPtr h){bool activated=SetForegroundWindow(h);return new {returned=activated,foregroundVerified=GetForegroundWindow()==h,foregroundOwner=Owner(GetForegroundWindow())};}
 public static void FocusEvent(IntPtr h){SendMessage(Surface(h),7,IntPtr.Zero,IntPtr.Zero);}
 public static void Click(IntPtr h,int x,int y){var s=Surface(h);var p=new Point{X=x,Y=y};if(!ScreenToClient(s,ref p))throw new Exception("ScreenToClient failed");var l=new IntPtr((p.Y<<16)|(p.X&65535));SendMessage(s,0x200,IntPtr.Zero,l);SendMessage(s,0x201,new IntPtr(1),l);SendMessage(s,0x202,IntPtr.Zero,l);}
 public static void Key(IntPtr h,int key){var s=Surface(h);SendMessage(s,0x100,new IntPtr(key),new IntPtr(1));SendMessage(s,0x101,new IntPtr(key),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Close(IntPtr h){if(!PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero))throw new Exception("Owned close failed");}
}
'@
[void][InspectorNestedR112]::SetThreadDpiAwarenessContext([IntPtr](-4))
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[InspectorNestedR112]::Verify($handle,$OwnerId)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$run=[ordered]@{schema=1;startedAt=[datetime]::UtcNow.ToString('o');finishedAt=$null;status='running';scope='Win32-injected clicks and keys on ownership-verified MyGoSurface; WM_SETFOCUS synthetic; no native-hover or hardware-input claim';fixture=@{pid=$OwnerId;path=$fixturePath;startTimeUtc=$ownedStart.ToString('o');closed=$false;exited=$false};activation=$null;states=@();error=$null}
function Verify-Owner {
 $current=Get-Process -Id $OwnerId -ErrorAction Stop
 if($current.StartTime.ToUniversalTime() -ne $ownedStart){throw 'Owned PID start time changed'}
 [InspectorNestedR112]::Verify($handle,$OwnerId)
}
function Metrics {
 Verify-Owner
 $path=$prefix+'-metrics.json'
 $previous=if(Test-Path -LiteralPath $path){(Get-Item -LiteralPath $path).LastWriteTimeUtc}else{[datetime]::MinValue}
 [InspectorNestedR112]::Key($handle,0x78)
 for($attempt=0;$attempt -lt 100;$attempt++){
  if((Test-Path -LiteralPath $path) -and (Get-Item -LiteralPath $path).LastWriteTimeUtc -gt $previous){
   try{$value=Get-Content -LiteralPath $path -Raw|ConvertFrom-Json;if($null -ne $value.time){return $value}}catch{# The next bounded read handles a writer still replacing its JSON.
   }
  }
  Start-Sleep -Milliseconds 15
 }
 throw 'No fresh F9 acknowledgement'
}
function Click([string]$name){
 Verify-Owner
 $condition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,$name)
 $elements=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,$condition)
 $match=$elements|Where-Object {$_.Current.ControlType -eq [System.Windows.Automation.ControlType]::Button -and $_.Current.BoundingRectangle.Width -gt 0}|Select-Object -First 1
 if($null -eq $match){throw ('Missing owned button '+$name)}
 $r=$match.Current.BoundingRectangle
 [InspectorNestedR112]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))
}
function Check-Record([string]$record,[string]$call){
 Click $record
 $landed=$null
 for($attempt=0;$attempt -lt 40;$attempt++){
  $metric=Metrics
  if($metric.selected -eq $record -and $metric.highlight -eq 'tool:root' -and $metric.reveal.call -eq $call -and $metric.reveal.ageMs -ge 0 -and $metric.reveal.rect.H -gt 0){$landed=$metric;break}
  Start-Sleep -Milliseconds 20
 }
 if($null -eq $landed){throw ('No nested native landing '+$record+': '+($metric|ConvertTo-Json -Depth 10 -Compress))}
 $actual=$landed.reveal.rect
 $expected=@($landed.reveal.calls|Where-Object {$_.owner -eq 'tool:root' -and $_.call -eq $call})[0].visible
 foreach($field in @('X','Y','W','H')){if([Math]::Abs($actual.$field-$expected.$field) -gt 0.02){throw ('Wrong child flash rectangle '+$call)}}
 $bodies=@($landed.expanded.PSObject.Properties|Where-Object {$_.Name -like '*|tool:*' -and $_.Value})
 if($bodies.Count){throw 'Reveal opened tool argument/output bodies'}
 $run.states+=@{record=$record;call=$call;passed=$true;metrics=$landed}
 Write-Output ('PASS '+$record+' -> '+$call+' subtree')
 Click 'Close'
 Start-Sleep -Milliseconds 90
}
try {
 [void][InspectorNestedR112]::ShowWindow($handle,9)
 $run.activation=[InspectorNestedR112]::Activate($handle)
 [InspectorNestedR112]::FocusEvent($handle)
 $initial=Metrics
 if($initial.mode -ne 'session-log'){throw 'Wrong native fixture scenario'}
 Check-Record 'event:13' 'opaque-grandchild'
 Check-Record 'event:11' 'child'
 Check-Record 'event:14' 'unrelated-sibling'
 Check-Record 'event:10' 'root'
 $byCall=@{};foreach($state in $run.states){$byCall[$state.call]=$state.metrics.reveal.rect}
 $parent=$byCall['child'];$leaf=$byCall['opaque-grandchild'];$whole=$byCall['root']
 if($parent.H -le $leaf.H -or $parent.Y -gt $leaf.Y -or $parent.Y+$parent.H -lt $leaf.Y+$leaf.H -or $whole.H -le $parent.H){throw 'Parent subtree ancestry was lost'}
 if($VerifyTreeLayout -or $VerifyToolRows){
  $scale=$run.states[0].metrics.reveal.scale
  $indent=30+[Math]::Max(1,[Math]::Floor(.5*$scale))/$scale
  $sibling=$byCall['unrelated-sibling']
  $checks=@(
   @{name='child indent';actual=$parent.X-$whole.X;expected=$indent},
   @{name='grandchild indent';actual=$leaf.X-$parent.X;expected=$indent},
   @{name='first child top';actual=$parent.Y-$whole.Y;expected=28},
   @{name='grandchild top';actual=$leaf.Y-$parent.Y;expected=28},
   @{name='root subtree';actual=$whole.H;expected=110},
   @{name='child subtree';actual=$parent.H;expected=54},
   @{name='leaf height';actual=$leaf.H;expected=24},
   @{name='sibling height';actual=$sibling.H;expected=24},
   @{name='sibling gap';actual=$sibling.Y-$parent.Y-$parent.H;expected=4},
   @{name='child right edge';actual=$parent.X+$parent.W-$whole.X-$whole.W;expected=0},
   @{name='grandchild right edge';actual=$leaf.X+$leaf.W-$whole.X-$whole.W;expected=0}
  )
  foreach($check in $checks){if([Math]::Abs($check.actual-$check.expected) -gt .025){throw ('Native tree geometry '+$check.name+': '+$check.actual+' expected '+$check.expected)}}
  $run['layoutChecks']=$checks
  Write-Output ('PASS native tree layout at '+$scale+' scale: '+$checks.Count+' measured assertions')
 }
 $run.status='pass-injected-surface'
 Write-Output 'PASS four actual native Inspector -> tool subtree selections; ancestor rectangles and closed call bodies'
} catch {
 $run.error=$_.Exception.Message
 $run.status='failed'
 throw
} finally {
 try{Verify-Owner;[InspectorNestedR112]::Close($handle);$run.fixture.closed=$true;for($attempt=0;$attempt -lt 100;$attempt++){if($null -eq (Get-Process -Id $OwnerId -ErrorAction SilentlyContinue)){$run.fixture.exited=$true;break};Start-Sleep -Milliseconds 30}}catch{$run.fixture['closeError']=$_.Exception.Message}
 $run.finishedAt=[datetime]::UtcNow.ToString('o')
 $run|ConvertTo-Json -Depth 24|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
}
