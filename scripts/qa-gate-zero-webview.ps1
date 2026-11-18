# Core startup gate. Only explicit isolated data is used; optional tools may own
# extra Node processes. Readiness is the Host handshake, not process existence.
param(
    [string]$Executable,
    [string]$UserDataDir,
    [ValidateRange(1,3600)][int]$WarmupSeconds = 15,
    [ValidateRange(2,3600)][int]$TimeoutSeconds = 30,
    [ValidateRange(0,3600)][int]$IdleSeconds = 0,
    [string]$ReportPath,
    [Alias("LauncherOnly")][switch]$WorkbenchOnly,
    [switch]$ResumeOwnedProfile,
    [switch]$RequireWorkspaceReady,
    [switch]$RequireProcessStartEvents,
    # Default measurements include the real desktop background schedule.
    [switch]$SuppressScheduledUpdates,
    # Explicitly authorized desktop-input QA; default remains hidden.
    [switch]$Visible,
    [switch]$DryRun
)
$ErrorActionPreference = 'Stop'
if ($WarmupSeconds -ge $TimeoutSeconds) { throw 'WarmupSeconds must be shorter than TimeoutSeconds.' }
if ($IdleSeconds -gt 0 -and $WorkbenchOnly) { throw 'Idle performance requires the complete desktop Host.' }
if ($RequireWorkspaceReady -and $WorkbenchOnly) { throw 'Workspace readiness requires the complete desktop Host.' }
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot
. (Join-Path $PSScriptRoot 'env.ps1')

function Get-ProcessDescendants([uint32]$rootPid) {
    $all = @(Get-CimInstance Win32_Process)
    $known = [Collections.Generic.HashSet[uint32]]::new()
    $null = $known.Add($rootPid)
    $found = [Collections.Generic.List[object]]::new()
    do {
        $added = $false
        foreach ($item in $all) {
            if ($known.Contains([uint32]$item.ParentProcessId) -and $known.Add([uint32]$item.ProcessId)) {
                $found.Add($item); $added = $true
            }
        }
    } while ($added)
    return $found.ToArray()
}

# Process identities include creation time so PID reuse never joins unrelated
# activity to this run. CPU percentages use one logical core, not machine cores.
function Read-OwnedResources($identities, [uint32]$desktopPid, [uint32]$hostPid) {
    $rows = @()
    foreach ($entry in @($identities.GetEnumerator())) {
        $live = Get-Process -Id $entry.Key -ErrorAction SilentlyContinue
        if (-not $live) { continue }
        try {
            if ([Math]::Abs(($live.StartTime.ToUniversalTime()-$entry.Value).TotalSeconds) -ge 1) { continue }
            $role = if ($entry.Key -eq $desktopPid) { 'desktop' } elseif ($entry.Key -eq $hostPid) { 'host' } else { 'helper' }
            $rows += [ordered]@{
                pid=[uint32]$entry.Key; startedAt=$entry.Value.ToString('o'); name=$live.ProcessName; role=$role
                cpuSeconds=$live.TotalProcessorTime.TotalSeconds; workingSetBytes=$live.WorkingSet64
                privateBytes=$live.PrivateMemorySize64; handles=$live.HandleCount; threads=$live.Threads.Count
            }
        } catch [InvalidOperationException] { }
    }
    return $rows
}
if ($DryRun) {
    $items = @(Get-ProcessDescendants ([uint32]$PID))
    Write-Host "Process inventory available; descendants=$($items.Count). No application started."
    return
}
$tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\','/')
$workPrefix = 'whaleisle-qa-webview-gate-'
$ownsUserData = -not $PSBoundParameters.ContainsKey('UserDataDir')
if ($ResumeOwnedProfile -and $ownsUserData) { throw 'ResumeOwnedProfile requires an explicit profile created by this gate.' }
if (-not $UserDataDir) { $UserDataDir = Join-Path $tempParent ($workPrefix + [guid]::NewGuid().ToString('N')) }
$UserDataDir = [IO.Path]::GetFullPath($UserDataDir)
$ownerPath = Join-Path $UserDataDir '.whaleisle-qa-owned-profile.json'
if (Test-Path -LiteralPath $UserDataDir) {
    if ((Get-Item -LiteralPath $UserDataDir).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'QA profile cannot be a link.' }
    if ($ResumeOwnedProfile) {
        $owner = Get-Content -LiteralPath $ownerPath -Raw | ConvertFrom-Json
        if ($owner.schema -ne 1 -or $owner.purpose -ne 'whaleisle-zero-webview-qa' -or $owner.path -ne $UserDataDir) { throw 'Profile is not owned by this QA gate; existing data was retained.' }
    } elseif (@(Get-ChildItem -LiteralPath $UserDataDir -Force).Count) { throw 'QA requires an empty profile; existing data was retained.' }
} elseif ($ResumeOwnedProfile) {
    throw 'Owned profile does not exist.'
}
if (-not $Executable) {
    $Executable = @((Join-Path $ProjectRoot 'whaleisle.exe'),(Join-Path $ProjectRoot 'dist\whaleisle\whaleisle.exe')) |
        Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
}
if (-not $Executable -or -not (Test-Path -LiteralPath $Executable -PathType Leaf)) { throw 'Supply a built candidate using -Executable.' }
$Executable = (Get-Item -LiteralPath $Executable).FullName
$names = @('WHALEISLE_USER_DATA','WHALEISLE_VENDOR','WHALEISLE_NODE_DIR','WHALEISLE_DISABLE_AUTO_UPDATE','WHALEISLE_ISOLATED','WHALEISLE_ISOLATED_VISIBLE','WHALEISLE_STARTUP_TRACE','LOCALAPPDATA','PSModuleAnalysisCachePath','WHALEISLE_CPU_PROFILE','WHALEISLE_ALLOC_PROFILE','WHALEISLE_UI_SAMPLE','MYGO_FRAME_STATS')
$saved = @{}
foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name,'Process') }
$proc = $null
$observed = @{}
$eventSource = 'whaleisle-process-start-' + [guid]::NewGuid().ToString('N')
$watcher = $null
$retain = $false
$profileKind = if ($ResumeOwnedProfile) { 'resumed-owned-isolated' } else { 'fresh-isolated' }
$report = [ordered]@{schema=1; executable=$Executable; hidden=(-not $Visible); profile=$profileKind; status='failed'; startup=@(); idle=$null}
if ($ReportPath) { $ReportPath = [IO.Path]::GetFullPath($ReportPath) }
try {
    New-Item -ItemType Directory -Force -Path $UserDataDir | Out-Null
    if (-not $ResumeOwnedProfile) {
        $config = [ordered]@{autoDownloadUpdates=$false;autoStartDesktop=(-not $WorkbenchOnly);remoteEnabled=$false;harnessAutoRestart=$false;closeToTray=$false}
        if (-not $WorkbenchOnly) {
            $config.workspace = Join-Path $UserDataDir 'workspace'
            New-Item -ItemType Directory -Path $config.workspace | Out-Null
        }
        $config | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $UserDataDir 'config.json') -Encoding UTF8
        New-Item -ItemType File -Path (Join-Path $UserDataDir 'welcome-shown') | Out-Null
        [ordered]@{schema=1;purpose='whaleisle-zero-webview-qa';path=$UserDataDir} | ConvertTo-Json | Set-Content -LiteralPath $ownerPath -Encoding UTF8
    }
    $env:WHALEISLE_USER_DATA = $UserDataDir
    $env:WHALEISLE_ISOLATED = '1'
    $env:WHALEISLE_ISOLATED_VISIBLE = if ($Visible) { '1' } else { '0' }
    if ($SuppressScheduledUpdates) { $env:WHALEISLE_DISABLE_AUTO_UPDATE = '1' }
    $report.scheduledUpdatesSuppressed = @('1','true','yes','on') -contains ([string]$env:WHALEISLE_DISABLE_AUTO_UPDATE).Trim()
    $report.warmupSeconds = $WarmupSeconds
    $env:LOCALAPPDATA = $UserDataDir
    $env:PSModuleAnalysisCachePath = Join-Path $UserDataDir 'psmodulecache'
    $tracePath = Join-Path $UserDataDir 'startup-gate.json'
    $env:WHALEISLE_STARTUP_TRACE = $tracePath
    # Opt-in profiles and per-frame logs perturb the steady-state sample.
    if ($IdleSeconds -gt 0) {
        foreach ($name in @('WHALEISLE_CPU_PROFILE','WHALEISLE_ALLOC_PROFILE','WHALEISLE_UI_SAMPLE','MYGO_FRAME_STATS')) {
            [Environment]::SetEnvironmentVariable($name,$null,'Process')
        }
    }
    $exeDir = Split-Path $Executable -Parent
    foreach ($pair in @(@('WHALEISLE_VENDOR','vendor'),@('WHALEISLE_NODE_DIR','node'))) {
        if (-not [Environment]::GetEnvironmentVariable($pair[0],'Process')) {
            $selected = @((Join-Path $exeDir "runtime\$($pair[1])"),(Join-Path $ProjectRoot "runtime-rc4\$($pair[1])"),(Join-Path $ProjectRoot "runtime\$($pair[1])")) |
                Where-Object { Test-Path -LiteralPath $_ -PathType Container } | Select-Object -First 1
            if ($selected) { [Environment]::SetEnvironmentVariable($pair[0],$selected,'Process') }
        }
    }
    $eventCoverage = 'sampled-startup'
    try {
        $watcher = Register-CimIndicationEvent -Query 'SELECT * FROM Win32_ProcessStartTrace' -SourceIdentifier $eventSource
        $eventCoverage = 'process-start-events'
    } catch {
        if ($RequireProcessStartEvents) { throw 'Process-start events are unavailable; full lifecycle acceptance was not run.' }
        Write-Warning 'Process-start events are unavailable. This run covers sampled startup only, not every short-lived process.'
    }
    $info = New-Object Diagnostics.ProcessStartInfo
    $info.FileName = $Executable
    $info.WorkingDirectory = $ProjectRoot
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.WindowStyle = if ($Visible) { [Diagnostics.ProcessWindowStyle]::Normal } else { [Diagnostics.ProcessWindowStyle]::Hidden }
    $proc = [Diagnostics.Process]::Start($info)
    $observed[[uint32]$proc.Id] = $proc.StartTime.ToUniversalTime()
    $clock = [Diagnostics.Stopwatch]::StartNew()
    $readyAt = $null
    while ($clock.Elapsed.TotalSeconds -lt $TimeoutSeconds) {
        foreach ($event in @(Get-Event -SourceIdentifier $eventSource -ErrorAction SilentlyContinue | Sort-Object TimeGenerated)) {
            $created = $event.SourceEventArgs.NewEvent
            if ($observed.ContainsKey([uint32]$created.ParentProcessID)) {
                $observed[[uint32]$created.ProcessID] = [DateTime]::FromFileTimeUtc([int64]$created.TIME_CREATED)
                if ($created.ProcessName -like '*msedgewebview2*') { throw 'Route A started WebView2 during core startup.' }
            }
            Remove-Event -EventIdentifier $event.EventIdentifier
        }
        foreach ($child in @(Get-ProcessDescendants ([uint32]$proc.Id))) {
            $observed[[uint32]$child.ProcessId] = $child.CreationDate.ToUniversalTime()
            if ($child.Name -like '*msedgewebview2*') { throw 'Route A has an active WebView2 process.' }
        }
        if ($proc.HasExited) { throw "Target exited before readiness (code $($proc.ExitCode))." }
        if (Test-Path -LiteralPath $tracePath) {
            try { $trace = Get-Content -LiteralPath $tracePath -Raw | ConvertFrom-Json } catch { $trace = $null }
            if ($trace -and $trace.pid -eq $proc.Id) {
                $stages = @($trace.events | ForEach-Object { $_.stage })
                $ready = if ($WorkbenchOnly) { ($stages -contains 'window.created') -and (-not $Visible -or $stages -contains 'ui.first_view_built') } else { ($stages -contains 'host.ready') -and $trace.finished -and $trace.reason -eq 'ready' }
                if ($Visible) { $ready = $ready -and ($stages -contains 'ui.first_view_built') }
                if ($IdleSeconds -gt 0 -or $RequireWorkspaceReady) { $ready = $ready -and ($stages -contains 'workspace.ready') }
                if ($ready -and $null -eq $readyAt) { $readyAt = $clock.Elapsed.TotalSeconds }
            }
        }
        if ($null -ne $readyAt -and $clock.Elapsed.TotalSeconds -ge $WarmupSeconds -and $clock.Elapsed.TotalSeconds -ge ($readyAt+2)) { break }
        Start-Sleep -Milliseconds 200
    }
    if ($null -eq $readyAt -or $clock.Elapsed.TotalSeconds -ge $TimeoutSeconds) { throw "Readiness/observation did not finish within ${TimeoutSeconds}s." }
    if ($WorkbenchOnly -and @($trace.events | Where-Object { $_.stage -eq 'harness.start_requested' }).Count) { throw 'Host was started during workbench-only acceptance.' }
    $report.coverage = $eventCoverage
    $report.readiness = if ($IdleSeconds -gt 0 -or $RequireWorkspaceReady) { 'workspace' } elseif ($WorkbenchOnly -and $Visible) { 'native-workbench-view-built' } elseif ($WorkbenchOnly) { 'main-window-created-hidden' } else { 'host' }
    # startuptrace already redacts Host credentials before recording each line.
    # Keep the bounded timeline so Node load and readiness can be distinguished.
    $report.startup = @($trace.events)
    $report.runtime = [ordered]@{vendor=$env:WHALEISLE_VENDOR;nodeDirectory=$env:WHALEISLE_NODE_DIR}
    $hostEvent = @($trace.events | Where-Object { $_.stage -eq 'host.spawned' }) | Select-Object -Last 1
    [uint32]$hostPid = 0
    if ($hostEvent) { $hostPid = [uint32]$hostEvent.detail }
    if ($IdleSeconds -gt 0) {
        if ($hostPid -eq 0) { throw 'Host process identity missing from startup trace.' }
        $samples = [Collections.Generic.List[object]]::new()
        $idleClock = [Diagnostics.Stopwatch]::StartNew()
        do {
            foreach ($event in @(Get-Event -SourceIdentifier $eventSource -ErrorAction SilentlyContinue | Sort-Object TimeGenerated)) {
                $created = $event.SourceEventArgs.NewEvent
                if ($observed.ContainsKey([uint32]$created.ParentProcessID)) {
                    $observed[[uint32]$created.ProcessID] = [DateTime]::FromFileTimeUtc([int64]$created.TIME_CREATED)
                    if ($created.ProcessName -like '*msedgewebview2*') { throw 'Route A started WebView2 during idle observation.' }
                }
                Remove-Event -EventIdentifier $event.EventIdentifier
            }
            foreach ($child in @(Get-ProcessDescendants ([uint32]$proc.Id))) {
                $observed[[uint32]$child.ProcessId] = $child.CreationDate.ToUniversalTime()
                if ($child.Name -like '*msedgewebview2*') { throw 'Route A has an active WebView2 process during idle observation.' }
            }
            $rows = @(Read-OwnedResources $observed ([uint32]$proc.Id) $hostPid)
            if (-not @($rows | Where-Object { $_.role -eq 'desktop' }).Count -or -not @($rows | Where-Object { $_.role -eq 'host' }).Count) { throw 'Desktop or Host exited during the idle observation.' }
            if (@($rows | Where-Object { $_.name -like '*msedgewebview2*' }).Count) { throw 'An owned detached WebView2 survived during idle observation.' }
            $samples.Add([ordered]@{elapsedSeconds=$idleClock.Elapsed.TotalSeconds; processes=$rows})
            if ($idleClock.Elapsed.TotalSeconds-$samples[0].elapsedSeconds -ge $IdleSeconds) { break }
            Start-Sleep -Milliseconds 1000
        } while ($true)
        $first, $last = $samples[0], $samples[$samples.Count-1]
        $duration = $last.elapsedSeconds-$first.elapsedSeconds
        $totals = @()
        foreach ($end in $last.processes) {
            $begin = @($first.processes | Where-Object { $_.pid -eq $end.pid -and $_.startedAt -eq $end.startedAt }) | Select-Object -First 1
            if (-not $begin) { continue }
            $totals += [ordered]@{
                pid=$end.pid; name=$end.name; role=$end.role; observedSeconds=$duration
                cpuPercentOneCore=100*($end.cpuSeconds-$begin.cpuSeconds)/$duration
                initialWorkingSetBytes=$begin.workingSetBytes; finalWorkingSetBytes=$end.workingSetBytes
                initialPrivateBytes=$begin.privateBytes; finalPrivateBytes=$end.privateBytes
                initialHandles=$begin.handles; finalHandles=$end.handles; finalThreads=$end.threads
            }
        }
        $report.idle = [ordered]@{requestedSeconds=$IdleSeconds; observedSeconds=$duration; samples=$samples.ToArray(); stableProcesses=$totals; summaryScope='processes alive for the entire observation; transient processes remain in samples'}
        foreach ($item in $totals) {
            Write-Host ("IDLE {0} ({1}): CPU={2:N3}% one core; private={3:N2}->{4:N2} MiB; handles={5}->{6}" -f $item.role,$item.name,$item.cpuPercentOneCore,($item.initialPrivateBytes/1MB),($item.finalPrivateBytes/1MB),$item.initialHandles,$item.finalHandles)
        }
    }
    $active = @(Get-CimInstance Win32_Process | Where-Object {
        $observed.ContainsKey([uint32]$_.ProcessId) -and [Math]::Abs(($_.CreationDate.ToUniversalTime()-$observed[[uint32]$_.ProcessId]).TotalSeconds) -lt 1
    })
    if (@($active | Where-Object { $_.Name -like '*msedgewebview2*' }).Count) { throw 'WebView2 survived in the owned process set.' }
    if (@(Get-ChildItem -LiteralPath $UserDataDir -Recurse -Directory -Filter EBWebView).Count) { throw 'An EBWebView profile was created.' }
    [int64]$workingSet = 0
    [int64]$privateBytes = 0
    foreach ($item in $active) {
        $live = Get-Process -Id $item.ProcessId -ErrorAction SilentlyContinue
        if ($live) { $workingSet += $live.WorkingSet64; $privateBytes += $live.PrivateMemorySize64 }
    }
    Write-Host "PASS ($eventCoverage): core startup ready; WebView2=0; owned processes=$($active.Count); WS=$([Math]::Round($workingSet/1MB,2)) MiB; private=$([Math]::Round($privateBytes/1MB,2)) MiB."
    $report.status = 'passed'
    $report.webViewCount = 0
} catch {
    $report.error = $_.Exception.Message
    throw
} finally {
    if ($watcher) {
        Unregister-Event -SourceIdentifier $eventSource -ErrorAction SilentlyContinue
        Get-Event -SourceIdentifier $eventSource -ErrorAction SilentlyContinue | Remove-Event
    }
    if ($proc) {
        foreach ($child in @(Get-ProcessDescendants ([uint32]$proc.Id))) { $observed[[uint32]$child.ProcessId] = $child.CreationDate.ToUniversalTime() }
        foreach ($entry in @($observed.GetEnumerator())) {
            $live = Get-Process -Id $entry.Key -ErrorAction SilentlyContinue
            if ($live -and [Math]::Abs(($live.StartTime.ToUniversalTime()-$entry.Value).TotalSeconds) -lt 1) {
                try { if (-not $live.HasExited) { $live.Kill(); if (-not $live.WaitForExit(5000)) { $retain=$true } } } catch { $retain=$true }
            }
        }
    }
    foreach ($name in $saved.Keys) {
        if ($null -eq $saved[$name]) { Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue }
        else { [Environment]::SetEnvironmentVariable($name,$saved[$name],'Process') }
    }
    if ($ownsUserData -and (Test-Path -LiteralPath $UserDataDir)) {
        if ($retain) { Write-Warning "Owned process has not exited; retained: $UserDataDir" }
        else {
            & node (Join-Path $PSScriptRoot 'build-storage.mjs') owned-dir $UserDataDir $tempParent $workPrefix
            if ($LASTEXITCODE -ne 0) { throw "Gate cleanup failed; retained: $UserDataDir" }
        }
    }
    if ($ReportPath) {
        $report | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $ReportPath -Encoding utf8
    }
}
