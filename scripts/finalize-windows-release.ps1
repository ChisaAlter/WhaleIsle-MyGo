# Finalize the exact installer bytes consumed by the updater. A supplied signing
# script receives -FilePath; private keys stay in the operator's signing service.
param(
    [Parameter(Mandatory=$true)][string]$Installer,
    [string]$SigningScript = $env:WHALEISLE_SIGNING_SCRIPT,
    [string]$ExpectedPublisher = $env:WHALEISLE_UPDATE_PUBLISHER,
    [switch]$RequireSignature = (([string]$env:WHALEISLE_UPDATE_REQUIRE_SIGNATURE).Trim() -notin @('', '0', 'false', 'no')),
    [switch]$Development
)
$ErrorActionPreference = 'Stop'
$installerPath = (Get-Item -LiteralPath $Installer -ErrorAction Stop).FullName
if ($SigningScript) {
    $signer = (Get-Item -LiteralPath $SigningScript -ErrorAction Stop).FullName
    $global:LASTEXITCODE = 0
    & $signer -FilePath $installerPath
    if ($LASTEXITCODE -ne 0) { throw "Installer signing failed (exit $LASTEXITCODE); candidate retained: $installerPath" }
}
# Unsigned Windows releases are the default. A requested signer, publisher or
# strict policy still requires a valid final signature before metadata is made.
$signatureRequired = $RequireSignature -or [bool]$ExpectedPublisher -or ([bool]$SigningScript -and -not $Development)
if ($signatureRequired) {
    $signature = Get-AuthenticodeSignature -LiteralPath $installerPath
    if ($signature.Status -ne 'Valid') {
        throw "Release finalization requires a valid Authenticode signature ($($signature.Status)). Supply WHALEISLE_SIGNING_SCRIPT or sign this retained installer, then rerun finalize-windows-release.ps1: $installerPath"
    }
    if ($ExpectedPublisher -and $signature.SignerCertificate.Subject.IndexOf($ExpectedPublisher, [StringComparison]::OrdinalIgnoreCase) -lt 0) {
        throw 'Installer publisher does not match WHALEISLE_UPDATE_PUBLISHER.'
    }
}
$stream = [IO.File]::OpenRead($installerPath)
$fileHash = [Security.Cryptography.SHA256]::Create()
$blockHash = [Security.Cryptography.SHA256]::Create()
$blocks = [Collections.Generic.List[object]]::new()
try {
    $buffer = New-Object byte[] 65536
    [int64]$offset = 0
    while (($count = $stream.Read($buffer, 0, $buffer.Length)) -gt 0) {
        $digest = $blockHash.ComputeHash($buffer, 0, $count)
        $null = $fileHash.TransformBlock($buffer, 0, $count, $null, 0)
        $blocks.Add([ordered]@{offset=$offset;size=[int64]$count;sha256=([BitConverter]::ToString($digest) -replace '-', '').ToLowerInvariant()})
        $offset += $count
    }
    $null = $fileHash.TransformFinalBlock([byte[]]@(), 0, 0)
    $sum = ([BitConverter]::ToString($fileHash.Hash) -replace '-', '').ToLowerInvariant()
} finally {
    $stream.Dispose(); $fileHash.Dispose(); $blockHash.Dispose()
}
$blockmapPath = "$installerPath.blockmap.json"
$blockmap = [ordered]@{fileSize=$offset;blocks=@($blocks.ToArray())} | ConvertTo-Json -Depth 4
[IO.File]::WriteAllText($blockmapPath, $blockmap + [Environment]::NewLine)
$sumPath = Join-Path (Split-Path $installerPath -Parent) 'SHA256SUMS.txt'
[IO.File]::WriteAllText($sumPath, "$sum  $(Split-Path $installerPath -Leaf)" + [Environment]::NewLine)
Write-Host "Finalized $(if ($signatureRequired) { 'signed release' } elseif ($Development) { 'development' } else { 'unsigned release' }) installer: $installerPath"
