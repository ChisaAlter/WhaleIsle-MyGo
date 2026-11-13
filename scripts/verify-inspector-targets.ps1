# Source-backed forward Inspector matching plus affected native application checks.
param([ValidateSet('all','affected','build')][string]$Stage='all')
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$evidence=Join-Path $root 'docs/review-20261022/evidence'
$env:GOTOOLCHAIN='go1.27.2'
$env:CGO_ENABLED='0'
$env:GOOS='windows'
$env:GOARCH='amd64'
$results=@()
function Check([string]$Name,[string]$Program,[string[]]$Arguments){
 $log=Join-Path $evidence ('inspector-pass253-'+$Name+'.txt')
 $start=[datetime]::UtcNow
 & $Program @Arguments 2>&1 | Tee-Object -FilePath $log | Write-Output
 $code=$LASTEXITCODE
 $text=if(Test-Path -LiteralPath $log){Get-Content -LiteralPath $log -Raw}else{''}
 Set-Content -LiteralPath $log -Value $text -Encoding UTF8
 $script:results += [pscustomobject]@{name=$Name;program=$Program;args=$Arguments;exitCode=$code;seconds=([datetime]::UtcNow-$start).TotalSeconds}
 if($code -ne 0){throw "$Name failed with exit $code"}
}
Push-Location $root
try {
 if($Stage -in 'all','affected'){
  Check 'reference-functions' 'node' @('scripts/inspector-reference-targets.mjs')
  Check 'affected-tests' 'go' @('test','./internal/store/session','./internal/ui/chat','./internal/ui/frame','-count=1')
  Check 'affected-vet' 'go' @('vet','./internal/store/session','./internal/ui/chat','./internal/ui/frame')
 }
 if($Stage -eq 'all'){
  Check 'command-tests' 'go' @('test','./cmd/whaleisle/...','./cmd/whaleisle-preview/...','-count=1')
 }
 if($Stage -in 'all','build'){
  Check 'production-build' 'go' @('build','-o','whaleisle-pass253.exe','./cmd/whaleisle')
 }
} finally {
 Pop-Location
 $results|ConvertTo-Json -Depth 5|Set-Content -LiteralPath (Join-Path $evidence ('inspector-pass253-'+$Stage+'-checks.json')) -Encoding UTF8
}
