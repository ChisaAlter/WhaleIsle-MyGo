param([int]$OwnerId,[string]$Step='filtered')
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
if($process.ExecutablePath -ne 'C:\AI\WhaleIsle-MyGo\.devdata\r103-inspector-fixture.exe'){throw 'Unexpected fixture owner'}
$handle=(Get-Process -Id $OwnerId).MainWindowHandle
[void][InspectorR103]::ShowWindow($handle,9)
[void][InspectorR103]::SetForegroundWindow($handle)
[void][InspectorR103]::SetWindowPos($handle,[IntPtr]::Zero,0,0,1193,912,0x0040)
$root=[System.Windows.Automation.AutomationElement]::FromHandle($handle)
function Find([string]$name){$cond=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty,$name);$root.FindFirst([System.Windows.Automation.TreeScope]::Descendants,$cond)}
function Click([string]$name){$e=Find $name;if($null -eq $e){throw ('Missing '+$name)};$r=$e.Current.BoundingRectangle;[InspectorR103]::Click($handle,[int]($r.X+$r.Width/2),[int]($r.Y+$r.Height/2));Start-Sleep -Milliseconds 250}
function Apply-InspectorQuery([string]$query){Click 'Inspector type filter';$in=Find 'Inspector type query';$in.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern).SetValue($query);Start-Sleep -Milliseconds 250;Click 'Apply filter'}
Start-Sleep -Milliseconds 200
switch($Step){
 'filtered' {Apply-InspectorQuery ' *CUSTOM* '}
 'folded' {Click 'Collapse children 2'}
 'picked' {[InspectorR103]::Key($handle,0x75)}
 'reopened' {Click 'Inspector details divider';[InspectorR103]::Key($handle,0x23);Start-Sleep -Milliseconds 250;Click 'Close';Click 'future/custom'}
 'reconfirmed' {Apply-InspectorQuery 'CUSTOM'}
 'excluded-pick' {[InspectorR103]::Key($handle,0x76)}
 default {throw 'Unknown step'}
}
Start-Sleep -Milliseconds 400
$names=@();$records=@()
foreach($e in $root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)){
 $name=$e.Current.Name
 if($name){$names+=$name}
 if($name -like 'Inspector record *'){$r=$e.Current.BoundingRectangle;$selection=$e.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern);$records+=@{key=$name.Substring(17);selected=$selection.Current.IsSelected;x=$r.X;y=$r.Y;w=$r.Width;h=$r.Height}}
}
$wanted=switch($Step){'filtered' {@('event:1','event:2','event:4')};'folded' {@('event:1','event:2')};'picked' {@('event:1','event:2','event:4')};'reopened' {@('event:1','event:2','event:4')};'reconfirmed' {@('event:1','event:2','event:4')};'excluded-pick' {@('event:0','event:1','event:2','event:3','event:4','event:5','event:6')}}
if((@($records.key|Sort-Object)-join ',') -ne (@($wanted|Sort-Object)-join ',')){throw ('Unexpected records '+($records.key-join ','))}
$details='Raw data' -in $names
$expectedDetails=$Step -in @('picked','reopened','excluded-pick')
if($details -ne $expectedDetails){throw ('Wrong details state '+$details)}
$selected=@($records|Where-Object{$_.selected})
if($Step -in @('picked','reopened') -and ($selected.Count -ne 1 -or $selected[0].key -ne 'event:4')){throw 'Included pick not selected'}
if($Step -eq 'excluded-pick' -and ($selected.Count -ne 1 -or $selected[0].key -ne 'event:3')){throw 'Excluded pick not selected'}
if($Step -ne 'excluded-pick' -and (([string][char]0xb7)+' 1 / 7') -notin $names){throw 'Matching filter lost'}
if($Step -eq 'excluded-pick' -and (([string][char]0xb7)+' 7') -notin $names){throw 'Excluded pick did not clear filter'}
$divider=Find 'Inspector details divider'
$dividerY=if($divider){$divider.Current.BoundingRectangle.Y}else{$null}
if($Step -eq 'reopened' -and ($dividerY -lt 539 -or $dividerY -gt 543)){throw ('Reopened details ratio wrong '+$dividerY)}
$out='C:\AI\WhaleIsle-MyGo\docs\review-20261022\evidence\inspector-native-pass247-'+$Step
@{step=$Step;records=$records;details=$details;dividerY=$dividerY;names=$names;physicalScale=1.5}|ConvertTo-Json -Depth 7|Set-Content -Encoding UTF8 -LiteralPath ($out+'-state.json')
$rect=New-Object InspectorR103+Rect;[void][InspectorR103]::GetWindowRect($handle,[ref]$rect)
$bitmap=New-Object System.Drawing.Bitmap ($rect.R-$rect.L),($rect.B-$rect.T);$graphics=[System.Drawing.Graphics]::FromImage($bitmap);$dc=$graphics.GetHdc()
try{if(-not[InspectorR103]::PrintWindow($handle,$dc,2)){throw 'Capture failed'}}finally{$graphics.ReleaseHdc($dc);$graphics.Dispose()}
try{$bitmap.Save(($out+'.png'),[System.Drawing.Imaging.ImageFormat]::Png)}finally{$bitmap.Dispose()}
'PASS: '+$Step
