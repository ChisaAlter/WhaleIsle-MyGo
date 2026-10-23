param([int]$OwnerId,[string]$Step='open')
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class InspectorR103 {
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
[void][InspectorR103]::SetThreadDpiAwarenessContext([IntPtr](-4))
$process=Get-CimInstance Win32_Process -Filter ('ProcessId = '+$OwnerId)
if($process.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r104-inspector-fixture.exe'){throw 'Unexpected fixture owner'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[void][InspectorR103]::ShowWindow($handle,9)
[void][InspectorR103]::SetForegroundWindow($handle)
[void][InspectorR103]::SetWindowPos($handle,[IntPtr]::Zero,0,0,1193,912,0x0040)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
function Find([string]$name){$cond=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,$name);$root.FindFirst([System.Windows.Automation.TreeScope]::Descendants,$cond)}
function Click([string]$name){$e=Find $name;if($null -eq $e){throw ('Missing '+$name)};$r=$e.Current.BoundingRectangle;[InspectorR103]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 250}
function Click-ReasoningDisclosure([string]$label){$row=Find 'Inspector record attempt:stream-a/chunk:0';if($null -eq $row){throw 'Missing reasoning block row'};$actionName=$label+' 0';$button=$null;$rowRect=$row.Current.BoundingRectangle;foreach($candidate in $root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)){$r=$candidate.Current.BoundingRectangle;if($candidate.Current.Name -eq $actionName -and $r.Y -ge $rowRect.Y -and $r.Y -lt ($rowRect.Y+$rowRect.Height)){$button=$candidate;break}};if($null -eq $button){throw ('Missing block action '+$label)};$r=$button.Current.BoundingRectangle;[InspectorR103]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 250}
function Apply-InspectorQuery([string]$query){Click 'Inspector type filter';$in=Find 'Inspector type query';$in.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern).SetValue($query);Start-Sleep -Milliseconds 250;Click 'Apply filter'}
Start-Sleep -Milliseconds 200
switch($Step){
 'open' {}
 'manual-fold' {Click-ReasoningDisclosure 'Collapse children'}
 'closed' {Click-ReasoningDisclosure 'Expand children';[InspectorR103]::Key($handle,0x77)}
 'reopened' {Click-ReasoningDisclosure 'Expand children'}
 'filtered' {Apply-InspectorQuery 'reasoning-delta'}
 'filtered-fold' {Click-ReasoningDisclosure 'Collapse children'}
 'cleared' {Apply-InspectorQuery ''}
 'settled' {[InspectorR103]::Key($handle,0x77)}
 'retired' {[InspectorR103]::Key($handle,0x77)}
 default {throw 'Unknown step'}
}
Start-Sleep -Milliseconds 400
$names=@();$records=@()
foreach($e in $root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)){
 $name=$e.Current.Name
 if($name){$names+=$name}
 if($name -like 'Inspector record *'){$r=$e.Current.BoundingRectangle;$selection=$e.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern);$records+=@{key=$name.Substring(17);selected=$selection.Current.IsSelected;x=$r.X;y=$r.Y;w=$r.Width;h=$r.Height}}
}
$wanted=switch($Step){'open'{8};'manual-fold'{6};'closed'{6};'reopened'{9};'filtered'{6};'filtered-fold'{4};'cleared'{6};'settled'{10};'retired'{8}}
if($records.Count -ne $wanted){throw ('Unexpected record count '+$records.Count+' != '+$wanted+' ('+($records.key-join ',')+')')}
if($Step -in @('manual-fold','closed','cleared','settled','retired') -and ('blockType: reasoning '+[char]0xb7+' first second') -notin $names){throw 'Folded reasoning preview missing'}
if($Step -eq 'settled' -and ('attempt:stream-a' -notin $records.key -or 'event:2' -notin $records.key)){throw 'Success must retain separate live and durable roots'}
if($Step -eq 'retired' -and @($records.key|Where-Object{$_ -like 'attempt:*'}).Count -ne 0){throw 'Matching Step end did not retire live records'}
$out='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-native-pass248-'+$Step
@{step=$Step;records=$records;names=$names;physicalScale=1.5}|ConvertTo-Json -Depth 7|Set-Content -Encoding UTF8 -LiteralPath ($out+'-state.json')
$rect=New-Object InspectorR103+Rect;[void][InspectorR103]::GetWindowRect($handle,[ref]$rect)
$bitmap=New-Object System.Drawing.Bitmap ($rect.R-$rect.L),($rect.B-$rect.T);$graphics=[System.Drawing.Graphics]::FromImage($bitmap);$dc=$graphics.GetHdc()
try{if(-not[InspectorR103]::PrintWindow($handle,$dc,2)){throw 'Capture failed'}}finally{$graphics.ReleaseHdc($dc);$graphics.Dispose()}
try{$bitmap.Save(($out+'.png'),[System.Drawing.Imaging.ImageFormat]::Png)}finally{$bitmap.Dispose()}
'PASS: '+$Step
