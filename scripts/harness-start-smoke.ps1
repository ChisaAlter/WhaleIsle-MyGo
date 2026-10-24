# Start->boot->ready smoke for a Whale Isle build: throwaway profile, real
# config copied and retargeted (workspace/port), welcome marker, then a sampled
# timeline of port state, top-level windows and process responsiveness. When the
# readiness criteria hold it dumps the main window's UI Automation text, records
# the spawned Host child command lines, kills the process tree and removes the
# temp files.
#
# No PrintWindow/BitBlt: that hangs these windows (verified). Geometry comes from
# GetWindowRect in a per-monitor-v2 process.
#
#   pwsh -File tmp/harness-start-smoke.ps1 -Exe whaleisle-pass274.exe `
#     -Out docs/review-20261022/evidence/harness-start-pass274.json
param(
  [Parameter(Mandatory = $true)][string]$Exe,
  [Parameter(Mandatory = $true)][string]$Out,
  [int]$Port = 3099,
  [int]$TimeoutSeconds = 180,
  [int]$SettleSeconds = 6,
  [string]$RealConfig = "$env:APPDATA\Deepseek-Harness-Desktop\config.json",
  [string]$KeepProfile = '',
  [string]$LogPath = ''
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

# Progress log survives a hard kill of this shell, which is exactly what a
# parent-console/page-file failure would look like.
function Write-Step([string]$Message) {
  if (-not $LogPath) { return }
  $line = '{0}  {1}' -f ([datetime]::UtcNow.ToString('o')), $Message
  [System.IO.File]::AppendAllText($LogPath, $line + "`r`n", (New-Object System.Text.UTF8Encoding($false)))
}
Write-Step "start exe=$Exe out=$Out port=$Port"

Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public class SmokeWin {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
  public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

  [DllImport("user32.dll")] static extern bool EnumWindows(EnumWindowsProc cb, IntPtr lParam);
  [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint pid);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern int GetWindowTextLength(IntPtr hWnd);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int max);
  [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern int GetClassName(IntPtr hWnd, StringBuilder name, int max);
  [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
  [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr hWnd);
  [DllImport("user32.dll")] static extern bool IsIconic(IntPtr hWnd);
  [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr context);
  [DllImport("user32.dll")] static extern uint GetDpiForWindow(IntPtr hWnd);
  [DllImport("user32.dll", SetLastError = true)] static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint msg, IntPtr wParam, IntPtr lParam, uint flags, uint timeout, out IntPtr result);

  // WM_NULL with SMTO_ABORTIFHUNG: the same liveness question as
  // Process.Responding, asked of a specific window instead of MainWindowHandle.
  public static bool Ping(IntPtr hWnd, uint timeoutMs) {
    if (hWnd == IntPtr.Zero) return false;
    IntPtr result;
    return SendMessageTimeout(hWnd, 0, IntPtr.Zero, IntPtr.Zero, 0x0002, timeoutMs, out result) != IntPtr.Zero;
  }

  public static void UsePhysicalPixels() { SetThreadDpiAwarenessContext(new IntPtr(-4)); }

  public static string Title(IntPtr hWnd) {
    int length = GetWindowTextLength(hWnd);
    if (length <= 0) return string.Empty;
    var text = new StringBuilder(length + 1);
    GetWindowText(hWnd, text, text.Capacity);
    return text.ToString();
  }

  public static string ClassName(IntPtr hWnd) {
    var name = new StringBuilder(256);
    GetClassName(hWnd, name, name.Capacity);
    return name.ToString();
  }

  // One record per top-level window of pid: hwnd, title, class, physical rect, state, dpi.
  public static List<string> Windows(uint pid) {
    var rows = new List<string>();
    EnumWindows(delegate(IntPtr hWnd, IntPtr lParam) {
      uint owner;
      GetWindowThreadProcessId(hWnd, out owner);
      if (owner != pid) return true;
      RECT rect;
      if (!GetWindowRect(hWnd, out rect)) return true;
      string title = Title(hWnd);
      string cls = ClassName(hWnd);
      if (title.Length == 0 && cls.Length == 0) return true;
      uint dpi = 0;
      try { dpi = GetDpiForWindow(hWnd); } catch { dpi = 0; }
      rows.Add(string.Join("\u001f", new string[] {
        hWnd.ToInt64().ToString(), title.Replace("\u001f", " "), cls,
        rect.Left.ToString(), rect.Top.ToString(),
        (rect.Right - rect.Left).ToString(), (rect.Bottom - rect.Top).ToString(),
        IsWindowVisible(hWnd) ? "1" : "0", IsIconic(hWnd) ? "1" : "0",
        dpi.ToString(),
      }));
      return true;
    }, IntPtr.Zero);
    return rows;
  }
}
'@
[void][SmokeWin]::UsePhysicalPixels()

Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$exePath = (Resolve-Path -LiteralPath $Exe).Path
$outPath = [System.IO.Path]::GetFullPath((Join-Path (Get-Location).Path $Out))
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$base = Join-Path ([System.IO.Path]::GetTempPath()) ("whaleisle-start-$stamp")
$profile = if ($KeepProfile) { [System.IO.Path]::GetFullPath($KeepProfile) } else { Join-Path $base 'profile' }
$workspace = Join-Path $base 'workspace'
$createdBase = -not (Test-Path -LiteralPath $base)

function Read-Windows([uint32]$ProcId) {
  $windows = @()
  foreach ($row in [SmokeWin]::Windows($ProcId)) {
    $f = $row.Split([char]31)
    $dpi = [int]$f[9]
    if ($dpi -le 0) { $dpi = 96 }
    $w = [int]$f[5]; $h = [int]$f[6]
    $windows += [ordered]@{
      hwnd      = [long]$f[0]
      title     = $f[1]
      class     = $f[2]
      rect      = [ordered]@{ left = [int]$f[3]; top = [int]$f[4]; width = $w; height = $h }
      dip       = [ordered]@{ width = [int][math]::Round($w * 96.0 / $dpi); height = [int][math]::Round($h * 96.0 / $dpi) }
      visible   = $f[7] -eq '1'
      minimized = $f[8] -eq '1'
      dpi       = $dpi
    }
  }
  return $windows
}

function Test-PortOpen([int]$P, [int]$TimeoutMs = 250) {
  $client = New-Object System.Net.Sockets.TcpClient
  try {
    $task = $client.ConnectAsync('127.0.0.1', $P)
    if (-not $task.Wait($TimeoutMs)) { return $false }
    return $client.Connected
  } catch { return $false } finally { $client.Dispose() }
}

function Get-Responding($Process) {
  try {
    if ($null -eq $Process -or $Process.HasExited) { return $false }
    return [bool]$Process.Responding
  } catch { return $false }
}

function Get-Children([int]$ParentId) {
  Get-CimInstance Win32_Process -Filter "ParentProcessId=$ParentId" -ErrorAction SilentlyContinue |
    ForEach-Object {
      [ordered]@{ pid = [int]$_.ProcessId; name = $_.Name; commandLine = $_.CommandLine }
      Get-Children ([int]$_.ProcessId)
    }
}

# Kill a process tree by walking ParentProcessId downwards, then the root.
function Stop-Tree([int]$RootId) {
  foreach ($child in @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$RootId" -ErrorAction SilentlyContinue)) {
    Stop-Tree ([int]$child.ProcessId)
  }
  try { Stop-Process -Id $RootId -Force -ErrorAction Stop } catch { }
}

function Get-UiaDump([long]$Hwnd) {
  $dump = [ordered]@{ hwnd = $Hwnd; available = $false; nodes = @(); error = $null }
  try {
    $root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$Hwnd)
    if ($null -eq $root) { $dump.error = 'FromHandle returned null'; return $dump }
    $dump.available = $true
    $all = $root.FindAll([System.Windows.Automation.TreeScope]::Descendants, [System.Windows.Automation.Condition]::TrueCondition)
    $nodes = @()
    foreach ($element in $all) {
      $current = $element.Current
      $name = $current.Name
      if ([string]::IsNullOrWhiteSpace($name)) { continue }
      $nodes += [ordered]@{
        name         = $name
        controlType  = $current.ControlType.ProgrammaticName
        className    = $current.ClassName
        automationId = $current.AutomationId
        offscreen    = [bool]$current.IsOffscreen
      }
      if ($nodes.Count -ge 4000) { break }
    }
    $dump.nodes = $nodes
  } catch {
    $dump.error = $_.Exception.Message
  }
  return $dump
}

$startedAt = [datetime]::UtcNow
$process = $null
$status = 'ok'
$errorText = $null
$timeline = New-Object System.Collections.Generic.List[object]
$uiaDump = $null
$uiaLauncher = $null
$clearedSockets = @()
$clearedSocketList = @()
$settleSamples = @()
$windowStates = @()
$readiness = [ordered]@{ portListening = $false; mainVisible = $false; launcherGone = $true; responding = $false }
$children = @()
$crashReports = @()
$dshHomeListing = @()
$harnessHttp = $null
$mainHwnd = [long]0
$launcherHwnd = [long]0
$originalPort = 0
$originalWorkspace = ''
$respondAtReady = $false

try {
  if ($createdBase) { [void](New-Item -ItemType Directory -Path $base -Force) }
  if (-not (Test-Path -LiteralPath $profile)) { [void](New-Item -ItemType Directory -Path $profile -Force) }
  [void](New-Item -ItemType Directory -Path $workspace -Force)

  # Real config copied verbatim, then only the two fields the smoke owns change.
  $configText = [System.IO.File]::ReadAllText($RealConfig)
  $cfg = $configText | ConvertFrom-Json
  $originalPort = $cfg.port
  $originalWorkspace = $cfg.workspace
  $cfg.workspace = $workspace
  $cfg.host = '127.0.0.1'
  $cfg.port = $Port
  $patched = $cfg | ConvertTo-Json -Depth 100
  [System.IO.File]::WriteAllText((Join-Path $profile 'config.json'), $patched, (New-Object System.Text.UTF8Encoding($false)))
  # welcome-shown marker: cmd/whaleisle/welcome_window.go welcomeMarker
  [System.IO.File]::WriteAllText((Join-Path $profile 'welcome-shown'), '')
  Write-Step "profile prepared ($profile); config workspace=$workspace port=$Port"

  # A force-killed instance leaves mygo's single-instance socket behind; on
  # Windows a later launch then hands over to the dead instance and exits 0 with
  # no window. With no whaleisle running these sockets are stale.
  if (-not (Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.ProcessName -like 'whaleisle*' })) {
    Get-ChildItem ([System.IO.Path]::GetTempPath()) -Filter 'mygo-*.sock' -Force -ErrorAction SilentlyContinue |
      ForEach-Object { $clearedSockets += $_.FullName; Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue }
  }

  $previousProfile = $env:WHALEISLE_USER_DATA
  $env:WHALEISLE_USER_DATA = $profile
  try {
    $process = Start-Process -FilePath $exePath -WorkingDirectory (Split-Path -Parent $exePath) -PassThru
  } finally {
    if ($null -eq $previousProfile) { Remove-Item Env:WHALEISLE_USER_DATA -ErrorAction SilentlyContinue }
    else { $env:WHALEISLE_USER_DATA = $previousProfile }
  }
  Write-Step "app started pid=$($process.Id)"

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $lastSample = [datetime]::MinValue
  $nextSampleAt = $startedAt
  $ready = $false
  while ((Get-Date) -lt $deadline) {
    if ($process.HasExited) {
      Write-Step "app exited before readiness exit=$($process.ExitCode)"
      throw "whaleisle exited before readiness (exit $($process.ExitCode))"
    }
    $now = [datetime]::UtcNow
    $elapsed = [int]($now - $startedAt).TotalSeconds
    $portOpen = Test-PortOpen -P $Port
    $resp = Get-Responding $process
    $windows = Read-Windows ([uint32]$process.Id)
    $mainWin = @($windows | Where-Object { $_.visible -and $_.dip.width -eq 1440 -and $_.dip.height -eq 920 })[0]
    $launcherWin = @($windows | Where-Object { $_.visible -and $_.dip.width -eq 1060 -and $_.dip.height -eq 660 })[0]
    $loopPing = if ($null -ne $mainWin) { [bool][SmokeWin]::Ping([IntPtr][long]$mainWin.hwnd, 2000) } else { $false }
    $isReady = $portOpen -and ($null -ne $mainWin) -and ($null -eq $launcherWin) -and ($resp -or $loopPing)
    if ($now -ge $nextSampleAt -or $isReady) {
      $timeline.Add([ordered]@{
        t            = $elapsed
        at           = $now.ToString('o')
        portListening = $portOpen
        responding   = $resp
        windows      = @($windows | ForEach-Object { [ordered]@{ title = $_.title; dip = "$($_.dip.width)x$($_.dip.height)"; px = "$($_.rect.width)x$($_.rect.height)"; visible = $_.visible } })
      })
      $lastSample = $now
      $nextSampleAt = $now.AddSeconds($(if ($elapsed -lt 20) { 1 } elseif ($elapsed -lt 60) { 2 } else { 5 }))    }
    if ($isReady) {
      $ready = $true
      $respondAtReady = $resp -or $loopPing
      Write-Step "ready at ${elapsed}s (port=$portOpen main=$($null -ne $mainWin) launcherGone=$($null -eq $launcherWin) responding=$($resp -or $loopPing))"
      break
    }
    Start-Sleep -Milliseconds 400
  }
  if (-not $ready) {
    $status = 'timeout'
    $errorText = "readiness not reached within $TimeoutSeconds s"
  }

  # Settle: confirm the ready state holds and the shell stays responsive.
  # Process.Responding alone can be false when MainWindowHandle is zero or when
  # the first ready-frame paint is long; ask the specific window too and keep
  # every sample so a transient is not reported as a hang.
  $settleSamples = @()
  $settleDeadline = (Get-Date).AddSeconds($SettleSeconds)
  $index = 0
  while ((Get-Date) -lt $settleDeadline) {
    if ($process.HasExited) { throw "whaleisle exited while settling (exit $($process.ExitCode))" }
    $pingHwnd = [long]0
    $live = @(Read-Windows ([uint32]$process.Id) | Where-Object { $_.visible -and $_.dip.width -eq 1440 })[0]
    if ($null -ne $live) { $pingHwnd = [long]$live.hwnd }
    $settleSamples += [ordered]@{
      t                  = $index
      processResponding  = (Get-Responding $process)
      mainWindowHandle   = [long]$process.MainWindowHandle
      mainHwnd           = $pingHwnd
      mainPing           = if ($pingHwnd -ne 0) { [bool][SmokeWin]::Ping([IntPtr]$pingHwnd, 3000) } else { $false }
    }
    $index++
    Start-Sleep -Milliseconds 1000
  }
  $windowStates = Read-Windows ([uint32]$process.Id)
  $readiness.portListening = Test-PortOpen -P $Port
  $mainWin = @($windowStates | Where-Object { $_.visible -and $_.dip.width -eq 1440 -and $_.dip.height -eq 920 })[0]
  $launcherWin = @($windowStates | Where-Object { $_.visible -and $_.dip.width -eq 1060 -and $_.dip.height -eq 660 })[0]
  $readiness.mainVisible = $null -ne $mainWin
  $readiness.launcherGone = $null -eq $launcherWin
  if ($null -ne $mainWin) { $mainHwnd = [long]$mainWin.hwnd }
  if ($null -ne $launcherWin) { $launcherHwnd = [long]$launcherWin.hwnd }
  $finalPing = if ($mainHwnd -ne 0) { [bool][SmokeWin]::Ping([IntPtr]$mainHwnd, 3000) } else { $false }
  # The Process object caches MainWindowHandle on first use; after the launcher
  # window is destroyed its Responding answer stays pinned to that dead handle.
  # Ask a fresh Process object for the same pid as well.
  $staleResponding = Get-Responding $process
  $freshResponding = $false
  $freshMainWindowHandle = [long]0
  try {
    $fresh = [System.Diagnostics.Process]::GetProcessById($process.Id)
    $fresh.Refresh()
    $freshMainWindowHandle = [long]$fresh.MainWindowHandle
    $freshResponding = [bool]$fresh.Responding
  } catch { Write-Step "fresh process probe failed: $($_.Exception.Message)" }
  $readiness.respondingAtReady = $respondAtReady
  $readiness.respondingFresh = $freshResponding
  $readiness.respondingStaleHandle = $staleResponding
  $readiness.freshMainWindowHandle = $freshMainWindowHandle
  $readiness.mainWindowPing = $finalPing
  $readiness.responding = $respondAtReady -or $freshResponding -or $finalPing
  Write-Step "settle samples: $(($settleSamples | ForEach-Object { 't' + $_.t + ' resp=' + $_.processResponding + ' mwh=' + $_.mainWindowHandle + ' ping=' + $_.mainPing }) -join '; ')"
  Write-Step "respond at-ready=$respondAtReady fresh=$freshResponding (mwh fresh=$freshMainWindowHandle) staleProcessObject=$staleResponding mainPing=$finalPing"
  Write-Step "settled port=$($readiness.portListening) main=$($readiness.mainVisible) launcherGone=$($readiness.launcherGone) responding=$($readiness.responding) mainHwnd=$mainHwnd launcherHwnd=$launcherHwnd"

  # Host children, proving which runtime/probe the shell launched.
  $children = @(Get-Children $process.Id)
  Write-Step "host children: $((@($children | ForEach-Object { $_.name + ':' + $_.pid }) -join ','))"

  if ($readiness.portListening) {
    try {
      $resp = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/" -UseBasicParsing -TimeoutSec 10
      $harnessHttp = [ordered]@{ statusCode = [int]$resp.StatusCode; length = $resp.Content.Length; title = ([regex]::Match($resp.Content, '<title>(.*?)</title>').Groups[1].Value) }
    } catch { $harnessHttp = [ordered]@{ error = $_.Exception.Message } }
  }

  if ($mainHwnd -ne 0) { $uiaDump = Get-UiaDump $mainHwnd }
  if ($launcherHwnd -ne 0) { $uiaLauncher = Get-UiaDump $launcherHwnd }
  $uiaMainCount = 0
  $uiaLauncherCount = 0
  if ($null -ne $uiaDump) { $uiaMainCount = @($uiaDump.nodes).Count }
  if ($null -ne $uiaLauncher) { $uiaLauncherCount = @($uiaLauncher.nodes).Count }
  Write-Step "uia main=$uiaMainCount launcher=$uiaLauncherCount"

  $crashDir = Join-Path $profile 'logs\crash'
  if (Test-Path -LiteralPath $crashDir) {
    foreach ($file in Get-ChildItem -LiteralPath $crashDir -File) {
      $text = [System.IO.File]::ReadAllText($file.FullName)
      $crashReports += [ordered]@{ name = $file.Name; length = $file.Length; text = $text.Substring(0, [math]::Min(8000, $text.Length)) }
    }
  }
  $dshHome = Join-Path $profile 'dsh-home'
  if (Test-Path -LiteralPath $dshHome) {
    $dshHomeListing = @(Get-ChildItem -LiteralPath $dshHome -Force | ForEach-Object { [ordered]@{ name = $_.Name; mode = $_.Mode.ToString() } })
  }
  $clearedSocketList = $clearedSockets
} catch {
  $status = 'error'
  $errorText = $_.Exception.Message
} finally {
  Write-Step "kill tree pid=$($process.Id)"
  if ($null -ne $process -and -not $process.HasExited) {
    # Process tree: the shell plus its node Host child (and grandchildren).
    # A parent-tree walk avoids taskkill /T, which would also signal anything
    # sharing this console.
    try { Stop-Tree ([int]$process.Id) } catch { Write-Step "stop tree error: $($_.Exception.Message)" }
    try { [void]$process.WaitForExit(8000) } catch { }
    if (-not $process.HasExited) { try { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue } catch { } }
  }
  Start-Sleep -Milliseconds 800
  # Any Host child that outlived the tree kill (matched by this profile's dsh-home).
  Get-CimInstance Win32_Process -Filter "Name='node.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.CommandLine -and ($_.CommandLine -like "*$profile*" -or $_.CommandLine -like "*--port $Port*") } |
    ForEach-Object { try { Stop-Process -Id $_.ProcessId -Force -ErrorAction Stop } catch { } }
  if ($createdBase) {
    for ($i = 0; $i -lt 8; $i++) {
      try { Remove-Item -LiteralPath $base -Recurse -Force -ErrorAction Stop; break } catch { Start-Sleep -Milliseconds 400 }
    }
  }
  Write-Step "cleanup done"
}

$result = [ordered]@{
  schema      = 1
  smoke       = 'harness-start'
  method      = 'Start-Process with WHALEISLE_USER_DATA temp profile; sampled TcpClient port probe, EnumWindows+GetWindowText+GetWindowRect (per-monitor-v2 physical px, DIP derived from GetDpiForWindow), Process.Responding; UIAutomation descendants of the main/launcher HWND; recursive ParentProcessId tree kill; no PrintWindow'
  generatedAt = [datetime]::UtcNow.ToString('o')
  status      = $status
  error       = $errorText
  exe         = $exePath
  pid         = if ($null -ne $process) { $process.Id } else { 0 }
  userData    = $profile
  workspace   = $workspace
  port        = $Port
  realConfig  = $RealConfig
  configOriginal = [ordered]@{ workspace = $originalWorkspace; port = $originalPort }
  clearedSockets = [object[]]@($clearedSocketList)
  startedAt   = $startedAt.ToString('o')
  finishedAt  = [datetime]::UtcNow.ToString('o')
  readiness   = $readiness
  settleSamples = [object[]]$settleSamples
  timeline    = [object[]]$timeline
  windowStates = [object[]]$windowStates
  mainHwnd    = $mainHwnd
  launcherHwnd = $launcherHwnd
  uia         = $uiaDump
  uiaLauncher = $uiaLauncher
  hostChildren = [object[]]$children
  harnessHttp = $harnessHttp
  dshHome     = [object[]]$dshHomeListing
  crashReports = [object[]]$crashReports
}
$outDir = Split-Path -Parent $outPath
if ($outDir) { [void](New-Item -ItemType Directory -Path $outDir -Force) }
Write-Step "writing $outPath (status=$status readiness=$($readiness.portListening)/$($readiness.mainVisible)/$($readiness.launcherGone)/$($readiness.responding) samples=$($timeline.Count))"
[System.IO.File]::WriteAllText($outPath, ($result | ConvertTo-Json -Depth 12), (New-Object System.Text.UTF8Encoding($false)))
Write-Step 'written'

Write-Host ("harness-start: status={0} ready={1} port={2} main={3} launcherGone={4} responding={5} samples={6} -> {7}" -f `
  $status, $readiness.portListening, $readiness.portListening, $readiness.mainVisible, $readiness.launcherGone, $readiness.responding, $timeline.Count, $outPath)
if ($status -ne 'ok' -or -not ($readiness.portListening -and $readiness.mainVisible -and $readiness.launcherGone -and $readiness.responding)) { exit 1 }
