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

function Unquote-RDNValue {
    param([string]$Value)
    $s = $Value.Trim()
    if ($s.Length -ge 2 -and $s.StartsWith('"') -and $s.EndsWith('"')) {
        $s = $s.Substring(1, $s.Length - 2)
    }
    $sb = New-Object System.Text.StringBuilder $s.Length
    for ($i = 0; $i -lt $s.Length; $i++) {
        if ($s[$i] -eq '\' -and ($i + 1) -lt $s.Length) {
            $i++
            [void]$sb.Append($s[$i])
            continue
        }
        [void]$sb.Append($s[$i])
    }
    return $sb.ToString().Trim()
}

function Split-RDNToken {
    param([string]$Token)
    $eq = $Token.IndexOf('=')
    if ($eq -le 0) {
        return @{ HasKey = $false; Key = ''; Val = (Unquote-RDNValue $Token) }
    }
    $key = $Token.Substring(0, $eq).Trim()
    if ($key -notmatch '^[A-Za-z0-9.]+$') {
        return @{ HasKey = $false; Key = ''; Val = (Unquote-RDNValue $Token) }
    }
    $val = Unquote-RDNValue $Token.Substring($eq + 1)
    return @{ HasKey = $true; Key = $key.ToUpperInvariant(); Val = $val }
}

function Parse-PublisherTokens {
    param([string]$Raw)
    $parts = New-Object System.Collections.Generic.List[string]
    $sb = New-Object System.Text.StringBuilder
    $inQuotes = $false
    $escaped = $false
    for ($i = 0; $i -lt $Raw.Length; $i++) {
        $c = $Raw[$i]
        if ($escaped) {
            [void]$sb.Append('\')
            [void]$sb.Append($c)
            $escaped = $false
            continue
        }
        if ($c -eq '\') {
            $escaped = $true
            continue
        }
        if ($c -eq '"') {
            $inQuotes = -not $inQuotes
            [void]$sb.Append($c)
            continue
        }
        if (-not $inQuotes -and ($c -eq ',' -or $c -eq ';')) {
            $parts.Add($sb.ToString())
            $sb.Length = 0
            continue
        }
        [void]$sb.Append($c)
    }
    if ($escaped) {
        [void]$sb.Append('\')
    }
    $parts.Add($sb.ToString())
    $out = New-Object System.Collections.Generic.List[object]
    foreach ($part in $parts) {
        $trimmed = $part.Trim()
        if ($trimmed -eq '') { continue }
        $out.Add((Split-RDNToken $trimmed))
    }
    return ,$out.ToArray()
}

function Test-PublisherMatches {
    param([string]$Publisher, [string]$Expected)
    $pub = if ($null -eq $Publisher) { '' } else { $Publisher.Trim() }
    $exp = if ($null -eq $Expected) { '' } else { $Expected.Trim() }
    if ($pub -eq '' -or $exp -eq '') { return $false }
    $tokens = Parse-PublisherTokens $pub
    $expTok = Split-RDNToken $exp
    if ($expTok.HasKey) {
        foreach ($tok in $tokens) {
            if ($tok.Key -eq $expTok.Key -and [string]::Equals($tok.Val, $expTok.Val, [StringComparison]::OrdinalIgnoreCase)) {
                return $true
            }
        }
        return $false
    }
    $allowedKeys = @('', 'CN', 'O', 'AUTHORITY', 'TEAMID')
    foreach ($tok in $tokens) {
        if ($allowedKeys -contains $tok.Key -and [string]::Equals($tok.Val, $exp, [StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }
    return $false
}

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
    if ($ExpectedPublisher -and -not (Test-PublisherMatches -Publisher $signature.SignerCertificate.Subject -Expected $ExpectedPublisher)) {
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
