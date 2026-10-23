param([int]$OwnerId)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;using System.Threading;using System.Runtime.InteropServices;using System.Drawing;using System.Drawing.Imaging;
public class InspectorSmoothR107 {
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h,int c);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p,IntPtr a,string c,string t);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern bool PrintWindow(IntPtr h,IntPtr dc,uint f);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out Rect r);
 public struct Point{public int X,Y;} public struct Rect{public int L,T,R,B;}
 public static void Click(IntPtr h,int x,int y){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero)throw new Exception("owned surface absent");var p=new Point{X=x,Y=y};ScreenToClient(s,ref p);var l=new IntPtr((p.Y<<16)|(p.X&65535));PostMessage(s,0x200,IntPtr.Zero,l);PostMessage(s,0x201,new IntPtr(1),l);PostMessage(s,0x202,IntPtr.Zero,l);}
 public static void Key(IntPtr h,int k){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);PostMessage(s,0x100,new IntPtr(k),new IntPtr(1));PostMessage(s,0x101,new IntPtr(k),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Pulse(IntPtr h,int x,int y,int key){Click(h,x,y);Thread.Sleep(100);Key(h,key);}
 public static void Capture(IntPtr h,string path){Rect r;GetWindowRect(h,out r);using(var b=new Bitmap(r.R-r.L,r.B-r.T)){using(var g=Graphics.FromImage(b)){var dc=g.GetHdc();try{if(!PrintWindow(h,dc,2))throw new Exception("Capture failed");}finally{g.ReleaseHdc(dc);}}b.Save(path,ImageFormat.Png);}}
}
'@
[void][InspectorSmoothR107]::SetThreadDpiAwarenessContext([IntPtr](-4))
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($owner.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r107-inspector-fixture.exe'){throw 'Wrong owned fixture'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[void][InspectorSmoothR107]::ShowWindow($handle,9)
[void][InspectorSmoothR107]::SetForegroundWindow($handle)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-native-pass251-smooth-desktop'
function Find([string]$name){$all=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition);foreach($e in $all){if($e.Current.Name -eq $name){return $e.Current.BoundingRectangle}};throw ('Missing '+$name)}
function Metrics {[InspectorSmoothR107]::Key($handle,0x78);Start-Sleep -Milliseconds 350;Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw|ConvertFrom-Json}
function ClickRecord([string]$name,[int]$key){$r=Find $name;[InspectorSmoothR107]::Pulse($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2),$key)}
function Settled {for($i=0;$i -lt 10;$i++){$metric=Metrics;if(-not $metric.reveal.pending -and -not $metric.reveal.moving -and $metric.highlight -eq ''){return $metric};Start-Sleep -Milliseconds 200};throw ('Desktop reveal did not settle: '+($metric|ConvertTo-Json -Depth 5 -Compress))}
Start-Sleep -Milliseconds 200
$states=@();$initial=Metrics;$states+=@{phase='initial';metrics=$initial}
# Observe a committed moving frame, then send Escape as its own surface event.
$r=Find 'assistant/7';[InspectorSmoothR107]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))
$moving=$null
for($i=0;$i -lt 300;$i++){try{$frame=Get-Content -LiteralPath ($prefix+'-frame.json') -Raw|ConvertFrom-Json;if($frame.reveal.moving -and [math]::Abs($frame.scroll-$initial.scroll) -gt .5){$moving=$frame;break}}catch{};Start-Sleep -Milliseconds 10}
if($null -eq $moving){throw 'No actual moving desktop frame observed'}
[InspectorSmoothR107]::Key($handle,0x1b)
Start-Sleep -Milliseconds 200
$cancel=Metrics
if($cancel.reveal.pending -or $cancel.reveal.moving -or $cancel.highlight -ne ''){throw 'Escape did not cancel running desktop navigation'}
Start-Sleep -Milliseconds 1500
$late=Metrics
if([math]::Abs($late.scroll-$cancel.scroll) -gt .01 -or $late.reveal.ageMs -ge 0){throw 'Cancelled desktop reveal resumed'}
$states+=@{phase='escape';moving=$moving;metrics=$cancel;late=$late}
[InspectorSmoothR107]::Capture($handle,$prefix+'-cancel.png')
# Ordinary A does not interrupt; the earlier record lands and its flash expires.
ClickRecord 'assistant/7' 0x41
Start-Sleep -Milliseconds 2100
$landed=Settled
if($landed.scroll -ge $initial.scroll -or $landed.reveal.pending -or $landed.reveal.moving -or $landed.highlight -ne ''){throw 'Ordinary key prevented desktop landing'}
$states+=@{phase='ordinary-key-landed';metrics=$landed}
[InspectorSmoothR107]::Capture($handle,$prefix+'-landed.png')
ClickRecord 'assistant/30' 0x41
Start-Sleep -Milliseconds 2100
$latest=Settled
if([math]::Abs($latest.scroll-$latest.max) -gt 2 -or $latest.reveal.pending -or $latest.reveal.moving -or $latest.highlight -ne ''){throw 'Latest smooth desktop navigation failed'}
$states+=@{phase='latest';metrics=$latest}
[InspectorSmoothR107]::Capture($handle,$prefix+'-latest.png')
$states|ConvertTo-Json -Depth 8|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
'PASS desktop Inspector smooth navigation, Escape freeze, ordinary key continuation, latest center and expiry'
