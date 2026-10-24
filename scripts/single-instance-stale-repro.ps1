# Single-instance stale-socket reproduction: start the app with a throwaway
# profile, force-kill it (simulating a crash), then start it again and report
# whether the second launch opens a window.
#
# Before the round145 fix the second launch exited 0 with no window whenever the
# leftover socket had a live-looking listener (a crash, or a child process that
# inherited the listening handle). The app now logs "the single-instance socket
# has no live owner; taking it over" and shows its window.
#
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts/single-instance-stale-repro.ps1 `
#     -Exe C:\AI\WhaleIsle-MyGo\whaleisle.exe
param(
  [Parameter(Mandatory = $true)][string]$Exe,
  [int]$WindowTimeoutSeconds = 20
)
$ErrorActionPreference = 'Stop'
$exePath = (Resolve-Path -LiteralPath $Exe).Path
$profile = Join-Path ([System.IO.Path]::GetTempPath()) ('whaleisle-si-' + [guid]::NewGuid().ToString('N'))
$socket = Join-Path ([System.IO.Path]::GetTempPath()) 'mygo--1-0c44173c701a.sock' # MyGo鲸屿 single-instance socket
$result = [ordered]@{ exe = $exePath; userData = $profile; firstLaunch = $null; staleSocketLeft = $false; secondLaunch = $null; ok = $false }

function Wait-Window([System.Diagnostics.Process]$Process) {
  $deadline = (Get-Date).AddSeconds($WindowTimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    $Process.Refresh() # Process properties are cached until refreshed
    if ($Process.HasExited) { return $null }
    if ($Process.MainWindowTitle) { return $Process.MainWindowTitle }
    Start-Sleep -Milliseconds 250
  }
  return $null
}

try {
  [void](New-Item -ItemType Directory -Path $profile -Force)
  # Skip the first-run welcome window so the launcher/main window appears.
  [System.IO.File]::WriteAllText((Join-Path $profile 'welcome-shown'), '')
  $env:WHALEISLE_USER_DATA = $profile
  try {
    $first = Start-Process -FilePath $exePath -WorkingDirectory (Split-Path -Parent $exePath) -PassThru
  } finally {
    Remove-Item Env:WHALEISLE_USER_DATA -ErrorAction SilentlyContinue
  }
  $result.firstLaunch = Wait-Window $first
  if (-not $result.firstLaunch) { throw "the first launch opened no window" }
  Stop-Process -Id $first.Id -Force   # a crash leaves the socket behind
  Start-Sleep -Seconds 2
  $result.staleSocketLeft = Test-Path -LiteralPath $socket

  try {
    $second = Start-Process -FilePath $exePath -WorkingDirectory (Split-Path -Parent $exePath) -PassThru
  } finally {
    Remove-Item Env:WHALEISLE_USER_DATA -ErrorAction SilentlyContinue
  }
  $result.secondLaunch = Wait-Window $second
  if (-not $second.HasExited) { Stop-Process -Id $second.Id -Force }
  $result.ok = [bool]$result.secondLaunch
} finally {
  Remove-Item Env:WHALEISLE_USER_DATA -ErrorAction SilentlyContinue
  Get-Process -Name ([System.IO.Path]::GetFileNameWithoutExtension($exePath)) -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $exePath } | Stop-Process -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $profile -Recurse -Force -ErrorAction SilentlyContinue
  if ($result.staleSocketLeft) { Remove-Item -LiteralPath $socket -Force -ErrorAction SilentlyContinue }
}

$result | ConvertTo-Json -Depth 4
if (-not $result.ok) {
  Write-Host "FAILED: the launch after a force kill opened no window"
  exit 1
}
Write-Host ("ok: first '{0}', second '{1}'" -f $result.firstLaunch, $result.secondLaunch)
