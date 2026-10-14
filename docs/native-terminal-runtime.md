# Native terminal runtime packaging

The workspace terminal uses MyGo's native libghostty-vt binding and the existing
ConPTY implementation on Windows. Its input, output, resize, process exit and
restart paths already run through that plugin. The Host's `ghostty-vt.wasm`,
`ghostty-write-pty.wasm` and Nerd Font belong to the separate web terminal; they
cannot replace the native library.

## Fixed Windows x64 input

The authoritative ABI selection is
`third_party/mygo/plugins/terminal/mygo-plugin.json`:

- Ghostty source: `befcdfd2c3a1cb24d9ec886e93c95b2b5daa7028`
- Official asset: https://github.com/egoist/mygo/releases/download/libghostty-vt-befcdfd2c3a1/libghostty-vt-windows-amd64.dll
- Packaged filename: `ghostty-vt.dll`
- SHA256: `53e8e2dd979702e8f2c48107af19be5a76efc2e88eb7cf1097218e899665da41`
- Size of the verified release asset: 2,173,952 bytes
- License: MIT, from https://raw.githubusercontent.com/ghostty-org/ghostty/befcdfd2c3a1cb24d9ec886e93c95b2b5daa7028/LICENSE

Keep the downloaded binary outside git. Obtain that exact official asset on the
build machine, then set its local path before packaging:

```powershell
$env:WHALEISLE_TERMINAL_LIBRARY = 'C:\build-inputs\ghostty-vt.dll'
.\scripts\package.ps1
```

`package.ps1` invokes the offline `tools/stage-terminal` tool. It checks that the
embedded manifest still matches the MyGo binding and verifies the input SHA256
before copying the bytes. Without an explicit input it checks
`.toolchain/natives/windows-amd64/`, `resources/windows-amd64/`, then MyGo's
hash-addressed user cache. It never downloads or loads the library.

The staged executable directory contains:

- `whaleisle.exe`
- `ghostty-vt.dll`
- `ghostty-vt.dll.LICENSE`
- `terminal-runtime.json`, with platform, official source/release URL and digest

NSIS requires and installs all three terminal sidecar files beside the EXE. They
are included even in a shell-only package with `WHALEISLE_PACKAGE_SKIP_RUNTIME=1`.
Missing or mismatched input stops packaging rather than shipping a broken terminal.

For an existing Go-build output, stage directly without running any terminal:

```powershell
go run ./tools/stage-terminal -target windows-amd64 -source $env:WHALEISLE_TERMINAL_LIBRARY -out .\dist\whaleisle
```

## Runtime

Opening the terminal checks the library at MyGo's resource/executable/cache paths
(or the existing `MYGO_GHOSTTY_VT` override) against the pinned digest, then loads
that exact path. There is no first-open network fallback. A missing library is
reported before MyGo's one-time loader runs, so after supplying the correct file
the terminal's existing Restart action can try again. A native loader failure
requires restarting the application.

This closes the artifact and loader path. It is not evidence of Windows visual,
keyboard/IME, ConPTY or native DLL execution testing; those require a Windows run.
