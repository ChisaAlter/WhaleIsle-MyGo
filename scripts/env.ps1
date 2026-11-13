# Dot-source before running go: `. .\scripts\env.ps1; go test ./...`
# Use Go's user-level caches, shared with other projects. Do not override an
# explicitly isolated GOPATH/GOMODCACHE/GOCACHE/GOTMPDIR supplied by the caller.
# Go owns and removes its ordinary temporary build directories under system TEMP.
$env:GOTOOLCHAIN = 'go1.27.2+auto'
$env:CGO_ENABLED = '0'
Remove-Item Env:ELECTRON_RUN_AS_NODE -ErrorAction SilentlyContinue
if ($env:GOTMPDIR) { New-Item -ItemType Directory -Force $env:GOTMPDIR | Out-Null }
if (Test-Path 'C:\AI\tools\go\bin\go.exe') {
    $env:PATH = 'C:\AI\tools\go\bin;' + $env:PATH
}
