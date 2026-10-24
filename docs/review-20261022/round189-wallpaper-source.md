# Round 189 — native wallpaper storage and the MyGo seeding recipe

Date: 2026-10-25. Task: task-72. Scope read-only: native app and its profile were not modified; no commit, no passN exe, no `runtime/` or vendor edits. Temp profile under `.devdata\wallpaper72` (deleted at the end; nothing committed).

## 1. Storage trace (where the native wallpaper lives)

| step | evidence |
| --- | --- |
| the harness theme plugin owns the value | `runtime/vendor/deepseek-harness/packages/client/ui-theme/lib/types/theme-settings.js:11` — `export const THEME_SETTINGS_NAMESPACE = 'ui-theme'`; file header: "Theme preferences stored in the Host user-settings document" |
| field name and shape | `theme-settings.js:29` — `THEME_WALLPAPER_IMAGE_FIELD = 'wallpaperImage'`, documented as "Field carrying the wallpaper data URL"; `lib/types/index.js:18` marks it `volatile()` |
| only data URLs are accepted | `lib/types/client/index.js:384-386` — `patch.wallpaperImage === '' \|\| isWallpaperDataUrl(...) ? patch : ''` (a path or a gallery URL is rejected, so a stored value is always `data:image/…`) |
| the writer | `lib/types/client/index.js:95-112` — `queueWrite(field, value)` debounces 300 ms, then for each pending `[field, value]` calls `this.host.set(field, value)`; `:441-442` queues `THEME_WALLPAPER_IMAGE_FIELD` on every patch |
| where it is persisted | the settings service edits one document per config entry through `configEditor`: `packages/settings/settings/lib/index.js:326` (`inject = ["configEditor", "profileContext"]`), `:456-463` (`documentPath` = `ownerContext.configEditor.documentPath`), and the legacy per-profile `settings.yaml` migration at `:350` (`join(profile.home, "settings.yaml")`). The wallpaper therefore lives in the profile's settings document (the `ui-theme` entry section), not in a separate database row |
| how the shell renders it | `reference/electron/src/main/harness-chrome-inject.js:164` injects the `#dsh-wallpaper` layer; `packages/client/ui-theme/lib/types/appearance-apply.js:59-77` turns `extras.wallpaperImage` into that layer; `reference/electron/src/main/ipc.js:284-290` exposes the gallery (`shell:list-wallpaper-catalog` / `shell:download-wallpaper`, HARNESS_ONLY) that fills the picker, but the picked bytes end up as the same data URL |

Consequence: the native value is **not a path** — it is a `data:image/...` data URL inside the `ui-theme` settings namespace, written over the harness settings RPC after the harness is up.

## 2. MyGo side

- Host read: `internal/ui/settings/host_theme.go:33` (`BindHostTheme`) → `:44-59` snapshot reads `s.Namespaces[0].Value` through `theme.ReadHostSettings` (`internal/ui/theme/host_preferences.go:48`), which yields `HostSettings.WallpaperImage` (the same data URL).
- Window backdrop: **config only** — `config.Config.WallpaperPath` (`json:"goWallpaperPath"`, `internal/config/config.go:150-152`) → `theme.WindowWallpaperPath` (`internal/ui/theme/wallpaper.go:19`) → `chrome.WallpaperBackdrop` (`internal/ui/chrome/wallpaper.go:28`) painted in the main window root.
- The loader takes a file **or** a data URL: `chrome/wallpaperImageBytes` (`internal/ui/chrome/wallpaper.go:67-70`) reads the file unless the value starts with `data:`, in which case it decodes base64/percent-encoded payloads.
- Gap (by design, documented rather than "fixed"): the Host `wallpaperImage` never reaches the window backdrop — the backdrop is painted by the shell before the harness RPC exists, and the config field is the pre-harness source. A Host value can still be used verbatim because the loader accepts data URLs, so no code path needs to change for acceptance.

## 3. Seeding recipe (verified)

```
<profile>/config.json:
{
  "theme": "deepseek.dark",
  "host": "127.0.0.1",
  "port": 3099,                        // never 3080: the dev GUI owns it
  "workspace": "<absolute temp workspace dir>",
  "goWallpaperPath": "<absolute local png>"   // or the native data:image/… value
}
```

- `goWallpaperPath` accepts any local PNG/JPEG/WebP/GIF (and, per §2, a `data:` URL copied from the native profile — read it locally, never commit it).
- Verification: a real shell-painter render with this fixture, `docs/review-20261022/evidence/screens/wallpaper-seeded-render-1440x860.png` — dark window, the seeded dark-green image present edge to edge, not the theme's aurora gradient. It is produced by `chrome.WallpaperBackdrop` itself (the same call the window makes), rendered at 1440×860 with a translucent panel over it.
- Capture generation (temporary, removed afterwards): a `ui.NewTester` frame painting `WallpaperBackdrop(c, <goWallpaperPath>)` over `ui.RGBA(21,21,23,1)` and writing the PNG.

## 4. Live-window attempt and the blocker

Running `whaleisle-pass286.exe` with the seeded profile (dark + wallpaper + temp workspace, port 3099) opened a **549×460 px (366×307 DIP) window** and the process exited on its own after ≈ 20 s. The profile afterwards had no `last-desktop-start.json`, no `go-resume.json`, no logs and no `dsh-home`, i.e. the run failed before the harness hand-off (a startup/launcher error dialog, not a wallpaper failure). Not chased further inside the timebox; the exact dialog text was not captured. Consequence: the wallpaper evidence is the shell painter render above rather than a live window capture. Re-running with the dialog captured (or with a profile whose harness can boot in this environment) is the remaining step to turn it into a live capture; the paint path is identical, so the recipe itself is verified.

## 5. Residual

- A live seeded capture still needs a profile whose harness boots here (the run above dies before boot; the dialog text is the missing evidence).
- `wallpaperBlur` / `wallpaperPixelate` / `transparentTheme` from the same namespace are not applied by the MyGo window backdrop (config path only) — out of scope for this task, listed for the theme-owner.
- The native value stays a data URL in the settings document; the fixture should copy it verbatim into `goWallpaperPath` rather than converting it to a file, so both shells paint identical bytes.

## 6. Gates

`go test ./internal/ui/theme ./internal/ui/settings ./internal/config -count=1` → ok; `go vet` on the same three packages → clean. No source change was needed: the reading gap is documented, the fixture path already works.
