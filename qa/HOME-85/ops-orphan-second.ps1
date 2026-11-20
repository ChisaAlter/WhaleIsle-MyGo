#Requires -Version 5.1
<#
Second half of the orphan run: relaunch on the profile the killed shell left
behind and prove the new launch ended the surviving Host and bound the port it
held, instead of starting a second service beside it.
#>
function Read-PoliteHostRecordFile {
    param([Parameter(Mandatory)]$Session)
    $path = Join-Path $Session.Profile 'runtime\host-process.json'
    if (-not (Test-Path -LiteralPath $path)) { throw "no host record at $path" }
    Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
}

function Get-PoliteListenerPidFor {
    param([Parameter(Mandatory)][int]$Port)
    $rows = @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue)
    if ($rows.Count -eq 0) { return 0 }
    [int]$rows[0].OwningProcess
}

function Invoke-PoliteOps {
    param($Session)

    $oldPid = [int]$env:POLITE_ORPHAN_PID
    $oldPort = [int]$env:POLITE_ORPHAN_PORT
    if ($oldPid -le 0 -or $oldPort -le 0) { throw 'POLITE_ORPHAN_PID and POLITE_ORPHAN_PORT must name the orphan' }

    $record = Read-PoliteHostRecordFile -Session $Session
    $deadline = (Get-Date).AddSeconds(120)
    while ((Get-Date) -lt $deadline -and $record.pid -eq $oldPid) {
        Start-Sleep -Milliseconds 500
        $record = Read-PoliteHostRecordFile -Session $Session
    }

    Invoke-PoliteStep -Session $Session -Name 'snapshot after relaunch' -Action {
        Read-PoliteSnapshot -Session $Session
    } | Out-Null

    $oldAlive = Get-Process -Id $oldPid -ErrorAction SilentlyContinue
    if ($oldAlive) { throw "the orphaned Host pid $oldPid is still running after the next launch" }
    if ($record.pid -eq $oldPid) { throw "the record still names the orphan pid $oldPid" }

    $listener = Get-PoliteListenerPidFor -Port $oldPort
    if ($listener -ne $record.pid) {
        throw "port $oldPort is owned by pid $listener, not by the Host this launch recorded ($($record.pid))"
    }

    Write-Host "reap              : orphan pid $oldPid is gone; record now names pid $($record.pid)"
    Write-Host "port              : $oldPort now held by pid $listener, the Host this launch started"
}
