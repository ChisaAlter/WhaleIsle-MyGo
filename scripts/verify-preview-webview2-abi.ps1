# Pure SDK source check; does not start WebView2 or create an AppContainer.
# If HeaderPath is omitted, obtain the pinned official NuGet SDK as data only.
param([string]$HeaderPath)
$ErrorActionPreference = "Stop"
$temp = $null
try {
    if (-not $HeaderPath) {
        $temp = Join-Path ([IO.Path]::GetTempPath()) ("whaleisle-webview2-sdk-" + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $temp | Out-Null
        $version = "1.0.3537.50"
        $uri = "https://api.nuget.org/v3-flatcontainer/microsoft.web.webview2/$version/microsoft.web.webview2.$version.nupkg"
        $zip = Join-Path $temp "sdk.zip"
        Invoke-WebRequest -Uri $uri -OutFile $zip
        Expand-Archive -Path $zip -DestinationPath (Join-Path $temp "sdk")
        $HeaderPath = Join-Path $temp "sdk/build/native/include/WebView2.h"
    }
    $header = [IO.File]::ReadAllText((Resolve-Path $HeaderPath))
    $table = [regex]::Match($header, '(?s)typedef\s+struct\s+ICoreWebView2Vtbl\s*\{(.*?)\}\s*ICoreWebView2Vtbl\s*;')
    if (-not $table.Success) { throw "ICoreWebView2Vtbl not found in SDK header" }
    $members = [regex]::Matches($table.Groups[1].Value, '\(\s*STDMETHODCALLTYPE\s*\*\s*(\w+)\s*\)')
    $names = @($members | ForEach-Object { $_.Groups[1].Value })
    if ($names.Count -lt 39 -or $names[0] -ne "QueryInterface" -or $names[1] -ne "AddRef" -or $names[2] -ne "Release") {
        throw "Unexpected SDK vtable parse; refusing to infer ABI"
    }
    $slot = [Array]::IndexOf($names, "get_BrowserProcessId")
    if ($slot -ne 37 -or $names[36] -ne "CallDevToolsProtocolMethod" -or $names[38] -ne "get_CanGoBack") {
        throw "SDK BrowserProcessId ABI does not match binding"
    }
    $binding = Join-Path $PSScriptRoot "../third_party/mygo/internal/windows/preview_process.go"
    if ([IO.File]::ReadAllText((Resolve-Path $binding)) -notmatch 'const\s+previewBrowserPIDSlot\s*=\s*37\b') {
        throw "Binding slot differs from SDK check"
    }
    $hash = (Get-FileHash -Algorithm SHA256 -Path $HeaderPath).Hash
    Write-Output "Verified official SDK ICoreWebView2::get_BrowserProcessId slot=$slot; header SHA256=$hash"
} finally {
    if ($temp -and (Test-Path $temp)) { Remove-Item -Recurse -Force $temp }
}
