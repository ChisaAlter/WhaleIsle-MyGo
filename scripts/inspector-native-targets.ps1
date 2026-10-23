param([Parameter(Mandatory=$true)][int]$OwnerId)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;using System.Runtime.InteropServices;using System.Drawing;using System.Drawing.Imaging;
public class InspectorTargetR109 {
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
 public static void Close(IntPtr h){PostMessage(h,0x10,IntPtr.Zero,IntPtr.Zero);}
 public static void Capture(IntPtr h,string path){Rect r;GetWindowRect(h,out r);using(var b=new Bitmap(r.R-r.L,r.B-r.T)){using(var g=Graphics.FromImage(b)){var dc=g.GetHdc();try{if(!PrintWindow(h,dc,2))throw new Exception("Capture failed");}finally{g.ReleaseHdc(dc);}}b.Save(path,ImageFormat.Png);}}
}
'@
[void][InspectorTargetR109]::SetThreadDpiAwarenessContext([IntPtr](-4))
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($owner.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r109-inspector-fixture.exe' -or $OwnerId -in 20012,16008){throw 'Wrong owned fixture'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
if($handle -eq 0){throw 'Owned native window missing'}
[void][InspectorTargetR109]::ShowWindow($handle,9)
[void][InspectorTargetR109]::SetForegroundWindow($handle)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-pass253-native-targets'
function Find([string]$name){$all=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition);foreach($element in $all){if($element.Current.Name -eq $name){return $element.Current.BoundingRectangle}};throw ('Missing '+$name)}
function Metrics {[InspectorTargetR109]::Key($handle,0x78);Start-Sleep -Milliseconds 45;Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw|ConvertFrom-Json}
function ClickRecord([string]$name){$r=Find $name;[InspectorTargetR109]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))}
function Check([string]$name,[string]$want,[bool]$keepFold=$false){
 ClickRecord $name
 $landed=$null
 for($i=0;$i -lt 16;$i++){$metric=Metrics;if($metric.highlight -eq $want -and $metric.reveal.ageMs -ge 0 -and $metric.reveal.rect.W -gt 0 -and $metric.reveal.rect.H -gt 0){$landed=$metric;break};Start-Sleep -Milliseconds 25}
 if($null -eq $landed){throw ('Wrong desktop target '+$name+': '+($metric|ConvertTo-Json -Depth 8 -Compress))}
 if($keepFold -and $landed.expanded.'transcript-turn:fixture:fixture:0'){throw 'Exact answer opened unrelated Turn process group'}
 $script:states+=@{record=$name;expected=$want;metrics=$landed}
 [InspectorTargetR109]::Capture($handle,$prefix+'-'+($name -replace '[^a-zA-Z0-9]','-')+'.png')
 Start-Sleep -Milliseconds 760
 ClickRecord 'Close'
 Start-Sleep -Milliseconds 120
}
$states=@()
try {
 Check 'chat:m40/text/2' 'm40/text/2' $true
 Check 'chat:m40' 'm40'
 Check 'tool/result' 'tool:root'
 Check 'tool/ptc-dispatch' 'tool:root'
 Check 'block-start' 'm40/reasoning/0'
 Check 'text-delta' 'm40'
 $states|ConvertTo-Json -Depth 10|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
 'PASS desktop Inspector exact text, second text, root result, nested result, reasoning and response part ownership'
} finally { [InspectorTargetR109]::Close($handle) }
