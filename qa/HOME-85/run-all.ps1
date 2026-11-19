#Requires -Version 5.1
<#
Every live scenario for the Host front door, in one command.

The operator's quiet window is the scarce resource, so this file exists to spend
one run on the whole sequence instead of six separate orchestrations: it prepares
each fixture, runs the scenario in order, and stops at the first failure with
that scenario's evidence already printed. A run that passes prints one PASS line
per scenario.

    .\qa\HOME-85\run-all.ps1 -Exe .\dist\polite\whaleisle.exe -VendorDir C:\AI\WhaleIsle-MyGo\runtime-rc4\vendor

Nothing here launches the app itself; every launch goes through
tools/polite/run.ps1, so the foreground check, the 界面未响应 tripwire, the
one-instance rule and the close-by-own-pid all still apply.
#>
[CmdletBinding()]
param(
    # Required to run scenarios; absent only when this file is dot-sourced for
    # its fixtures by qa/HOME-85/test-fixtures.ps1.
    [string]$Exe,
    [string]$RepoRoot = (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)),
    [string]$EvidenceRoot,
    # The staged Host runtime the launched app should use, read-only. The one the
    # lanes already stage is C:\AI\WhaleIsle-MyGo\runtime-rc4\vendor.
    [string]$VendorDir,
    [string[]]$Only = @(),
    [switch]$SkipOrphan
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false

. (Join-Path $RepoRoot 'tools\polite\run.ps1')

$exeFull = ''
if ($Exe) { $exeFull = [IO.Path]::GetFullPath($Exe) }
if (-not $EvidenceRoot) { $EvidenceRoot = Join-Path $RepoRoot 'qa-evidence\HOME-85' }
New-Item -ItemType Directory -Path $EvidenceRoot -Force | Out-Null

$fixtures = Join-Path $EvidenceRoot 'fixtures'
$seeds = Join-Path $EvidenceRoot 'seeds'
New-Item -ItemType Directory -Path $fixtures, $seeds -Force | Out-Null

$workspace = Join-Path $RepoRoot 'qa\HOME-85\workspace'
New-Item -ItemType Directory -Path $workspace -Force | Out-Null

# --- fixtures -------------------------------------------------------------

function New-SeedConfig {
    param([Parameter(Mandatory)][string]$Name, [hashtable]$Config)
    $path = Join-Path $seeds "$Name.json"
    $Config | ConvertTo-Json | Set-Content -LiteralPath $path -Encoding UTF8
    $path
}

# A vendor tree that is present but is not the runtime this build was built
# against: it carries an incomplete session-event registry, which is the exact
# refusal the copy calls a version mismatch.
function New-MismatchVendor {
    $root = Join-Path $fixtures 'mismatch-vendor'
    $harness = Join-Path $root 'deepseek-harness'
    $files = @(
        'apps\cli\lib\bin.js',
        'apps\web\dist\index.html'
    )
    foreach ($file in $files) {
        $path = Join-Path $harness $file
        New-Item -ItemType Directory -Path (Split-Path -Parent $path) -Force | Out-Null
        Set-Content -LiteralPath $path -Value '' -Encoding UTF8
    }
    $reader = Join-Path $harness 'packages\core\session\lib\index.js'
    New-Item -ItemType Directory -Path (Split-Path -Parent $reader) -Force | Out-Null
    Set-Content -LiteralPath $reader -Value 'const KNOWN_SESSION_EVENT_TYPES = new Set(["job/registered"]);' -Encoding UTF8
    $root
}

# A vendor directory with nothing in it, so the launch cannot find a runtime.
function New-EmptyVendor {
    $root = Join-Path $fixtures 'empty-vendor'
    New-Item -ItemType Directory -Path $root -Force | Out-Null
    $root
}

# Occupy the whole port range the launcher scans from the configured port, so
# the next start has nowhere to listen.
function Start-PortWall {
    param([int]$First = 3080, [int]$Count = 50)
    $listeners = New-Object System.Collections.Generic.List[object]
    try {
        for ($port = $First; $port -lt $First + $Count; $port++) {
            $listener = New-Object System.Net.Sockets.TcpListener ([System.Net.IPAddress]::Loopback), $port
            $listener.Server.ExclusiveAddressUse = $true
            $listener.Start()
            $listeners.Add($listener)
        }
    } catch {
        foreach ($open in $listeners) { try { $open.Stop() } catch { } }
        throw "could not occupy ports $First..$($First + $Count - 1): $_"
    }
    $listeners.ToArray()
}

function Stop-PortWall {
    param([AllowEmptyCollection()]$Listeners)
    foreach ($listener in $Listeners) { try { $listener.Stop() } catch { } }
}

# --- scenarios ------------------------------------------------------------

$script:Results = New-Object System.Collections.Generic.List[string]

function Invoke-CopyScenario {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$Scenario,
        [string]$SeedConfig,
        [string]$Vendor,
        [int]$Settle = 0,
        [scriptblock]$Before,
        [scriptblock]$After
    )
    $savedVendor = $env:WHALEISLE_VENDOR
    $savedScenario = $env:POLITE_SCENARIO
    $savedSettle = $env:POLITE_SETTLE
    $dir = Join-Path $EvidenceRoot $Name
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    try {
        if ($Vendor) { $env:WHALEISLE_VENDOR = $Vendor } else { $env:WHALEISLE_VENDOR = $savedVendor }
        $env:POLITE_SCENARIO = $Scenario
        $env:POLITE_SETTLE = "$Settle"
        if ($Before) { & $Before }
        $parameters = @{
            ExePath           = $exeFull
            OpsPath           = (Join-Path $PSScriptRoot 'ops-assert.ps1')
            EvidenceDirectory = $dir
        }
        if ($SeedConfig) { $parameters.SeedConfig = $SeedConfig }
        $session = Invoke-PoliteRun @parameters
        if ($After) { & $After }
        $verdict = Test-PoliteSession -Session $session
        if (-not $verdict.Passed) { throw "politeness verdict failed: $($verdict | ConvertTo-Json -Compress)" }
        $script:Results.Add("$Name : PASS")
        Write-Host "$Name : PASS"
    } catch {
        Write-Host ''
        Write-Host "SCENARIO FAILED: $Name"
        Write-Host "$_"
        if (Test-Path -LiteralPath (Join-Path $dir 'evidence.txt')) {
            Write-Host "evidence: $(Join-Path $dir 'evidence.txt')"
        }
        $script:Results.Add("$Name : FAIL")
        throw
    } finally {
        if ($After) { try { & $After } catch { } }
        $env:WHALEISLE_VENDOR = $savedVendor
        $env:POLITE_SCENARIO = $savedScenario
        $env:POLITE_SETTLE = $savedSettle
    }
}

function Invoke-OrphanScenario {
    $savedVendor = $env:WHALEISLE_VENDOR
    $dir = Join-Path $EvidenceRoot 'orphan'
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    $env:WHALEISLE_VENDOR = $VendorDir
    $profile = $null
    try {
        $first = Invoke-PoliteRun -ExePath $exeFull -OpsPath (Join-Path $PSScriptRoot 'ops-orphan-first.ps1') `
            -SeedConfig (New-SeedConfig -Name 'workspace-zh' -Config @{ locale = 'zh'; autoStartDesktop = $true; workspace = $workspace; host = '127.0.0.1'; port = 3080; theme = 'deepseek' }) `
            -EvidenceDirectory (Join-Path $dir 'first') -StartupTimeoutSec 120
        $profile = $first.Profile
        $record = Get-Content -LiteralPath (Join-Path $profile 'runtime\host-process.json') -Raw | ConvertFrom-Json
        Write-Host "orphan            : pid $($record.pid) on port $($record.port), profile $profile"

        $env:POLITE_ORPHAN_PID = "$($record.pid)"
        $env:POLITE_ORPHAN_PORT = "$($record.port)"
        $second = Invoke-PoliteRun -ExePath $exeFull -OpsPath (Join-Path $PSScriptRoot 'ops-orphan-second.ps1') `
            -ProfilePath $profile -EvidenceDirectory (Join-Path $dir 'second') -StartupTimeoutSec 120
        $verdict = Test-PoliteSession -Session $second
        if (-not $verdict.Passed) { throw "politeness verdict failed: $($verdict | ConvertTo-Json -Compress)" }
        $script:Results.Add('orphan : PASS')
        Write-Host 'orphan : PASS'
    } catch {
        Write-Host ''
        Write-Host 'SCENARIO FAILED: orphan'
        Write-Host "$_"
        if ($profile) { Write-Host "profile kept: $profile" }
        $script:Results.Add('orphan : FAIL')
        throw
    } finally {
        Remove-Item Env:\POLITE_ORPHAN_PID -ErrorAction SilentlyContinue
        Remove-Item Env:\POLITE_ORPHAN_PORT -ErrorAction SilentlyContinue
        $env:WHALEISLE_VENDOR = $savedVendor
    }
}

# --- sequence -------------------------------------------------------------

# Dot-sourcing this file is how the window-free fixture test reaches New-SeedConfig,
# New-MismatchVendor and Start-PortWall without running a scenario.
if ($MyInvocation.InvocationName -eq '.') { return }

if (-not $Exe) { throw 'no desktop binary: pass -Exe .\dist\polite\whaleisle.exe' }
if (-not (Test-Path -LiteralPath $exeFull -PathType Leaf)) { throw "no desktop binary at $exeFull" }

$steps = New-Object System.Collections.Generic.List[string]
$steps.Add('no-workspace-zh')
$steps.Add('no-workspace-en')
if ($VendorDir) { $steps.Add('starting-zh') }
$steps.Add('missing-runtime-zh')
$steps.Add('mismatch-zh')
$steps.Add('port-taken-zh')
if ($VendorDir -and -not $SkipOrphan) { $steps.Add('orphan') }
if ($Only.Count -gt 0) { $steps = @($steps | Where-Object { $Only -contains $_ }) }

$zhWorkspaceSeed = New-SeedConfig -Name 'workspace-zh' -Config @{
    locale = 'zh'; autoStartDesktop = $true; workspace = $workspace; host = '127.0.0.1'; port = 3080; theme = 'deepseek'
}
$enNoWorkspaceSeed = New-SeedConfig -Name 'no-workspace-en' -Config @{ locale = 'en'; autoStartDesktop = $true }
$mismatchVendor = New-MismatchVendor
$emptyVendor = New-EmptyVendor

$portWall = $null
try {
    foreach ($step in $steps) {
        switch ($step) {
            'no-workspace-zh' {
                Invoke-CopyScenario -Name 'no-workspace-zh' -Scenario 'no-workspace-zh'
            }
            'no-workspace-en' {
                Invoke-CopyScenario -Name 'no-workspace-en' -Scenario 'no-workspace-en' -SeedConfig $enNoWorkspaceSeed
            }
            'starting-zh' {
                Invoke-CopyScenario -Name 'starting-zh' -Scenario 'starting-zh' -SeedConfig $zhWorkspaceSeed -Vendor $VendorDir
            }
            'missing-runtime-zh' {
                Invoke-CopyScenario -Name 'missing-runtime-zh' -Scenario 'missing-runtime-zh' -SeedConfig $zhWorkspaceSeed -Vendor $emptyVendor
            }
            'mismatch-zh' {
                Invoke-CopyScenario -Name 'mismatch-zh' -Scenario 'mismatch-zh' -SeedConfig $zhWorkspaceSeed -Vendor $mismatchVendor
            }
            'port-taken-zh' {
                $portWall = Start-PortWall
                try {
                    Invoke-CopyScenario -Name 'port-taken-zh' -Scenario 'port-taken-zh' -SeedConfig $zhWorkspaceSeed -Vendor $VendorDir
                } finally {
                    Stop-PortWall -Listeners $portWall
                    $portWall = $null
                }
            }
            'orphan' {
                Invoke-OrphanScenario
            }
            default { throw "unknown scenario '$step'" }
        }
    }
} finally {
    if ($portWall) { Stop-PortWall -Listeners $portWall }
}

Write-Host ''
Write-Host '=== live scenarios ==='
foreach ($line in $script:Results) { Write-Host $line }
Write-Host "evidence root: $EvidenceRoot"
