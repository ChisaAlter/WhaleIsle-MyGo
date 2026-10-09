# Dot-source before running go: `. .\scripts\env.ps1; go test ./...`
# Keeps the Go toolchain, module cache and build cache inside .toolchain.
$root = Split-Path -Parent $PSScriptRoot
$tc = Join-Path $root '.toolchain'
$env:GOPATH = Join-Path $tc 'gopath'
$env:GOMODCACHE = Join-Path $tc 'gopath\pkg\mod'
$env:GOCACHE = Join-Path $tc 'gocache'
$env:GOTMPDIR = Join-Path $tc 'gotmp'
$env:GOTOOLCHAIN = 'go1.27.2+auto'
$env:CGO_ENABLED = '0'
Remove-Item Env:ELECTRON_RUN_AS_NODE -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $env:GOTMPDIR | Out-Null
if (Test-Path 'C:\AI\tools\go\bin\go.exe') {
    $env:PATH = 'C:\AI\tools\go\bin;' + $env:PATH
}
