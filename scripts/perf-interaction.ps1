# Interaction-latency measurement against the real application binary, run in
# the background: it never moves the operator's mouse, types on their keyboard,
# takes focus, or covers their windows.
#
# - Profile: a snapshot copy of the real profile (or -UserData) in isolated mode,
#   so the operator's own instance and data are untouched.
# - Window: started hidden, then shown without activation off the visible desktop.
# - Input: posted window messages to the MyGo surface, the same channel as
#   scripts/frame-probe.ps1. The binary, window, renderer and input handling are
#   the real ones; the input is a posted message, not a hardware event.
# - Trace: the engine's own per-frame trace (MYGO_FRAME_TRACE); the report
#   (scripts/perf-interaction-report.mjs) aligns it with the step timestamps and
#   checks the interaction gates in ARCHITECTURE.md section 11.
#
#   powershell -ExecutionPolicy Bypass -File scripts\perf-interaction.ps1 -Exe tmp\latest-direct\whaleisle.exe -Label before
#   ... -UserData <dir>      use this isolated profile instead of a snapshot of the real one
#   ... -Screenshots         save a PrintWindow capture after every step
param(
    [Parameter(Mandatory = $true)][string]$Exe,
    [string]$OutDir = '',
    [string]$Label = '',
    [string]$UserData = '',
    [string]$Vendor = '',
    [string]$NodeDir = '',
    [int]$ReadyTimeoutSec = 240,
    [int]$SettleSec = 8,
    [int]$StepMs = 1800,
    [int]$Width = 1322,
    [int]$Height = 828,
    [switch]$Screenshots
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$Exe = (Resolve-Path -LiteralPath $Exe).Path
# Performance numbers mean nothing unless they measure the pinned shipping
# runtime: resolve vendor/node from runtime-pin.json before any legacy path.
$pin = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'runtime-pin.json') -Raw | ConvertFrom-Json
$pinStage = @($pin.stageDirNames) | ForEach-Object { Join-Path $root $_ } | Where-Object { Test-Path -LiteralPath (Join-Path $_ 'vendor') -PathType Container } | Select-Object -First 1
if (-not $Vendor) { $Vendor = $(if ($pinStage) { Join-Path $pinStage 'vendor' } else { Join-Path $root 'runtime-rc4\vendor' }) }
if (-not $NodeDir) { $NodeDir = $(if ($pinStage -and (Test-Path (Join-Path $pinStage 'node\node.exe') -PathType Leaf)) { Join-Path $pinStage 'node' } else { Join-Path $root 'runtime-rc4\node' }) }
if (-not $OutDir) { $OutDir = Join-Path $root ('tmp\perf\' + (Get-Date -Format 'yyyyMMdd-HHmmss') + $(if ($Label) { '-' + $Label } else { '' })) }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$OutDir = (Resolve-Path -LiteralPath $OutDir).Path

Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes, System.Drawing, System.Windows.Forms
Add-Type -ReferencedAssemblies System.Drawing -TypeDefinition @'
using System; using System.Runtime.InteropServices;
public static class PerfWin {
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int c);
  [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
  [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
  [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p, IntPtr a, string c, string t);
  [DllImport("user32.dll")] static extern bool ScreenToClient(IntPtr h, ref POINT p);
  [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] static extern bool PrintWindow(IntPtr h, IntPtr dc, uint f);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern IntPtr FindWindowW(string c, string t);
  [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] static extern bool SetThreadDpiAwarenessContext(IntPtr c);
  public struct POINT { public int X, Y; }
  public struct RECT { public int L, T, R, B; }
  public static void Dpi() { SetThreadDpiAwarenessContext(new IntPtr(-4)); }
  public static IntPtr Surface(IntPtr h) { var s = FindWindowEx(h, IntPtr.Zero, "MyGoSurface", null); return s == IntPtr.Zero ? h : s; }
  public static IntPtr WindowOf(int pid, string title) {
    var h = FindWindowW(null, title);
    if (h == IntPtr.Zero) return IntPtr.Zero;
    uint owner; GetWindowThreadProcessId(h, out owner);
    return owner == (uint)pid ? h : IntPtr.Zero;
  }
  static IntPtr XY(int x, int y) { return new IntPtr((y << 16) | (x & 0xFFFF)); }
  // ClickScreen posts a move, press and release at a screen point, converted to
  // the surface's client coordinates.
  public static void ClickScreen(IntPtr h, int sx, int sy) {
    var s = Surface(h); var p = new POINT { X = sx, Y = sy }; ScreenToClient(s, ref p);
    PostMessage(s, 0x200, IntPtr.Zero, XY(p.X, p.Y));
    System.Threading.Thread.Sleep(16);
    PostMessage(s, 0x201, new IntPtr(1), XY(p.X, p.Y));
    System.Threading.Thread.Sleep(40);
    PostMessage(s, 0x202, IntPtr.Zero, XY(p.X, p.Y));
  }
  public static void Key(IntPtr h, int vk) {
    var s = Surface(h);
    PostMessage(s, 0x100, new IntPtr(vk), new IntPtr(1));
    System.Threading.Thread.Sleep(16);
    PostMessage(s, 0x101, new IntPtr(vk), new IntPtr(unchecked((int)0xC0000001)));
  }
  public static bool Capture(IntPtr h, string path) {
    RECT r; if (!GetWindowRect(h, out r) || r.R <= r.L || r.B <= r.T) return false;
    using (var bmp = new System.Drawing.Bitmap(r.R - r.L, r.B - r.T)) {
      using (var g = System.Drawing.Graphics.FromImage(bmp)) { var dc = g.GetHdc(); PrintWindow(h, dc, 2); g.ReleaseHdc(dc); }
      bmp.Save(path, System.Drawing.Imaging.ImageFormat.Png);
    }
    return true;
  }
}
'@
[PerfWin]::Dpi()

function NowNs { [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() * 1000000 }

function Find-MainWindow([int]$procId) {
    $cond = New-Object Windows.Automation.PropertyCondition([Windows.Automation.AutomationElement]::ProcessIdProperty, $procId)
    $wins = [Windows.Automation.AutomationElement]::RootElement.FindAll([Windows.Automation.TreeScope]::Children, $cond)
    foreach ($w in $wins) { if ($w.Current.Name -like '*MyGo*') { return $w } }
    return $null
}

function Find-Named($win, [string]$name) {
    if (-not $name.Contains('*')) {
        $cond = New-Object Windows.Automation.PropertyCondition([Windows.Automation.AutomationElement]::NameProperty, $name)
        return $win.FindFirst([Windows.Automation.TreeScope]::Descendants, $cond)
    }
    foreach ($el in $win.FindAll([Windows.Automation.TreeScope]::Descendants, [Windows.Automation.Condition]::TrueCondition)) {
        if ($el.Current.Name -like $name) { return $el }
    }
    return $null
}

function Stop-Tree([int]$procId) {
    $children = Get-CimInstance Win32_Process -Filter "ParentProcessId=$procId" -ErrorAction SilentlyContinue
    foreach ($c in $children) { Stop-Tree ([int]$c.ProcessId) }
    Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
}

# Removes a directory this run created. Junctions are unlinked, never followed.
function Remove-Owned([string]$dir) {
    if (-not $dir -or -not (Test-Path -LiteralPath $dir)) { return }
    Get-ChildItem -LiteralPath $dir -Recurse -Force -Attributes ReparsePoint -ErrorAction SilentlyContinue | ForEach-Object { cmd /c rmdir "$($_.FullName)" | Out-Null }
    Remove-Item -LiteralPath $dir -Recurse -Force -ErrorAction SilentlyContinue
}

# The fixed sequence. The plugin page decodes its artwork in the background, so
# it dwells longer and its allocation is attributed to its own step; revisiting
# it measures whether that work survives a page change.
$sequence = @(
    @{ kind = 'click'; name = '插件'; dwellMs = 6000 },
    @{ kind = 'click'; name = '自动化任务' },
    @{ kind = 'click'; name = '鲸鱼娘' },
    @{ kind = 'click'; name = '新会话' },
    @{ kind = 'click'; name = '标准模式' },
    @{ kind = 'key'; name = 'Escape' },
    @{ kind = 'click'; name = '账号菜单' },
    @{ kind = 'key'; name = 'Escape' },
    @{ kind = 'click'; name = '插件'; dwellMs = 6000 },
    @{ kind = 'click'; name = '新会话' }
)

$snapshot = ''
if (-not $UserData) {
    $real = Join-Path $env:APPDATA 'WhaleIsle-MyGo\data'
    if (-not (Test-Path -LiteralPath $real)) { throw "no real profile at $real; pass -UserData" }
    $snapshot = Join-Path $OutDir 'profile'
    New-Item -ItemType Directory -Force -Path $snapshot | Out-Null
    # Snapshot of the real profile without update payloads, logs, crash dumps or
    # the live instance's runtime/lock files.
    & robocopy $real $snapshot /E /XJ /R:0 /W:0 /NFL /NDL /NJH /NJS /NP /XD updates logs runtime crash /XF lockfile *.lock | Out-Null
    if ($LASTEXITCODE -ge 8) { throw "profile snapshot failed (robocopy $LASTEXITCODE)" }
    $UserData = $snapshot
}

$saved = @{}
$envs = @{
    MYGO_FRAME_TRACE           = (Join-Path $OutDir 'frames.csv')
    WHALEISLE_LOG_FILE         = (Join-Path $OutDir 'app.log')
    WHALEISLE_VENDOR           = $Vendor
    WHALEISLE_NODE_DIR         = $NodeDir
    WHALEISLE_USER_DATA        = $UserData
    WHALEISLE_ISOLATED         = '1'
    WHALEISLE_ISOLATED_VISIBLE = '0'
}
$proc = $null
$meta = [ordered]@{ label = $Label; exe = $Exe; profile = $(if ($snapshot) { 'snapshot of real' } else { $UserData }); input = 'posted messages, background window'; steps = @() }
try {
    foreach ($k in $envs.Keys) { $saved[$k] = [Environment]::GetEnvironmentVariable($k, 'Process'); [Environment]::SetEnvironmentVariable($k, $envs[$k], 'Process') }
    $foregroundBefore = [PerfWin]::GetForegroundWindow()
    $meta.launch_ns = NowNs
    $proc = Start-Process -FilePath $Exe -WorkingDirectory (Split-Path -Parent $Exe) -WindowStyle Hidden -RedirectStandardError (Join-Path $OutDir 'stderr.log') -RedirectStandardOutput (Join-Path $OutDir 'stdout.log') -PassThru
    $meta.pid = $proc.Id

    $deadline = (Get-Date).AddSeconds($ReadyTimeoutSec)
    $hwnd = [IntPtr]::Zero
    while ((Get-Date) -lt $deadline -and -not $proc.HasExited -and $hwnd -eq [IntPtr]::Zero) {
        $hwnd = [PerfWin]::WindowOf($proc.Id, '鲸屿 MyGo')
        if ($hwnd -eq [IntPtr]::Zero) { Start-Sleep -Milliseconds 200 }
    }
    if ($proc.HasExited) { throw "whaleisle.exe exited during startup (code $($proc.ExitCode)); see $OutDir\stderr.log" }
    if ($hwnd -eq [IntPtr]::Zero) { throw "no main window within $ReadyTimeoutSec s" }
    # Shown without activation, at the bottom of the z-order, past the right edge
    # of the virtual desktop: it renders and composes but covers nothing.
    $vs = [System.Windows.Forms.SystemInformation]::VirtualScreen
    $offX = $vs.Right + 200
    [void][PerfWin]::ShowWindow($hwnd, 4)
    [void][PerfWin]::SetWindowPos($hwnd, [IntPtr]::new(1), $offX, $vs.Top + 40, $Width, $Height, 0x0010 -bor 0x0040)
    $meta.window = "$offX,$($vs.Top + 40) ${Width}x$Height"

    $win = $null
    while ((Get-Date) -lt $deadline -and -not $proc.HasExited) {
        $win = Find-MainWindow $proc.Id
        if ($win -and (Find-Named $win '新会话')) { break }
        Start-Sleep -Milliseconds 250
    }
    if ($proc.HasExited) { throw "whaleisle.exe exited during startup (code $($proc.ExitCode)); see $OutDir\stderr.log" }
    if (-not $win -or -not (Find-Named $win '新会话')) { throw "the workbench was not ready within $ReadyTimeoutSec s" }
    $meta.ready_ns = NowNs
    Start-Sleep -Seconds $SettleSec

    $i = 0
    foreach ($step in $sequence) {
        $entry = [ordered]@{ index = $i; kind = $step.kind; name = $step.name; found = $true; t_ns = 0 }
        $win = Find-MainWindow $proc.Id
        if (-not $win) { throw "the main window disappeared before step $i" }
        if ($step.kind -eq 'click') {
            $el = Find-Named $win $step.name
            if (-not $el) {
                $entry.found = $false
            } else {
                $r = $el.Current.BoundingRectangle
                $entry.t_ns = NowNs
                [PerfWin]::ClickScreen($hwnd, [int]($r.X + $r.Width / 2), [int]($r.Y + $r.Height / 2))
            }
        } else {
            $entry.t_ns = NowNs
            [PerfWin]::Key($hwnd, 0x1B)
        }
        $meta.steps += $entry
        Start-Sleep -Milliseconds $(if ($step.dwellMs) { $step.dwellMs } else { $StepMs })
        if ($Screenshots) { [void][PerfWin]::Capture($hwnd, (Join-Path $OutDir ('step-{0:D2}.png' -f $i))) }
        $i++
    }
    $meta.end_ns = NowNs
    $meta.focus_kept = ([PerfWin]::GetForegroundWindow() -ne $hwnd)
} finally {
    if ($proc) { Stop-Tree $proc.Id }
    foreach ($k in $saved.Keys) { [Environment]::SetEnvironmentVariable($k, $saved[$k], 'Process') }
    $meta | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $OutDir 'run.json') -Encoding UTF8
    if ($snapshot) { Start-Sleep -Milliseconds 500; Remove-Owned $snapshot }
}

$node = Join-Path $NodeDir 'node.exe'
if (-not (Test-Path -LiteralPath $node)) { $node = 'node' }
& $node (Join-Path $PSScriptRoot 'perf-interaction-report.mjs') $OutDir
exit $LASTEXITCODE
