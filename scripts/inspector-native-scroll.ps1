param([int]$OwnerId,[ValidateSet('manual-open','manual-fold','manual-latest','auto-open','auto-closed','auto-appended','auto-latest')][string]$Step)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class InspectorScrollR105 {
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h,int c);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h,IntPtr a,int x,int y,int w,int ht,uint f);
 [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h,IntPtr dc,uint f);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out Rect r);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h,uint m,IntPtr wp,IntPtr lp);
 [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p,IntPtr a,string c,string t);
 [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h,ref Point p);
 public struct Point {public int X,Y;}
 public struct Rect {public int L,T,R,B;}
 public static void Click(IntPtr h,int x,int y){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);if(s==IntPtr.Zero)throw new Exception("No owned surface");var p=new Point{X=x,Y=y};ScreenToClient(s,ref p);var lp=new IntPtr((p.Y<<16)|(p.X&65535));PostMessage(s,0x200,IntPtr.Zero,lp);PostMessage(s,0x201,new IntPtr(1),lp);PostMessage(s,0x202,IntPtr.Zero,lp);PostMessage(s,0x200,IntPtr.Zero,new IntPtr((400<<16)|600));}
 public static void Key(IntPtr h,int k){var s=FindWindowEx(h,IntPtr.Zero,"MyGoSurface",null);PostMessage(s,0x100,new IntPtr(k),new IntPtr(1));PostMessage(s,0x101,new IntPtr(k),new IntPtr(unchecked((int)0xc0000001)));}
}
'@
[void][InspectorScrollR105]::SetThreadDpiAwarenessContext([IntPtr](-4))
$process=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($process.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r105-inspector-fixture.exe'){throw 'Unexpected fixture owner'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[void][InspectorScrollR105]::ShowWindow($handle,9)
[void][InspectorScrollR105]::SetForegroundWindow($handle)
[void][InspectorScrollR105]::SetWindowPos($handle,[IntPtr]::Zero,0,0,1193,912,0x0040)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
function Elements {$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)}
function Find([string]$name){foreach($e in (Elements)){if($e.Current.Name -eq $name){return $e}}}
function Click([string]$name){$e=Find $name;if($null -eq $e){throw ('Missing '+$name)};$r=$e.Current.BoundingRectangle;[InspectorScrollR105]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 250}
Start-Sleep -Milliseconds 200
switch($Step){
 'manual-fold' {
  $row=Find 'Inspector record event:20';if($null -eq $row){throw 'Missing Turn'}
  $rect=$row.Current.BoundingRectangle;$button=$null
  foreach($e in (Elements)){$r=$e.Current.BoundingRectangle;if($e.Current.Name -eq 'Collapse children 20' -and $r.Y -ge $rect.Y -and $r.Y -lt ($rect.Y+$rect.Height)){$button=$e;break}}
  if($null -eq $button){throw 'Missing Turn disclosure'}
  $r=$button.Current.BoundingRectangle;[InspectorScrollR105]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2))
 }
 'manual-latest' {Click 'Follow Latest'}
 'auto-closed' {[InspectorScrollR105]::Key($handle,0x77)}
 'auto-appended' {[InspectorScrollR105]::Key($handle,0x77)}
 'auto-latest' {Click 'Follow Latest'}
}
Start-Sleep -Milliseconds 400
[InspectorScrollR105]::Key($handle,0x78)
Start-Sleep -Milliseconds 200
$prefix='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-native-pass249'
$metrics=Get-Content -LiteralPath ($prefix+'-metrics.json') -Raw | ConvertFrom-Json
$records=@();$header=Find 'Inspector table header';$origin=$header.Current.BoundingRectangle.Y
foreach($e in (Elements)){$name=$e.Current.Name;if($name -like 'Inspector record *'){$r=$e.Current.BoundingRectangle;$records+=@{key=$name.Substring(17);top=($r.Y-$origin)/1.5;height=$r.Height/1.5}}}
$state=@{offset=$metrics.offset;max=$metrics.max;padding=$metrics.padding;mode=$metrics.mode;viewportHeight=$metrics.viewportHeight;records=$records;physicalScale=1.5}
if($Step -in @('manual-fold','auto-closed','auto-appended') -and [math]::Abs($metrics.offset-600) -gt 1){throw ('Anchor moved: '+($state|ConvertTo-Json -Depth 4 -Compress))}
if($Step -like '*latest' -and ($metrics.padding -ne 0 -or [math]::Abs($metrics.offset-$metrics.max) -gt 1)){throw 'Latest did not clear blank'}
$state|ConvertTo-Json -Depth 6|Set-Content -Encoding UTF8 -LiteralPath ($prefix+'-'+$Step+'-state.json')
$rect=New-Object InspectorScrollR105+Rect;[void][InspectorScrollR105]::GetWindowRect($handle,[ref]$rect)
$bitmap=New-Object System.Drawing.Bitmap ($rect.R-$rect.L),($rect.B-$rect.T);$graphics=[System.Drawing.Graphics]::FromImage($bitmap);$dc=$graphics.GetHdc()
try{if(-not[InspectorScrollR105]::PrintWindow($handle,$dc,2)){throw 'Capture failed'}}finally{$graphics.ReleaseHdc($dc);$graphics.Dispose()}
try{$bitmap.Save(($prefix+'-'+$Step+'.png'),[System.Drawing.Imaging.ImageFormat]::Png)}finally{$bitmap.Dispose()}
'PASS '+$Step+' offset='+$metrics.offset+' padding='+$metrics.padding+' mode='+$metrics.mode
