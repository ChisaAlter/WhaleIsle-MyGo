$ErrorActionPreference = 'Stop'
. .\scripts\env.ps1
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
$taskGo = 'C:\AI\WhaleIsle-MyGo\.toolchain\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.2.windows-amd64\bin\go.exe'
& 'C:\AI\WhaleIsle-MyGo\runtime\node\node.exe' .\cmd\whaleisle\plugin_session_investigation.cjs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $taskGo test ./internal/desktop/pluginui -run '^Test(BridgeActions|SessionWire|SessionOutlet)' -count=1 -v
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $taskGo test ./cmd/whaleisle -run '^TestPluginSession(Identity|Hidden|Switch)' -count=1 -v
exit $LASTEXITCODE
