# Frame-time probe: drives one scripted interaction in a real MyGo window and
# leaves a per-frame trace for scripts/frame-time-report.mjs to read.
#
# The window is shown without activation and every input goes in as a posted
# message, so a run never takes the operator's mouse or keyboard. Nothing here
# reads the screen: the trace comes from the engine's own frame clock
# (MYGO_FRAME_TRACE, off unless this script sets it).
#
#   powershell -File scripts/frame-probe.ps1 -Action display
#   powershell -File scripts/frame-probe.ps1 -Action run -Interaction all
#
# Build the fixture first:
#   go test -tags inspectorfixture -c -o .devdata/frameprobe-fixture.exe ./internal/ui/frame
param(
    [ValidateSet('run','display')][string]$Action = 'run',
    [ValidateSet('scroll','stream','expand','picker','theme','resize','all')][string]$Interaction = 'all',
    [string]$FixturePath = $env:WHALEISLE_FIXTURE_EXE,
    [string]$OutDir = '',
    [int]$Repeat = 1
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $FixturePath) { $FixturePath = Join-Path $projectRoot '.devdata\frameprobe-fixture.exe' }
if (-not $OutDir) { $OutDir = Join-Path $projectRoot '.devdata\frame-probe' }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public class MyGoFrameProbe {
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int c);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
 [DllImport("user32.dll")] static extern IntPtr FindWindowEx(IntPtr p, IntPtr a, string c, string t);
 [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h, ref POINT p);
 [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h, out RECT r);
 [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern IntPtr FindWindowW(string c, string t);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll")] static extern bool SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll", CharSet = CharSet.Unicode)] static extern bool EnumDisplaySettingsW(string dev, int mode, ref DEVMODE dm);
 [DllImport("dwmapi.dll")] public static extern int DwmFlush();
 [DllImport("kernel32.dll")] static extern bool QueryPerformanceCounter(out long c);
 [DllImport("kernel32.dll")] static extern bool QueryPerformanceFrequency(out long f);
 public struct POINT { public int X, Y; }
 public struct RECT { public int L, T, R, B; }
 [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)] public struct DEVMODE {
  [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 32)] public string dmDeviceName;
  public ushort dmSpecVersion, dmDriverVersion, dmSize, dmDriverExtra; public uint dmFields;
  public int dmPositionX, dmPositionY; public uint dmDisplayOrientation, dmDisplayFixedOutput;
  public short dmColor, dmDuplex, dmYResolution, dmTTOption, dmCollate;
  [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 32)] public string dmFormName;
  public ushort dmLogPixels; public uint dmBitsPerPel, dmPelsWidth, dmPelsHeight, dmDisplayFlags, dmDisplayFrequency;
  public uint dmICMMethod, dmICMIntent, dmMediaType, dmDitherType, dmReserved1, dmReserved2, dmPanningWidth, dmPanningHeight;
 }
 public static void Dpi() { SetThreadDpiAwarenessContext(new IntPtr(-4)); }
 public static IntPtr Surface(IntPtr h) { return FindWindowEx(h, IntPtr.Zero, "MyGoSurface", null); }
 public static string TitleOf(string title) { return FindWindowW(null, title) == IntPtr.Zero ? "" : title; }
 // WindowOf returns the window with this title that belongs to pid. A hidden
 // window has no MainWindowHandle until it is shown, so it is found by title
 // and checked against the process that owns it.
 public static IntPtr WindowOf(int pid, string title) {
  var h = FindWindowW(null, title);
  if (h == IntPtr.Zero) { return IntPtr.Zero; }
  uint owner; GetWindowThreadProcessId(h, out owner);
  return owner == (uint)pid ? h : IntPtr.Zero;
 }
 public static POINT OriginOf(IntPtr h) {
  var s = Surface(h); if (s == IntPtr.Zero) s = h;
  var p = new POINT { X = 0, Y = 0 }; ClientToScreen(s, ref p); return p;
 }
 public static RECT Client(IntPtr h) {
  var s = Surface(h); if (s == IntPtr.Zero) s = h;
  RECT r; GetClientRect(s, out r); return r;
 }
 static IntPtr XY(int x, int y) { return new IntPtr((y << 16) | (x & 0xFFFF)); }
 public static void Move(IntPtr h, int x, int y) {
  PostMessage(Surface(h), 0x200, IntPtr.Zero, XY(x, y));
 }
 public static void Click(IntPtr h, int x, int y) {
  var s = Surface(h);
  PostMessage(s, 0x200, IntPtr.Zero, XY(x, y));
  System.Threading.Thread.Sleep(16);
  PostMessage(s, 0x201, new IntPtr(1), XY(x, y));
  System.Threading.Thread.Sleep(24);
  PostMessage(s, 0x202, IntPtr.Zero, XY(x, y));
 }
 public static void Key(IntPtr h, int vk) {
  var s = Surface(h);
  PostMessage(s, 0x100, new IntPtr(vk), new IntPtr(1));
  System.Threading.Thread.Sleep(16);
  PostMessage(s, 0x101, new IntPtr(vk), new IntPtr(unchecked((int)0xC0000001)));
 }
 public static void Wheel(IntPtr h, int screenX, int screenY, int delta) {
  PostMessage(Surface(h), 0x20A, new IntPtr((delta << 16) & unchecked((int)0xFFFF0000)), XY(screenX, screenY));
 }
 public static int RefreshHz() {
  var dm = new DEVMODE(); dm.dmSize = (ushort)Marshal.SizeOf(dm);
  if (!EnumDisplaySettingsW(null, -1, ref dm)) { return 0; }
  return (int)dm.dmDisplayFrequency;
 }
 public static string Mode() {
  var dm = new DEVMODE(); dm.dmSize = (ushort)Marshal.SizeOf(dm);
  if (!EnumDisplaySettingsW(null, -1, ref dm)) { return "unknown"; }
  return dm.dmPelsWidth + "x" + dm.dmPelsHeight + "@" + dm.dmDisplayFrequency + "Hz";
 }
 public static double CompositionRate() {
  long f; QueryPerformanceFrequency(out f);
  long start, now; QueryPerformanceCounter(out start);
  int n = 0;
  while (true) {
   DwmFlush(); n++;
   QueryPerformanceCounter(out now);
   if ((now - start) / (double)f >= 2.0) { break; }
  }
  return n / ((now - start) / (double)f);
 }
}
'@
[void][MyGoFrameProbe]::Dpi()

if ($Action -eq 'display') {
    $before = [MyGoFrameProbe]::GetForegroundWindow()
    "display_mode=$([MyGoFrameProbe]::Mode())"
    "enum_display_hz=$([MyGoFrameProbe]::RefreshHz())"
    "dwm_flush_rate_hz=$([math]::Round([MyGoFrameProbe]::CompositionRate(),1))"
    "foreground_before=$before"
    "foreground_after=$([MyGoFrameProbe]::GetForegroundWindow())"
    return
}

$fixturePath = [IO.Path]::GetFullPath($FixturePath)
if (-not (Test-Path -LiteralPath $fixturePath -PathType Leaf)) {
    throw "Fixture executable not found at $fixturePath. Build it: go test -tags inspectorfixture -c -o .devdata/frameprobe-fixture.exe ./internal/ui/frame"
}
$interactions = if ($Interaction -eq 'all') { @('scroll', 'stream', 'expand', 'picker', 'theme', 'resize') } else { @($Interaction) }

function Assert-NoFocusTheft([IntPtr]$hwnd, [IntPtr]$before, [string]$step) {
    $fg = [MyGoFrameProbe]::GetForegroundWindow()
    if ($fg -eq $hwnd) {
        throw "focus stolen after $step (foreground=$fg, before=$before)"
    }
}

for ($round = 1; $round -le $Repeat; $round++) {
    foreach ($name in $interactions) {
        $tag = ''
        if ($Repeat -gt 1) { $tag = "run$round-" }
        $prefix = Join-Path $OutDir ($tag + $name)
        $trace = "$prefix-trace.csv"
        Remove-Item -LiteralPath $trace -ErrorAction SilentlyContinue
        $foregroundBefore = [MyGoFrameProbe]::GetForegroundWindow()
        $env:INSPECTOR_NATIVE_FIXTURE = Join-Path $projectRoot 'internal\ui\frame\testdata\inspector-records.json'
        $env:INSPECTOR_NATIVE_FIXTURE_SCENARIO = 'frameprobe'
        $env:INSPECTOR_NATIVE_FIXTURE_OUTPUT = $prefix
        $env:MYGO_FRAME_TRACE = $trace
        $process = Start-Process -FilePath $fixturePath -PassThru
        $hwnd = [IntPtr]::Zero
        for ($i = 0; $i -lt 300; $i++) {
            Start-Sleep -Milliseconds 100
            $process.Refresh()
            if ($process.MainWindowHandle -ne [IntPtr]::Zero) { $hwnd = $process.MainWindowHandle; break }
            $found = [MyGoFrameProbe]::WindowOf($process.Id, 'MyGo frame probe fixture')
            if ($found -ne [IntPtr]::Zero) { $hwnd = $found; break }
        }
        if ($hwnd -eq [IntPtr]::Zero) { Stop-Process -Id $process.Id -Force; throw "no window for $name" }
        [void][MyGoFrameProbe]::ShowWindow($hwnd, 4)
        [void][MyGoFrameProbe]::SetWindowPos($hwnd, [System.IntPtr]::new(-1), 80, 80, 1280, 720, 0x0010 -bor 0x0040)
        Start-Sleep -Milliseconds 700
        Assert-NoFocusTheft $hwnd $foregroundBefore "show"
        $client = [MyGoFrameProbe]::Client($hwnd)
        $origin = [MyGoFrameProbe]::OriginOf($hwnd)
        $chatX = [int]($client.R / 2)
        $chatY = [int]($client.B / 2)
        $pickerX = $client.R - 130
        try {
            switch ($name) {
                'scroll' {
                    [MyGoFrameProbe]::Key($hwnd, 0x70)
                    Start-Sleep -Milliseconds 120
                    [MyGoFrameProbe]::Move($hwnd, $chatX, $chatY)
                    Start-Sleep -Milliseconds 120
                    for ($i = 0; $i -lt 40; $i++) {
                        [MyGoFrameProbe]::Wheel($hwnd, $origin.X + $chatX, $origin.Y + $chatY, -120)
                        Start-Sleep -Milliseconds 16
                    }
                }
                'stream' {
                    [MyGoFrameProbe]::Key($hwnd, 0x71)
                    Start-Sleep -Milliseconds 2500
                    [MyGoFrameProbe]::Key($hwnd, 0x76)
                }
                'expand' {
                    [MyGoFrameProbe]::Key($hwnd, 0x72); Start-Sleep -Milliseconds 600
                    [MyGoFrameProbe]::Key($hwnd, 0x72)
                }
                'picker' {
                    [MyGoFrameProbe]::Key($hwnd, 0x73)
                    Start-Sleep -Milliseconds 120
                    [MyGoFrameProbe]::Click($hwnd, $pickerX, 22); Start-Sleep -Milliseconds 700
                    [MyGoFrameProbe]::Click($hwnd, $pickerX, 22); Start-Sleep -Milliseconds 700
                }
                'theme' {
                    [MyGoFrameProbe]::Key($hwnd, 0x74); Start-Sleep -Milliseconds 600
                    [MyGoFrameProbe]::Key($hwnd, 0x74); Start-Sleep -Milliseconds 600
                    [MyGoFrameProbe]::Key($hwnd, 0x74)
                }
                'resize' {
                    [MyGoFrameProbe]::Key($hwnd, 0x75); Start-Sleep -Milliseconds 600
                    [MyGoFrameProbe]::Key($hwnd, 0x75); Start-Sleep -Milliseconds 600
                    [MyGoFrameProbe]::Key($hwnd, 0x75)
                }
            }
            Start-Sleep -Milliseconds 800
            Assert-NoFocusTheft $hwnd $foregroundBefore "$name"
        }
        finally {
            $alive = Get-Process -Id $process.Id -ErrorAction SilentlyContinue
            if ($alive) { Stop-Process -Id $process.Id -Force }
            try { $process.WaitForExit(10000) | Out-Null } catch {}
            $still = Get-Process -Id $process.Id -ErrorAction SilentlyContinue
            $unresponsive = [MyGoFrameProbe]::TitleOf([char]0x754C + [char]0x9762 + [char]0x672A + [char]0x54CD + [char]0x5E94)
            $frames = 0
            if (Test-Path -LiteralPath $trace) {
                $frames = (Get-Content -LiteralPath $trace | Measure-Object).Count - 1
            }
            "{0}: frames={1} trace={2} foreground_before={3} foreground_after={4} exited={5} unresponsive_window={6}" -f $name, $frames, $trace, $foregroundBefore, ([MyGoFrameProbe]::GetForegroundWindow()), ($null -eq $still), $(if ($unresponsive) { $unresponsive } else { 'none' })
        }
    }
}
