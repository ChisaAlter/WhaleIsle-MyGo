$ErrorActionPreference = 'Stop'
. .\scripts\env.ps1
$petCoordinateFiles = @('geometry.go','geometry_windows_test.go','motion.go','motion_state.go','interaction.go','interaction_physics.go','display_policy_windows_test.go','display_center_test.go','coordinate_contract_test.go')
foreach ($petCoordinateFile in $petCoordinateFiles) { & gofmt -w (Join-Path '.\internal\desktop\pet' $petCoordinateFile); if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE } }
$env:WHALEISLE_PET_POLICY_RESULT = Join-Path (Get-Location).Path 'tmp\pet-policy-measurement.json'
& node --test .\internal\desktop\pet\testdata\reference-centre.test.mjs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& go test -count=1 -v -run '^Test(MeasurePetDisplayPolicies|Coordinate.*|DisplayCentre.*|PhysicalDisplay.*)$' ./internal/desktop/pet
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$env:WHALEISLE_PET_RENDER_ASSETS = Join-Path (Get-Location).Path 'dist\whaleisle\pet-render-assets'
& go test -count=1 ./internal/desktop/pet ./cmd/whaleisle
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& node --test .\internal\desktop\pet\testdata\renderer.test.mjs
exit $LASTEXITCODE
