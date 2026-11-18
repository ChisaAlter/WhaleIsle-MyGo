# Dot-source before running go: `. .\scripts\env.ps1; go test ./...`
# Use Go's user-level caches, shared with other projects. Do not override an
# explicitly isolated GOPATH/GOMODCACHE/GOCACHE/GOTMPDIR/PSModuleAnalysisCachePath
# supplied by the caller.
# Go owns and removes its ordinary temporary build directories under system TEMP.
$env:GOTOOLCHAIN = 'go1.27.2+auto'
$env:CGO_ENABLED = '0'
Remove-Item Env:ELECTRON_RUN_AS_NODE -ErrorAction SilentlyContinue
if ($env:GOTMPDIR) { New-Item -ItemType Directory -Force $env:GOTMPDIR | Out-Null }
if (-not $env:PSModuleAnalysisCachePath) {
    $psCacheRoot = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { [IO.Path]::GetTempPath().TrimEnd('\', '/') }
    $env:PSModuleAnalysisCachePath = Join-Path $psCacheRoot 'Microsoft\Windows\PowerShell\ModuleAnalysisCache'
}
if ($env:WHALEISLE_GO_BIN -and (Test-Path (Join-Path $env:WHALEISLE_GO_BIN 'go.exe'))) {
    $env:PATH = $env:WHALEISLE_GO_BIN + [IO.Path]::PathSeparator + $env:PATH
}
