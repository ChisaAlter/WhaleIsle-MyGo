param(
  [Parameter(Mandatory = $true)][long]$Hwnd,
  [Parameter(Mandatory = $true)][string]$Out,
  [int]$Flags = 2
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class CapWin {
  [DllImport("user32.dll")] public static extern bool SetProcessDpiAwarenessContext(IntPtr ctx);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr dc, uint flags);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern int GetDpiForWindow(IntPtr h);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
  public static int[] Measure(IntPtr h) { RECT r; GetWindowRect(h, out r); return new[] { r.L, r.T, r.R - r.L, r.B - r.T }; }
  public static int Dpi(IntPtr h) { return GetDpiForWindow(h); }
}
'@
# Per-monitor-v2 so GetWindowRect and PrintWindow use physical pixels.
[void][CapWin]::SetProcessDpiAwarenessContext([IntPtr](-4))
$hwndPtr = [IntPtr]$Hwnd
$m = [CapWin]::Measure($hwndPtr)
$w = [int]$m[2]; $h = [int]$m[3]
if ($w -le 0 -or $h -le 0) { throw "window has no area: ${w}x${h}" }
$dpi = [CapWin]::Dpi($hwndPtr)
$bmp = New-Object System.Drawing.Bitmap($w, $h)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$dc = $g.GetHdc()
try {
  $ok = [CapWin]::PrintWindow($hwndPtr, $dc, [uint32]$Flags)
} finally {
  $g.ReleaseHdc($dc); $g.Dispose()
}
$dir = Split-Path -Parent ([System.IO.Path]::GetFullPath($Out))
if ($dir) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
$bmp.Save([System.IO.Path]::GetFullPath($Out), [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
[pscustomobject]@{ ok = $ok; width = $w; height = $h; dpi = $dpi; dipWidth = [math]::Round($w * 96.0 / $dpi, 1); dipHeight = [math]::Round($h * 96.0 / $dpi, 1); path = [System.IO.Path]::GetFullPath($Out); bytes = (Get-Item $Out).Length }
