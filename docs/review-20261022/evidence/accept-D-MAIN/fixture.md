# D-MAIN acceptance fixture (recipe, state, and what is still blocked)

Row: **D-MAIN** — 主壳·终端关·空会话 (main shell, terminal closed, empty session).
Artifacts: `%TEMP%\whaleisle-accept-<UTC date>\D-MAIN\`, mirrored in
`docs/review-20261022/evidence/accept-D-MAIN/`.

## Fixture constants (both sides)

| constant | value | why |
| --- | --- | --- |
| client size | **1440×860 DIP** @ 150 % = 2160×1290 px | 1440×920 DIP does not fit the working area with the Windows taskbar; the bottom strip would be the taskbar, not the app (round167 finding) |
| window position | both anchored at (0,0) | a window that hangs below the taskbar cannot be copied cleanly |
| DPI | 144 (150 %), both windows | the pipeline fails loudly if they differ |
| locale | zh-CN | both apps configured `zh` |
| session | one empty session selected (hero visible) | D-MAIN is defined as 空会话 |
| terminal | closed | D-MAIN is defined as 终端关 |
| theme | **light** | the recipe below; the native app currently runs `跟随系统` and the desktop is dark, so this is the one step that is not yet automated |

## Native (installed Electron, read-only on disk)

1. Start the app (already running for the pilot: pid 15560 `Whale Isle.exe` 0.3.10).
2. Select an empty session: sidebar `新建会话` (UIA `Button` named `新建会话`, physical rect
   (221,107) 378×57 → DIP (14,70) 252×38). Verified: this produces the hero state
   （`探索未至之鲸` + `大肥鱼版` + `WhaleIsle-MyGo` + `标准模式` + empty composer).
3. Light theme: open settings, 外观 page, press `浅色`.
   - The appearance controls exist and were read through UIA once (round167 probe):
     `颜色模式` group with buttons `浅色` at physical (1052,265), `深色` at (1335,265),
     `跟随系统` at (1619,265), each ≈200×47.
   - **Automation is blocked**: `Ctrl+,` opened the settings window on the first attempt of
     round167 but not on later attempts; `账号菜单` → `设置` did not reach the menu item (the
     popup items were not in the main window's UIA tree, 0 matches over 8 retries);
     `Alt+F` + arrow walk to the `设置` item also left the tree unchanged (83 nodes, no settings).
     The three appearance buttons expose no `InvokePattern`/`SelectionItemPattern` either, so a
     UIA-only click is impossible — a real mouse click at the button's physical centre is the
     working route **once the settings window is open**.
   - Human recipe (3 clicks, ~5 s): `账号菜单` (bottom-left) → `设置` → `外观` → `浅色` → close the
     settings window. Restore afterwards with `跟随系统`.
4. Record: window class `Chrome_WidgetWin_1`, exe path, version, client rect.

## MyGo (delivered build, throwaway profile)

1. Build from the delivered tree (not a passN exe): `go build -o tmp\whaleisle-accept.exe ./cmd/whaleisle`.
2. Profile: copy the real `%APPDATA%\Deepseek-Harness-Desktop\config.json`, then set
   `host=127.0.0.1`, `port=3098`, `autoStartDesktop=true`, `goPreviewNoticeAck=2026-09-28.1`,
   `locale=zh`, and a temp `workspace`; write `welcome-shown`.
3. Theme: `theme` accepts `deepseek.dark`, `deepseek.light`, `dark`, `light`, `system`
   (`internal/ui/theme/theme.go:24-27,212-215`), but **the captured window ignored both
   `deepseek.dark` and `dark`** and rendered light. That is a real defect (task-52 owns the mapping);
   for this fixture it happens to coincide with the light target, so MyGo needs no change today —
   but a *dark* row cannot be produced until it is fixed.
4. Start with `WHALEISLE_USER_DATA=<profile>`, wait for the harness connection to settle
   (~45-50 s: the client shows `连接中…` until the socket handshake and catalogue land; the capture
   must not run before that).

## What the matched pair would contain vs what round167 measured

Round167's pair was **not** fixture-matched: native dark + real profile (workspaces
`whale-code`, `WhaleBridge`, `Whale Isle`, `Deepseek-Harness-Desktop`, `Whalelsle-MyGo`, `无工作目录`,
long session list, signed-in `Ayase`), MyGo light + one workspace, no sessions, signed out.

Consequences measured in round170 (`structure.json`):

- 64 named native controls vs 44 MyGo; only 15 match by (type, name); 48 native-only, 29 MyGo-only.
- Of the 15 matches only 4 are within the 1-DIP tolerance; the 11 others are dominated by
  data-driven layout (the session tree shifts the sidebar) and by state (the signed-in footer is
  44 DIP, the signed-out merged seat 36 DIP).
- `标准模式` sits 472 DIP to the right in MyGo because the hero block centres in a column without the
  native session list — a fixture consequence, not a renderer delta.

Until the fixture above is applied on both sides, a pixel comparison measures the fixture, not MyGo.
