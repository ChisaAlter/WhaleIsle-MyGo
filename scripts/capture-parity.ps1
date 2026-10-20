# Capture the native shell's main window for UI parity acceptance.
#
# Launches an isolated instance (own userData + port + single-instance identity),
# optionally with the dark theme and a synthetic wallpaper, then captures the
# main window with PrintWindow. The installed Electron client is only read for
# a reference capture.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts\capture-parity.ps1
#   ... -Exe .\whaleisle.exe -OutDir docs\review-20261020\evidence\after -Theme dark
param(
    [string]$Exe = "$(Split-Path -Parent $PSScriptRoot)\whaleisle.exe",
    [string]$OutDir = "$(Split-Path -Parent $PSScriptRoot)\docs\review-20261020\evidence\after",
    [string]$Theme = "deepseek.dark",
    [int]$Port = 3191,
    [string]$Wallpaper = "",
    [string]$Vendor = "C:\软件\Whale Isle\resources\vendor",
    [int]$Width = 1440,
    [int]$Height = 900
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $root '.devdata\accept') | Out-Null
$data = Join-Path $root ('.devdata\accept\' + (Get-Date -Format 'yyyyMMdd-HHmmss'))

# Synthetic wallpaper so translucent surfaces have something to sample when the
# caller does not pass the real one.
if ($Wallpaper -eq "") {
    Add-Type -AssemblyName System.Drawing
    $wp = Join-Path $data 'wallpaper.png'
    New-Item -ItemType Directory -Force -Path $data | Out-Null
    $bmp = New-Object System.Drawing.Bitmap 2560, 1440
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $rect = New-Object System.Drawing.Rectangle 0, 0, 2560, 1440
    $brush = New-Object System.Drawing.Drawing2D.LinearGradientBrush $rect, ([System.Drawing.Color]::FromArgb(255, 18, 22, 34)), ([System.Drawing.Color]::FromArgb(255, 72, 46, 92)), 35
    $g.FillRectangle($brush, $rect)
    $dots = @(
        @{ X = 420; Y = 300; R = 260; C = [System.Drawing.Color]::FromArgb(120, 120, 200, 255) },
        @{ X = 1900; Y = 420; R = 340; C = [System.Drawing.Color]::FromArgb(100, 255, 170, 220) },
        @{ X = 1300; Y = 1000; R = 300; C = [System.Drawing.Color]::FromArgb(90, 120, 255, 210) }
    )
    foreach ($d in $dots) {
        $b2 = New-Object System.Drawing.Drawing2D.GraphicsPath
        $b2.AddEllipse($d.X - $d.R, $d.Y - $d.R, $d.R * 2, $d.R * 2)
        $br = New-Object System.Drawing.SolidBrush $d.C
        $g.FillPath($br, $b2)
        $br.Dispose(); $b2.Dispose()
    }
    $brush.Dispose(); $g.Dispose()
    $bmp.Save($wp, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    $Wallpaper = $wp
}

New-Item -ItemType Directory -Force -Path $data | Out-Null
$config = @{
    theme           = $Theme
    port            = $Port
    goWallpaperPath = $Wallpaper
    locale          = 'zh'
    closeToTray     = $false
    askOnUpdate     = $false
    autoStartDesktop = $true
} | ConvertTo-Json -Depth 4
[System.IO.File]::WriteAllText((Join-Path $data 'config.json'), $config, (New-Object System.Text.UTF8Encoding($false)))

$env:WHALEISLE_USER_DATA = $data
$env:WHALEISLE_VENDOR = $Vendor
$env:WHALEISLE_ISOLATED = (Join-Path $data 'native')

$p = Start-Process -FilePath $Exe -WorkingDirectory $root -PassThru
$deadline = (Get-Date).AddSeconds(90)
$hwnd = [IntPtr]::Zero
while ((Get-Date) -lt $deadline) {
    Start-Sleep -Milliseconds 500
    $p.Refresh()
    if ($p.HasExited) { throw "native shell exited early: $($p.ExitCode)" }
    if ($p.MainWindowHandle -ne 0) { $hwnd = $p.MainWindowHandle; break }
}
if ($hwnd -eq [IntPtr]::Zero) { Stop-Process -Id $p.Id -Force; throw 'main window did not appear' }
Start-Sleep -Seconds 6  # boot -> ready transition

Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public class ParityCapture {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT r);
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdc, uint flags);
    [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr hWnd, IntPtr after, int x, int y, int cx, int cy, uint flags);
}
'@
# Fixed size keeps before/after captures comparable.
[void][ParityCapture]::SetWindowPos($hwnd, [IntPtr]::Zero, 40, 40, $Width, $Height, 0x0004)
Start-Sleep -Milliseconds 800
$r = New-Object ParityCapture+RECT
[void][ParityCapture]::GetWindowRect($hwnd, [ref]$r)
$w = $r.Right - $r.Left; $h = $r.Bottom - $r.Top
$bmp = New-Object System.Drawing.Bitmap $w, $h
$g = [System.Drawing.Graphics]::FromImage($bmp)
$hdc = $g.GetHdc()
$ok = [ParityCapture]::PrintWindow($hwnd, $hdc, 2)
$g.ReleaseHdc($hdc); $g.Dispose()
$shot = Join-Path $OutDir ('native-' + $Theme + '-' + $w + 'x' + $h + '.png')
$bmp.Save($shot, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Write-Output "native=$shot ok=$ok size=${w}x${h} data=$data"

# The installed Electron client is a read-only reference.
$ref = Get-Process -Name 'Whale Isle' -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1
if ($ref) {
    $rr = New-Object ParityCapture+RECT
    [void][ParityCapture]::GetWindowRect($ref.MainWindowHandle, [ref]$rr)
    $rw = $rr.Right - $rr.Left; $rh = $rr.Bottom - $rr.Top
    $rb = New-Object System.Drawing.Bitmap $rw, $rh
    $rg = [System.Drawing.Graphics]::FromImage($rb)
    $rhdc = $rg.GetHdc()
    $rok = [ParityCapture]::PrintWindow($ref.MainWindowHandle, $rhdc, 2)
    $rg.ReleaseHdc($rhdc); $rg.Dispose()
    $rshot = Join-Path $OutDir ('original-' + $rw + 'x' + $rh + '.png')
    $rb.Save($rshot, [System.Drawing.Imaging.ImageFormat]::Png)
    $rb.Dispose()
    Write-Output "original=$rshot ok=$rok size=${rw}x${rh}"
} else {
    Write-Output 'original=not-running'
}

Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
Write-Output "capture-complete data=$data"
