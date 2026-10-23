param([Parameter(Mandatory=$true)][int]$OwnerId)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;using System.Runtime.InteropServices;using System.Drawing;using System.Drawing.Imaging;
public class InspectorPickR110 {
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h,int c);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p,IntPtr a,string c,string t);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern bool PrintWindow(IntPtr h,IntPtr dc,uint f);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out Rect r);
 public struct Point{public int X,Y;} public struct Rect{public int L,T,R,B;}
 public static void Click(IntPtr h,int x,int y){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero)throw new Exception("Owned surface absent");var p=new Point{X=x,Y=y};ScreenToClient(s,ref p);var l=new IntPtr((p.Y<<16)|(p.X&65535));PostMessage(s,0x200,IntPtr.Zero,l);PostMessage(s,0x201,new IntPtr(1),l);PostMessage(s,0x202,IntPtr.Zero,l);}
 public static void Key(IntPtr h,int k){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);PostMessage(s,0x100,new IntPtr(k),new IntPtr(1));PostMessage(s,0x101,new IntPtr(k),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Text(IntPtr h,string text){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);foreach(var ch in text)PostMessage(s,0x102,new IntPtr(ch),new IntPtr(1));}
 public static void Close(IntPtr h){PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero);}
 public static void Capture(IntPtr h,string path){Rect r;GetWindowRect(h,out r);using(var b=new Bitmap(r.R-r.L,r.B-r.T)){using(var g=Graphics.FromImage(b)){var dc=g.GetHdc();try{if(!PrintWindow(h,dc,2))throw new Exception("Capture failed");}finally{g.ReleaseHdc(dc);}}b.Save(path,ImageFormat.Png);}}
}
'@
[void][InspectorPickR110]::SetThreadDpiAwarenessContext([IntPtr](-4))
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($owner.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r110-inspector-fixture.exe' -or $OwnerId -in 20012,16008){throw 'Wrong owned fixture'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
if($handle -eq 0){throw 'Owned native window missing'}
[void][InspectorPickR110]::ShowWindow($handle,9)
[void][InspectorPickR110]::SetForegroundWindow($handle)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-pass254-native-picks'
function Elements { $root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition) }
function Find([string]$name){foreach($element in (Elements)){if($element.Current.Name -eq $name -and $element.Current.BoundingRectangle.Width -gt 0){return $element.Current.BoundingRectangle}};throw ('Missing '+$name)}
function Click([string]$name){$r=Find $name;[InspectorPickR110]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 100}
function Key([int]$key){[InspectorPickR110]::Key($handle,$key);Start-Sleep -Milliseconds 120}
function Metrics {Key 0x78;Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw|ConvertFrom-Json}
function PickRow([string]$label){
 $row=$null
 foreach($element in (Elements)){if($element.Current.Name -eq ('Pick this chat node '+$label) -and $element.Current.ControlType -eq [System.Windows.Automation.ControlType]::ListItem){$row=$element;break}}
 if($null -eq $row){throw ('Missing Chat row '+$label)}
 $children=$row.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
 foreach($element in $children){if($element.Current.Name -eq 'Pick this chat node'){$r=$element.Current.BoundingRectangle;[InspectorPickR110]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 120;return}}
 throw ('Missing picker in Chat row '+$label)
}
function Check([string]$name,[string]$row,[string]$want,[string]$mode='session-log'){
 Click 'Pick an element in Chat View'
 PickRow $row
 $metric=Metrics
 if($metric.selected -ne $want -or $metric.mode -ne $mode -or $metric.picking -or $metric.notice -ne '' -or $metric.targetPending){throw ('Wrong desktop reverse target '+$name+': '+($metric|ConvertTo-Json -Depth 8 -Compress))}
 $script:states+=@{name=$name;row=$row;expected=$want;metrics=$metric}
 $script:states|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
 [InspectorPickR110]::Capture($handle,$prefix+'-'+$name+'.png')
 Click 'Close'
}
$states=@()
try {
 Key 0x75
 Check 'zero-anchor' 'Sequence zero message.' 'event:0'
 Key 0x75
 Check 'group-first-member' 'Work details' 'event:10'
 Key 0x75
 Check 'response-root' 'Second answer block.' 'event:40'
 Click 'Work details'
 Click 'Used tools'
 Click 'Inspector type filter'
 [InspectorPickR110]::Text($handle,'user/message')
 Start-Sleep -Milliseconds 160
 Click 'Apply filter'
 $before=Metrics
 if($before.filter -ne 'user/message'){throw ('Filter probe selected another suggestion: '+$before.filter)}
 Check 'reasoning-header' 'Thinking Reasoning details.' 'event:40/chunk:0'
 if($states[-1].metrics.filter -ne ''){throw 'Excluded reasoning target retained the tool filter'}
 Click 'Work details'
 Click 'Presentation'
 Click 'Chat Group'
 Click 'Pick an element in Chat View'
 $metric=Metrics
 $group=$metric.groups.PSObject.Properties|Select-Object -First 1
 if($null -eq $group){throw 'No native transcript group'}
 PickRow 'Work details'
 $metric=Metrics
 if($metric.selected -ne ('chat:'+$group.Name) -or $metric.mode -ne 'chat-node' -or $metric.picking -or $metric.notice -ne ''){throw ('Group header did not select its own Chat row: '+($metric|ConvertTo-Json -Depth 8 -Compress))}
 $states+=@{name='chat-group-header';expected=('chat:'+$group.Name);metrics=$metric}
 [InspectorPickR110]::Capture($handle,$prefix+'-chat-group-header.png')
 $states|ConvertTo-Json -Depth 12|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
 'PASS desktop reverse picks: zero anchor, group first member, response root, reasoning header/filter, Chat group header'
} finally { [InspectorPickR110]::Close($handle) }
