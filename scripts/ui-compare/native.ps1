param(
  [Parameter(Mandatory=$true)][int]$ProcId,
  [Parameter(Mandatory=$true)][string]$Command,   # dump|find|invoke|expand|collapse|setclient|capture
  [string]$Name = "",
  [string]$Prefix = "",
  [string]$Type = "",
  [int]$Index = 0,
  [int]$Up = 0,
  [string]$UpToType = "",
  [int]$Width = 0,
  [int]$Height = 0,
  [string]$Out = "",
  [int]$Depth = 8
)
# UIA helper for scripts/ui-compare/run.mjs. All output is JSON on stdout
# (UTF-8). Every rect is physical pixels; the caller divides by scale.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class WIUWin {
  [DllImport("user32.dll")] public static extern bool SetProcessDpiAwarenessContext(IntPtr ctx);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr ins, int x, int y, int cx, int cy, uint fl);
  [DllImport("user32.dll")] public static extern int GetDpiForWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool ClientToScreen(IntPtr h, ref POINT p);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr dc, uint flags);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
  [StructLayout(LayoutKind.Sequential)] public struct POINT { public int x, y; }
}
'@
[void][WIUWin]::SetProcessDpiAwarenessContext([IntPtr](-4))
[Console]::OutputEncoding = [Text.Encoding]::UTF8

$proc = Get-Process -Id $ProcId
$hwnd = $proc.MainWindowHandle
if ($hwnd -eq [IntPtr]::Zero) { throw "pid $ProcId has no main window" }

function Rect-Of($r) {
  [pscustomobject]@{ x=[math]::Round($r.X,1); y=[math]::Round($r.Y,1); w=[math]::Round($r.Width,1); h=[math]::Round($r.Height,1) }
}
function Find-Elements($root, $name, $type, $prefix) {
  $cond = [System.Windows.Automation.Condition]::TrueCondition
  $out = @()
  try {
    $all = $root.FindAll([System.Windows.Automation.TreeScope]::Descendants, $cond)
  } catch {
    return @()
  }
  foreach ($el in $all) {
    try {
      $n = $el.Current.Name
      if ($name -ne "" -and $n -ne $name) { continue }
      if ($prefix -ne "" -and -not $n.StartsWith($prefix)) { continue }
      if ($type -ne "" -and $el.Current.ControlType.ProgrammaticName -ne "ControlType.$type") { continue }
      $out += $el
    } catch {}
  }
  return $out
}

$root = [System.Windows.Automation.AutomationElement]::FromHandle($hwnd)
if ($null -eq $root) { throw "no automation element for hwnd $hwnd" }

switch ($Command) {
  'dump' {
    function Walk($el, $d) {
      $rows = ,([pscustomobject]@{ depth=$d; name=$el.Current.Name; type=$el.Current.ControlType.ProgrammaticName.Replace('ControlType.',''); rect=(Rect-Of $el.Current.BoundingRectangle) })
      if ($d -lt $Depth) {
        $child = [System.Windows.Automation.TreeWalker]::ContentViewWalker.GetFirstChild($el)
        while ($null -ne $child) {
          $rows += Walk $child ($d+1)
          $child = [System.Windows.Automation.TreeWalker]::ContentViewWalker.GetNextSibling($child)
        }
      }
      return $rows
    }
    Walk $root 0 | ConvertTo-Json -Depth 4
  }
  'find' {
    $els = @(Find-Elements $root $Name $Type $Prefix)
    if ($Up -gt 0 -or $UpToType -ne "") {
      # Map a found element to an ancestor: Up levels of RawView parents, or
      # until the parent's ControlType matches UpToType (whichever ends first).
      $walker = [System.Windows.Automation.TreeWalker]::RawViewWalker
      $els = @($els | ForEach-Object {
        $el = $_
        $n = $Up
        while ($el -ne $null) {
          if ($UpToType -ne "" -and $el.Current.ControlType.ProgrammaticName -eq "ControlType.$UpToType") { break }
          $p = $walker.GetParent($el)
          if ($null -eq $p) { $el = $null; break }
          $el = $p
          if ($UpToType -eq "") { $n--; if ($n -le 0) { break } }
        }
        if ($el -ne $null) { $el }
      } | Where-Object { $null -ne $_ })
    }
    $res = @()
    foreach ($el in $els) {
      $pats = @()
      foreach ($p in $el.GetSupportedPatterns()) { $pats += ($p.ProgrammaticName -replace 'PatternIdentifiers\.Pattern$','' -replace 'Pattern$','') }
      $res += [pscustomobject]@{ name=$el.Current.Name; type=$el.Current.ControlType.ProgrammaticName; rect=(Rect-Of $el.Current.BoundingRectangle); patterns=$pats; enabled=$el.Current.IsEnabled }
    }
    [pscustomobject]@{ count=$els.Count; index=$Index; hits=$res } | ConvertTo-Json -Depth 4
  }
  { $_ -in 'invoke','expand','collapse' } {
    $els = @(Find-Elements $root $Name $Type $Prefix)
    if ($els.Count -le $Index) { [pscustomobject]@{ ok=$false; error="no match: $Name/$Type[$Index] ($($els.Count) hits)" } | ConvertTo-Json; break }
    $el = $els[$Index]
    try {
      switch ($Command) {
        'invoke'   { $el.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke() }
        'expand'   { $el.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Expand() }
        'collapse' { $el.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Collapse() }
      }
      [pscustomobject]@{ ok=$true; action=$Command; name=$Name; index=$Index } | ConvertTo-Json
    } catch {
      [pscustomobject]@{ ok=$false; error=$_.Exception.Message; name=$Name; index=$Index } | ConvertTo-Json
    }
  }
  'setclient' {
    $wr = New-Object WIUWin+RECT; [WIUWin]::GetWindowRect($hwnd,[ref]$wr) | Out-Null
    $cr = New-Object WIUWin+RECT; [WIUWin]::GetClientRect($hwnd,[ref]$cr) | Out-Null
    $dx = ($wr.R-$wr.L)-$cr.R; $dy = ($wr.B-$wr.T)-$cr.B
    # -Width/-Height are DIP targets; SetWindowPos takes physical pixels.
    $dpi = [WIUWin]::GetDpiForWindow($hwnd); $px = $dpi / 96.0
    [WIUWin]::SetWindowPos($hwnd,[IntPtr]::Zero,0,0,[int]($Width*$px)+$dx,[int]($Height*$px)+$dy,0x16) | Out-Null
    Start-Sleep -Milliseconds 400
    [WIUWin]::GetWindowRect($hwnd,[ref]$wr) | Out-Null
    [WIUWin]::GetClientRect($hwnd,[ref]$cr) | Out-Null
    $pt = New-Object WIUWin+POINT; [WIUWin]::ClientToScreen($hwnd,[ref]$pt) | Out-Null
    [pscustomobject]@{ ok=$true; hwnd=$hwnd.ToInt64(); winW=$wr.R-$wr.L; winH=$wr.B-$wr.T; clientW=$cr.R; clientH=$cr.B; dpi=[WIUWin]::GetDpiForWindow($hwnd); clientX=$pt.x; clientY=$pt.y } | ConvertTo-Json
  }
  'capture' {
    $m = New-Object WIUWin+RECT; [WIUWin]::GetWindowRect($hwnd,[ref]$m) | Out-Null
    $w = $m.R-$m.L; $h = $m.B-$m.T
    if ($w -le 0 -or $h -le 0) { throw "window has no area: ${w}x${h}" }
    $bmp = New-Object System.Drawing.Bitmap($w, $h)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $dc = $g.GetHdc()
    try { $ok = [WIUWin]::PrintWindow($hwnd, $dc, 2) } finally { $g.ReleaseHdc($dc); $g.Dispose() }
    $dir = Split-Path -Parent ([IO.Path]::GetFullPath($Out))
    if ($dir) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
    $bmp.Save([IO.Path]::GetFullPath($Out), [Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    $pt = New-Object WIUWin+POINT; [WIUWin]::ClientToScreen($hwnd,[ref]$pt) | Out-Null
    [pscustomobject]@{ ok=$ok; width=$w; height=$h; dpi=[WIUWin]::GetDpiForWindow($hwnd); clientX=$pt.x; clientY=$pt.y; path=[IO.Path]::GetFullPath($Out) } | ConvertTo-Json
  }
  default { throw "unknown command $Command" }
}
