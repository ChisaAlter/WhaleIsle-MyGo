param([int]$OwnerId)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System;using System.Runtime.InteropServices;using System.Threading;using System.Drawing;using System.Drawing.Imaging;
public class InspectorChatR106 {
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h,int c);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h,IntPtr a,int x,int y,int w,int ht,uint f);
 [DllImport("user32.dll")] static extern bool PrintWindow(IntPtr h,IntPtr dc,uint f);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out Rect r);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr wp,IntPtr lp);
 [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p,IntPtr a,string c,string t);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 public struct Point {public int X,Y;}
 public struct Rect {public int L,T,R,B;}
 public static void Click(IntPtr h,int x,int y){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero)throw new Exception("No owned surface");var p=new Point{X=x,Y=y};ScreenToClient(s,ref p);var lp=new IntPtr((p.Y<<16)|(p.X&65535));PostMessage(s,0x200,IntPtr.Zero,lp);PostMessage(s,0x201,new IntPtr(1),lp);PostMessage(s,0x202,IntPtr.Zero,lp);}
 public static void Key(IntPtr h,int k){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);PostMessage(s,0x100,new IntPtr(k),new IntPtr(1));PostMessage(s,0x101,new IntPtr(k),new IntPtr(unchecked((int)0xc0000001)));}
 public static void Capture(IntPtr h,string path){Rect r;GetWindowRect(h,out r);using(var b=new Bitmap(r.R-r.L,r.B-r.T)){using(var g=Graphics.FromImage(b)){var dc=g.GetHdc();try{if(!PrintWindow(h,dc,2))throw new Exception("Capture failed");}finally{g.ReleaseHdc(dc);}}b.Save(path,ImageFormat.Png);}}
 public static void ClickCapture(IntPtr h,int x,int y,string path){Click(h,x,y);Thread.Sleep(120);Key(h,0x78);Thread.Sleep(45);Capture(h,path);}
}
'@
[void][InspectorChatR106]::SetThreadDpiAwarenessContext([IntPtr](-4))
$owner=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($owner.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r106-inspector-fixture.exe'){throw 'Unexpected fixture owner'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[void][InspectorChatR106]::ShowWindow($handle,9)
[void][InspectorChatR106]::SetForegroundWindow($handle)
[void][InspectorChatR106]::SetWindowPos($handle,[IntPtr]::Zero,0,0,1820,1000,0x0040)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-native-pass250-chat-desktop'
function ReadMetrics {[InspectorChatR106]::Key($handle,0x78);Start-Sleep -Milliseconds 180;Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw|ConvertFrom-Json}
function Find([string]$name){$all=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition);foreach($e in $all){if($e.Current.Name -eq $name){return $e}}}
Start-Sleep -Milliseconds 300
$states=@();$initial=ReadMetrics;$states+=@{phase='initial';metrics=$initial};[InspectorChatR106]::Capture($handle,$prefix+'-initial.png')
foreach($phase in @('locate','repeat','latest')){
 $name='assistant/7';if($phase -eq 'latest'){$name='assistant/30'}
 $button=Find $name;if($null -eq $button){throw ('Missing '+$name)}
 $r=$button.Current.BoundingRectangle
 [InspectorChatR106]::ClickCapture($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2),$prefix+'-'+$phase+'-active.png')
 $active=Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw|ConvertFrom-Json
 $expected='answer-7';if($phase -eq 'latest'){$expected='answer-30'}
 if($active.highlight -ne $expected -and -not $active.targetPending){throw ('Click did not activate requested target: '+$phase+' '+($active|ConvertTo-Json -Compress))}
 if($phase -eq 'repeat' -and ($active.reveal.pending -or $active.reveal.ageMs -lt 0 -or $active.reveal.ageMs -ge 700 -or $active.reveal.rect.W -le 0)){throw 'Repeated click did not enter active overlay phase'}
 Start-Sleep -Milliseconds 950
 $metrics=ReadMetrics;$states+=@{phase=$phase;active=$active;metrics=$metrics}
 if($metrics.highlight -ne '' -or $metrics.targetPending){throw 'Highlight or deferred request survived pulse'}
 if($phase -ne 'latest' -and $metrics.scroll -ge $initial.scroll){throw 'Offscreen target did not reveal earlier content'}
 if($phase -eq 'latest' -and [math]::Abs($metrics.scroll-$metrics.max) -gt 2){throw 'Latest target did not land at end'}
 [InspectorChatR106]::Capture($handle,$prefix+'-'+$phase+'-expired.png')
}
$states|ConvertTo-Json -Depth 5|Set-Content -LiteralPath ($prefix+'-state.json') -Encoding UTF8
'PASS real native Inspector clicks: earlier target, repeat pulse, latest target; all expire'
