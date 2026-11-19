#Requires -Version 5.1
<#
One live assertion of the startup page's copy. Dot-sourced by tools/polite/run.ps1
as -Ops; the scenario comes from $env:POLITE_SCENARIO and the text it expects from
the table below, so the Chinese never travels through a command line.
#>
$script:PoliteExpectations = @{
    'no-workspace-zh' = @{
        Want  = @('还没有选择工作区', '选择工作区')
        Avoid = @('Host 尚未启动', 'Host 未运行', '选择一个工作区开始', '连接后显示会话')
    }
    'no-workspace-en' = @{
        Want  = @('No workspace is selected yet', 'Choose folder')
        Avoid = @('Host is not running', 'Connect to load sessions')
    }
    'starting-zh' = @{
        Want  = @('正在启动本地服务', '正在打开工作区')
        Avoid = @('Host 尚未启动', '本地服务没有运行')
    }
    'missing-runtime-zh' = @{
        Want  = @('桌面运行时文件缺失')
        Avoid = @('Host 尚未启动', '操作没有完成')
    }
    'mismatch-zh' = @{
        Want  = @('桌面运行时与当前版本不匹配')
        Avoid = @('Host 尚未启动', '操作没有完成')
    }
    'port-taken-zh' = @{
        Want  = @('端口被其他程序占用')
        Avoid = @('Host 尚未启动', '操作没有完成')
    }
    'any' = @{ Want = @(); Avoid = @() }
}

function Get-PoliteNodeLines {
    param([Parameter(Mandatory)]$Snapshot)
    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($node in (Get-PoliteNodes -Snapshot $Snapshot)) {
        foreach ($field in @('role', 'name', 'value', 'description', 'helptext')) {
            $value = [string]$node.$field
            if ($value) { $lines.Add(("{0}:{1}" -f $field, $value)) }
        }
    }
    $lines.ToArray()
}

function Invoke-PoliteOps {
    param($Session)

    if ($env:POLITE_SETTLE) {
        Start-Sleep -Seconds ([int]$env:POLITE_SETTLE)
    }

    $shot = Invoke-PoliteStep -Session $Session -Name 'snapshot' -Action {
        Read-PoliteSnapshot -Session $Session
    }

    $scenario = 'any'
    if ($env:POLITE_SCENARIO) { $scenario = $env:POLITE_SCENARIO }
    if (-not $script:PoliteExpectations.ContainsKey($scenario)) {
        throw "no expectation table for scenario '$scenario'"
    }
    $expect = $script:PoliteExpectations[$scenario]
    $lines = Get-PoliteNodeLines -Snapshot $shot
    $joined = $lines -join "`n"
    Write-Host "--- window text ($scenario, $($shot.ref_count) refs) ---"
    foreach ($line in $lines) { Write-Host $line }
    Write-Host '--- end window text ---'

    foreach ($want in @($expect.Want)) {
        if (-not $joined.Contains($want)) {
            throw "[$scenario] the window does not say '$want'"
        }
        Write-Host "[$scenario] says '$want'"
    }
    foreach ($avoid in @($expect.Avoid)) {
        if ($joined.Contains($avoid)) {
            throw "[$scenario] the window still says '$avoid'"
        }
    }
    Write-Host "[$scenario] ok"
}
