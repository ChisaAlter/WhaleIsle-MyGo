# qa-gate-zero-webview.ps1 - Route A Zero-WebView Automated Acceptance Gate
#
# Asserts that WhaleIsle MyGo runs purely native with 0 persistent WebView2 instances,
# 0 EBWebView user data directories, and no extra background node workers.
# Complies with AGENTS.md build/storage and lifecycle teardown rules.

param(
    [string]$Executable,
    [string]$UserDataDir,
    [int]$WarmupSeconds = 15,
    [int]$TimeoutSeconds = 30,
    [switch]$LauncherOnly,
    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

. (Join-Path $PSScriptRoot 'env.ps1')

function Get-ProcessDescendants([uint32]$rootPid) {
    $allProcesses = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue
    if (-not $allProcesses) { return @() }
    $lookup = $allProcesses | Group-Object ParentProcessId -AsHashTable
    $descendants = [System.Collections.Generic.List[object]]::new()
    $q = [System.Collections.Generic.Queue[uint32]]::new()
    $q.Enqueue($rootPid)
    while ($q.Count -gt 0) {
        $curr = $q.Dequeue()
        if ($lookup.ContainsKey($curr)) {
            foreach ($child in $lookup[$curr]) {
                $descendants.Add($child)
                $q.Enqueue([uint32]$child.ProcessId)
            }
        }
    }
    return $descendants
}

$tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/')
$workPrefix = 'whaleisle-qa-webview-gate-'
$ownsUserData = -not $PSBoundParameters.ContainsKey('UserDataDir')
if (-not $UserDataDir) {
    $UserDataDir = Join-Path $tempParent ($workPrefix + [guid]::NewGuid().ToString('N'))
}
$UserDataDir = [IO.Path]::GetFullPath($UserDataDir)

if (-not $Executable) {
    $candidateExe = @(
        (Join-Path $ProjectRoot 'whaleisle.exe'),
        (Join-Path $ProjectRoot 'dist\whaleisle\whaleisle.exe')
    ) | Where-Object { Test-Path $_ -PathType Leaf } | Select-Object -First 1
    if ($candidateExe) {
        $Executable = $candidateExe
    } else {
        $Executable = Join-Path $ProjectRoot 'whaleisle.exe'
    }
}

Write-Host "=== Route A Zero-WebView Gate ==="
Write-Host "Target Executable : $Executable"
Write-Host "Isolated UserData : $UserDataDir"
Write-Host "Warmup / Timeout  : ${WarmupSeconds}s / ${TimeoutSeconds}s"
Write-Host "Mode              : $(if ($LauncherOnly) { 'LauncherOnly' } else { 'FullWorkbench (DSH Host required)' })"

if ($DryRun) {
    Write-Host "DryRun requested: verifying script structure and process tree helpers."
    $testSelf = @(Get-ProcessDescendants ([uint32]$PID))
    Write-Host "Self process ($PID) descendant query OK (count: $($testSelf.Count))."
    Write-Host "Route A Zero-WebView Gate structure verified (LauncherOnly=$LauncherOnly, WarmupSeconds=$WarmupSeconds)."
    exit 0
}

if (-not (Test-Path $Executable -PathType Leaf)) {
    throw "Target executable not found: $Executable. Build whaleisle.exe before running acceptance gate."
}

# Save environment for full teardown
$savedEnv = @{
    WHALEISLE_USER_DATA         = $env:WHALEISLE_USER_DATA
    WHALEISLE_VENDOR            = $env:WHALEISLE_VENDOR
    WHALEISLE_NODE_DIR          = $env:WHALEISLE_NODE_DIR
    WHALEISLE_NO_AUTO_UPDATE    = $env:WHALEISLE_NO_AUTO_UPDATE
    WHALEISLE_FORCE_MAIN        = $env:WHALEISLE_FORCE_MAIN
    LOCALAPPDATA                = $env:LOCALAPPDATA
    PSModuleAnalysisCachePath   = $env:PSModuleAnalysisCachePath
}

$proc = $null
try {
    New-Item -ItemType Directory -Force -Path $UserDataDir | Out-Null
    $configPath = Join-Path $UserDataDir 'config.json'
    # Mark first-run welcome window as already shown
    $welcomeMarker = Join-Path $UserDataDir 'welcome-shown'
    New-Item -ItemType File -Force -Path $welcomeMarker | Out-Null

    $configObj = [ordered]@{
        autoDownloadUpdates = $false
        autoStartDesktop    = $true
    }

    if (-not $LauncherOnly) {
        $workspaceDir = Join-Path $UserDataDir 'workspace'
        New-Item -ItemType Directory -Force -Path $workspaceDir | Out-Null
        $configObj['workspace'] = $workspaceDir
        $env:WHALEISLE_FORCE_MAIN = '1'
    } else {
        $env:WHALEISLE_FORCE_MAIN = $null
    }

    Set-Content -Path $configPath -Value ($configObj | ConvertTo-Json -Depth 4) -Encoding UTF8

    $env:WHALEISLE_USER_DATA = $UserDataDir
    $env:WHALEISLE_NO_AUTO_UPDATE = '1'
    $env:LOCALAPPDATA = $UserDataDir
    $env:PSModuleAnalysisCachePath = Join-Path $UserDataDir 'psmodulecache'

    $exeDir = Split-Path -Parent (Resolve-Path $Executable).Path
    $stageCandidate = Join-Path $exeDir 'runtime\vendor'
    if (Test-Path $stageCandidate) {
        $vendorCandidate = $stageCandidate
    } else {
        $vendorCandidate = Join-Path $ProjectRoot 'runtime-rc4\vendor'
        if (-not (Test-Path $vendorCandidate)) { $vendorCandidate = Join-Path $ProjectRoot 'runtime\vendor' }
    }
    if (-not $env:WHALEISLE_VENDOR -and (Test-Path $vendorCandidate)) {
        $env:WHALEISLE_VENDOR = $vendorCandidate
    }

    $stageNodeCandidate = Join-Path $exeDir 'runtime\node'
    if (Test-Path $stageNodeCandidate) {
        $nodeCandidate = $stageNodeCandidate
    } else {
        $nodeCandidate = Join-Path $ProjectRoot 'runtime-rc4\node'
        if (-not (Test-Path $nodeCandidate)) { $nodeCandidate = Join-Path $ProjectRoot 'runtime\node' }
    }
    if (-not $env:WHALEISLE_NODE_DIR -and (Test-Path $nodeCandidate)) {
        $env:WHALEISLE_NODE_DIR = $nodeCandidate
    }

    Write-Host "Resolved Vendor   : $env:WHALEISLE_VENDOR"
    Write-Host "Resolved Node Dir : $env:WHALEISLE_NODE_DIR"

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $Executable
    $psi.WorkingDirectory = $ProjectRoot
    $psi.UseShellExecute = $false
    # Do not redirect stdout/stderr to avoid child process pipe deadlock (4KB buffer limit)
    $psi.RedirectStandardOutput = $false
    $psi.RedirectStandardError = $false

    Write-Host "Launching $Executable in isolated environment..."
    $proc = [System.Diagnostics.Process]::Start($psi)
    if ($null -eq $proc) {
        throw "Failed to launch target process: $Executable"
    }

    Write-Host "Waiting up to ${WarmupSeconds}s for target stabilization and DSH Host readiness..."
    $pollIntervalMs = 500
    $maxPollIterations = [math]::Ceiling(($WarmupSeconds * 1000) / $pollIntervalMs)
    $hostObserved = $false

    for ($i = 0; $i -lt $maxPollIterations; $i++) {
        Start-Sleep -Milliseconds $pollIntervalMs
        if ($proc.HasExited) {
            throw "Target process exited prematurely with code $($proc.ExitCode)."
        }
        if (-not $LauncherOnly) {
            $currentDescendants = @(Get-ProcessDescendants ([uint32]$proc.Id))
            $currentNodes = @($currentDescendants | Where-Object { $_.Name -like '*node*' })
            if ($currentNodes.Count -ge 1) {
                Write-Host "DSH Host node detected after $([math]::Round(($i + 1) * $pollIntervalMs / 1000, 1))s. Observing stability for 3s..."
                Start-Sleep -Seconds 3
                $hostObserved = $true
                break
            }
        }
    }

    if ($proc.HasExited) {
        throw "Target process exited prematurely with code $($proc.ExitCode)."
    }

    # Recursive BFS process tree inspection
    $descendants = @(Get-ProcessDescendants ([uint32]$proc.Id))
    $allInTree = @($proc.Id) + @($descendants | ForEach-Object { $_.ProcessId })

    $webviews = @($descendants | Where-Object { $_.Name -like '*msedgewebview2*' })
    $nodes = @($descendants | Where-Object { $_.Name -like '*node*' })
    $ebDir = Join-Path $UserDataDir 'EBWebView'

    Write-Host "--- Process Tree & Resource Analysis ---"
    Write-Host "Total Tree Processes : $($allInTree.Count)"

    $totalWsBytes = 0
    $totalPmBytes = 0

    # Print memory & handle metrics for main process and DSH Host node
    try {
        $mainProc = Get-Process -Id $proc.Id -ErrorAction SilentlyContinue
        if ($mainProc) {
            $wsMiB = [math]::Round($mainProc.WorkingSet64 / 1MB, 2)
            $pmMiB = [math]::Round($mainProc.PrivateMemorySize64 / 1MB, 2)
            $totalWsBytes += $mainProc.WorkingSet64
            $totalPmBytes += $mainProc.PrivateMemorySize64
            Write-Host "Main UI ($($mainProc.ProcessName) PID $($mainProc.Id)): WS = $wsMiB MiB, Private = $pmMiB MiB, Handles = $($mainProc.HandleCount), Threads = $($mainProc.Threads.Count)"
        }
    } catch {}

    foreach ($n in $nodes) {
        try {
            $nodeProc = Get-Process -Id $n.ProcessId -ErrorAction SilentlyContinue
            if ($nodeProc) {
                $nws = [math]::Round($nodeProc.WorkingSet64 / 1MB, 2)
                $npm = [math]::Round($nodeProc.PrivateMemorySize64 / 1MB, 2)
                $totalWsBytes += $nodeProc.WorkingSet64
                $totalPmBytes += $nodeProc.PrivateMemorySize64
                Write-Host "Host Node ($($n.Name) PID $($n.ProcessId)): WS = $nws MiB, Private = $npm MiB, Handles = $($nodeProc.HandleCount)"
                if ($n.CommandLine) {
                    Write-Host "  CommandLine: $($n.CommandLine)"
                }
            }
        } catch {}
    }

    $totalWsMiB = [math]::Round($totalWsBytes / 1MB, 2)
    $totalPmMiB = [math]::Round($totalPmBytes / 1MB, 2)
    Write-Host "Combined Resource Usage: WS = $totalWsMiB MiB, Private = $totalPmMiB MiB"

    # Strict Assertions
    $failed = $false
    if ($webviews.Count -gt 0) {
        [Console]::Error.WriteLine("ASSERTION FAILED: Route A requires msedgewebview2.exe count == 0; found $($webviews.Count) instances.")
        $webviews | ForEach-Object { [Console]::Error.WriteLine("  - PID $($_.ProcessId) ($($_.Name))") }
        $failed = $true
    } else {
        Write-Host "[PASS] Zero WebView2 instances in descendant tree (msedgewebview2.exe == 0)."
    }

    if (Test-Path $ebDir) {
        [Console]::Error.WriteLine("ASSERTION FAILED: EBWebView user data directory exists at $ebDir.")
        $failed = $true
    } else {
        Write-Host "[PASS] No EBWebView user data directory detected."
    }

    if (-not $LauncherOnly) {
        if ($nodes.Count -ne 1) {
            [Console]::Error.WriteLine("ASSERTION FAILED: Route A workbench expects exactly 1 DSH Host node.exe process; found $($nodes.Count).")
            foreach ($n in $nodes) {
                [Console]::Error.WriteLine("  - Node PID $($n.ProcessId): $($n.CommandLine)")
            }
            $failed = $true
        } else {
            Write-Host "[PASS] Exactly 1 DSH Host node.exe running in workbench."
        }
    } else {
        if ($nodes.Count -gt 0) {
            Write-Warning "NOTICE: LauncherOnly mode observed $($nodes.Count) node.exe processes."
        }
    }

    if ($failed) {
        throw "Route A Zero-WebView Gate FAILED."
    }

    Write-Host "=== Route A Zero-WebView Gate PASSED ==="
} finally {
    if ($proc -and -not $proc.HasExited) {
        Write-Host "Stopping target process tree..."
        try {
            # Collect descendants before killing parent
            $toKill = Get-ProcessDescendants ([uint32]$proc.Id)
            $proc.Kill()
            foreach ($child in $toKill) {
                try {
                    $cp = Get-Process -Id $child.ProcessId -ErrorAction SilentlyContinue
                    if ($cp -and -not $cp.HasExited) { $cp.Kill() }
                } catch {}
            }
            $proc.WaitForExit(5000) | Out-Null
        } catch {}
    }

    # Restore environment
    foreach ($k in $savedEnv.Keys) {
        if ($null -eq $savedEnv[$k]) {
            Remove-Item "Env:$k" -ErrorAction SilentlyContinue
        } else {
            Set-Item "Env:$k" $savedEnv[$k]
        }
    }

    # Safe reclamation of isolated UserDataDir
    if ($ownsUserData -and (Test-Path $UserDataDir)) {
        Start-Sleep -Milliseconds 500
        & node (Join-Path $PSScriptRoot 'build-storage.mjs') owned-dir $UserDataDir $tempParent $workPrefix
    }
}
