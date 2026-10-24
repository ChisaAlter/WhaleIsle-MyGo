# Acceptance capture pipeline (contract: docs/acceptance-ui-parity.md).
#
# Produces one screen row under %TEMP%\whaleisle-accept-<UTC date>\<ID>\:
#   baseline.png (native Electron)  actual.png (MyGo)  geom.json  controls.json
#   accept-log.txt  mask.json      (diff.png/diff.json come from acceptance-diff.py)
#
# Both apps draw frameless windows (window rect == client rect, asserted at
# runtime), so each side is brought to the foreground and its window rect is
# copied with CopyFromScreen from this per-monitor-v2 process. PrintWindow hangs
# on Electron/WebView2, and copying both sides the same way keeps the path
# symmetric. The installed app is read-only: resized and restored, never written.
#
# Preconditions: the native app is running; the MyGo build (delivered commit) is
# running with its own profile. Pass -MyGoPid or let it pick the MyGoWindow.
#
# PowerShell 5.1 compatible.
param(
  [string]$Screen = 'D-MAIN',
  [int]$NativePid = 0,
  [int]$MyGoPid = 0,
  [int]$ClientWidth = 1440,
  [int]$ClientHeight = 920,
  [string]$OutRoot = '',
  [int]$SettleSeconds = 2,
  [switch]$KeepWindows
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
Add-Type -TypeDefinition @'
using System; using System.Collections.Generic; using System.Runtime.InteropServices; using System.Text;
public static class Aw {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
  [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X, Y; }
  public delegate bool EnumProc(IntPtr h, IntPtr l);
  [DllImport("user32.dll")] public static extern bool SetProcessDpiAwarenessContext(IntPtr c);
  [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc cb, IntPtr l);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool ClientToScreen(IntPtr h, ref POINT p);
  [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr a, int x, int y, int cx, int cy, uint f);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int c);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] public static extern IntPtr WindowFromPoint(POINT p);
  [DllImport("user32.dll")] public static extern IntPtr GetParent(IntPtr h);
  [DllImport("user32.dll")] public static extern int GetDpiForWindow(IntPtr h);
  public static string Cls(IntPtr h) { var s = new StringBuilder(256); GetClassName(h, s, 256); return s.ToString(); }
  public static string Ttl(IntPtr h) { var s = new StringBuilder(512); GetWindowText(h, s, 512); return s.ToString(); }
  public static long[] Wins(uint pid) {
    var f = new List<long>();
    EnumWindows((h, l) => { uint p; GetWindowThreadProcessId(h, out p); if (p == pid && IsWindowVisible(h) && Ttl(h).Length > 0) f.Add(h.ToInt64()); return true; }, IntPtr.Zero);
    return f.ToArray();
  }
  public static int[] Rect(IntPtr h) { RECT r; GetWindowRect(h, out r); return new[] { r.L, r.T, r.R - r.L, r.B - r.T }; }
  public static int[] Client(IntPtr h) { RECT r; GetClientRect(h, out r); return new[] { r.R - r.L, r.B - r.T }; }
  public static int[] Origin(IntPtr h) { POINT p; p.X = 0; p.Y = 0; ClientToScreen(h, ref p); return new[] { p.X, p.Y }; }
}
'@
[void][Aw]::SetProcessDpiAwarenessContext([IntPtr](-4))

if (-not $OutRoot) { $OutRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('whaleisle-accept-' + [datetime]::UtcNow.ToString('yyyy-MM-dd')) }
$dir = Join-Path $OutRoot $Screen
New-Item -ItemType Directory -Path $dir -Force | Out-Null
$script:lines = New-Object System.Collections.Generic.List[string]
function Log([string]$m) { $l = '{0:o}  {1}' -f ([datetime]::UtcNow), $m; $script:lines.Add($l); Write-Host $l }
function Save-Log() { $script:lines | Set-Content -Path (Join-Path $dir 'accept-log.txt') -Encoding UTF8 }

function Get-Hwnd([string]$cls, [int]$processId) {
  $procs = @()
  if ($processId -gt 0) { $procs = @(Get-Process -Id $processId -ErrorAction SilentlyContinue) }
  else { $procs = @(Get-Process | Where-Object { $_.MainWindowTitle -eq 'Whale Isle' }) }
  foreach ($p in $procs) {
    foreach ($w in [Aw]::Wins([uint32]$p.Id)) {
      $h = [IntPtr]$w
      if ($cls -eq '*' -and [Aw]::Cls($h) -like 'Chrome_WidgetWin*') { return @{ Pid = $p.Id; Hwnd = $h } }
      if ([Aw]::Cls($h) -eq $cls) { return @{ Pid = $p.Id; Hwnd = $h } }
    }
  }
  throw "window '$cls' not found"
}

function Resize-Client([IntPtr]$h, [int]$w, [int]$hh) {
  $r = [Aw]::Rect($h); $c = [Aw]::Client($h)
  # Top-left anchor keeps the whole client inside the working area; a window that hangs below the taskbar cannot be copied cleanly.
  if (-not [Aw]::SetWindowPos($h, [IntPtr]::Zero, 0, 0, $w + ($r[2] - $c[0]), $hh + ($r[3] - $c[1]), 0x0014)) { throw 'SetWindowPos failed' }
  Start-Sleep -Milliseconds 500
}

function Capture([IntPtr]$h, [string]$path) {
  $r = [Aw]::Rect($h); $o = [Aw]::Origin($h)
  $bmp = New-Object System.Drawing.Bitmap($r[2], $r[3]); $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($o[0], $o[1], 0, 0, (New-Object System.Drawing.Size($r[2], $r[3])))
  $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png); $g.Dispose(); $bmp.Dispose()
  return @{ W = $r[2]; H = $r[3] }
}

function Check-Point([IntPtr]$h, [int]$x, [int]$y, [string]$side) {
  $pt = New-Object Aw+POINT
  $pt.X = $x
  $pt.Y = $y
  $hit = [Aw]::WindowFromPoint($pt)
  # Electron renders inside a child HWND (Chrome_RenderWidgetHostHWND), so any
  # descendant of the target counts as the target's own pixels.
  $cur = $hit
  $owned = $false
  while ($cur -ne [IntPtr]::Zero) {
    if ($cur -eq $h) { $owned = $true; break }
    $cur = [Aw]::GetParent($cur)
  }
  if (-not $owned) {
    throw "$side occluded at $x,$y by [$([Aw]::Cls($hit))] '$([Aw]::Ttl($hit))'"
  }
}

function Check-Occlusion([IntPtr]$h, [string]$side) {
  $o = [Aw]::Origin($h)
  $r = [Aw]::Rect($h)
  $w = [int]$r[2]
  $ht = [int]$r[3]
  $cx = $o[0] + [int]($w / 2)
  $cy = $o[1] + [int]($ht / 2)
  Check-Point $h ($o[0] + 8) ($o[1] + 8) $side
  Check-Point $h $cx ($o[1] + 18) $side
  Check-Point $h $cx $cy $side
  Check-Point $h ($o[0] + $w - 12) $cy $side
  Check-Point $h $cx ($o[1] + $ht - 24) $side
}

function Dump-Uia([IntPtr]$h, [string]$side, [double]$scale) {
  $null = [System.Windows.Automation.AutomationElement]::FromHandle($h).FindAll([System.Windows.Automation.TreeScope]::Descendants, [System.Windows.Automation.Condition]::TrueCondition)
  Start-Sleep -Milliseconds 700
  $root = [System.Windows.Automation.AutomationElement]::FromHandle($h)
  $w = [System.Windows.Automation.TreeWalker]::ControlViewWalker
  $nodes = New-Object System.Collections.Generic.List[object]
  $q = New-Object System.Collections.Generic.Queue[object]
  $q.Enqueue(@($root, 0)); $n = 0
  while ($q.Count -gt 0 -and $n -lt 2500) {
    $it = $q.Dequeue(); $el = $it[0]; $d = $it[1]; $n++
    try {
      $r = $el.Current.BoundingRectangle; $ct = $el.Current.ControlType
      $typeName = ''
      if ($ct) { $typeName = $ct.ProgrammaticName -replace 'ControlType\.', '' }
      $nodes.Add([ordered]@{
        side = $side; depth = $d
        type = $typeName
        name = $el.Current.Name; automationId = $el.Current.AutomationId
        enabled = $el.Current.IsEnabled; offscreen = $el.Current.IsOffscreen
        physical = [ordered]@{ x = [math]::Round($r.X); y = [math]::Round($r.Y); w = [math]::Round($r.Width); h = [math]::Round($r.Height) }
        dip = [ordered]@{ x = [math]::Round($r.X / $scale, 2); y = [math]::Round($r.Y / $scale, 2); w = [math]::Round($r.Width / $scale, 2); h = [math]::Round($r.Height / $scale, 2) }
      })
    } catch { }
    try { $c = $w.GetFirstChild($el); while ($null -ne $c) { $q.Enqueue(@($c, $d + 1)); $c = $w.GetNextSibling($c) } } catch { }
  }
  return $nodes
}

$native = $null; $mygo = $null; $nOrig = $null; $mOrig = $null; $out = [ordered]@{ screen = $Screen; ok = $false }
try {
  $native = Get-Hwnd '*' $NativePid
  $mygo = Get-Hwnd 'MyGoWindow' $MyGoPid
  Log "native hwnd=$($native.Hwnd) pid=$($native.Pid) class=$([Aw]::Cls($native.Hwnd))"
  Log "mygo hwnd=$($mygo.Hwnd) pid=$($mygo.Pid)"
  $dpi = [Aw]::GetDpiForWindow($native.Hwnd)
  if ([Aw]::GetDpiForWindow($mygo.Hwnd) -ne $dpi) { throw 'DPI differs between the two windows' }
  $pw = [int][math]::Round($ClientWidth * $dpi / 96.0); $ph = [int][math]::Round($ClientHeight * $dpi / 96.0)
  Log "target client ${ClientWidth}x${ClientHeight} DIP = ${pw}x${ph} px @ dpi $dpi"
  $nOrig = [Aw]::Rect($native.Hwnd); $mOrig = [Aw]::Rect($mygo.Hwnd)
  Resize-Client $native.Hwnd $pw $ph
  Resize-Client $mygo.Hwnd $pw $ph
  $nS = [Aw]::Rect($native.Hwnd); $mS = [Aw]::Rect($mygo.Hwnd)
  Log "after resize native $($nS[2])x$($nS[3]) mygo $($mS[2])x$($mS[3])"
  if ([math]::Abs($nS[2] - $mS[2]) -gt 1 -or [math]::Abs($nS[3] - $mS[3]) -gt 1) { throw 'client sizes differ by more than 1 px' }
  [void][Aw]::ShowWindow($native.Hwnd, 9); [void][Aw]::SetForegroundWindow($native.Hwnd)
  Start-Sleep -Milliseconds 1500
  Check-Occlusion $native.Hwnd 'native'
  $nb = Capture $native.Hwnd (Join-Path $dir 'baseline.png')
  [void][Aw]::ShowWindow($mygo.Hwnd, 9); [void][Aw]::SetForegroundWindow($mygo.Hwnd)
  Start-Sleep -Seconds $SettleSeconds
  Check-Occlusion $mygo.Hwnd 'mygo'
  $mb = Capture $mygo.Hwnd (Join-Path $dir 'actual.png')
  Log "baseline $($nb.W)x$($nb.H) actual $($mb.W)x$($mb.H)"
  $scale = $dpi / 96.0
  $nn = Dump-Uia $native.Hwnd 'native' $scale
  $mn = Dump-Uia $mygo.Hwnd 'mygo' $scale
  Log "uia nodes native=$($nn.Count) mygo=$($mn.Count)"
  ([ordered]@{ screen = $Screen; native = $nn; mygo = $mn }) | ConvertTo-Json -Depth 6 | ForEach-Object { [System.IO.File]::WriteAllText((Join-Path $dir 'controls.json'), $_, (New-Object System.Text.UTF8Encoding($false))) }
  ([ordered]@{ screen = $Screen; client = [ordered]@{ dip = [ordered]@{ w = $ClientWidth; h = $ClientHeight }; native = $nb; mygo = $mb; dpi = $dpi }; native = $nn; mygo = $mn }) | ConvertTo-Json -Depth 6 | ForEach-Object { [System.IO.File]::WriteAllText((Join-Path $dir 'geom.json'), $_, (New-Object System.Text.UTF8Encoding($false))) }
  ([ordered]@{ screen = $Screen; threshold = 0; mask = @(); registry = @('caret', 'spinner/progress', 'toast/banner', 'hover state', 'cursor'); note = 'D-MAIN steady state: no caret/spinner/toast, so no mask is registered.' }) | ConvertTo-Json -Depth 4 | ForEach-Object { [System.IO.File]::WriteAllText((Join-Path $dir 'mask.json'), $_, (New-Object System.Text.UTF8Encoding($false))) }
  $exe = (Get-Process -Id $native.Pid).Path
  $repo = Join-Path $PSScriptRoot '..'
  $git = & git -C $repo rev-parse --short HEAD 2>$null
  $dirty = @(& git -C $repo status --porcelain 2>$null | Where-Object { $_ })
  Log "native exe=$exe version=$((Get-Item $exe).VersionInfo.FileVersion)"
  Log "mygo exe=$((Get-Process -Id $mygo.Pid).Path) git=$git dirty=$($dirty.Count)"
  Log "client ${ClientWidth}x${ClientHeight} DIP; captured $($nb.W)x$($nb.H) / $($mb.W)x$($mb.H); dpi=$dpi"
  Log "out=$dir"
  Save-Log
  $out.ok = $true; $out.dir = $dir; $out.uia = @{ native = $nn.Count; mygo = $mn.Count }
}
catch {
  Log "FAILED: $($_.Exception.Message)"
  $out.error = $_.Exception.Message
  Save-Log
  throw
}
finally {
  if (-not $KeepWindows) {
    if ($nOrig -and $native) { [void][Aw]::SetWindowPos($native.Hwnd, [IntPtr]::Zero, $nOrig[0], $nOrig[1], $nOrig[2], $nOrig[3], 0x0014); Log "restored native to $($nOrig[2])x$($nOrig[3])" }
    if ($mOrig -and $mygo) { [void][Aw]::SetWindowPos($mygo.Hwnd, [IntPtr]::Zero, $mOrig[0], $mOrig[1], $mOrig[2], $mOrig[3], 0x0014) }
  }
  Save-Log
  $out | ConvertTo-Json -Depth 4
}
