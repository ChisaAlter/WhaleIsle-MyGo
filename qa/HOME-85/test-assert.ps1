#Requires -Version 5.1
<#
Window-free proof that the copy assertions can fail.

A row that can only pass is worse than no row: it reports a green tree and a
regression walks through. This runs the same Test-PoliteWindowText the live ops
use, against trees built here, and requires every planted wrong tree to be
rejected and every right one to be accepted.

    pwsh -NoProfile -File qa/HOME-85/test-assert.ps1

Exit 0 only when all cases agree with expectation.
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
# run.ps1 is dot-sourced for its tree walk (Get-PoliteNodes); it loads as a
# library and runs nothing, because dot-sourcing is what marks it as one.
. (Join-Path $PSScriptRoot '..\..\tools\polite\run.ps1')
. (Join-Path $PSScriptRoot 'ops-assert.ps1')

function New-FakeTree {
    param([Parameter(Mandatory)][string[]]$Texts)
    $root = [pscustomobject]@{ role = 'window'; name = '鲸屿 · MyGo'; children = @() }
    $children = New-Object System.Collections.Generic.List[object]
    foreach ($text in $Texts) {
        $children.Add([pscustomobject]@{ role = 'text'; name = $text; children = @() })
    }
    $root.children = $children.ToArray()
    [pscustomobject]@{ tree = $root; ref_count = $children.Count }
}

# A tree that carries the copy the merged fix paints for the operator's report.
$rightZH = New-FakeTree -Texts @(
    '启动加载页',
    '鲸屿 · MyGo',
    '还没有选择工作区。选择要使用的文件夹后，本地服务会自动启动。',
    '选择工作区',
    '下载日志'
)
$rightEN = New-FakeTree -Texts @(
    'Startup loading page',
    'No workspace is selected yet. Choose a folder and the local service starts on its own.',
    'Choose folder'
)
$rightMissing = New-FakeTree -Texts @('桌面端启动失败', '桌面运行时文件缺失，请重新安装桌面端后重试；详情见日志。', '重试')
$rightMismatch = New-FakeTree -Texts @('桌面端启动失败', '桌面运行时与当前版本不匹配，请重新安装桌面端后重试；详情见日志。', '重试')
$rightPort = New-FakeTree -Texts @('桌面端启动失败', '启动所需端口被其他程序占用，请关闭占用程序后重试。', '重试')
$rightStarting = New-FakeTree -Texts @('正在启动本地服务，请稍候。', '鲸屿 · MyGo')

# The page the operator reported, and other regressions the table must catch.
$oldDeadStrip = New-FakeTree -Texts @('Host 尚未启动', '启动 Host', '连接后显示会话', '选择一个工作区开始')
$stoppedOnly = New-FakeTree -Texts @('已停止')
$genericOperation = New-FakeTree -Texts @('桌面端启动失败', '操作没有完成。请重试；若仍然失败，请查看日志。')
$wrongFamily = New-FakeTree -Texts @('桌面端启动失败', '启动所需端口被其他程序占用，请关闭占用程序后重试。')

$cases = @(
    @{ Name = 'right zh no-workspace';    Tree = $rightZH;       Scenario = 'no-workspace-zh';   ShouldPass = $true }
    @{ Name = 'right en no-workspace';    Tree = $rightEN;       Scenario = 'no-workspace-en';   ShouldPass = $true }
    @{ Name = 'right missing runtime';    Tree = $rightMissing;  Scenario = 'missing-runtime-zh'; ShouldPass = $true }
    @{ Name = 'right mismatch';           Tree = $rightMismatch; Scenario = 'mismatch-zh';       ShouldPass = $true }
    @{ Name = 'right port taken';         Tree = $rightPort;     Scenario = 'port-taken-zh';     ShouldPass = $true }
    @{ Name = 'right starting';           Tree = $rightStarting; Scenario = 'starting-zh';       ShouldPass = $true }
    @{ Name = 'planted dead strip';       Tree = $oldDeadStrip;  Scenario = 'no-workspace-zh';   ShouldPass = $false }
    @{ Name = 'planted stopped only';     Tree = $stoppedOnly;   Scenario = 'no-workspace-zh';   ShouldPass = $false }
    @{ Name = 'planted generic failure';  Tree = $genericOperation; Scenario = 'mismatch-zh';    ShouldPass = $false }
    @{ Name = 'planted wrong family';     Tree = $wrongFamily;   Scenario = 'mismatch-zh';       ShouldPass = $false }
)

$failures = 0
foreach ($case in $cases) {
    $lines = Get-PoliteNodeLines -Snapshot $case.Tree
    $accepted = $true
    $reason = ''
    try {
        Test-PoliteWindowText -Lines $lines -Scenario $case.Scenario | Out-Null
    } catch {
        $accepted = $false
        $reason = "$_"
    }
    $agrees = ($accepted -eq $case.ShouldPass)
    if (-not $agrees) { $failures++ }
    $verdict = if ($agrees) { 'PASS' } else { 'FAIL' }
    $detail = if ($reason) { " ($reason)" } else { '' }
    Write-Host ("case {0,-24}: asserted={1} expected={2} {3}{4}" -f $case.Name, $(if ($accepted) { 'accepted' } else { 'rejected' }), $(if ($case.ShouldPass) { 'accepted' } else { 'rejected' }), $verdict, $detail)
}

if ($failures -gt 0) {
    Write-Host "RESULT: FAIL ($failures case(s) disagreed)"
    exit 1
}
Write-Host "RESULT: PASS ($($cases.Count) cases)"
exit 0
