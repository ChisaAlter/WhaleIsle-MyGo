# Round 152 — chat surface pixel audit against the native harness UI

Scope: the conversation surface only — sidebar/titlebar/footer chrome, hero, composer, and the
first-run 预览版说明 notice. The reference is the *staged* harness web UI (the same client the
installed app serves), captured at 1440×920 DIP with device scale 1.5 → 2160×1380 PNG. MyGo is
the real window capture at 150% (`mygo-chat-pass275-150dpi.png`), so MyGo pixels are divided by 1.5.

## 1. Reference pipeline (working, reusable)

```powershell
# 1) fresh profile so the first-run notice is part of the reference
$home1 = Join-Path $env:TEMP ("dsh-native-ref-" + [guid]::NewGuid().ToString('N').Substring(0,8))
New-Item -ItemType Directory -Path $home1 -Force | Out-Null
$cmd = "set `"DSH_HOME=$home1`" && cd /d C:\AI\WhaleIsle-MyGo && " +
       "runtime\node\node.exe runtime\vendor\deepseek-harness\apps\cli\lib\bin.js web " +
       "--host 127.0.0.1 --port 3097 --no-open > tmp\native-web-3097.log 2> tmp\native-web-3097.err.log"
Start-Process -FilePath "cmd.exe" -ArgumentList @("/c", $cmd) -WindowStyle Hidden
# stdout: dsh web: http://127.0.0.1:3097/?token=…   (plain / is 401 — the token is required)

# 2) notice state (blocking modal on a fresh home)
runtime\node\node.exe scripts\native-web-shot.mjs <token-url> evidence/screens/native-harness-chat-modal-1440x920.png 1440 920 1.5

# 3) hero state (dismisses 预览版说明 + the sign-in prompt, then measures the live DOM)
runtime\node\node.exe scripts/native-web-shot.mjs <token-url> evidence/screens/native-harness-chat-hero-1440x920.png `
    1440 920 1.5 --eval tmp/hero-map.js --map tmp/native-hero-map.json
```

`scripts/native-web-shot.mjs` grew three options this round: `--click-text <text>` (repeatable, clicks
the leaf with that exact text, e.g. `继续`), `--settle <ms>`, and `--eval <js> --map <json>`
(awaits an async expression, writes its JSON). Notes learned the hard way:

- Direct `chrome --screenshot` hangs on this app; CDP works (that is what the script does).
- The harness server exits when its wrapper shell exits; if a capture comes back as
  `ERR_CONNECTION_REFUSED`, restart it — the token changes per start, `DSH_HOME` does not.
- Acknowledging 继续 persists into `DSH_HOME`, so the second run of the same home already shows the
  hero; a second `开始使用` (sign-in) prompt still appears and must be closed.
- Measuring the DOM (`getBoundingClientRect` + `getComputedStyle` at `deviceScaleFactor: 1`) is far
  more precise than reading pixels; every native number below is from that map, not from eyeballing.

Evidence in `docs/review-20261022/evidence/screens/`:

| file | what |
| --- | --- |
| `native-harness-chat-modal-1440x920.png` | first-run 预览版说明 over the fresh hero |
| `native-harness-chat-hero-1440x920.png` | hero without any dialog (1440×920 @1.5) |
| `mygo-chat-pass275-150dpi.png` | MyGo real window, 2160×1380 = 1440×920 @1.5 |

## 2. Measurements (DIP)

Native values are DOM rects/`computedStyle` on the live page; MyGo values come from the pass275 PNG
(pixel boxes ÷ 1.5) or from the layout code named in the row.

| # | item | native | MyGo (before) | verdict |
| --- | --- | --- | --- | --- |
| 1 | sidebar column | 280 wide, `ymWifq_sidebarCol` | 281.3 incl. hairline | equal |
| 2 | panel list rows | 插件 y=120 h=36, 自动化任务 y=160 h=36 → pitch **40**; panelList 256×76 at x=12 | pitch **36** (36-high rows, no gap) | **fixed** |
| 3 | page row insets | icon x=22, label x=46 | icon 21.3, label ≈45.3 | equal |
| 4 | collapse control | `收起侧边栏` 28×28 r8 at (240,22) | ink 13.4×12.7 at 249.3..262.7 / 30..42.7 → control ≈(242,29) 30×30 | difference ≤2 DIP, open |
| 5 | footer seat | `账号菜单` button 252×36 r12 pad 7/8, gear 16, gap 8, label 14/22 `设置与账号` | 252×**44**, pad 6, avatar 24 + person 14, label 13/500 `账号与登录` | **fixed** |
| 6 | 新会话 button | 252×38 r12 pad 8/12 at (14,70) | not measured this round | unverified |
| 7 | workspace toolbar | 4×28×28 r8 at y=210, x=144/176/208/240 | not measured this round | unverified |
| 8 | hero headline | 26/32 w500, gap 10–12, fish icon 24 | ink 25.3 tall at y=366..391 | equal (≈1–2 DIP) |
| 9 | hero badge | `大肥鱼版` (installed `hero.preview`; en `Preview`), 12/18 pill r999 pad 1/7 | `预览版` | **fixed** |
| 10 | composer card | 774.39×**114** r28 pad 8/0/0, max-width 774.4, beam 782×122 | 772×**126** at x=470.7 | width equal, height +12 → open |
| 11 | composer placeholder | `描述你想要构建的内容，/ 调用指令，@ 文件或对话` | `描述你想要构建的内容, / 调用指令, @ 文件或对话` | **fixed** |
| 12 | model seat | `deepseek-official/deepseek-flash` (no spaces), 28 high | `deepseek-official / deepseek-flash` | **fixed** |
| 13 | model seat row | 218.94×28 r8 at x=970.75 in a 1153 seat | same geometry (fill) | equal |
| 14 | send button | 34×34 r999 at (1201.7,532) | 34×34 at (1194,524) | equal (≤8 DIP) |
| 15 | access-mode chip | 126×28 r8 | same recipe | equal |
| 16 | page header band | 52 high (`_0H3Aha_header`), corner 28×28 at y=16, two 32×32 buttons at y=8 | 48 high, content 12..44 (round151 caption fix) | difference, open (see §5) |
| 17 | chat header trailing icons | 切换终端抽屉 32×32, 切换右侧栏 32×32, then desktop window buttons | same order minus the injected window buttons | equal (window buttons are injection) |
| 18 | 预览版说明 dialog | 600×344 centred at (420,288), r28, content pad 28, title 20/28 w500, copy 14/24, body mt 20, p gap 12, action mt 24, button 120×36 r12 14/22 | absent | **implemented** |

## 3. Fixed in this round

1. **Sidebar footer is the merged settings/account seat** — `internal/ui/sidebar/account.go`.
   A profile-less footer now renders what the installed `AccountMenu` trigger does with
   `data-signed-out`: 36 DIP tall, `7/8` padding, a 16 DIP settings glyph instead of the avatar, and
   the `launcherLabel` copy `设置与账号` (`Settings & account`) at 14/22. The button keeps the native
   accessible name `账号菜单` (`AccountMenu` `aria-label` = `t('menu')`), so the menu and tooltip do not
   change. New key `i18n.KeySettingsAccountMenu`. Before → after:
   `账号与登录 13/500 + 24 DIP avatar, 44 tall` → `设置与账号 14/400 + 16 DIP gear, 36 tall`.
2. **Nav row pitch 36 → 40** — `internal/ui/sidebar/sidebar.go` panel list gained `Gap(4)`. The old
   code claimed "the pass-2 measurement" said the rows butt together; the native DOM contradicts it
   (`插件` y=120, `自动化任务` y=160, both 36 high, panelList 76 = 36+4+36). The new test asserts the
   40 pitch.
3. **Hero badge copy** — `i18n.KeyChatHeroPreview` zh `预览版` → `大肥鱼版` (installed
   `ui-conversation` `hero.preview`; English stays `Preview`).
4. **Composer placeholder punctuation** — `i18n.KeyComposerPlaceholderHero` now uses the native
   full-width commas and single spaces: `描述你想要构建的内容，/ 调用指令，@ 文件或对话`.
5. **Model seat separator** — `internal/store/session/models.go` fallback label is
   `provider/model` instead of `provider / model`, matching the installed seat.
6. **First-run 预览版说明 notice** — new `internal/ui/chat/preview_notice.go` +
   `cmd/whaleisle/preview_notice_layer.go`:
   - copy and paragraph split from the installed dictionary (`welcomeTitle`, `welcomeBody`,
     `welcomeContinue`, `welcomeError`) and version `2026-09-28.1` from `onboarding-copy.ts`;
   - geometry from the installed `OnboardingModal/WelcomeNotice` CSS and the rendered dialog:
     600 wide, r28, content padding 28, title 20/28 w500, body margin-top 20, 12 between
     paragraphs, copy 14/24 secondary, action row margin-top 24 ending in a 120×36 r12 primary
     button labelled `继续`;
   - it is a blocking layer over the whole window (painted last, below the closing cover), appears
     only once the harness is `Ready`, and persists the acknowledgement in the profile config
     (`goPreviewNoticeAck`), compared for exact equality so a future notice version re-shows it;
   - a failed write keeps the notice open and shows `暂时无法保存确认状态，请重试。`.

Tests added/changed: `chat/preview_notice_test.go` (copy, version equality, 600-wide centring,
title at the padding box, 120×36 end-aligned button, single `继续` press, closed = nothing, save
error), `sidebar/account_test.go` (36-DIP merged seat + 280 width + menu), `store/session/models_label_test.go`
(slash separator), and the frame/sidebar tests that used to look for the old footer label now look up
the seat by its accessible name.

## 4. Expected injection differences (not bugs)

The native browser reference has no WhaleIsle desktop shell, so these are absent by design:

| item | why |
| --- | --- |
| 鲸鱼娘 entry (sidebar) | WhaleIsle plugin entry, absent from the pure harness page |
| 分支 `main` pill + 提交并推送 in the header | `desktop` injection in the caption trailing cluster |
| window buttons (最小化/最大化/关闭) top-right | desktop caption, not the page |
| desktop wallpaper/aurora behind the chat | shell wallpaper; the page draws its own gradient |
| custom workspace/`无工作目录` row and the `默认工作区` seed | native page had a seeded workspace, MyGo's capture had none |
| `开始使用` sign-in prompt | depends on credentials; the reference home had none |

## 5. Open / blocked

- **Composer card height +12 DIP** (native 114 vs MyGo 126 at the same 774 width). The native card is
  `padding: 8px 0 0` with a fixed 28-DIP control row; MyGo's composer adds its own vertical padding and
  a taller input row. Owner: `internal/ui/composer` — not touched this round because the composer is
  mid-refactor for other rows and the delta needs a full control-row audit, not a single number.
- **Page header band 48 vs 52** (native corner 28×28 at y=16, 32×32 buttons at y=8). MyGo deliberately
  merges the desktop caption and the conversation header into one 48-DIP band (round151). Changing it
  is a frame/caption decision, not a chat fix; recorded for the frame owner.
- **Collapse control** measured ≈2 DIP right/low of the native 28×28 at (240,22). The pixel box is
  fuzzy (icon ink, not the button); needs a headless MyGo rect probe before changing anything.
- **新会话 button and the workspace toolbar icon group** were not measured this round; the native
  numbers are recorded above (252×38 r12 pad 8/12 at (14,70); four 28×28 r8 at y=210, x=144/176/208/240)
  so the next pass can diff them directly.
- **No MyGo after-capture**: producing one needs a temp exe rebuild plus a real `start → boot → ready`
  (~70 s) and the round's timebox was spent on the pipeline and the fixes; the geometry that was fixed
  is asserted headlessly instead (chat/sidebar/frame tests above). The native reference captures are
  the durable half of this round.

## 6. Checks run

```
go vet ./internal/ui/chat ./internal/ui/sidebar ./internal/ui/i18n ./internal/config \
        ./internal/store/session ./cmd/whaleisle
go test ./internal/ui/chat ./internal/ui/sidebar ./internal/ui/frame ./internal/ui/i18n \
        ./internal/config ./internal/ui/composer ./internal/store/session ./cmd/whaleisle -count=1
```

All green. No commit, no passN binary, `runtime/vendor`, `third_party/mygo` and the installed app were
not touched.
