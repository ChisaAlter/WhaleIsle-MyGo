$ErrorActionPreference = 'Stop'
. .\scripts\env.ps1
& gofmt -w .\internal\desktop\pet\real_coordinates_windows_test.go .\internal\desktop\pet\geometry_windows.go
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& go test -c -ldflags=-H=windowsgui -o .\tmp\pet-coordinate-gui.test.exe ./internal/desktop/pet
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$env:WHALEISLE_PET_GUI = '1'
$env:WHALEISLE_PET_GUI_NEURAL = '1'
$env:WHALEISLE_PET_RENDER_ASSETS = Join-Path (Get-Location).Path 'dist\whaleisle\pet-render-assets'
$env:WHALEISLE_PET_GUI_RESULT = Join-Path (Get-Location).Path 'tmp\pet-coordinate-gui-result.json'
$petStart = [System.Diagnostics.ProcessStartInfo]::new()
$petStart.FileName = (Resolve-Path .\tmp\pet-coordinate-gui.test.exe).Path
$petStart.UseShellExecute = $false
$petStart.CreateNoWindow = $true
$petStart.Arguments = '-test.run=^TestRealPet(CoordinateContract|OverlayAndQuickChatCreation)$ -test.v -test.timeout=120s'
$petProcess = [System.Diagnostics.Process]::Start($petStart)
if (-not $petProcess.WaitForExit(120000)) { throw 'Owned pet coordinate GUI fixture did not exit' }
if ($petProcess.ExitCode -ne 0) { exit $petProcess.ExitCode }
$petEvidence = Get-Content -Raw -LiteralPath $env:WHALEISLE_PET_GUI_RESULT | ConvertFrom-Json
if (-not $petEvidence.pass -or $petEvidence.pid -ne $petProcess.Id -or -not $petEvidence.coordinateHardwareSamples) { throw 'Real coordinate measurements missing or failed' }
$petEvidence | ConvertTo-Json -Depth 10
exit 0
