# Verify the Windows preview ownership regression with an explicitly pinned
# Go toolchain. Logs are local evidence, not executable/browser profile data.
param([string]$Toolchain = 'go1.27.2')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$module = Join-Path $root 'third_party/mygo'
$evidence = Join-Path $root 'docs/review-20261022/evidence'
New-Item -ItemType Directory -Force -Path $evidence | Out-Null
$env:GOTOOLCHAIN = $Toolchain
$env:CGO_ENABLED = '0'
$results = @()
function Check([string]$Name,[string[]]$Arguments,[string]$Platform='windows',[string]$Architecture='amd64',[bool]$Native=$false) {
    $env:GOOS = $Platform
    $env:GOARCH = $Architecture
    $env:MYGO_PREVIEW_NATIVE_TEST = $(if($Native){'1'}else{''})
    $log = Join-Path $evidence ('preview-pass252-' + $Name + '.txt')
    $started = [datetime]::UtcNow
    Push-Location $module
    try {
        & go @Arguments 2>&1 | Tee-Object -FilePath $log | Write-Output
        $code = $LASTEXITCODE
        # Windows PowerShell Tee-Object writes UTF-16. Keep evidence readable
        # by the repository's UTF-8 text tools, including silent success logs.
        $text = if(Test-Path -LiteralPath $log){Get-Content -LiteralPath $log -Raw}else{''}
        Set-Content -LiteralPath $log -Value $text -Encoding UTF8
    } finally { Pop-Location }
    $script:results += [pscustomobject]@{name=$Name;args=$Arguments;goos=$Platform;goarch=$Architecture;native=$Native;exitCode=$code;seconds=([datetime]::UtcNow-$started).TotalSeconds;log=[IO.Path]::GetFileName($log)}
    if($code -ne 0){throw "$Name failed with exit $code"}
}
try {
    Check 'native-and-filesystem' @('test','./internal/windows','-run','^TestPreview','-count=1','-v','-timeout=100s') 'windows' 'amd64' $true
    Check 'windows-package' @('test','./internal/windows','-count=1')
    Check 'windows-build' @('build','./...')
    Check 'windows-vet' @('vet','./...')
    Check 'production-vet' @('vet','-tags','mygo_noinspector','./...')
    Check 'arm64-build' @('build','./...') 'windows' 'arm64'
    Check 'linux-build' @('build','./...') 'linux' 'amd64'
} finally {
    $results | ConvertTo-Json -Depth 5 | Set-Content -Encoding UTF8 (Join-Path $evidence 'preview-pass252-toolkit-checks.json')
}
