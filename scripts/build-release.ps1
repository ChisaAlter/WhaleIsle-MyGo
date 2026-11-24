# Shared native production build. The diagnostic variant keeps symbols only;
# it uses the same source, tags, resources and module inputs as the release.
param(
    [Parameter(Mandatory=$true)][string]$Package,
    [Parameter(Mandatory=$true)][string]$Output,
    [string]$Version = '0.0.0-dev',
    [string]$GoExecutable = 'go',
    [switch]$KeepSymbols
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$contract = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'release-build.json') -Raw | ConvertFrom-Json
$outputPath = [IO.Path]::GetFullPath($Output)
if ($Version -notmatch '^[0-9A-Za-z.+-]+$') { throw 'Invalid build version.' }
$packagePath = [IO.Path]::GetFullPath((Join-Path $projectRoot $Package))
$cmdRoot = [IO.Path]::GetFullPath((Join-Path $projectRoot 'cmd')) + [IO.Path]::DirectorySeparatorChar
if (-not $packagePath.StartsWith($cmdRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Release package must be beneath cmd/.' }
$targetOS = (& $GoExecutable env GOOS).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve Go target OS.' }
$targetArch = (& $GoExecutable env GOARCH).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve Go target architecture.' }
$parent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/')
function Resolve-RealPath([string]$Path) {
    $full = [IO.Path]::GetFullPath($Path)
    if ([IO.Path]::DirectorySeparatorChar -eq '\') {
        if (-not ('WhaleIslePathResolver' -as [type])) {
            Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;
using Microsoft.Win32.SafeHandles;

public static class WhaleIslePathResolver {
    private const uint FILE_FLAG_BACKUP_SEMANTICS = 0x02000000;
    private const uint FILE_SHARE_ALL = 0x00000001 | 0x00000002 | 0x00000004;
    private const uint OPEN_EXISTING = 3;

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern SafeFileHandle CreateFileW(
        string lpFileName, uint dwDesiredAccess, uint dwShareMode,
        IntPtr lpSecurityAttributes, uint dwCreationDisposition,
        uint dwFlagsAndAttributes, IntPtr hTemplateFile);

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern uint GetFinalPathNameByHandleW(
        SafeFileHandle hFile, StringBuilder lpszFilePath, uint cchFilePath, uint dwFlags);

    public static string RealPath(string path) {
        using (SafeFileHandle h = CreateFileW(path, 0, FILE_SHARE_ALL, IntPtr.Zero, OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS, IntPtr.Zero)) {
            if (h.IsInvalid) return Path.GetFullPath(path);
            uint len = GetFinalPathNameByHandleW(h, null, 0, 0);
            if (len == 0) return Path.GetFullPath(path);
            StringBuilder sb = new StringBuilder((int)len + 1);
            uint got = GetFinalPathNameByHandleW(h, sb, (uint)sb.Capacity, 0);
            if (got == 0) return Path.GetFullPath(path);
            string res = sb.ToString(0, (int)got);
            if (res.StartsWith(@"\\?\UNC\", StringComparison.OrdinalIgnoreCase)) return @"\\" + res.Substring(8);
            if (res.StartsWith(@"\\?\", StringComparison.Ordinal)) return res.Substring(4);
            return res;
        }
    }
}
'@
        }
        return [WhaleIslePathResolver]::RealPath($full)
    }
    $resolved = & realpath -- $full 2>$null
    if ($LASTEXITCODE -eq 0 -and $resolved) { return $resolved.Trim() }
    return $full
}
$lockBase = [Environment]::GetFolderPath('LocalApplicationData')
if (-not $lockBase) { $lockBase = $parent }
$lockDir = Join-Path (Join-Path ([IO.Path]::GetFullPath($lockBase).TrimEnd('\', '/')) 'mygo') 'locks'
$work = Join-Path $parent ('whaleisle-release-build-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
$resourceTarget = $null
$lockStream = $null
$lockPath = $null
try {
    $flags = @()
    if (-not $KeepSymbols) { $flags += @($contract.ldflags) }
    if ((Split-Path $packagePath -Leaf) -eq 'whaleisle') { $flags += "-X main.buildVersion=$Version" }
    if ($targetOS -eq 'windows' -and $contract.windowsGUI) { $flags += '-H windowsgui' }
    $buildArgs = @('build', '-mod=readonly', '-tags', ($contract.tags -join ','))
    if ($contract.trimpath) { $buildArgs += '-trimpath' }
    if ($flags.Count) { $buildArgs += @('-ldflags', ($flags -join ' ')) }
    if ($targetOS -eq 'windows') {
        $resource = Join-Path $work 'windows.syso'
        $savedOS, $savedArch = $env:GOOS, $env:GOARCH
        try {
            $env:GOOS = (& $GoExecutable env GOHOSTOS).Trim()
            $env:GOARCH = (& $GoExecutable env GOHOSTARCH).Trim()
            & $GoExecutable tool mygo winres -arch $targetArch -version $Version -out $resource $projectRoot
        } finally {
            if ($null -eq $savedOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $savedOS }
            if ($null -eq $savedArch) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $savedArch }
        }
        if ($LASTEXITCODE -ne 0) { throw 'Windows resource generation failed.' }
        # Go's archive packer opens .syso paths directly, bypassing -overlay.
        # Lock on the resolved real package directory in a TEMP-independent
        # LocalApplicationData lock root so junction aliases and per-run TEMP
        # workspaces still serialize the physical .syso writer.
        New-Item -ItemType Directory -Force -Path $lockDir | Out-Null
        $resolvedPackagePath = Resolve-RealPath $packagePath
        $resolvedCandidate = Join-Path $resolvedPackagePath "mygo_windows_$targetArch.syso"
        $lockHash = [BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash([Text.Encoding]::UTF8.GetBytes($resolvedCandidate.ToUpperInvariant()))).Replace('-', '').ToLowerInvariant()
        $lockPath = Join-Path $lockDir "whaleisle-syso-$lockHash.lock"
        try {
            $lockStream = [IO.File]::Open($lockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
        } catch {
            throw "Another build holds the Windows resource lock for $Package; run one release build at a time."
        }
        $candidate = Join-Path $packagePath "mygo_windows_$targetArch.syso"
        $resourceTarget = $candidate
        $owned = [IO.File]::Open($candidate, [IO.FileMode]::Create, [IO.FileAccess]::Write, [IO.FileShare]::Read)
        try {
            $source = [IO.File]::OpenRead($resource)
            try { $source.CopyTo($owned) } finally { $source.Dispose() }
        } finally { $owned.Dispose() }
    }
    $buildArgs += @('-o', $outputPath, $packagePath)
    Push-Location $projectRoot
    try { & $GoExecutable @buildArgs } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }
    if (-not $KeepSymbols -and $targetOS -eq 'windows' -and $targetArch -eq 'amd64' -and (Split-Path $packagePath -Leaf) -eq 'whaleisle') {
        $size = (Get-Item -LiteralPath $outputPath).Length
        Write-Host ('Production desktop: {0} bytes ({1:N3} MiB)' -f $size, ($size / 1MB))
    }
} finally {
    try {
        if ($resourceTarget -and (Test-Path -LiteralPath $resourceTarget)) {
            Remove-Item -LiteralPath $resourceTarget -Force
        }
    } finally {
        try {
            if ($lockStream) {
                $lockStream.Dispose()
                # Delete only the lock this build held; a build that re-acquired it
                # between dispose and delete keeps its file (its own cleanup runs).
                Remove-Item -LiteralPath $lockPath -Force -ErrorAction SilentlyContinue
            }
        } finally {
            if ([IO.Path]::GetFullPath((Split-Path $work -Parent)) -ne $parent -or (Split-Path $work -Leaf) -notmatch '^whaleisle-release-build-[a-f0-9]{32}$') { throw 'Unexpected release build workspace.' }
            # Only ordinary files written above exist here. Never clean package sources.
            foreach ($name in @('windows.syso')) {
                $file = Join-Path $work $name
                if (Test-Path -LiteralPath $file) { Remove-Item -LiteralPath $file -Force }
            }
            if (Test-Path -LiteralPath $work) { Remove-Item -LiteralPath $work -Force }
        }
    }
}
