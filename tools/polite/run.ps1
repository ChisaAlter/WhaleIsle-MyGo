#Requires -Version 5.1
<#
.SYNOPSIS
One polite run of the MyGo desktop on the operator's own machine.

.DESCRIPTION
Starts exactly one whaleisle.exe with --background against a fresh isolated
profile under %TEMP%, drives it only through agent-desktop headless semantic
actions, and fails loudly the moment the OS foreground window changes or the
native-UI watchdog raises its 界面未响应 window.

Dot-source this file to get the primitives, or run it to do one whole run:

    . .\tools\polite\run.ps1
    $s = Start-PoliteInstance
    Invoke-PoliteStep -Session $s -Name 'snapshot' -Action { Read-PoliteSnapshot -Session $s }
    Close-PoliteInstance -Session $s

    .\tools\polite\run.ps1 -Ops .\my-lane-ops.ps1

An -Ops file is dot-sourced after the window is up and must define
Invoke-PoliteOps -Session $Session. See README.md.
#>
[CmdletBinding()]
param(
    [Alias('Exe')][string]$CliExe,
    [Alias('Ops')][string]$CliOps,
    [Alias('TargetRef')][string]$CliTargetRef,
    [Alias('TargetAction')]
    [ValidateSet('click', 'clear', 'set-value', 'uncheck', 'expand', 'collapse')]
    [string]$CliTargetAction = 'click',
    [Alias('StartupTimeoutSec')][int]$CliStartupTimeoutSec = 90,
    [Alias('SnapshotTimeoutMs')][int]$CliSnapshotTimeoutMs = 6000,
    [Alias('EvidenceDir')][string]$CliEvidenceDir,
    [Alias('KeepProfile')][switch]$CliKeepProfile,
    # A lane that must reach a state the first frame cannot have (a configured
    # workspace, a broken runtime, a port already held) seeds the throwaway
    # profile before the launch instead of driving the app there through menus.
    [Alias('SeedConfig')][string]$CliSeedConfig,
    # Restart-style lanes (an orphan left by a killed shell) need the second
    # launch to open the same profile the first one left behind.
    [Alias('ProfilePath')][string]$CliProfilePath
)

$script:PoliteLoadedAsLibrary = ($MyInvocation.InvocationName -eq '.')

$script:OwnProcessNames = @('whaleisle', 'whaleisle-gw')
$script:SharedProcessNames = @('electron', 'cua_node')
$script:ForeignSharedProcess = @()
$script:HangTitles = @('界面未响应', 'Interface not responding')
$script:HarnessMutexName = 'Local\WhaleIsle-Polite-Desktop-Run'
$script:HarnessMutex = $null
$script:HarnessMutexHeld = $false
$script:WindowsTemp = $null

# Redirected output is written in the console code page, which turns every
# Chinese window title into mojibake in the evidence log.
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false

function Initialize-PoliteInterop {
    if ('PoliteDesktop.Native' -as [type]) { return }
    Add-Type -Namespace PoliteDesktop -Name Native -MemberDefinition @'
public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
[DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowTextW(IntPtr hWnd, System.Text.StringBuilder text, int max);
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetClassNameW(IntPtr hWnd, System.Text.StringBuilder text, int max);
[DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint pid);
[DllImport("user32.dll")] public static extern bool EnumWindows(EnumWindowsProc callback, IntPtr lParam);
[DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hWnd);
[DllImport("user32.dll")] public static extern bool IsWindow(IntPtr hWnd);
[DllImport("user32.dll")] public static extern bool PostMessageW(IntPtr hWnd, uint message, IntPtr wParam, IntPtr lParam);
[DllImport("user32.dll")] public static extern IntPtr GetParent(IntPtr hWnd);
[DllImport("user32.dll")] public static extern IntPtr GetWindow(IntPtr hWnd, uint command);
[DllImport("user32.dll")] public static extern bool EnumChildWindows(IntPtr parent, EnumWindowsProc callback, IntPtr lParam);
'@ | Out-Null
}

function Get-PoliteWindowsTemp {
    if ($script:WindowsTemp) { return $script:WindowsTemp }
    # agent-desktop validates the DACL of its state directory under %TEMP%, and
    # an MSYS-style /tmp makes it refuse the whole run, so the harness always
    # hands every child the real Windows temp directory.
    $local = [Environment]::GetFolderPath('LocalApplicationData')
    $temp = Join-Path $local 'Temp'
    if (-not (Test-Path -LiteralPath $temp)) { $temp = [IO.Path]::GetTempPath().TrimEnd('\') }
    $script:WindowsTemp = $temp
    return $temp
}

function Get-WindowDescription {
    param([IntPtr]$Handle)
    $title = ''
    $class = ''
    $ownerPid = 0
    if ($Handle -ne [IntPtr]::Zero) {
        $text = New-Object System.Text.StringBuilder 512
        [void][PoliteDesktop.Native]::GetWindowTextW($Handle, $text, $text.Capacity)
        $title = $text.ToString()
        $cls = New-Object System.Text.StringBuilder 256
        [void][PoliteDesktop.Native]::GetClassNameW($Handle, $cls, $cls.Capacity)
        $class = $cls.ToString()
        [uint32]$rawPid = 0
        [void][PoliteDesktop.Native]::GetWindowThreadProcessId($Handle, [ref]$rawPid)
        $ownerPid = [int]$rawPid
    }
    $processName = ''
    if ($ownerPid -gt 0) {
        $proc = Get-Process -Id $ownerPid -ErrorAction SilentlyContinue
        if ($proc) { $processName = $proc.ProcessName }
    }
    [pscustomobject]@{
        Handle      = $Handle
        Hwnd        = ('0x{0:X8}' -f $Handle.ToInt64())
        Title       = $title
        Class       = $class
        OwnerPid    = $ownerPid
        ProcessName = $processName
    }
}

function Get-ForegroundWindowInfo {
    Initialize-PoliteInterop
    $handle = [PoliteDesktop.Native]::GetForegroundWindow()
    $info = Get-WindowDescription -Handle $handle
    $info
}

# A raise is only actionable once the window that took the foreground is named.
# The class alone separates a menu popup from a surface child from a fresh window,
# and the owner/parent chain says which client window opened it.
# When a window of our own pid takes the foreground, the next question is
# whether it belongs to the app's UI or is a window the shell made alongside it.
# Every top-level window the pid owns answers that, and the stolen window's chain
# then shows whether it descends from the main window or stands alone.
function Get-OwnWindowInventory {
    param([int]$OwnerPid)
    Initialize-PoliteInterop
    $items = New-Object System.Collections.ArrayList
    foreach ($window in (Get-TopLevelWindows | Where-Object { $_.OwnerPid -eq $OwnerPid })) {
        $items.Add(('top {0}[{1}]"{2}" visible={3}' -f $window.Hwnd, $window.Class, $window.Title, [PoliteDesktop.Native]::IsWindowVisible($window.Handle))) | Out-Null
    }
    $items.ToArray()
}

function Get-WindowChain {
    param([IntPtr]$Handle, [int]$Depth = 4)
    Initialize-PoliteInterop
    $parts = New-Object System.Collections.ArrayList
    $current = $Handle
    for ($i = 0; $i -lt $Depth -and $current -ne [IntPtr]::Zero; $i++) {
        $description = Get-WindowDescription -Handle $current
        $parts.Add(('{0}[{1}]"{2}"' -f $description.Hwnd, $description.Class, $description.Title)) | Out-Null
        $parent = [PoliteDesktop.Native]::GetParent($current)
        if ($parent -eq [IntPtr]::Zero) { $parent = [PoliteDesktop.Native]::GetWindow($current, 4) }
        $current = $parent
    }
    $parts -join ' <- '
}

function Get-ChildWindows {
    param([IntPtr]$Handle)
    Initialize-PoliteInterop
    $children = New-Object System.Collections.ArrayList
    $callback = [PoliteDesktop.Native+EnumWindowsProc] {
        param([IntPtr]$child, [IntPtr]$state)
        $children.Add((Get-WindowDescription -Handle $child)) | Out-Null
        return $true
    }
    [void][PoliteDesktop.Native]::EnumChildWindows($Handle, $callback, [IntPtr]::Zero)
    $children.ToArray()
}

function Get-TopLevelWindows {
    Initialize-PoliteInterop
    $windows = New-Object System.Collections.ArrayList
    $callback = [PoliteDesktop.Native+EnumWindowsProc] {
        param([IntPtr]$handle, [IntPtr]$state)
        $windows.Add((Get-WindowDescription -Handle $handle)) | Out-Null
        return $true
    }
    [void][PoliteDesktop.Native]::EnumWindows($callback, [IntPtr]::Zero)
    $windows.ToArray()
}

# Runs only while the operator is not moving between windows, so a run started
# under a busy hand does not report their alt-tab as our focus theft.
function Wait-QuietForeground {
    param(
        [int]$QuietSeconds = 3,
        [int]$Attempts = 6,
        [int]$PollMs = 500
    )
    $samples = [int][Math]::Ceiling(($QuietSeconds * 1000) / $PollMs)
    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        $first = Get-ForegroundWindowInfo
        $stable = $true
        for ($i = 1; $i -lt $samples; $i++) {
            Start-Sleep -Milliseconds $PollMs
            $next = Get-ForegroundWindowInfo
            if ($next.Handle -ne $first.Handle) { $stable = $false; break }
        }
        if ($stable) { return $first }
        Start-Sleep -Seconds 5
    }
    Write-Host 'warning           : foreground kept moving before the run; an external window switch may mark this run failed'
    Get-ForegroundWindowInfo
}

function Find-HangWindow {
    $found = @(Get-TopLevelWindows | Where-Object {
            $title = $_.Title
            $title -and ($script:HangTitles | Where-Object { $_ -and $title.Equals($_, [StringComparison]::Ordinal) })
        })
    $found
}

# The watchdog exits when its parent's stdin closes, but a crashed run can leave
# one behind, so the harness also names it by parent pid and never by process name.
function Get-OwnedDescendant {
    param([Parameter(Mandatory)][int]$ParentPid, [Parameter(Mandatory)][string]$ExePath)
    $full = [IO.Path]::GetFullPath($ExePath)
    $children = @()
    try {
        $children = @(Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentPid" -ErrorAction Stop)
    } catch {
        return @()
    }
    @($children | Where-Object {
            $_.ExecutablePath -and ([IO.Path]::GetFullPath($_.ExecutablePath) -eq $full)
        } | ForEach-Object {
            [pscustomobject]@{ Id = [int]$_.ProcessId; Path = $_.ExecutablePath; CommandLine = $_.CommandLine }
        })
}

function Get-ProcessCommandLine {
    param([Parameter(Mandatory)][int]$ProcessId)
    try {
        $row = Get-CimInstance Win32_Process -Filter "ProcessId = $ProcessId" -ErrorAction Stop
        if ($row) { return [pscustomobject]@{ CommandLine = $row.CommandLine; Path = $row.ExecutablePath } }
    } catch { }
    return [pscustomobject]@{ CommandLine = ''; Path = '' }
}

function Assert-NoRivalInstance {
    param([Parameter(Mandatory)][string]$ExePath)

    # whaleisle and whaleisle-gw are only ever this product, so any of them is a
    # conflict. electron and cua_node are shared names: the operator runs other
    # Electron apps (the reference desktop is one), and those cannot touch our
    # window or our profile. Only a MyGo-owned one is a rival.
    #
    # Do not tighten this back to "refuse on any electron". Five electron.exe from
    # Deepseek-Harness-Desktop were running the first time this guard fired, and a
    # guard that refuses while the operator has the reference desktop open blocks
    # every lane until someone comments it out. Name the conflict, not the process
    # name.
    $names = New-Object System.Collections.Generic.List[string]
    foreach ($name in $script:OwnProcessNames) { $names.Add($name) }
    $own = [IO.Path]::GetFileNameWithoutExtension($ExePath)
    if (-not $names.Contains($own)) { $names.Add($own) }

    $running = @(foreach ($name in $names) {
            foreach ($proc in @(Get-Process -Name $name -ErrorAction SilentlyContinue)) {
                [pscustomobject]@{ Name = $proc.ProcessName; Id = $proc.Id; Path = $proc.Path }
            }
        })
    if ($running.Count -gt 0) {
        $lines = $running | ForEach-Object { "  $($_.Name) (pid $($_.Id)) $($_.Path)" }
        throw @"
refusing to start: another MyGo desktop instance is already running on this machine.
$($lines -join "`n")
One instance at a time. Close those first, then rerun. This harness never kills by process name.
"@
    }

    $script:ForeignSharedProcess = @()
    foreach ($name in $script:SharedProcessNames) {
        $rows = @()
        try {
            $rows = @(Get-CimInstance Win32_Process -Filter "Name = '$name.exe'" -ErrorAction Stop)
        } catch {
            continue
        }
        foreach ($row in $rows) {
            $owned = ($row.ExecutablePath -and $row.ExecutablePath -match 'WhaleIsle-MyGo') -or
                     ($row.CommandLine -and $row.CommandLine -match 'WhaleIsle-MyGo')
            if ($owned) {
                throw "refusing to start: a MyGo process from an earlier run is still alive — $name (pid $($row.ProcessId)) $($row.ExecutablePath). Close it, then rerun."
            }
            $script:ForeignSharedProcess += [pscustomobject]@{ Name = $name; Id = [int]$row.ProcessId; Path = $row.ExecutablePath }
        }
    }
}

function Enter-PoliteHarnessLock {
    if ($script:HarnessMutexHeld) { return }
    $script:HarnessMutex = New-Object -TypeName System.Threading.Mutex -ArgumentList @($false, $script:HarnessMutexName)
    try {
        $script:HarnessMutexHeld = $script:HarnessMutex.WaitOne(0)
    } catch [System.Threading.AbandonedMutexException] {
        $script:HarnessMutexHeld = $true
    }
    if (-not $script:HarnessMutexHeld) {
        throw 'refusing to start: another polite run already holds this machine. One harness run at a time.'
    }
}

function Exit-PoliteHarnessLock {
    if ($script:HarnessMutexHeld -and $script:HarnessMutex) {
        try { [void]$script:HarnessMutex.ReleaseMutex() } catch { }
    }
    $script:HarnessMutexHeld = $false
    if ($script:HarnessMutex) { $script:HarnessMutex.Dispose() }
    $script:HarnessMutex = $null
}

function Resolve-PoliteExe {
    param([string]$ExePath)
    if (-not $ExePath) {
        $root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
        $ExePath = Join-Path $root 'dist\polite\whaleisle.exe'
    }
    $full = [IO.Path]::GetFullPath($ExePath)
    if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
        throw "no desktop binary at $full. Build it first: go build -trimpath -ldflags '-H windowsgui' -o dist/polite/whaleisle.exe ./cmd/whaleisle"
    }
    $full
}

function Get-AgentDesktopCommand {
    if ($script:AgentDesktopCommand) { return $script:AgentDesktopCommand }
    $command = Get-Command agent-desktop -ErrorAction SilentlyContinue
    if (-not $command) {
        throw 'agent-desktop is not on PATH. Install it (npm i -g agent-desktop) instead of falling back to physical input.'
    }
    $script:AgentDesktopCommand = $command.Source
    $script:AgentDesktopCommand
}

function Invoke-AgentDesktop {
    param(
        [Parameter(Mandatory)][string[]]$Arguments,
        [switch]$AllowFailure
    )
    $exe = Get-AgentDesktopCommand
    $savedTemp = $env:TEMP
    $savedTmp = $env:TMP
    $savedConsole = [Console]::OutputEncoding
    $env:TEMP = Get-PoliteWindowsTemp
    $env:TMP = $env:TEMP
    # The CLI answers in UTF-8; Windows PowerShell decodes native output with the
    # console code page and mangles every non-ASCII window title into invalid JSON.
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false
    try {
        $raw = & $exe @Arguments 2>&1 | ForEach-Object { if ($_ -is [System.Management.Automation.ErrorRecord]) { $_.ToString() } else { [string]$_ } }
        $exit = $LASTEXITCODE
    } finally {
        [Console]::OutputEncoding = $savedConsole
        $env:TEMP = $savedTemp
        $env:TMP = $savedTmp
    }
    $json = @($raw | Where-Object { $_ -and $_.TrimStart().StartsWith('{') }) | Select-Object -First 1
    if (-not $json) {
        if ($AllowFailure) { return [pscustomobject]@{ ok = $false; Raw = ($raw -join "`n"); ExitCode = $exit } }
        throw "agent-desktop $($Arguments[0]) printed no JSON (exit $exit):`n$($raw -join "`n")"
    }
    $envelope = $json | ConvertFrom-Json
    if (-not $envelope.ok -and -not $AllowFailure) {
        $code = $envelope.error.code
        $message = $envelope.error.message
        throw "agent-desktop $($Arguments[0]) failed: $code — $message"
    }
    $envelope
}

function New-PoliteProfile {
    $path = Join-Path (Get-PoliteWindowsTemp) ('mygo-polite-' + [guid]::NewGuid().ToString('n').Substring(0, 12))
    New-Item -ItemType Directory -Path $path -Force | Out-Null
    $path
}

# A reused profile is how a lane tests a restart: the first launch leaves state
# in <profile> and the second must find it. It must still be a throwaway under
# the Windows temp directory, never the operator's own data directory.
function Assert-PoliteProfilePath {
    param([Parameter(Mandatory)][string]$Path)
    $full = [IO.Path]::GetFullPath($Path)
    $temp = [IO.Path]::GetFullPath((Get-PoliteWindowsTemp))
    if (-not $full.StartsWith($temp, [StringComparison]::OrdinalIgnoreCase)) {
        throw "refusing profile '$full': a polite run only ever uses a directory under $temp"
    }
    if (-not (Test-Path -LiteralPath $full -PathType Container)) {
        throw "no profile directory at $full"
    }
    $full
}

# The seed is written before the app reads its configuration, so a lane reaches
# a state the first frame cannot have without clicking through the product.
function Copy-PoliteSeedConfig {
    param([Parameter(Mandatory)][string]$ProfilePath, [Parameter(Mandatory)][string]$SeedFile)
    $source = [IO.Path]::GetFullPath($SeedFile)
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) { throw "no seed config at $source" }
    Copy-Item -LiteralPath $source -Destination (Join-Path $ProfilePath 'config.json') -Force
}

function Start-PoliteInstance {
    param(
        [string]$ExePath,
        [int]$StartupTimeoutSec = 90,
        [int]$SnapshotTimeoutMs = 6000,
        $ForegroundBaseline,
        [string]$SeedConfig,
        [string]$ProfilePath
    )
    Initialize-PoliteInterop
    $ExePath = Resolve-PoliteExe -ExePath $ExePath
    Enter-PoliteHarnessLock

    $session = [pscustomobject]@{
        Exe               = $ExePath
        Profile           = $null
        Process           = $null
        Pid               = 0
        StartTime         = $null
        CommandLine       = ''
        WindowId          = $null
        WindowTitle       = ''
        SnapshotId        = $null
        SnapshotPath      = ''
        Snapshot          = $null
        Steps             = @()
        ForegroundBefore  = $null
        HangSeen          = @()
        CloseEvidence     = $null
        SnapshotTimeoutMs = $SnapshotTimeoutMs
        EvidenceDir       = $EvidenceDir
        SeedConfig        = ''
        ReusedProfile     = $false
    }

    try {
        if ($ForegroundBaseline) { $session.ForegroundBefore = $ForegroundBaseline } else { $session.ForegroundBefore = Get-ForegroundWindowInfo }
        Assert-NoRivalInstance -ExePath $ExePath

        if ($ProfilePath) {
            $session.Profile = Assert-PoliteProfilePath -Path $ProfilePath
            $session.ReusedProfile = $true
        } else {
            $session.Profile = New-PoliteProfile
        }
        if ($SeedConfig) {
            Copy-PoliteSeedConfig -ProfilePath $session.Profile -SeedFile $SeedConfig
            $session.SeedConfig = [IO.Path]::GetFullPath($SeedConfig)
        }
        $saved = @{
            TEMP                     = $env:TEMP
            TMP                      = $env:TMP
            WHALEISLE_USER_DATA      = $env:WHALEISLE_USER_DATA
            WHALEISLE_ISOLATED       = $env:WHALEISLE_ISOLATED
            WHALEISLE_ISOLATED_VISIBLE = $env:WHALEISLE_ISOLATED_VISIBLE
        }
        try {
            $env:TEMP = Get-PoliteWindowsTemp
            $env:TMP = $env:TEMP
            # Isolated mode keeps the run off the operator's real profile, their
            # registry (URL scheme, autostart) and their tray. The visible flag
            # keeps the window real so the accessibility tree can read it.
            $env:WHALEISLE_USER_DATA = $session.Profile
            $env:WHALEISLE_ISOLATED = '1'
            $env:WHALEISLE_ISOLATED_VISIBLE = '1'
            $session.Process = Start-Process -FilePath $ExePath -ArgumentList @('--background') `
                -WorkingDirectory (Split-Path -Parent $ExePath) -PassThru
        } finally {
            $env:TEMP = $saved.TEMP
            $env:TMP = $saved.TMP
            $env:WHALEISLE_USER_DATA = $saved.WHALEISLE_USER_DATA
            $env:WHALEISLE_ISOLATED = $saved.WHALEISLE_ISOLATED
            $env:WHALEISLE_ISOLATED_VISIBLE = $saved.WHALEISLE_ISOLATED_VISIBLE
        }

        $session.Pid = $session.Process.Id
        $session.StartTime = $session.Process.StartTime
        $session.CommandLine = "$ExePath --background"

        Wait-PoliteWindow -Session $session -TimeoutSec $StartupTimeoutSec | Out-Null
        $session | Add-Member -NotePropertyName FirstWindowCommandLine -NotePropertyValue (Get-ProcessCommandLine -ProcessId $session.Pid).CommandLine -Force
        return $session
    } catch {
        Close-PoliteInstance -Session $session -Quiet
        throw
    }
}

function Assert-PoliteSessionShape {
    param([Parameter(Mandatory)]$Session)
    if ($Session -isnot [pscustomobject] -or -not $Session.Pid) {
        throw 'harness bug: Start-PoliteInstance handed back an unexpected object; refusing to run against it'
    }
}

function Wait-PoliteWindow {
    param(
        [Parameter(Mandatory)]$Session,
        [int]$TimeoutSec = 90
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    $lastProblem = 'no window yet'
    while ((Get-Date) -lt $deadline) {
        if (-not (Get-Process -Id $Session.Pid -ErrorAction SilentlyContinue)) {
            throw "the app exited before it showed a window (pid $($Session.Pid)). Logs: $($Session.Profile)\native\logs"
        }
        $hang = @(Find-HangWindow)
        if ($hang.Count -gt 0) {
            $Session.HangSeen += $hang
            throw "probe too heavy: 界面未响应 window appeared while waiting for the app window (title '$($hang[0].Title)'). Stopping."
        }
        $listed = Invoke-AgentDesktop -Arguments @('list-windows') -AllowFailure
        if ($listed.ok) {
            $window = @($listed.data | Where-Object { $_.pid -eq $Session.Pid -and $_.visible -and -not $_.minimized }) | Select-Object -First 1
            if ($window) {
                $Session.WindowId = $window.id
                $Session.WindowTitle = $window.title
                return $window
            }
            $lastProblem = "pid $($Session.Pid) has no visible window in the accessibility tree"
        } else {
            $lastProblem = "list-windows failed: $($listed.Raw)"
        }
        Start-Sleep -Milliseconds 500
    }
    throw "no accessible window for pid $($Session.Pid) after $TimeoutSec s ($lastProblem)"
}

function Read-PoliteSnapshot {
    param(
        [Parameter(Mandatory)]$Session,
        [int]$MaxDepth = 8
    )
    if (-not $Session.WindowId) { throw 'no window id yet; Start-PoliteInstance waits for it' }
    $envelope = Invoke-AgentDesktop -Arguments @(
        'snapshot', '--window-id', $Session.WindowId, '-i', '--compact',
        '--max-depth', "$MaxDepth", '--timeout-ms', "$($Session.SnapshotTimeoutMs)"
    )
    $Session.Snapshot = $envelope.data
    $Session.SnapshotId = $envelope.data.snapshot_id
    if ($Session.EvidenceDir) {
        New-Item -ItemType Directory -Path $Session.EvidenceDir -Force | Out-Null
        $path = Join-Path $Session.EvidenceDir ('snapshot-' + $envelope.data.snapshot_id + '.json')
        $envelope | ConvertTo-Json -Depth 32 -Compress | Set-Content -LiteralPath $path -Encoding UTF8
        $Session.SnapshotPath = $path
    }
    $envelope.data
}

function Get-PoliteNodes {
    param([Parameter(Mandatory)]$Snapshot)
    $nodes = New-Object System.Collections.ArrayList
    $stack = New-Object System.Collections.Stack
    if ($Snapshot.tree) { $stack.Push($Snapshot.tree) }
    while ($stack.Count -gt 0) {
        $node = $stack.Pop()
        $nodes.Add($node) | Out-Null
        if ($node.children) {
            foreach ($child in @($node.children)) { if ($child) { $stack.Push($child) } }
        }
    }
    $nodes.ToArray()
}

function Get-PoliteNodeState {
    param([Parameter(Mandatory)]$Node, [Parameter(Mandatory)][string]$State)
    @($Node.states) -contains $State
}

# A step must be idempotent: the profile is disposable, but a probe that opens a
# modal or a menu raises a window on the operator's screen. Preference order is
# "cannot change anything" first, and running out of candidates is a recorded
# failure, never a fallback to physical input.
function Test-PoliteNodeUsable {
    param([Parameter(Mandatory)]$Node)
    if ($null -ne $Node.enabled -and -not $Node.enabled) { return $false }
    foreach ($state in @('disabled', 'unavailable', 'offscreen')) {
        if (Get-PoliteNodeState -Node $Node -State $state) { return $false }
    }
    return $true
}

function Find-PoliteHarmlessTarget {
    param([Parameter(Mandatory)]$Snapshot)
    $nodes = Get-PoliteNodes -Snapshot $Snapshot
    $textRoles = @('textfield', 'textbox', 'edit', 'searchbox', 'searchfield', 'document', 'textarea', 'combobox', 'richtext', 'text')
    $checkRoles = @('checkbox', 'switch', 'toggle')
    $navRoles = @('tab', 'tabitem', 'navigationitem', 'menuitem', 'link', 'listitem', 'treeitem')

    foreach ($node in $nodes) {
        if (-not $node.ref_id) { continue }
        if (-not (Test-PoliteNodeUsable -Node $node)) { continue }
        $role = ([string]$node.role).ToLower()
        $value = [string]$node.value
        if (($textRoles -contains $role) -and [string]::IsNullOrEmpty($value)) {
            return @{ Ref = $node.ref_id; Action = 'clear'; Name = [string]$node.name; Why = "empty ${role}: clear is a no-op" }
        }
    }
    foreach ($node in $nodes) {
        if (-not $node.ref_id) { continue }
        if (-not (Test-PoliteNodeUsable -Node $node)) { continue }
        $role = ([string]$node.role).ToLower()
        if (($checkRoles -contains $role) -and -not (Get-PoliteNodeState -Node $node -State 'checked')) {
            return @{ Ref = $node.ref_id; Action = 'uncheck'; Name = [string]$node.name; Why = "unchecked ${role}: uncheck is idempotent" }
        }
    }
    foreach ($node in $nodes) {
        if (-not $node.ref_id) { continue }
        if (-not (Test-PoliteNodeUsable -Node $node)) { continue }
        $role = ([string]$node.role).ToLower()
        if (($navRoles -contains $role) -and (Get-PoliteNodeState -Node $node -State 'selected')) {
            return @{ Ref = $node.ref_id; Action = 'click'; Name = [string]$node.name; Why = "already selected ${role}: click re-selects the same route" }
        }
    }
    foreach ($node in $nodes) {
        if (-not $node.ref_id) { continue }
        if (-not (Test-PoliteNodeUsable -Node $node)) { continue }
        $role = ([string]$node.role).ToLower()
        if (($textRoles -contains $role) -and -not [string]::IsNullOrEmpty([string]$node.value)) {
            return @{ Ref = $node.ref_id; Action = 'set-value'; Name = [string]$node.name; Value = [string]$node.value; Why = "writes a ${role} back to the value it already holds" }
        }
    }
    return $null
}

function Invoke-PoliteStep {
    param(
        [Parameter(Mandatory)]$Session,
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][scriptblock]$Action
    )
    $before = Get-ForegroundWindowInfo
    $hangBefore = @(Find-HangWindow)
    if ($hangBefore.Count -gt 0) {
        $Session.HangSeen += $hangBefore
        throw "probe too heavy: 界面未响应 window is already up before step '$Name'. Stopping; do not push through."
    }

    $actionError = $null
    $result = $null
    try { $result = & $Action } catch { $actionError = $_ }
    Start-Sleep -Milliseconds 200

    $after = Get-ForegroundWindowInfo
    $hangAfter = @(Find-HangWindow)
    $stolen = ($before.Handle -ne $after.Handle)
    if ($hangAfter.Count -gt 0) { $Session.HangSeen += $hangAfter }

    $ownerPid = $Session.Pid
    $owned = $false
    $wasOurs = $false
    if ($stolen) {
        $owned = ($after.OwnerPid -eq $Session.Pid)
        if (-not $owned -and $after.OwnerPid -gt 0) {
            $owned = @(Get-OwnedDescendant -ParentPid $Session.Pid -ExePath $Session.Exe | Where-Object { $_.Id -eq $after.OwnerPid }).Count -gt 0
        }
    }
    # A window of ours moving focus between its own child windows is not the
    # theft this harness exists to catch: the operator already had the app in
    # front, so they lost nothing. Only a step that takes the foreground from a
    # window that is not ours is a politeness failure. Without this distinction a
    # run started while the operator was looking at the app reports a steal for
    # every click that lands inside it.
    if ($owned -and $before.OwnerPid -gt 0) {
        $wasOurs = ($before.OwnerPid -eq $Session.Pid)
    }

    $actionErrorText = ''
    if ($actionError) { $actionErrorText = "$actionError" }
    # What the driver says it delivered is part of the evidence: the same click
    # that moves the operator's foreground is either a confirmed semantic call or
    # an unverified delivery, and those two strings accuse different things.
    $deliveryText = ''
    $stepsText = ''
    if ($result -and $result.data) {
        $disposition = $result.data.disposition
        if ($disposition) {
            $deliveryText = "$($disposition.delivery)"
            if ($disposition.retry) { $deliveryText += " retry=$($disposition.retry)" }
        }
        if ($result.data.steps) {
            $stepsText = (@($result.data.steps | ForEach-Object { '{0}/{1}/{2}' -f $_.label, $_.mechanism, $_.outcome }) -join '; ')
        }
    }

    $StepResult = [pscustomobject]@{
        Name             = $Name
        ForegroundBefore = $before.Hwnd
        ForegroundAfter  = $after.Hwnd
        ForegroundTitle  = $after.Title
        ForegroundClass  = $after.Class
        ForegroundChain  = $(if ($stolen) { Get-WindowChain -Handle $after.Handle } else { '' })
        ForegroundChildren = $(if ($stolen) { @(Get-ChildWindows -Handle $after.Handle | ForEach-Object { '{0}[{1}]"{2}"' -f $_.Hwnd, $_.Class, $_.Title }) } else { @() })
        OwnWindows       = $(if ($stolen -and $owned) { @(Get-OwnWindowInventory -OwnerPid $Session.Pid) } else { @() })
        ForegroundPid    = $after.OwnerPid
        ForegroundOwned  = $owned
        FocusStolen      = ($stolen -and $owned -and -not $wasOurs)
        InternalMove     = ($stolen -and $owned -and $wasOurs)
        ExternalChange   = ($stolen -and -not $owned)
        HangWindow       = ($hangAfter.Count -gt 0)
        ActionError      = $actionErrorText
        ActionDelivery   = $deliveryText
        ActionSteps      = $stepsText
        ActionTrace      = $(if ($result -and $result.PolitenessTrace) { $result.PolitenessTrace } else { '' })
    }
    $Session.Steps = @($Session.Steps) + $StepResult
    $verdict = 'unchanged'
    if ($stolen -and $owned -and $wasOurs) { $verdict = 'internal-move' }
    elseif ($stolen -and $owned) { $verdict = 'STOLEN' }
    elseif ($stolen) { $verdict = 'external' }
    $ownerText = ''
    if ($stolen) { $ownerText = " (pid $($after.OwnerPid) $($after.ProcessName) class '$($after.Class)' title '$($after.Title)')" }
    Write-Host ("step {0,-24}: {1} -> {2} {3}{4}" -f $Name, $before.Hwnd, $after.Hwnd, $verdict, $ownerText)

    if ($hangAfter.Count -gt 0) {
        throw "probe too heavy: step '$Name' raised 界面未响应 ('$($hangAfter[0].Title)'). Stopping."
    }
    if ($stolen -and $owned -and -not $wasOurs) {
        throw "focus stolen during step '$Name': foreground went $($before.Hwnd) -> $($after.Hwnd), a window of our own process tree (class '$($after.Class)' title '$($after.Title)', chain $($StepResult.ForegroundChain)). Stopping."
    }
    if ($stolen) {
        # Someone else's window — the operator switching apps, usually. Nothing we
        # did, but this step can no longer measure anything, so the run is void.
        throw "foreground-changed-externally: step '$Name' saw the foreground go $($before.Hwnd) -> $($after.Hwnd), a window owned by pid $($after.OwnerPid) ($($after.ProcessName)) '$($after.Title)'. Not our process, so this run cannot prove politeness; rerun while the machine is quiet."
    }
    if ($actionError) { throw $actionError }
    $result
}

function Invoke-PoliteAction {
    param(
        [Parameter(Mandatory)]$Session,
        [Parameter(Mandatory)][string]$Ref,
        [Parameter(Mandatory)][string]$Action,
        [string]$Value
    )
    $arguments = switch ($Action) {
        'click' { @('click', $Ref) }
        'clear' { @('clear', $Ref) }
        'uncheck' { @('uncheck', $Ref) }
        'expand' { @('expand', $Ref) }
        'collapse' { @('collapse', $Ref) }
        'set-value' { @('set-value', $Ref, $Value) }
        default { throw "unsupported polite action '$Action'" }
    }
    # The driver's own reliability trace is the only record of what it did inside
    # a step, and it refuses a path that is not a private user file, so the trace
    # is written under a directory this harness locks down first.
    $tracePath = $null
    if ($Session -and $Session.EvidenceDir -and $Action -eq 'click') {
        $traceDir = Join-Path $env:USERPROFILE '.capyworkad-trace'
        New-Item -ItemType Directory -Path $traceDir -Force | Out-Null
        & icacls.exe $traceDir /inheritance:r /grant:r ("$($env:USERNAME):(OI)(CI)F") | Out-Null
        $tracePath = Join-Path $traceDir ('click-' + [guid]::NewGuid().ToString('n').Substring(0, 8) + '.jsonl')
        $arguments = $arguments + @('--trace', $tracePath)
    }
    $envelope = Invoke-AgentDesktop -Arguments $arguments -AllowFailure
    if ($tracePath -and (Test-Path -LiteralPath $tracePath)) {
        $copy = Join-Path $Session.EvidenceDir ('driver-trace-' + (Get-Date).ToString('HHmmss') + '.jsonl')
        Copy-Item -LiteralPath $tracePath -Destination $copy -Force
        $envelope | Add-Member -NotePropertyName PolitenessTrace -NotePropertyValue $copy -Force
    }
    if (-not $envelope.ok) {
        $code = 'unknown'
        $delivery = ''
        $message = "$($envelope.Raw)"
        if ($envelope.error) {
            $code = $envelope.error.code
            $message = $envelope.error.message
            if ($envelope.error.disposition) { $delivery = $envelope.error.disposition.delivery }
        }
        # not_delivered is the only delivery value that proves nothing reached the
        # app. Anything else may have landed, so it must stop the run.
        if ($delivery -eq 'not_delivered') { throw "action-not-sent: $code ($message)" }
        throw "action-uncertain: $code after delivery '$delivery' — it may have landed; inspect before rerunning ($message)"
    }
    $envelope
}

function Close-PoliteInstance {
    param(
        [Parameter(Mandatory)]$Session,
        [switch]$Quiet
    )
    try {
        if ($Session -and $Session.Pid -gt 0 -and $Session.Process) {
            $evidence = [pscustomobject]@{
                Pid                = $Session.Pid
                WmCloseSent        = $false
                ExitedGracefully   = $false
                Forced             = $false
                ExitCode           = $null
                ReapedDescendants  = @()
                SurvivingProcesses = @()
            }
            $proc = Get-Process -Id $Session.Pid -ErrorAction SilentlyContinue
            if ($proc) {
                foreach ($window in @(Get-TopLevelWindows | Where-Object { $_.OwnerPid -eq $Session.Pid })) {
                    [void][PoliteDesktop.Native]::PostMessageW($window.Handle, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero)
                    $evidence.WmCloseSent = $true
                }
                $deadline = (Get-Date).AddSeconds(25)
                while ((Get-Date) -lt $deadline -and -not $proc.HasExited) {
                    Start-Sleep -Milliseconds 250
                    try { $proc.Refresh() } catch { }
                }
                $evidence.ExitedGracefully = $proc.HasExited
                if ($proc.HasExited) {
                    $evidence.ExitCode = $proc.ExitCode
                } else {
                    # Only this pid, and only after proving it is the process this
                    # run started. Never a process name: other lanes share the machine.
                    $live = Get-Process -Id $Session.Pid -ErrorAction SilentlyContinue
                    $ours = $false
                    if ($live -and $live.Path) {
                        $ours = ([IO.Path]::GetFullPath($live.Path) -eq [IO.Path]::GetFullPath($Session.Exe))
                    }
                    if ($ours) {
                        Stop-Process -Id $Session.Pid -Force -ErrorAction Stop
                        $evidence.Forced = $true
                        $deadline = (Get-Date).AddSeconds(10)
                        while ((Get-Date) -lt $deadline -and -not $proc.HasExited) {
                            Start-Sleep -Milliseconds 200
                            try { $proc.Refresh() } catch { }
                        }
                    }
                }
            } else {
                $evidence.ExitedGracefully = $true
                if ($Session.Process.HasExited) { $evidence.ExitCode = $Session.Process.ExitCode }
            }

            $descendants = @(Get-OwnedDescendant -ParentPid $Session.Pid -ExePath $Session.Exe)
            if ($descendants.Count -gt 0) {
                Start-Sleep -Milliseconds 1500
                foreach ($child in $descendants) {
                    if (Get-Process -Id $child.Id -ErrorAction SilentlyContinue) {
                        Stop-Process -Id $child.Id -Force -ErrorAction SilentlyContinue
                        $evidence.ReapedDescendants += $child.Id
                    }
                }
            }
            $evidence.SurvivingProcesses = @(Get-OwnedDescendant -ParentPid $Session.Pid -ExePath $Session.Exe | ForEach-Object { $_.Id })
            $Session.CloseEvidence = $evidence
        }
    } finally {
        $keep = $KeepProfile -or ($Session -and $Session.PSObject.Properties['ReusedProfile'] -and $Session.ReusedProfile)
        if ($Session -and $Session.Profile -and -not $keep -and (Test-Path -LiteralPath $Session.Profile)) {
            Remove-Item -LiteralPath $Session.Profile -Recurse -Force -ErrorAction SilentlyContinue
        }
        Exit-PoliteHarnessLock
    }
    if (-not $Quiet -and $Session.CloseEvidence) {
        if ($Session.CloseEvidence.SurvivingProcesses.Count -gt 0) {
            throw "close failed: pid(s) $($Session.CloseEvidence.SurvivingProcesses -join ', ') from our own tree are still running"
        }
        if (-not $Session.CloseEvidence.ExitedGracefully -and -not $Session.CloseEvidence.Forced) {
            throw "close failed: pid $($Session.Pid) never exited"
        }
    }
}

function Show-PoliteEvidence {
    param([Parameter(Mandatory)]$Session)
    $close = $Session.CloseEvidence
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add('=== polite run evidence ===')
    $lines.Add("binary          : $($Session.Exe)")
    $lines.Add("command line     : $($Session.CommandLine)")
    if ($Session.PSObject.Properties['FirstWindowCommandLine'] -and $Session.FirstWindowCommandLine) {
        $lines.Add("os command line  : $($Session.FirstWindowCommandLine)")
    }
    $lines.Add("profile          : $($Session.Profile)")
    if ($Session.PSObject.Properties['SeedConfig'] -and $Session.SeedConfig) {
        $lines.Add("profile seed     : $($Session.SeedConfig)")
    }
    if ($Session.PSObject.Properties['ReusedProfile'] -and $Session.ReusedProfile) {
        $lines.Add("profile reused   : yes")
    }
    $startedText = "$($Session.StartTime)"
    if ($Session.StartTime -is [datetime]) { $startedText = $Session.StartTime.ToString('o') }
    $lines.Add("app pid          : $($Session.Pid) started $startedText")
    $lines.Add("window           : $($Session.WindowId) '$($Session.WindowTitle)'")
    $lines.Add("foreground before: $($Session.ForegroundBefore.Hwnd) '$($Session.ForegroundBefore.Title)' (pid $($Session.ForegroundBefore.OwnerPid) $($Session.ForegroundBefore.ProcessName))")
    foreach ($step in @($Session.Steps)) {
        $verdict = 'unchanged'
        if ($step.InternalMove) { $verdict = "internal-move '$($step.ForegroundClass)'" }
        elseif ($step.FocusStolen) { $verdict = "STOLEN class '$($step.ForegroundClass)' title '$($step.ForegroundTitle)'" }
        elseif ($step.ExternalChange) { $verdict = "external pid $($step.ForegroundPid) '$($step.ForegroundTitle)'" }
        $note = ''
        if ($step.ActionError) { $note = " action-error: $($step.ActionError)" }
        if ($step.ActionDelivery) { $note += " delivery: $($step.ActionDelivery)" }
        if ($step.ActionSteps) { $note += " steps: $($step.ActionSteps)" }
        if ($step.ActionTrace) { $note += " driver-trace: $($step.ActionTrace)" }
        $stepLine = "step {0,-24}: {1} -> {2} {3}{4}" -f $step.Name, $step.ForegroundBefore, $step.ForegroundAfter, $verdict, $note
        $lines.Add($stepLine)
        if ($step.FocusStolen) {
            $lines.Add("  chain            : $($step.ForegroundChain)")
            foreach ($child in @($step.ForegroundChildren)) { $lines.Add("  child            : $child") }
            foreach ($own in @($step.OwnWindows)) { $lines.Add("  own window       : $own") }
        }
    }
    $final = Get-ForegroundWindowInfo
    $lines.Add("foreground after : $($final.Hwnd) '$($final.Title)' (pid $($final.OwnerPid) $($final.ProcessName))")
    if ($close) {
        $lines.Add("wm_close sent    : $($close.WmCloseSent)")
        $lines.Add("pid exited       : $($close.ExitedGracefully -or $close.Forced) (graceful=$($close.ExitedGracefully) forced=$($close.Forced) exit=$($close.ExitCode))")
        if ($close.ReapedDescendants.Count -gt 0) { $lines.Add("watchdog reaped  : pid(s) $($close.ReapedDescendants -join ', ')") }
        $lines.Add("survivors        : $(if ($close.SurvivingProcesses.Count -gt 0) { $close.SurvivingProcesses -join ', ' } else { 'none' })")
    }
    $hangText = if (@($Session.HangSeen).Count -gt 0) { "$(@($Session.HangSeen).Count) window(s)" } else { 'none' }
    if (@($script:ForeignSharedProcess).Count -gt 0) {
        $foreign = @($script:ForeignSharedProcess) | ForEach-Object { "$($_.Name) $($_.Id)" }
        $lines.Add("foreign electron : $($foreign -join ', ') — not a MyGo process, ignored")
    }
    $lines.Add("watchdog window  : $hangText")
    $text = $lines -join "`n"
    # A run's evidence is worth more than the console that printed it: one shell
    # mistake in the caller's pipeline lost a whole run's step log, so the block
    # is always written beside the other evidence as well.
    if ($Session.EvidenceDir) {
        $stamp = (Get-Date).ToString('yyyyMMdd-HHmmss')
        $path = Join-Path $Session.EvidenceDir "run-evidence-$stamp.txt"
        try { Set-Content -LiteralPath $path -Value $text -Encoding UTF8 } catch { }
    }
    $text
}

function Test-PoliteSession {
    param([Parameter(Mandatory)]$Session)
    $close = $Session.CloseEvidence
    $steps = @($Session.Steps)
    $stolenByApp = @($steps | Where-Object { $_.FocusStolen }).Count -gt 0
    $external = @($steps | Where-Object { $_.ExternalChange }).Count -gt 0
    $noHang = @($Session.HangSeen).Count -eq 0
    $exited = [bool]($close -and ($close.ExitedGracefully -or $close.Forced) -and $close.SurvivingProcesses.Count -eq 0)
    [pscustomobject]@{
        AppFocusStolen   = $stolenByApp
        ExternalChange   = $external
        ForegroundStable = ((-not $stolenByApp) -and (-not $external))
        NoHangWindow     = $noHang
        PidExited        = $exited
        Passed           = ((-not $stolenByApp) -and (-not $external) -and $noHang -and $exited)
    }
}

function Invoke-PoliteRun {
    param(
        [string]$ExePath,
        [string]$OpsPath,
        [string]$TargetRef,
        [string]$TargetActionName = 'click',
        [int]$StartupTimeoutSec = 90,
        [int]$SnapshotTimeoutMs = 6000,
        [string]$EvidenceDirectory,
        [switch]$KeepProfile,
        [string]$SeedConfig,
        [string]$ProfilePath
    )
    $session = $null
    $failure = $null
    try {
        $session = Start-PoliteInstance -ExePath $ExePath -StartupTimeoutSec $StartupTimeoutSec -SnapshotTimeoutMs $SnapshotTimeoutMs -SeedConfig $SeedConfig -ProfilePath $ProfilePath
        Assert-PoliteSessionShape -Session $session
        $session.EvidenceDir = $EvidenceDirectory
        $actions = 0

        if ($OpsPath) {
            $opsFull = [IO.Path]::GetFullPath($OpsPath)
            if (-not (Test-Path -LiteralPath $opsFull -PathType Leaf)) { throw "no ops file at $opsFull" }
            . $opsFull
            if (-not (Get-Command Invoke-PoliteOps -ErrorAction SilentlyContinue)) {
                throw "$opsFull must define Invoke-PoliteOps -Session <session>"
            }
            Invoke-PoliteOps -Session $session
            $actions++
        } elseif ($TargetRef) {
            Invoke-PoliteStep -Session $session -Name "action $TargetActionName $TargetRef" -Action {
                Invoke-PoliteAction -Session $session -Ref $TargetRef -Action $TargetActionName
            } | Out-Null
            $actions++
        }

        if ($actions -eq 0) {
            Invoke-PoliteStep -Session $session -Name 'snapshot' -Action {
                Read-PoliteSnapshot -Session $session | Out-Null
            } | Out-Null
        }
    } catch {
        $failure = $_
    } finally {
        if ($session) {
            try { Close-PoliteInstance -Session $session -Quiet:$false } catch { if (-not $failure) { $failure = $_ } }
        } else {
            Exit-PoliteHarnessLock
        }
    }

    $evidence = if ($session) { Show-PoliteEvidence -Session $session } else { '' }
    if ($evidence) { Write-Host $evidence }
    if ($session -and $EvidenceDirectory) {
        New-Item -ItemType Directory -Path $EvidenceDirectory -Force | Out-Null
        Set-Content -LiteralPath (Join-Path $EvidenceDirectory 'evidence.txt') -Value $evidence -Encoding UTF8
    }
    if ($failure) { throw $failure }
    $session
}

function Invoke-PoliteRunCli {
    try {
        $session = Invoke-PoliteRun -ExePath $CliExe -OpsPath $CliOps -TargetRef $CliTargetRef -TargetActionName $CliTargetAction `
            -StartupTimeoutSec $CliStartupTimeoutSec -SnapshotTimeoutMs $CliSnapshotTimeoutMs -EvidenceDirectory $CliEvidenceDir -KeepProfile:$CliKeepProfile `
            -SeedConfig $CliSeedConfig -ProfilePath $CliProfilePath
        $verdict = Test-PoliteSession -Session $session
        if ($verdict.Passed) { Write-Host 'RESULT: PASS'; return 0 }
        Write-Host 'RESULT: FAIL'
        return 1
    } catch {
        $message = "$_"
        Write-Host "RESULT: FAIL"
        Write-Host $message
        if ($message -like 'refusing to start*') { return 2 }
        return 1
    }
}

if (-not $script:PoliteLoadedAsLibrary) {
    exit (Invoke-PoliteRunCli)
}
