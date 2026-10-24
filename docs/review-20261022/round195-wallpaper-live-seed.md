# Round 195 — live wallpaper seed and the 549×460 startup dialog

Date: 2026-10-25. Task: task-76. Two real defects found and fixed; the seeded live capture now shows the wallpaper. No commit, no passN exe (temp build only), native app/profile untouched, no `runtime/` or `third_party/` edits, temp profile deleted.

## 1. The 549×460 window is the config-read-failure dialog (root cause)

Reproduced with pass287 and a seeded profile whose `config.json` was written by PowerShell 5.1 (`Set-Content -Encoding UTF8` adds a UTF-8 BOM):

- visible window 549×460 px (366×307 DIP), captured: `evidence/screens/wallpaper-bom-dialog.png`
- title **配置读取失败**, body: `config: C:\AI\WhaleIsle-MyGo\…\config.json: invalid character '\ufeff' looking for beginning of value` + "The original file is preserved. Back up and repair config.json, then restart. / 原文件已保留，请备份并修复 config.json 后重启。"
- the process then quits (main.go's `config.Open` failure path: warning dialog → `mygo.App.Quit()`), which matches round189's "exited after ~20 s, no logs/go-resume/last-desktop-start".

**Root cause = the fixture writer, not the wallpaper path**: `encoding/json` rejects a leading BOM. Fixed in `internal/config/config.go:246-251` — the parse strips a UTF-8 BOM (`bytes.TrimPrefix`) while `baseline` still holds the original bytes, so the next save is unchanged. Test: `internal/config/bom_test.go` `TestOpenToleratesUTF8BOM` (theme + `goWallpaperPath` load, file bytes untouched).

## 2. Second real defect: an empty Host appearance dropped the seeded wallpaper

With the BOM fixed the profile booted, but the window painted the theme's aurora effect instead of the seeded image. Cause: `cmd/whaleisle/wallpaper_background.go` `wallpaperPainter` replaced the config path with `s.hostWallpaperImage` whenever the Host appearance was accepted — and a profile that never set a Host wallpaper has `wallpaperImage: ""`, so the seeded `goWallpaperPath` was silently discarded once the harness connected.

Fix: `wallpaperSourcePath(hostAccepted, hostImage, configPath)` — the Host value wins only when it is non-empty; otherwise the config path stays in force. Test: `cmd/whaleisle/wallpaper_background_test.go` `TestWallpaperSourcePath` (7 cases: host not accepted, empty/whitespace Host, Host data URL wins, Host only, nothing set, trimmed config path).

## 3. Seeded fixture recipe (working)

Throwaway `WHALEISLE_USER_DATA` profile, `config.json` **written without a BOM** (`[System.IO.File]::WriteAllText(path, json, New-Object System.Text.UTF8Encoding($false))`):

```json
{
  "theme": "deepseek.dark", "locale": "zh", "host": "127.0.0.1", "port": 3099,
  "workspace": "<absolute temp dir>",
  "goWallpaperPath": "<absolute local PNG, or the native data:image/… value>",
  "autoStartDesktop": true, "goPreviewNoticeAck": "2026-09-28.1",
  "goUnifiedShortcuts": { "schemaVersion": 2, "profiles": { "desktop:windows": {} } }
}
```

plus the empty `welcome-shown` marker. Start with `WHALEISLE_USER_DATA=<profile>`, wait ~45-50 s for the harness hand-off, then capture. Note the app rewrites `config.json` in its own full schema on first run, so seed before the first launch (the rewritten file keeps `goWallpaperPath`).

## 4. Live evidence

`evidence/screens/wallpaper-seeded-live-1440x860.png` — temp build (`go build` from this tree, not a passN exe), 2160×1290 px @144 dpi = **1440×860 DIP**, dark: the seeded image's three discs (blue, purple, green) are visible through the shell instead of the aurora gradient. Earlier capture with the bug (`wallpaper-seeded-live-1440x860.png` overwritten) showed the aurora effect in the same fixture.

## 5. `wallpaperBlur` / `wallpaperPixelate` / `transparentTheme`

Not applied by MyGo's window backdrop: the painter receives only the image path plus the locally stored `wallpapers.Preferences` (`cmd/whaleisle/wallpaper_background.go:80-104`), and the config has no fields for these. They live in the `ui-theme` namespace (`theme-settings.js` fields `wallpaperBlur`, `wallpaperPixelate`, `transparentTheme`) and reach MyGo only via `HostSettings` → the settings UI, not the backdrop. Recorded as a theme-owner follow-up (implementing them blindly without the Host page's own compositing would not match native).

## 6. Residual

- The wallpaper is painted by the **native shell** behind the page; the native app instead injects `#dsh-wallpaper` *into* the harness page and makes it transparent, so a Host-page-drawn wallpaper (and page-level blur/pixelate) is still a structural difference — the live capture shows the native-drawn image, not a page-level one.
- Host-side blur/pixelate/transparentTheme plumbing (theme-owner).
- A BOM'd config now loads; a config with *other* corruption still shows the same dialog (intended).

## 7. Gates

`go test ./cmd/whaleisle ./internal/config ./internal/ui/theme -count=1` → ok (19.6 s / 2.1 s / 1.3 s); `go vet` on the same three packages → clean.

Files: `internal/config/config.go`, `internal/config/bom_test.go` (new), `cmd/whaleisle/wallpaper_background.go`, `cmd/whaleisle/wallpaper_background_test.go` (new), `docs/review-20261022/round195-wallpaper-live-seed.md`, `docs/review-20261022/evidence/screens/wallpaper-bom-dialog.png`, `docs/review-20261022/evidence/screens/wallpaper-seeded-live-1440x860.png`.

## 追加（round220）· components 包的非 Windows 交叉编译修复

`internal/desktop/components` 在 `GOOS=linux/darwin` 下编译失败：`alerts_process_other.go` 与 `process_other.go` 只 import 了 `"fmt"`，函数体却用了 `errors.New(...)` 与 `storelocale.T(...)`，报 `"fmt" imported and not used` + `undefined: errors / storelocale`。两个文件是同名 Windows 实现的 `//go:build !windows` 半边，Windows 半边（`alerts_process_windows.go`、`process_windows.go`）编译正常，所以是重构时漏掉了导入。

修复（只动构建标签半边，不改行为）：两个文件的 import 换成 `"errors"` 与 `"whaleisle/internal/store/storelocale"`（后者与包内其它文件一致，如 `process_windows.go:14`），并补一行注释说明与 Windows 半边相同的契约——本平台无法校验进程身份/管理进程，返回本地化的「不支持」错误、进程为 nil，调用方按 unsupported 处理而不是信任。

验证：
- `GOOS=linux go build ./internal/desktop/components` → 0；`GOOS=darwin` → 0；`go build`（windows）→ 0。
- 更大范围扫描：`GOOS=linux go build ./internal/desktop/...` → 0，`GOOS=darwin go build ./internal/desktop/...` → 0（无其它同类破损的构建标签文件）。
- `go test ./internal/desktop/components` → ok 2.4 s；`go test ./cmd/whaleisle -run Component` → ok 2.8 s；`go vet ./internal/desktop/components` → 0。

残留：本轮只修「明显编译不过」的构建标签文件，没有对 linux/darwin 的运行期行为做审计（这些 `_other` 半边本身是明确的 unsupported 桩）。未提交、未构建 passN、未改 runtime/vendor 与 third_party。