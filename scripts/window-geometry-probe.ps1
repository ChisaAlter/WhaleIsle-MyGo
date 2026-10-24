# Window geometry probe: start a whaleisle build with a throwaway profile, then
# record every top-level window's title and rectangle.
#
# It never calls PrintWindow or BitBlt: PrintWindow(PW_RENDERFULLCONTENT) hangs
# this app's WebView2 windows (verified), so geometry comes from GetWindowRect
# only. The process it starts is stopped in finally, and the temp profile is
# removed.
#
#   pwsh -File scripts/window-geometry-probe.ps1 `
#     -Exe C:\AI\WhaleIsle-MyGo\whaleisle.exe `
#     -Out docs/review-20261022/evidence/window-geometry-pass272.json
param(
  [Parameter(Mandatory = $true)][string]$Exe,
  [Parameter(Mandatory = $true)][string]$Out,
  [int]$TimeoutSeconds = 60,
  [int]$SettleMilliseconds = 2500,
  [string]$UserData = ''
)
$ErrorActionPreference = 'Stop'

$exePath = (Resolve-Path -LiteralPath $Exe).Path
$outPath = [System.IO.Path]::GetFullPath((Join-Path (Get-Location).Path $Out))
$profile = if ($UserData) { [System.IO.Path]::GetFullPath($UserData) } else { Join-Path ([System.IO.Path]::GetTempPath()) ('whaleisle-geometry-' + [guid]::NewGuid().ToString('N')) }
$createdProfile = -not (Test-Path -LiteralPath $profile)

Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public class WindowGeometryProbe {
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

  // One record per top-level window of pid: title, class, physical rect, state.
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
        title.Replace("\u001f", " "),
        cls,
        rect.Left.ToString(), rect.Top.ToString(),
        (rect.Right - rect.Left).ToString(), (rect.Bottom - rect.Top).ToString(),
        IsWindowVisible(hWnd) ? "1" : "0",
        IsIconic(hWnd) ? "1" : "0",
        dpi.ToString(),
      }));
      return true;
    }, IntPtr.Zero);
    return rows;
  }
}
'@

[void][WindowGeometryProbe]::UsePhysicalPixels()

function Read-Windows([uint32]$ProcessId) {
  $rows = [WindowGeometryProbe]::Windows($ProcessId)
  $windows = @()
  foreach ($row in $rows) {
    $f = $row.Split([char]31)
    $windows += [ordered]@{
      title     = $f[0]
      class     = $f[1]
      rect      = [ordered]@{ left = [int]$f[2]; top = [int]$f[3]; width = [int]$f[4]; height = [int]$f[5] }
      visible   = $f[6] -eq '1'
      minimized = $f[7] -eq '1'
      dpi       = [int]$f[8]
    }
  }
  return $windows
}

$startedAt = [datetime]::UtcNow
$process = $null
$windows = @()
$status = 'ok'
$errorText = $null
try {
  if ($createdProfile) { [void](New-Item -ItemType Directory -Path $profile -Force) }
  # A profile that already saw the welcome window starts straight into the
  # launcher (cmd/whaleisle/welcome_window.go: welcomeMarker), which is the
  # window this probe measures.
  [System.IO.File]::WriteAllText((Join-Path $profile 'welcome-shown'), '')

  $previousProfile = $env:WHALEISLE_USER_DATA
  $env:WHALEISLE_USER_DATA = $profile
  try {
    $process = Start-Process -FilePath $exePath -WorkingDirectory (Split-Path -Parent $exePath) -PassThru
  } finally {
    if ($null -eq $previousProfile) { Remove-Item Env:WHALEISLE_USER_DATA -ErrorAction SilentlyContinue }
    else { $env:WHALEISLE_USER_DATA = $previousProfile }
  }

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $seen = $false
  while ((Get-Date) -lt $deadline) {
    if ($process.HasExited) { throw "whaleisle exited before a window appeared (exit $($process.ExitCode))" }
    $windows = Read-Windows ([uint32]$process.Id)
    if (@($windows | Where-Object { $_.visible -and $_.title }).Count -gt 0) { $seen = $true; break }
    Start-Sleep -Milliseconds 250
  }
  if (-not $seen) { throw "no titled top-level window within $TimeoutSeconds s" }

  # Let the window finish its first layout so the rect is the settled one.
  Start-Sleep -Milliseconds $SettleMilliseconds
  $windows = Read-Windows ([uint32]$process.Id)
} catch {
  $status = 'error'
  $errorText = $_.Exception.Message
} finally {
  if ($null -ne $process -and -not $process.HasExited) {
    [void]$process.CloseMainWindow()
    if (-not $process.WaitForExit(5000)) {
      # Windows PowerShell 5.1 has no Process.Kill(tree); the owned PID is ours.
      try { Stop-Process -Id $process.Id -Force -ErrorAction Stop } catch { }
      try { [void]$process.WaitForExit(5000) } catch { }
    }
  }
  if ($createdProfile) {
    for ($i = 0; $i -lt 10; $i++) {
      try { Remove-Item -LiteralPath $profile -Recurse -Force -ErrorAction Stop; break } catch { Start-Sleep -Milliseconds 250 }
    }
  }
}

$result = [ordered]@{
  schema      = 1
  probe       = 'window-geometry'
  method      = 'EnumWindows + GetWindowThreadProcessId + GetWindowText + GetWindowRect + IsWindowVisible + IsIconic (per-monitor-v2 physical pixels); never PrintWindow'
  generatedAt = [datetime]::UtcNow.ToString('o')
  status      = $status
  error       = $errorText
  exe         = $exePath
  pid         = if ($null -ne $process) { $process.Id } else { 0 }
  userData    = $profile
  startedAt   = $startedAt.ToString('o')
  windows     = $windows
}
$outDir = Split-Path -Parent $outPath
if ($outDir) { [void](New-Item -ItemType Directory -Path $outDir -Force) }
$json = $result | ConvertTo-Json -Depth 8
[System.IO.File]::WriteAllText($outPath, $json, (New-Object System.Text.UTF8Encoding($false)))

$windows | ForEach-Object { '{0}  {1}x{2} at {3},{4}  visible={5} minimized={6} dpi={7}' -f $_.title, $_.rect.width, $_.rect.height, $_.rect.left, $_.rect.top, $_.visible, $_.minimized, $_.dpi }
Write-Host ("window-geometry: status={0} windows={1} -> {2}" -f $status, @($windows).Count, $outPath)
if ($status -ne 'ok') { exit 1 }

