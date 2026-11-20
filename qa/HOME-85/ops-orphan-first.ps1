#Requires -Version 5.1
<#
First half of the orphan run: start with a workspace, wait for the Host to be
recorded and listening, then kill only this shell the way a crash would. The
Host it detached is what the next launch has to find.
#>
function Read-PoliteHostRecord {
    param([Parameter(Mandatory)]$Session)
    $path = Join-Path $Session.Profile 'runtime\host-process.json'
    $deadline = (Get-Date).AddSeconds(120)
    while ((Get-Date) -lt $deadline -and -not (Test-Path -LiteralPath $path)) { Start-Sleep -Milliseconds 500 }
    if (-not (Test-Path -LiteralPath $path)) { throw "no host record at $path after a start" }
    Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
}

function Get-PoliteListenerPid {
    param([Parameter(Mandatory)][int]$Port)
    $rows = @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue)
    if ($rows.Count -eq 0) { return 0 }
    [int]$rows[0].OwningProcess
}

function Invoke-PoliteOps {
    param($Session)

    $record = Read-PoliteHostRecord -Session $Session
    Invoke-PoliteStep -Session $Session -Name 'snapshot before kill' -Action {
        Read-PoliteSnapshot -Session $Session
    } | Out-Null

    $hostProcess = Get-Process -Id $record.pid -ErrorAction SilentlyContinue
    if (-not $hostProcess) { throw "the recorded Host pid $($record.pid) is not running" }
    $listener = Get-PoliteListenerPid -Port $record.port
    if ($listener -ne $record.pid) { throw "the recorded Host pid $($record.pid) does not own port $($record.port) (listener $listener)" }

    Write-Host "orphan setup      : host pid $($record.pid) ($($hostProcess.ProcessName)) listening on $($record.port), shell pid $($Session.Pid)"
    Write-Host "record            : $(Get-Content -LiteralPath (Join-Path $Session.Profile 'runtime\host-process.json') -Raw)"

    $live = Get-Process -Id $Session.Pid -ErrorAction SilentlyContinue
    if (-not $live -or -not $live.Path) { throw 'the shell we started is already gone' }
    if ([IO.Path]::GetFullPath($live.Path) -ne [IO.Path]::GetFullPath($Session.Exe)) { throw 'pid identity changed before the kill' }

    # Only our own pid, and only after proving it is the binary this run started.
    Stop-Process -Id $Session.Pid -Force
    Start-Sleep -Milliseconds 1500
    $hostAfter = Get-Process -Id $record.pid -ErrorAction SilentlyContinue
    if (-not $hostAfter) { throw "the Host pid $($record.pid) died with its shell; there is nothing left to reap" }
    $listenerAfter = Get-PoliteListenerPid -Port $record.port
    if ($listenerAfter -ne $record.pid) { throw "port $($record.port) is no longer held by the orphan ($listenerAfter)" }

    Write-Host "orphan after kill : host pid $($record.pid) still running ($($hostAfter.ProcessName)), still listening on $($record.port); shell $($Session.Pid) gone"
}
