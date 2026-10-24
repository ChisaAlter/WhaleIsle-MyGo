# Round183 · S03 按键实测 / G10 glyph 放大 / G03 菜单栏 三项收尾

结论速览：**C 完成并得出结论（无差异）；A 未能在本轮跑通（附精确原因与下一步）；B 未做（时间被 A 占用，附参数化配方）。** 没有为交差改动任何代码；`pass285` 与原生窗口均未被修改。

## A) S03 Ctrl+Shift+I 实按验证 —— **未完成（不是"已通过"）**

做了什么：
1. 用 `tmp/g65.ps1` 起一个 throwaway profile（真实 config + 临时 workspace + `mygoDesktopPet.enabled=true` + welcome marker + 默认 `runtime/vendor`），等 20 s，确认宠物 webview 窗口存在（`Whale Isle companion` 360×657），主窗 2160×1380。
2. `ShowWindow(SW_RESTORE)` + `SetForegroundWindow(pet)`，用 `keybd_event` 发送 Ctrl↓ Shift↓ I↓ I↑ Shift↑ Ctrl↑，间隔 4 s 后枚举**所有可见有标题的顶层窗口**做前后差集，再按一次做第二次差集。

结果：
- 第一次运行用的是 `whaleisle-pass285.exe`（3:56 构建）——**它早于本轮 S03 接线**（`Page.OnBeforeInput` 与 host 接线是 4:20 左右才落的），所以「没有新窗口」是预期的，不能算反证。
- 用当前树临时构建 `tmp/whaleisle-g65.exe` 重跑时，脚本在开头清理旧 profile 时被 `dsh-usage-panel.sqlite` 占用卡住（并且清理 node 子进程时打断了 job），**没有产出新结果**（读到的 JSON 仍是 pass285 那次）。因此本轮**没有拿到"按下真的打开 inspector"的证据**。

下一步（精确、约 10 分钟）：
1. 每次运行用**新的** profile 目录（不要复用 `%TEMP%\wl-g65`），避免 sqlite 占用；
2. 起 `tmp/whaleisle-g65.exe` 后先 `SetForegroundWindow` 主窗、再用 UIA 把焦点放进宠物页（或直接对宠物窗口 `SetFocus`），确认前台窗口确实是 pet；
3. 在 `installDevToolsShortcut` 里临时加一行 `petDebug`/`fmt.Println` 打点到文件，证明回调确实收到 `KeyInput{i, ctrl, shift, down}`——若收不到，说明合成按键没有走 WebView2 `AcceleratorKeyPressed`（需要真实键盘或 `SendInput` + 焦点），而不是 hook 逻辑错误；
4. 打开后用 `EnumWindows` 找 title 含 `DevTools` 的窗口（属于 `msedgewebview2.exe` 进程，不是 app pid）并截图，再按一次确认关闭。

## B) G10 面板开关键 glyph（14 vs 16/18） —— **未做**

原因：时间盒被 A 的两次运行占用。配方（下一轮 10 分钟可完成）：
1. 原生：`CopyFromScreen` 抓 `x = 200+2268-150`、`y = 2`、48×48 物理像素（32 DIP）的终端开关，再 4× 最近邻放大；
2. MyGo：`pass285`（或新构建）同样抓 `captionPanelToggle` 的 48×48，4× 放大；
3. 对放大图做二值化扫描内层 glyph 的包围盒（去掉 1px 边框），除以 1.5 得 DIP；只有两次测得**差 ≥2 DIP 且边界清晰**时才改 `workspace_trailing.go` 的 `.Size(14, 14)`，并补一条「glyph 尺寸 = N」的断言测试。

## C) G03 菜单栏 —— **完成，无差异**

Win32 实测（per-monitor-v2，150%，dpi 144）：

| | 原生 `Chrome_WidgetWin_1`（pid 15560） | MyGo 主窗 `MyGoWindow`（pass285） |
| --- | --- | --- |
| `GetMenu(hwnd)` | **0**（无原生菜单） | **0** |
| `GetMenuBarInfo(hwnd, OBJID_MENU, 0)` | **false** | **false** |
| 窗口矩形 | 2268×1366 | 2160×1380 |
| 客户区 | **2268×1366**（= 窗口，无边框/无菜单占位） | 2160×1379（≈窗口，1 px 舍入） |
| 菜单占用的客户区高度 | 0 | 0 |

结论：两侧都是**自绘标题栏、没有 Win32 菜单栏**，`AutoHideMenuBar` 也没有给菜单预留任何客户区高度（客户区高度只差 1 物理像素，属于 150% 下的取整，不是菜单带）。G03「菜单栏是否下压/缩小客户区」在两侧都是**否**，该行可判为**无差异（已核实）**，无需代码改动。

## 证据与清理

- 结果 JSON：`%TEMP%\wl-g65-result.json`（记录 windows、MyGo 菜单/客户区，A 的两次差集为空）。
- 临时脚本：`tmp/g65.ps1`（gitignored），临时构建 `tmp/whaleisle-g65.exe`。
- 已 `Stop-Process` 所有 `whaleisle*`，删除 `%TEMP%\wl-g65`，清理 `%TEMP%\mygo-*.sock`；原生窗口只做只读查询，未移动/未关闭。

## 残留

1. A 需要在干净 profile + 明确前台焦点下重跑（步骤已写在上面），并加一次落盘打点区分「回调没收到」与「收不到合成按键」。
2. B 的 glyph 尺寸问题仍开放：当前 MyGo 是 14 DIP，未经 4× 放大对比前不要改。
3. C 已关闭：两侧都没有原生菜单栏，客户区不被下压。
