#Requires -Version 5.1
<#
.SYNOPSIS
One bounded polite cycle, used to prove the harness itself is safe to run.

.DESCRIPTION
Fresh isolated profile, launch with --background, one accessibility snapshot,
one harmless semantic action, close. Exit 0 only when the OS foreground window
handle is identical before, during and after the run, and no 界面未响应 window
ever appeared.

    .\tools\polite\selftest.ps1 -Exe .\dist\polite\whaleisle.exe -EvidenceDir .\qa-evidence

Exit codes: 0 pass, 1 politeness failure, 2 refused (a rival instance, a
missing binary, or no agent-desktop), 3 no harmless target found (a harness
limitation, no action was taken).
#>
[CmdletBinding()]
param(
    [string]$Exe,
    [string]$EvidenceDir,
    [int]$StartupTimeoutSec = 90,
    [int]$SnapshotTimeoutMs = 6000,
    [int]$QuietSeconds = 3,
    [switch]$KeepProfile
)

$ErrorActionPreference = 'Stop'
# The evidence is read back as a file, and Windows PowerShell writes redirected
# output in the console code page: every Chinese window title came back as
# mojibake in the run logs. UTF-8 keeps the evidence readable.
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false
. (Join-Path $PSScriptRoot 'run.ps1')

$session = $null
$failure = $null
$noTarget = $false
$code = 1

# The first measurement of the run. Everything after this is compared to it.
$before = Wait-QuietForeground -QuietSeconds $QuietSeconds
Write-Host "foreground before : $($before.Hwnd) '$($before.Title)' (pid $($before.OwnerPid) $($before.ProcessName))"

try {
    $session = Start-PoliteInstance -ExePath $Exe -StartupTimeoutSec $StartupTimeoutSec -SnapshotTimeoutMs $SnapshotTimeoutMs -ForegroundBaseline $before
    Assert-PoliteSessionShape -Session $session
    $session.EvidenceDir = $EvidenceDir
    Write-Host "launched          : $($session.CommandLine)"
    Write-Host "profile           : $($session.Profile)"
    Write-Host "app pid           : $($session.Pid)"
    Write-Host "window            : $($session.WindowId) '$($session.WindowTitle)'"

    $snapshot = Invoke-PoliteStep -Session $session -Name 'snapshot' -Action {
        Read-PoliteSnapshot -Session $session
    }
    Write-Host "tree              : $($snapshot.ref_count) refs, complete=$($snapshot.complete)"

    $picked = Find-PoliteHarmlessTarget -Snapshot $snapshot
    if (-not $picked) {
        $noTarget = $true
        throw "no harmless semantic target in the tree (nothing idempotent to click, clear, or uncheck). Nothing was sent to the app."
    }
    Write-Host "action            : $($picked.Action) $($picked.Ref) '$($picked.Name)' — $($picked.Why)"

    Invoke-PoliteStep -Session $session -Name ("$($picked.Action) $($picked.Ref)") -Action {
        Invoke-PoliteAction -Session $session -Ref $picked.Ref -Action $picked.Action -Value $picked.Value
    } | Out-Null

    Invoke-PoliteStep -Session $session -Name 'snapshot after' -Action {
        Read-PoliteSnapshot -Session $session | Out-Null
    } | Out-Null
} catch {
    $failure = $_
} finally {
    if ($session) {
        try { Close-PoliteInstance -Session $session } catch { if (-not $failure) { $failure = $_ } }
    } else {
        Exit-PoliteHarnessLock
    }
}

$after = Get-ForegroundWindowInfo
Write-Host "foreground after  : $($after.Hwnd) '$($after.Title)' (pid $($after.OwnerPid) $($after.ProcessName))"
if ($failure) { Write-Host "failure           : $failure" }
$evidence = ''
if ($session) { $evidence = Show-PoliteEvidence -Session $session }
if ($evidence) { Write-Host $evidence }
Write-Host "hwnd before/during/after: $($before.Hwnd) / $(@($session.Steps | ForEach-Object { $_.ForegroundAfter }) -join ' / ') / $($after.Hwnd)"

if ($session -and $EvidenceDir) {
    New-Item -ItemType Directory -Path $EvidenceDir -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $EvidenceDir 'selftest-evidence.txt') -Value $evidence -Encoding UTF8
}

if ($failure) {
    Write-Host "selftest failure  : $failure"
    $message = "$failure"
    if ($noTarget) { $code = 3 }
    elseif ($message -like 'refusing to start*' -or $message -like 'no desktop binary*' -or $message -like 'agent-desktop is not on PATH*') { $code = 2 }
    elseif ($message -like 'action-not-sent*') { $code = 3 }
    elseif ($message -like 'foreground-changed-externally*') { $code = 4 }
    else { $code = 1 }
} elseif ($session) {
    $verdict = Test-PoliteSession -Session $session
    if ($verdict.Passed) {
        Write-Host 'RESULT: PASS — foreground unchanged, no watchdog window, our pid exited'
        $code = 0
    } elseif ($verdict.AppFocusStolen) {
        Write-Host "RESULT: FAIL — our own process took the foreground; stable=$($verdict.ForegroundStable) nohang=$($verdict.NoHangWindow) exited=$($verdict.PidExited)"
        $code = 1
    } elseif ($verdict.ExternalChange) {
        Write-Host "RESULT: INCONCLUSIVE — a window that is not ours took the foreground during a step; nohang=$($verdict.NoHangWindow) exited=$($verdict.PidExited). Rerun while the machine is quiet."
        $code = 4
    } else {
        Write-Host "RESULT: FAIL — stable=$($verdict.ForegroundStable) nohang=$($verdict.NoHangWindow) exited=$($verdict.PidExited)"
        $code = 1
    }
} else {
    Write-Host 'RESULT: FAIL — no session'
    $code = 1
}

exit $code
